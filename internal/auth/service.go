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
		IsActive:     true,
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
