// cmd/api-gateway/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tradingbothub/platform/internal/config"
	"github.com/tradingbothub/platform/internal/gateway"
	"github.com/tradingbothub/platform/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize gateway
	gw, err := gateway.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize gateway: %v", err)
	}

	// Setup Gin router
	router := setupRouter(gw)

	// Create HTTP server
	srv := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Gateway listening on %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	gw.Close()
	log.Println("API Gateway stopped")
}

func setupRouter(gw *gateway.Gateway) *gin.Engine {
	// Set Gin mode
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RateLimit())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"service":   "api-gateway",
		})
	})

	// API versioning
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", gw.Register)
			auth.POST("/login", gw.Login)
			auth.POST("/refresh", gw.RefreshToken)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(gw.AuthClient))
		{
			// User routes
			user := protected.Group("/user")
			{
				user.GET("/profile", gw.GetProfile)
				user.PUT("/profile", gw.UpdateProfile)
				user.POST("/change-password", gw.ChangePassword)
			}

			// Bot routes
			bots := protected.Group("/bots")
			{
				bots.GET("", gw.ListBots)
				bots.POST("", gw.CreateBot)
				bots.GET("/:id", gw.GetBot)
				bots.PUT("/:id", gw.UpdateBot)
				bots.DELETE("/:id", gw.DeleteBot)
				bots.POST("/:id/start", gw.StartBot)
				bots.POST("/:id/stop", gw.StopBot)
				bots.GET("/:id/logs", gw.GetBotLogs)
			}

			// Strategy routes
			strategies := protected.Group("/strategies")
			{
				strategies.GET("", gw.ListStrategies)
				strategies.POST("", gw.CreateStrategy)
				strategies.GET("/:id", gw.GetStrategy)
				strategies.PUT("/:id", gw.UpdateStrategy)
				strategies.DELETE("/:id", gw.DeleteStrategy)
				strategies.POST("/:id/backtest", gw.BacktestStrategy)
			}

			// Market data routes
			market := protected.Group("/market")
			{
				market.GET("/symbols", gw.GetSymbols)
				market.GET("/ticker/:symbol", gw.GetTicker)
				market.GET("/candles/:symbol", gw.GetCandles)
				market.GET("/orderbook/:symbol", gw.GetOrderBook)
			}

			// Portfolio routes
			portfolio := protected.Group("/portfolio")
			{
				portfolio.GET("", gw.GetPortfolio)
				portfolio.GET("/positions", gw.GetPositions)
				portfolio.GET("/orders", gw.GetOrders)
				portfolio.GET("/trades", gw.GetTrades)
				portfolio.GET("/performance", gw.GetPerformance)
			}
		}
	}

	return router
}

// internal/gateway/gateway.go
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tradingbothub/platform/api/proto/auth"
	"github.com/tradingbothub/platform/internal/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Gateway struct {
	config     *config.Config
	AuthClient authpb.AuthServiceClient
	authConn   *grpc.ClientConn
}

func New(cfg *config.Config) (*Gateway, error) {
	gw := &Gateway{
		config: cfg,
	}

	// Connect to Auth Service
	authConn, err := grpc.Dial(
		"localhost"+cfg.Auth.Port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	gw.authConn = authConn
	gw.AuthClient = authpb.NewAuthServiceClient(authConn)

	return gw, nil
}

func (gw *Gateway) Close() {
	if gw.authConn != nil {
		gw.authConn.Close()
	}
}

// Auth handlers
func (gw *Gateway) Register(c *gin.Context) {
	var req authpb.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := gw.AuthClient.Register(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) Login(c *gin.Context) {
	var req authpb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := gw.AuthClient.Login(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) RefreshToken(c *gin.Context) {
	var req authpb.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := gw.AuthClient.RefreshToken(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// User handlers (placeholder implementations)
func (gw *Gateway) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"message": "Get profile - implementation needed",
	})
}

func (gw *Gateway) UpdateProfile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update profile - implementation needed"})
}

func (gw *Gateway) ChangePassword(c *gin.Context) {
	var req authpb.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get token from header
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		req.AccessToken = token[7:]
	}

	resp, err := gw.AuthClient.ChangePassword(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Bot handlers (placeholder implementations)
func (gw *Gateway) ListBots(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List bots - implementation needed"})
}

func (gw *Gateway) CreateBot(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Create bot - implementation needed"})
}

func (gw *Gateway) GetBot(c *gin.Context) {
	botID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"bot_id":  botID,
		"message": "Get bot - implementation needed",
	})
}

func (gw *Gateway) UpdateBot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update bot - implementation needed"})
}

func (gw *Gateway) DeleteBot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete bot - implementation needed"})
}

func (gw *Gateway) StartBot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Start bot - implementation needed"})
}

func (gw *Gateway) StopBot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Stop bot - implementation needed"})
}

func (gw *Gateway) GetBotLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get bot logs - implementation needed"})
}

// Strategy handlers (placeholder implementations)
func (gw *Gateway) ListStrategies(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "List strategies - implementation needed"})
}

func (gw *Gateway) CreateStrategy(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "Create strategy - implementation needed"})
}

func (gw *Gateway) GetStrategy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get strategy - implementation needed"})
}

func (gw *Gateway) UpdateStrategy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Update strategy - implementation needed"})
}

func (gw *Gateway) DeleteStrategy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Delete strategy - implementation needed"})
}

func (gw *Gateway) BacktestStrategy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Backtest strategy - implementation needed"})
}

// Market data handlers (placeholder implementations)
func (gw *Gateway) GetSymbols(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get symbols - implementation needed"})
}

func (gw *Gateway) GetTicker(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"message": "Get ticker - implementation needed",
	})
}

func (gw *Gateway) GetCandles(c *gin.Context) {
	symbol := c.Param("symbol")
	interval := c.DefaultQuery("interval", "1h")
	limit := c.DefaultQuery("limit", "100")
	
	limitInt, _ := strconv.Atoi(limit)
	
	c.JSON(http.StatusOK, gin.H{
		"symbol":   symbol,
		"interval": interval,
		"limit":    limitInt,
		"message":  "Get candles - implementation needed",
	})
}

func (gw *Gateway) GetOrderBook(c *gin.Context) {
	symbol := c.Param("symbol")
	c.JSON(http.StatusOK, gin.H{
		"symbol":  symbol,
		"message": "Get order book - implementation needed",
	})
}

// Portfolio handlers (placeholder implementations)
func (gw *Gateway) GetPortfolio(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get portfolio - implementation needed"})
}

func (gw *Gateway) GetPositions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get positions - implementation needed"})
}

func (gw *Gateway) GetOrders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get orders - implementation needed"})
}

func (gw *Gateway) GetTrades(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get trades - implementation needed"})
}

func (gw *Gateway) GetPerformance(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Get performance - implementation needed"})
}