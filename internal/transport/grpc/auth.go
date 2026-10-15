package grpc

import (
	"context"
	"strings"

	"github.com/shdmitri/booking-service/pkg/security"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func authenticatedClaims(ctx context.Context, jwtSecret []byte) (*security.CustomClaims, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	parts := strings.SplitN(strings.TrimSpace(values[0]), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization metadata")
	}

	claims, err := security.ParseToken(strings.TrimSpace(parts[1]), jwtSecret)
	if err != nil || strings.TrimSpace(claims.UserID) == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid or expired access token")
	}
	return claims, nil
}
