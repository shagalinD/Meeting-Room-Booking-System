package grpc

import (
	"context"
	"strings"

	authpb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ authpb.AuthServiceServer = (*AuthServer)(nil)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
	service domain.AuthService
}

func NewAuthServer(service domain.AuthService) *AuthServer {
	return &AuthServer{service: service}
}

func (s *AuthServer) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.AuthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "registration details are required")
	}

	accessToken, refreshToken, err := s.service.Register(
		ctx,
		req.GetEmail(),
		req.GetPassword(),
		req.GetFirstName(),
		req.GetLastName(),
		"user",
	)
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.AuthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "login credentials are required")
	}

	accessToken, refreshToken, err := s.service.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *AuthServer) RefreshTokens(ctx context.Context, req *authpb.RefreshTokensRequest) (*authpb.AuthResponse, error) {
	if req == nil || strings.TrimSpace(req.GetRefreshToken()) == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	accessToken, refreshToken, err := s.service.RefreshTokens(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toStatusError(err)
	}

	return &authpb.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
