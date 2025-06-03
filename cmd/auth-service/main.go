// cmd/auth-service/main.go
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	authpb "github.com/tradingbothub/platform/api/proto/auth"
	"github.com/tradingbothub/platform/internal/auth"
	"github.com/tradingbothub/platform/internal/config"
	"github.com/tradingbothub/platform/internal/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.Connect(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize auth service
	authRepo := auth.NewRepository(db)
	tokenService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.ExpirationTime)
	authService := auth.NewService(authRepo, tokenService)

	// Create gRPC server
	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, auth.NewGRPCServer(authService))

	// Enable reflection for development
	reflection.Register(s)

	// Start server
	lis, err := net.Listen("tcp", cfg.Auth.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Auth service listening on %s", cfg.Auth.Port)

	// Graceful shutdown
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down auth service...")
	s.GracefulStop()
}
