// cmd/auth-service/main.go
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/tradingbothub/platform/internal/auth"
	"github.com/tradingbothub/platform/internal/config"
	"github.com/tradingbothub/platform/internal/database"
	"github.com/tradingbothub/platform/api/proto/auth"
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

// internal/auth/service.go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	repo         Repository
	tokenService TokenService
}

func NewService(repo Repository, tokenService TokenService) *Service {
	return &Service{
		repo:         repo,
		tokenService: tokenService,
	}
}

func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	// Check if user exists
	if _, err := s.repo.GetByEmail(ctx, req.Email); err == nil {
		return nil, ErrUserExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		Username:     req.Username,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    3600, // 1 hour
	}, nil
}

func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	user.LastLoginAt = time.Now()
	s.repo.Update(ctx, user)

	// Generate tokens
	accessToken, err := s.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    3600,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	userID, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Generate new access token
	accessToken, err := s.tokenService.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken: accessToken,
		User:        user,
		ExpiresIn:   3600,
	}, nil
}

func (s *Service) ValidateToken(ctx context.Context, token string) (*User, error) {
	// Validate token
	userID, err := s.tokenService.ValidateAccessToken(token)
	if err != nil {
		return nil, err
	}

	// Get user
	return s.repo.GetByID(ctx, userID)
}

// internal/auth/models.go
package auth

import "time"

type User struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"uniqueIndex"`
	Username     string    `json:"username" gorm:"uniqueIndex"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	PasswordHash string    `json:"-"`
	Avatar       string    `json:"avatar"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	User         *User  `json:"user"`
	ExpiresIn    int64  `json:"expires_in"`
}

// internal/auth/repository.go
package auth

import (
	"context"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) GetByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *repository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

// internal/auth/jwt.go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type TokenService interface {
	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken(userID string) (string, error)
	ValidateAccessToken(token string) (string, error)
	ValidateRefreshToken(token string) (string, error)
}

type jwtService struct {
	secret           []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

type Claims struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

func NewJWTService(secret string, accessTokenTTL time.Duration) TokenService {
	return &jwtService{
		secret:          []byte(secret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: 24 * 7 * time.Hour, // 7 days
	}
}

func (j *jwtService) GenerateAccessToken(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *jwtService) GenerateRefreshToken(userID string) (string, error) {
	claims := Claims{
		UserID: userID,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

func (j *jwtService) ValidateAccessToken(tokenString string) (string, error) {
	return j.validateToken(tokenString, "access")
}

func (j *jwtService) ValidateRefreshToken(tokenString string) (string, error) {
	return j.validateToken(tokenString, "refresh")
}

func (j *jwtService) validateToken(tokenString, tokenType string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secret, nil
	})

	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	if claims.Type != tokenType {
		return "", ErrInvalidToken
	}

	if claims.ExpiresAt.Before(time.Now()) {
		return "", ErrExpiredToken
	}

	return claims.UserID, nil
}
