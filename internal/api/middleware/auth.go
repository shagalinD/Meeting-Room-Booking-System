package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/shdmitri/booking-service/internal/domain"
	"github.com/shdmitri/booking-service/pkg/security"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	roleContextKey   contextKey = "role"
)

type Middleware struct {
	service domain.AuthService
	JWTAccessSecret  []byte
	JWTRefreshSecret []byte
}

func (m *Middleware) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}

		claims, err := security.ParseToken(parts[1], m.JWTAccessSecret)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		ctx = context.WithValue(ctx, roleContextKey, claims.Role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

func RoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleContextKey).(string)
	return role, ok
}