package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shdmitri/booking-service/pkg/security"
)

func TestAuthMiddlewareRejectsInvalidAuthorizationHeaders(t *testing.T) {
	middleware := &AuthMiddleware{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		JWTAccessSecret: []byte("access-secret"),
	}
	tests := []struct {
		name   string
		header string
	}{
		{name: "missing header"},
		{name: "missing bearer scheme", header: "token"},
		{name: "wrong scheme", header: "Basic token"},
		{name: "empty token", header: "Bearer   "},
		{name: "invalid token", header: "Bearer not-a-token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			})
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				request.Header.Set("Authorization", tt.header)
			}
			response := httptest.NewRecorder()

			middleware.Auth(next).ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized || called {
				t.Fatalf("status=%d next called=%v, want 401 and no call", response.Code, called)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
			var body struct {
				Error string `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error == "" {
				t.Fatalf("invalid error response: %q (error %v)", response.Body.String(), err)
			}
		})
	}
}

func TestAuthMiddlewareAddsVerifiedClaimsToContext(t *testing.T) {
	const secret = "unit-test-secret"
	token, err := security.CreateAccessToken("user-1", "admin", []byte(secret))
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}

	middleware := &AuthMiddleware{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		JWTAccessSecret: []byte(secret),
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "bEaReR "+token)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		userID, hasUserID := UserIDFromContext(r.Context())
		role, hasRole := RoleFromContext(r.Context())
		if !hasUserID || userID != "user-1" || !hasRole || role != "admin" {
			t.Errorf("context claims = (%q, %q), presence = (%v, %v)", userID, role, hasUserID, hasRole)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	middleware.Auth(next).ServeHTTP(response, request)

	if !called || response.Code != http.StatusNoContent {
		t.Fatalf("next called=%v status=%d, want true and 204", called, response.Code)
	}
}

func TestContextClaimHelpersReportMissingClaims(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := UserIDFromContext(request.Context()); ok {
		t.Fatal("UserIDFromContext() returned a value for a context without claims")
	}
	if _, ok := RoleFromContext(request.Context()); ok {
		t.Fatal("RoleFromContext() returned a value for a context without claims")
	}
}
