package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/api"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultRPCTimeout = 5 * time.Second

type AuthClient struct {
	client  pb.AuthServiceClient
	timeout time.Duration
}

func NewAuthClient(conn grpc.ClientConnInterface, timeout time.Duration) *AuthClient {
	return &AuthClient{client: pb.NewAuthServiceClient(conn), timeout: withDefaultTimeout(timeout)}
}

func (c *AuthClient) Register(ctx context.Context, email, password, firstName, lastName, role string) (string, string, error) {
	if role != "user" {
		return "", "", &apperrors.Errors{Code: apperrors.ValidationError, Message: "only user self-registration is allowed"}
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.client.Register(callCtx, &pb.RegisterRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
	})
	if err != nil {
		return "", "", toApplicationError(err)
	}
	return response.GetAccessToken(), response.GetRefreshToken(), nil
}

func (c *AuthClient) Login(ctx context.Context, email, password string) (string, string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.client.Login(callCtx, &pb.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", "", toApplicationError(err)
	}
	return response.GetAccessToken(), response.GetRefreshToken(), nil
}

func (c *AuthClient) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.client.RefreshTokens(callCtx, &pb.RefreshTokensRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return "", "", toApplicationError(err)
	}
	return response.GetAccessToken(), response.GetRefreshToken(), nil
}

type BookingClient struct {
	client  pb.BookingServiceClient
	timeout time.Duration
}

func NewBookingClient(conn grpc.ClientConnInterface, timeout time.Duration) *BookingClient {
	return &BookingClient{client: pb.NewBookingServiceClient(conn), timeout: withDefaultTimeout(timeout)}
}

func (c *BookingClient) Create(ctx context.Context, booking *domain.Booking) (string, error) {
	token, ok := middleware.AccessTokenFromContext(ctx)
	if !ok || token == "" {
		return "", &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "missing authorization token"}
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+token)

	response, err := c.client.CreateBooking(callCtx, &pb.CreateBookingRequest{
		Title:     booking.Title,
		StartTime: timestamppb.New(booking.StartTime),
		EndTime:   timestamppb.New(booking.EndTime),
		RoomId:    booking.RoomId,
	})
	if err != nil {
		return "", toApplicationError(err)
	}
	return response.GetBookingId(), nil
}

func (c *BookingClient) GetByUserId(ctx context.Context, _ string) ([]*domain.Booking, error) {
	token, ok := middleware.AccessTokenFromContext(ctx)
	if !ok || token == "" {
		return nil, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "missing authorization token"}
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+token)

	response, err := c.client.ListMyBookings(callCtx, &emptypb.Empty{})
	if err != nil {
		return nil, toApplicationError(err)
	}

	bookings := make([]*domain.Booking, 0, len(response.GetBookings()))
	for _, booking := range response.GetBookings() {
		if booking == nil || booking.GetStartTime() == nil || booking.GetEndTime() == nil {
			return nil, &apperrors.Errors{
				Code:    apperrors.InternalServerError,
				Message: "booking service returned an invalid response",
			}
		}
		if err := booking.GetStartTime().CheckValid(); err != nil {
			return nil, &apperrors.Errors{Code: apperrors.InternalServerError, Message: "booking service returned an invalid response"}
		}
		if err := booking.GetEndTime().CheckValid(); err != nil {
			return nil, &apperrors.Errors{Code: apperrors.InternalServerError, Message: "booking service returned an invalid response"}
		}
		bookings = append(bookings, &domain.Booking{
			Id:        booking.GetId(),
			RoomId:    booking.GetRoomId(),
			UserId:    booking.GetUserId(),
			Title:     booking.GetTitle(),
			StartTime: booking.GetStartTime().AsTime(),
			EndTime:   booking.GetEndTime().AsTime(),
			Status:    bookingStatusName(booking.GetStatus()),
		})
	}
	return bookings, nil
}

type RoomClient struct {
	client  pb.RoomServiceClient
	timeout time.Duration
}

func NewRoomClient(conn grpc.ClientConnInterface, timeout time.Duration) *RoomClient {
	return &RoomClient{client: pb.NewRoomServiceClient(conn), timeout: withDefaultTimeout(timeout)}
}

func (c *RoomClient) Create(ctx context.Context, room *domain.Room) (string, error) {
	token, ok := middleware.AccessTokenFromContext(ctx)
	if !ok || token == "" {
		return "", &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "missing authorization token"}
	}

	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "authorization", "Bearer "+token)

	response, err := c.client.CreateRoom(callCtx, &pb.CreateRoomRequest{
		Name:        room.Name,
		Description: room.Description,
		Capacity:    int32(room.Capacity),
	})
	if err != nil {
		return "", toApplicationError(err)
	}
	return response.GetId(), nil
}

func (c *RoomClient) List(ctx context.Context, filter domain.RoomFilter) ([]*domain.Room, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.client.ListRooms(callCtx, &pb.ListRoomsRequest{
		MinCapacity: int32(filter.MinCapacity),
		MaxCapacity: int32(filter.MaxCapacity),
	})
	if err != nil {
		return nil, toApplicationError(err)
	}
	rooms := make([]*domain.Room, 0, len(response.GetRooms()))
	for _, room := range response.GetRooms() {
		if room == nil {
			return nil, &apperrors.Errors{Code: apperrors.InternalServerError, Message: "room service returned an invalid response"}
		}
		rooms = append(rooms, protoRoomToDomain(room))
	}
	return rooms, nil
}

func (c *RoomClient) GetByID(ctx context.Context, id string) (*domain.Room, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response, err := c.client.GetRoom(callCtx, &pb.GetRoomRequest{RoomId: id})
	if err != nil {
		return nil, toApplicationError(err)
	}
	if response == nil {
		return nil, &apperrors.Errors{Code: apperrors.InternalServerError, Message: "room service returned an invalid response"}
	}
	return protoRoomToDomain(response), nil
}

func protoRoomToDomain(room *pb.Room) *domain.Room {
	return &domain.Room{
		ID:          room.GetId(),
		Name:        room.GetName(),
		Description: room.GetDescription(),
		Capacity:    int(room.GetCapacity()),
	}
}

func bookingStatusName(value pb.BookingStatus) string {
	switch value {
	case pb.BookingStatus_BOOKING_STATUS_PENDING:
		return domain.BookingStatusPending
	case pb.BookingStatus_BOOKING_STATUS_CONFIRMED:
		return domain.BookingStatusConfirmed
	case pb.BookingStatus_BOOKING_STATUS_CANCELED:
		return domain.BookingStatusCanceled
	case pb.BookingStatus_BOOKING_STATUS_FINISHED:
		return domain.BookingStatusFinished
	default:
		return ""
	}
}

func withDefaultTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultRPCTimeout
	}
	return timeout
}

func toApplicationError(err error) error {
	if err == nil {
		return nil
	}

	grpcStatus, ok := status.FromError(err)
	if !ok {
		return &apperrors.Errors{Err: err, Code: apperrors.InternalServerError, Message: "upstream service error"}
	}

	switch grpcStatus.Code() {
	case codes.InvalidArgument:
		return &apperrors.Errors{Err: err, Code: apperrors.ValidationError, Message: grpcStatus.Message()}
	case codes.Unauthenticated:
		return &apperrors.Errors{Err: err, Code: apperrors.InvalidCredentialsError, Message: grpcStatus.Message()}
	case codes.PermissionDenied:
		return &apperrors.Errors{Err: err, Code: apperrors.UnauthorizedError, Message: grpcStatus.Message()}
	case codes.NotFound:
		return &apperrors.Errors{Err: err, Code: apperrors.NotFoundError, Message: grpcStatus.Message()}
	case codes.AlreadyExists:
		return &apperrors.Errors{Err: err, Code: apperrors.InternalServerError, Message: "conflict: " + grpcStatus.Message()}
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return &apperrors.Errors{Err: err, Code: apperrors.ServiceUnavailableError, Message: "service temporarily unavailable"}
	case codes.Internal:
		return &apperrors.Errors{Err: err, Code: apperrors.InternalServerError, Message: "upstream service error"}
	default:
		return &apperrors.Errors{Err: fmt.Errorf("upstream gRPC error (%s): %s", grpcStatus.Code(), strings.TrimSpace(grpcStatus.Message())), Code: apperrors.InternalServerError, Message: "upstream service error"}
	}
}

var _ domain.AuthService = (*AuthClient)(nil)
var _ domain.BookingService = (*BookingClient)(nil)
var _ api.RoomService = (*RoomClient)(nil)
