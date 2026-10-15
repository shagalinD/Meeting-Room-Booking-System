package grpc

import (
	"context"
	"errors"
	"testing"

	authpb "github.com/shagalinD/booking-service-proto/gen/go"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authServiceStub struct {
	registerArgs []string
	loginArgs    []string
	refreshToken string
	accessToken  string
	newRefresh   string
	err          error
}

func (s *authServiceStub) Register(_ context.Context, email, password, firstName, lastName, role string) (string, string, error) {
	s.registerArgs = []string{email, password, firstName, lastName, role}
	return s.accessToken, s.newRefresh, s.err
}

func (s *authServiceStub) Login(_ context.Context, email, password string) (string, string, error) {
	s.loginArgs = []string{email, password}
	return s.accessToken, s.newRefresh, s.err
}

func (s *authServiceStub) RefreshTokens(_ context.Context, refreshToken string) (string, string, error) {
	s.refreshToken = refreshToken
	return s.accessToken, s.newRefresh, s.err
}

func TestAuthRegisterAlwaysAssignsUserRole(t *testing.T) {
	service := &authServiceStub{accessToken: "access", newRefresh: "refresh"}
	server := NewAuthServer(service)

	response, err := server.Register(context.Background(), &authpb.RegisterRequest{
		Email:     "user@example.com",
		Password:  "password",
		FirstName: "Ada",
		LastName:  "Lovelace",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if len(service.registerArgs) != 5 || service.registerArgs[4] != "user" {
		t.Fatalf("Register() arguments = %v, want user role", service.registerArgs)
	}
	if response.GetAccessToken() != "access" || response.GetRefreshToken() != "refresh" {
		t.Fatalf("Register() response = %+v", response)
	}
}

func TestAuthLoginAndRefresh(t *testing.T) {
	service := &authServiceStub{accessToken: "new-access", newRefresh: "new-refresh"}
	server := NewAuthServer(service)

	loginResponse, err := server.Login(context.Background(), &authpb.LoginRequest{
		Email:    "user@example.com",
		Password: "password",
	})
	if err != nil || len(service.loginArgs) != 2 || service.loginArgs[0] != "user@example.com" {
		t.Fatalf("Login() response=%+v args=%v error=%v", loginResponse, service.loginArgs, err)
	}

	refreshResponse, err := server.RefreshTokens(context.Background(), &authpb.RefreshTokensRequest{
		RefreshToken: "old-refresh",
	})
	if err != nil || service.refreshToken != "old-refresh" {
		t.Fatalf("RefreshTokens() response=%+v token=%q error=%v", refreshResponse, service.refreshToken, err)
	}
	if refreshResponse.GetAccessToken() != "new-access" || refreshResponse.GetRefreshToken() != "new-refresh" {
		t.Fatalf("RefreshTokens() response = %+v", refreshResponse)
	}
}

func TestAuthRejectsMissingRequestsAndMapsErrors(t *testing.T) {
	service := &authServiceStub{}
	server := NewAuthServer(service)

	if _, err := server.Register(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Register(nil) code = %s, want InvalidArgument", status.Code(err))
	}
	if _, err := server.Login(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Login(nil) code = %s, want InvalidArgument", status.Code(err))
	}
	for _, req := range []*authpb.RefreshTokensRequest{nil, {}, {RefreshToken: "  "}} {
		if _, err := server.RefreshTokens(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("RefreshTokens(%+v) code = %s, want InvalidArgument", req, status.Code(err))
		}
	}

	service.err = &apperrors.Errors{Code: apperrors.InvalidCredentialsError, Message: "Invalid credentials"}
	if _, err := server.Login(context.Background(), &authpb.LoginRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("Login() code = %s, want Unauthenticated", status.Code(err))
	}

	service.err = &apperrors.Errors{Code: apperrors.ValidationError, Message: "email is required"}
	if _, err := server.Register(context.Background(), &authpb.RegisterRequest{}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("Register() code = %s, want InvalidArgument", status.Code(err))
	}

	service.err = &apperrors.Errors{Code: apperrors.InternalServerError, Message: "database password should not be returned"}
	if _, err := server.Login(context.Background(), &authpb.LoginRequest{}); status.Code(err) != codes.Internal ||
		status.Convert(err).Message() != "internal server error" {
		t.Fatalf("Login() leaked internal error: %v", err)
	}

	service.err = errors.New("unexpected error")
	if _, err := server.RefreshTokens(context.Background(), &authpb.RefreshTokensRequest{RefreshToken: "refresh"}); status.Code(err) != codes.Internal {
		t.Fatalf("RefreshTokens() code = %s, want Internal", status.Code(err))
	}
}
