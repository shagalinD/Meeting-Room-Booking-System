package grpc

import (
	"context"
	"errors"
	"strings"

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

var _ bookingpb.BookingServiceServer = (*BookingServer)(nil)

type BookingServer struct {
	bookingpb.UnimplementedBookingServiceServer
	service   domain.BookingService
	jwtSecret []byte
}

func NewBookingServer(service domain.BookingService, jwtSecret []byte) *BookingServer {
	return &BookingServer{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

func (s *BookingServer) CreateBooking(ctx context.Context, req *bookingpb.CreateBookingRequest) (*bookingpb.CreateBookingResponse, error) {
	if req == nil || req.GetStartTime() == nil || req.GetEndTime() == nil {
		return nil, status.Error(codes.InvalidArgument, "start_time and end_time are required")
	}
	if err := req.GetStartTime().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "start_time is invalid")
	}
	if err := req.GetEndTime().CheckValid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, "end_time is invalid")
	}

	userID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	id, err := s.service.Create(ctx, &domain.Booking{
		RoomId:    req.GetRoomId(),
		UserId:    userID,
		Title:     req.GetTitle(),
		StartTime: req.GetStartTime().AsTime(),
		EndTime:   req.GetEndTime().AsTime(),
		Status:    domain.BookingStatusConfirmed,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &bookingpb.CreateBookingResponse{BookingId: id}, nil
}

func (s *BookingServer) ListMyBookings(ctx context.Context, _ *emptypb.Empty) (*bookingpb.ListMyBookingsResponse, error) {
	userID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	bookings, err := s.service.GetByUserId(ctx, userID)
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &bookingpb.ListMyBookingsResponse{
		Bookings: make([]*bookingpb.Booking, 0, len(bookings)),
	}
	for _, booking := range bookings {
		converted, err := bookingToProto(booking)
		if err != nil {
			return nil, toStatusError(err)
		}
		response.Bookings = append(response.Bookings, converted)
	}

	return response, nil
}

func (s *BookingServer) authenticatedUserID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization metadata")
	}

	parts := strings.SplitN(strings.TrimSpace(values[0]), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", status.Error(codes.Unauthenticated, "invalid authorization metadata")
	}

	claims, err := security.ParseToken(strings.TrimSpace(parts[1]), s.jwtSecret)
	if err != nil || strings.TrimSpace(claims.UserID) == "" {
		return "", status.Error(codes.Unauthenticated, "invalid or expired access token")
	}
	return claims.UserID, nil
}

func bookingToProto(booking *domain.Booking) (*bookingpb.Booking, error) {
	if booking == nil {
		return nil, errors.New("booking service returned a nil booking")
	}

	startTime := timestamppb.New(booking.StartTime)
	if err := startTime.CheckValid(); err != nil {
		return nil, errors.New("booking has an invalid start time")
	}
	endTime := timestamppb.New(booking.EndTime)
	if err := endTime.CheckValid(); err != nil {
		return nil, errors.New("booking has an invalid end time")
	}

	statusValue, ok := map[string]bookingpb.BookingStatus{
		domain.BookingStatusPending:   bookingpb.BookingStatus_BOOKING_STATUS_PENDING,
		domain.BookingStatusConfirmed: bookingpb.BookingStatus_BOOKING_STATUS_CONFIRMED,
		domain.BookingStatusCanceled:  bookingpb.BookingStatus_BOOKING_STATUS_CANCELED,
		domain.BookingStatusFinished:  bookingpb.BookingStatus_BOOKING_STATUS_FINISHED,
	}[booking.Status]
	if !ok {
		return nil, errors.New("booking has an unknown status")
	}

	return &bookingpb.Booking{
		Id:        booking.Id,
		Title:     booking.Title,
		StartTime: startTime,
		EndTime:   endTime,
		UserId:    booking.UserId,
		RoomId:    booking.RoomId,
		Status:    statusValue,
	}, nil
}

func toStatusError(err error) error {
	var appErr *apperrors.Errors
	if !errors.As(err, &appErr) {
		return status.Error(codes.Internal, "internal server error")
	}

	switch appErr.Code {
	case apperrors.ValidationError:
		if strings.Contains(strings.ToLower(appErr.Message), "conflict") ||
			strings.Contains(strings.ToLower(appErr.Message), "already") {
			return status.Error(codes.AlreadyExists, appErr.Message)
		}
		return status.Error(codes.InvalidArgument, appErr.Message)
	case apperrors.NotFoundError:
		return status.Error(codes.NotFound, appErr.Message)
	case apperrors.UnauthorizedError, apperrors.InvalidCredentialsError:
		return status.Error(codes.Unauthenticated, appErr.Message)
	case apperrors.InternalServerError:
		return status.Error(codes.Internal, "internal server error")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
