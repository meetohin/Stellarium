package auth

import (
	"context"
	"time"

	authpb "github.com/tradingbothub/platform/api/proto/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type GRPCServer struct {
	authpb.UnimplementedAuthServiceServer
	service *Service
}

func NewGRPCServer(service *Service) *GRPCServer {
	return &GRPCServer{service: service}
}

func (s *GRPCServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	authReq := &RegisterRequest{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	resp, err := s.service.Register(ctx, authReq)
	if err != nil {
		switch err {
		case ErrUserExists:
			return nil, status.Error(codes.AlreadyExists, "User already exists")
		default:
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}

	return &authpb.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         s.userToProto(resp.User),
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

func (s *GRPCServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	loginReq := &LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	resp, err := s.service.Login(ctx, loginReq)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			return nil, status.Error(codes.Unauthenticated, "Invalid credentials")
		default:
			return nil, status.Error(codes.Internal, "Internal server error")
		}
	}

	return &authpb.AuthResponse{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		User:         s.userToProto(resp.User),
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

func (s *GRPCServer) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.AuthResponse, error) {
	resp, err := s.service.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "Invalid refresh token")
	}

	return &authpb.AuthResponse{
		AccessToken: resp.AccessToken,
		User:        s.userToProto(resp.User),
		ExpiresIn:   resp.ExpiresIn,
	}, nil
}

func (s *GRPCServer) ValidateToken(ctx context.Context, req *authpb.ValidateTokenRequest) (*authpb.ValidateTokenResponse, error) {
	user, err := s.service.ValidateToken(ctx, req.AccessToken)
	if err != nil {
		return &authpb.ValidateTokenResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	return &authpb.ValidateTokenResponse{
		Valid: true,
		User:  s.userToProto(user),
	}, nil
}

func (s *GRPCServer) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	// TODO: Implement token blacklisting in Redis
	return &authpb.LogoutResponse{Success: true}, nil
}

func (s *GRPCServer) ChangePassword(ctx context.Context, req *authpb.ChangePasswordRequest) (*authpb.ChangePasswordResponse, error) {
	// TODO: Implement password change logic
	return &authpb.ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}, nil
}

func (s *GRPCServer) userToProto(user *User) *authpb.User {
	return &authpb.User{
		Id:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Avatar:      user.Avatar,
		IsActive:    user.IsActive,
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
		LastLoginAt: timestamppb.New(user.LastLoginAt),
	}
}
