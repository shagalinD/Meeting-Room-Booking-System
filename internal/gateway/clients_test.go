package gateway

import (
	"errors"
	"testing"

	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToApplicationError(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{name: "invalid argument", code: codes.InvalidArgument, want: apperrors.ValidationError},
		{name: "unauthenticated", code: codes.Unauthenticated, want: apperrors.InvalidCredentialsError},
		{name: "permission denied", code: codes.PermissionDenied, want: apperrors.UnauthorizedError},
		{name: "not found", code: codes.NotFound, want: apperrors.NotFoundError},
		{name: "already exists", code: codes.AlreadyExists, want: apperrors.InternalServerError},
		{name: "unavailable", code: codes.Unavailable, want: apperrors.ServiceUnavailableError},
		{name: "deadline exceeded", code: codes.DeadlineExceeded, want: apperrors.ServiceUnavailableError},
		{name: "resource exhausted", code: codes.ResourceExhausted, want: apperrors.ServiceUnavailableError},
		{name: "internal", code: codes.Internal, want: apperrors.InternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := toApplicationError(status.Error(tt.code, "upstream detail"))
			var appErr *apperrors.Errors
			if !errors.As(err, &appErr) {
				t.Fatalf("error type = %T, want *apperrors.Errors", err)
			}
			if appErr.Code != tt.want {
				t.Fatalf("error code = %d, want %d", appErr.Code, tt.want)
			}
		})
	}
}

func TestClientsRejectMissingBearerToken(t *testing.T) {
	if _, err := NewBookingClient(nil, 0).Create(t.Context(), nil); errorCode(err) != apperrors.UnauthorizedError {
		t.Fatalf("BookingClient.Create error = %v, want unauthorized", err)
	}
	if _, err := NewRoomClient(nil, 0).Create(t.Context(), nil); errorCode(err) != apperrors.UnauthorizedError {
		t.Fatalf("RoomClient.Create error = %v, want unauthorized", err)
	}
}

func errorCode(err error) int {
	var appErr *apperrors.Errors
	if !errors.As(err, &appErr) {
		return -1
	}
	return appErr.Code
}
