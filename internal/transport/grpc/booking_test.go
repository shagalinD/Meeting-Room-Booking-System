package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	bookingpb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"github.com/shdmitri/booking-service/pkg/security"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type bookingServiceStub struct {
	createdID string
	created   *domain.Booking
	bookings  []*domain.Booking
	userID    string
	err       error
}

func (s *bookingServiceStub) Create(_ context.Context, booking *domain.Booking) (string, error) {
	s.created = booking
	return s.createdID, s.err
}

func (s *bookingServiceStub) GetByUserId(_ context.Context, userID string) ([]*domain.Booking, error) {
	s.userID = userID
	return s.bookings, s.err
}

func TestCreateBookingUsesAuthenticatedUser(t *testing.T) {
	service := &bookingServiceStub{createdID: "booking-id"}
	server, ctx := bookingServerWithToken(t, service, "user-id", "user")
	start := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)

	response, err := server.CreateBooking(ctx, &bookingpb.CreateBookingRequest{
		RoomId:    "room-id",
		Title:     "Planning",
		StartTime: timestamppb.New(start),
		EndTime:   timestamppb.New(start.Add(time.Hour)),
	})
	if err != nil {
		t.Fatalf("CreateBooking() error = %v", err)
	}
	if response.GetBookingId() != "booking-id" {
		t.Fatalf("CreateBooking() booking ID = %q", response.GetBookingId())
	}
	if service.created == nil || service.created.UserId != "user-id" ||
		service.created.RoomId != "room-id" ||
		service.created.Status != domain.BookingStatusConfirmed {
		t.Fatalf("CreateBooking() passed unexpected booking: %+v", service.created)
	}
}

func TestCreateBookingRequiresValidAuthorizationAndTimestamps(t *testing.T) {
	service := &bookingServiceStub{createdID: "booking-id"}
	server := NewBookingServer(service, []byte("test-secret"))
	start := timestamppb.Now()

	tests := []struct {
		name string
		ctx  context.Context
		req  *bookingpb.CreateBookingRequest
		code codes.Code
	}{
		{
			name: "missing token",
			ctx:  context.Background(),
			req:  &bookingpb.CreateBookingRequest{StartTime: start, EndTime: start},
			code: codes.Unauthenticated,
		},
		{
			name: "missing timestamps",
			ctx:  contextWithAuthorization("Bearer ignored"),
			req:  &bookingpb.CreateBookingRequest{StartTime: start},
			code: codes.InvalidArgument,
		},
		{
			name: "invalid timestamp",
			ctx:  contextWithAuthorization("Bearer ignored"),
			req: &bookingpb.CreateBookingRequest{
				StartTime: &timestamppb.Timestamp{Seconds: 1<<63 - 1},
				EndTime:   start,
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.CreateBooking(tt.ctx, tt.req)
			if status.Code(err) != tt.code {
				t.Fatalf("CreateBooking() code = %s, want %s (error %v)", status.Code(err), tt.code, err)
			}
		})
	}

	if service.created != nil {
		t.Fatal("booking service was called for invalid requests")
	}
}

func TestListMyBookingsMapsResultsAndErrors(t *testing.T) {
	start := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	service := &bookingServiceStub{
		bookings: []*domain.Booking{{
			Id:        "booking-id",
			UserId:    "user-id",
			RoomId:    "room-id",
			Title:     "Planning",
			StartTime: start,
			EndTime:   start.Add(time.Hour),
			Status:    domain.BookingStatusConfirmed,
		}},
	}
	server, ctx := bookingServerWithToken(t, service, "user-id", "user")

	response, err := server.ListMyBookings(ctx, &emptypb.Empty{})
	if err != nil {
		t.Fatalf("ListMyBookings() error = %v", err)
	}
	if service.userID != "user-id" || len(response.GetBookings()) != 1 {
		t.Fatalf("ListMyBookings() user=%q response=%+v", service.userID, response)
	}
	if response.Bookings[0].GetStatus() != bookingpb.BookingStatus_BOOKING_STATUS_CONFIRMED {
		t.Fatalf("ListMyBookings() status = %s", response.Bookings[0].GetStatus())
	}

	service.err = &apperrors.Errors{Code: apperrors.ValidationError, Message: "invalid user id"}
	_, err = server.ListMyBookings(ctx, &emptypb.Empty{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListMyBookings() error code = %s, want InvalidArgument", status.Code(err))
	}

	service.err = errors.New("database credentials leaked")
	_, err = server.ListMyBookings(ctx, &emptypb.Empty{})
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal server error" {
		t.Fatalf("ListMyBookings() returned unexpected internal error: %v", err)
	}
}

func bookingServerWithToken(t *testing.T, service *bookingServiceStub, userID, role string) (*BookingServer, context.Context) {
	t.Helper()
	secret := []byte("test-secret")
	token, err := security.CreateAccessToken(userID, role, secret)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	return NewBookingServer(service, secret), contextWithAuthorization("Bearer " + token)
}

func contextWithAuthorization(value string) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs("authorization", value),
	)
}
