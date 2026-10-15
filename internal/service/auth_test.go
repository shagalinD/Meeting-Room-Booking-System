package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"github.com/shdmitri/booking-service/pkg/security"
)

type userRepositoryStub struct {
	created     *domain.UserCommand
	user        *domain.User
	createID    string
	err         error
	createCalls int
	getCalls    int
}

func (r *userRepositoryStub) Create(_ context.Context, user *domain.UserCommand) (string, error) {
	r.createCalls++
	r.created = user
	if r.err != nil {
		return "", r.err
	}
	return r.createID, nil
}

func (r *userRepositoryStub) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	r.getCalls++
	return r.user, r.err
}

func (r *userRepositoryStub) GetByID(context.Context, string) (*domain.User, error) {
	panic("unexpected GetByID call")
}

func TestAuthServiceRegisterValidationPreventsRepositoryCall(t *testing.T) {
	service := &AuthService{Repo: &userRepositoryStub{}}
	tests := []struct {
		name     string
		email    string
		password string
		first    string
		last     string
		role     string
	}{
		{name: "missing email", email: "", password: "password123", first: "Ada", last: "Lovelace", role: "user"},
		{name: "invalid email", email: "not-an-email", password: "password123", first: "Ada", last: "Lovelace", role: "user"},
		{name: "short password", email: "ada@example.com", password: "short", first: "Ada", last: "Lovelace", role: "user"},
		{name: "missing first name", email: "ada@example.com", password: "password123", first: " ", last: "Lovelace", role: "user"},
		{name: "missing last name", email: "ada@example.com", password: "password123", first: "Ada", last: " ", role: "user"},
		{name: "missing role", email: "ada@example.com", password: "password123", first: "Ada", last: "Lovelace", role: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &userRepositoryStub{}
			service.Repo = repo
			access, refresh, err := service.Register(context.Background(), tt.email, tt.password, tt.first, tt.last, tt.role)
			var appErr *apperrors.Errors
			if access != "" || refresh != "" || !errors.As(err, &appErr) || appErr.Code != apperrors.ValidationError {
				t.Fatalf("Register() tokens=(%q,%q) error=%v, want validation error", access, refresh, err)
			}
			if repo.createCalls != 0 {
				t.Fatalf("repository called %d times for invalid input", repo.createCalls)
			}
		})
	}
}

func TestAuthServiceRegisterLoginAndRefresh(t *testing.T) {
	ctx := context.Background()
	const accessSecret = "access-secret-for-unit-tests"
	const refreshSecret = "refresh-secret-for-unit-tests"
	repo := &userRepositoryStub{createID: "user-1"}
	service := &AuthService{
		Repo:             repo,
		JWTAccessSecret:  []byte(accessSecret),
		JWTRefreshSecret: []byte(refreshSecret),
	}

	access, refresh, err := service.Register(ctx, "ada@example.com", "password123", "Ada", "Lovelace", "user")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if repo.createCalls != 1 || repo.created.Email != "ada@example.com" || repo.created.Role != "user" {
		t.Fatalf("Register() repository input = %+v; calls = %d", repo.created, repo.createCalls)
	}
	if repo.created.PasswordHash == "" || repo.created.PasswordHash == "password123" {
		t.Fatal("Register() did not store a password hash")
	}
	assertTokenClaims(t, access, []byte(accessSecret), "user-1", "user")
	assertTokenClaims(t, refresh, []byte(refreshSecret), "user-1", "user")

	repo.user = &domain.User{ID: "user-1", Role: "user", PasswordHash: repo.created.PasswordHash}
	loginAccess, loginRefresh, err := service.Login(ctx, "ada@example.com", "password123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	assertTokenClaims(t, loginAccess, []byte(accessSecret), "user-1", "user")
	assertTokenClaims(t, loginRefresh, []byte(refreshSecret), "user-1", "user")

	_, _, err = service.Login(ctx, "ada@example.com", "wrong-password")
	var appErr *apperrors.Errors
	if !errors.As(err, &appErr) || appErr.Code != apperrors.InvalidCredentialsError {
		t.Fatalf("Login() wrong password error = %v, want invalid credentials", err)
	}

	newAccess, newRefresh, err := service.RefreshTokens(ctx, refresh)
	if err != nil {
		t.Fatalf("RefreshTokens() error = %v", err)
	}
	assertTokenClaims(t, newAccess, []byte(accessSecret), "user-1", "user")
	assertTokenClaims(t, newRefresh, []byte(refreshSecret), "user-1", "user")

	_, _, err = service.RefreshTokens(ctx, "not-a-token")
	if !errors.As(err, &appErr) || appErr.Code != apperrors.UnauthorizedError {
		t.Fatalf("RefreshTokens() invalid token error = %v, want unauthorized", err)
	}
}

func TestAuthServiceRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("repository unavailable")

	t.Run("register wraps unknown repository error", func(t *testing.T) {
		repo := &userRepositoryStub{err: repoErr}
		service := &AuthService{Repo: repo}
		_, _, err := service.Register(ctx, "ada@example.com", "password123", "Ada", "Lovelace", "user")
		var appErr *apperrors.Errors
		if !errors.As(err, &appErr) || appErr.Code != apperrors.InternalServerError || !errors.Is(err, repoErr) {
			t.Fatalf("Register() error = %v, want wrapped internal error", err)
		}
	})

	t.Run("preserves application error", func(t *testing.T) {
		wantErr := &apperrors.Errors{Code: apperrors.NotFoundError, Message: "user not found"}
		repo := &userRepositoryStub{err: wantErr}
		service := &AuthService{Repo: repo}
		_, _, err := service.Login(ctx, "ada@example.com", "password123")
		if err != wantErr {
			t.Fatalf("Login() error = %v, want original application error %v", err, wantErr)
		}
	})

	t.Run("login wraps unknown repository error", func(t *testing.T) {
		repo := &userRepositoryStub{err: repoErr}
		service := &AuthService{Repo: repo}
		_, _, err := service.Login(ctx, "ada@example.com", "password123")
		var appErr *apperrors.Errors
		if !errors.As(err, &appErr) || appErr.Code != apperrors.InternalServerError || !errors.Is(err, repoErr) {
			t.Fatalf("Login() error = %v, want wrapped internal error", err)
		}
	})
}

func assertTokenClaims(t *testing.T, token string, secret []byte, userID, role string) {
	t.Helper()
	claims, err := security.ParseToken(token, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != userID || claims.Role != role {
		t.Fatalf("token claims = (%q, %q), want (%q, %q)", claims.UserID, claims.Role, userID, role)
	}
}
