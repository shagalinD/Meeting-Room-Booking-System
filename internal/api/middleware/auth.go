package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"github.com/shdmitri/booking-service/pkg/security"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	roleContextKey   contextKey = "role"
)

type AuthMiddleware struct {
	Logger *slog.Logger
	JWTAccessSecret  []byte
	JWTRefreshSecret []byte
}

func (m *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if authHeader == "" {
			writeErrorResponse(w, m.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			writeErrorResponse(w, m.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "invalid authorization header format"})
			return
		}

		claims, err := security.ParseToken(parts[1], m.JWTAccessSecret)
		if err != nil {
			writeErrorResponse(w, m.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "invalid or expired token"})
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