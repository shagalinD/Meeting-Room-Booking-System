package grpc

import (
	"context"
	"errors"
	"testing"

	roompb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"github.com/shdmitri/booking-service/pkg/security"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type roomServiceStub struct {
	createdID string
	created   *domain.Room
	roomID    string
	room      *domain.Room
	filter    domain.RoomFilter
	rooms     []*domain.Room
	err       error
}

func (s *roomServiceStub) Create(room *domain.Room) (string, error) {
	s.created = room
	return s.createdID, s.err
}

func (s *roomServiceStub) GetByID(id string) (*domain.Room, error) {
	s.roomID = id
	return s.room, s.err
}

func (s *roomServiceStub) List(filter domain.RoomFilter) ([]*domain.Room, error) {
	s.filter = filter
	return s.rooms, s.err
}

func (*roomServiceStub) Update(*domain.Room) error { panic("unexpected Update call") }
func (*roomServiceStub) Delete(string) error       { panic("unexpected Delete call") }

func TestCreateRoomRequiresAdministrator(t *testing.T) {
	service := &roomServiceStub{createdID: "room-id"}
	server, adminCtx := roomServerWithToken(t, service, "admin-id", "admin")

	response, err := server.CreateRoom(adminCtx, &roompb.CreateRoomRequest{
		Name:        "Board room",
		Description: "Large meeting room",
		Capacity:    12,
	})
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if response.GetId() != "room-id" || service.created == nil {
		t.Fatalf("CreateRoom() response=%+v room=%+v", response, service.created)
	}
	if service.created.Name != "Board room" || service.created.Description != "Large meeting room" ||
		service.created.Capacity != 12 {
		t.Fatalf("CreateRoom() passed unexpected room: %+v", service.created)
	}

	_, userCtx := roomServerWithToken(t, service, "user-id", "user")
	_, err = server.CreateRoom(userCtx, &roompb.CreateRoomRequest{Name: "Denied", Capacity: 1})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("CreateRoom() code = %s, want PermissionDenied", status.Code(err))
	}

	_, err = server.CreateRoom(context.Background(), &roompb.CreateRoomRequest{Name: "Denied", Capacity: 1})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("CreateRoom() without token code = %s, want Unauthenticated", status.Code(err))
	}
}

func TestListRoomsUsesCurrentDefaultsAndMapsRooms(t *testing.T) {
	service := &roomServiceStub{
		rooms: []*domain.Room{
			{ID: "room-id", Name: "Board room", Description: "Large", Capacity: 12},
		},
	}
	server := NewRoomServer(service, []byte("test-secret"))

	response, err := server.ListRooms(context.Background(), &roompb.ListRoomsRequest{
		MinCapacity: 4,
		MaxCapacity: 20,
	})
	if err != nil {
		t.Fatalf("ListRooms() error = %v", err)
	}
	if service.filter != (domain.RoomFilter{MinCapacity: 4, MaxCapacity: 20}) {
		t.Fatalf("ListRooms() filter = %+v", service.filter)
	}
	if len(response.GetRooms()) != 1 || response.Rooms[0].GetId() != "room-id" ||
		response.Rooms[0].GetDescription() != "Large" || response.Rooms[0].GetCapacity() != 12 {
		t.Fatalf("ListRooms() response = %+v", response)
	}

	_, err = server.ListRooms(context.Background(), &roompb.ListRoomsRequest{})
	if err != nil || service.filter != (domain.RoomFilter{MinCapacity: 0, MaxCapacity: 10000}) {
		t.Fatalf("ListRooms() defaults filter=%+v error=%v", service.filter, err)
	}

	_, err = server.ListRooms(context.Background(), &roompb.ListRoomsRequest{MinCapacity: -1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListRooms() invalid filter code = %s, want InvalidArgument", status.Code(err))
	}
}

func TestGetRoomMapsNotFoundAndRoom(t *testing.T) {
	service := &roomServiceStub{
		room: &domain.Room{ID: "00000000-0000-4000-8000-000000000001", Name: "Board room", Capacity: 12},
	}
	server := NewRoomServer(service, []byte("test-secret"))

	roomID := "00000000-0000-4000-8000-000000000001"
	room, err := server.GetRoom(context.Background(), &roompb.GetRoomRequest{RoomId: roomID})
	if err != nil {
		t.Fatalf("GetRoom() error = %v", err)
	}
	if service.roomID != roomID || room.GetId() != roomID || room.GetCapacity() != 12 {
		t.Fatalf("GetRoom() room=%+v requested ID=%q", room, service.roomID)
	}

	_, err = server.GetRoom(context.Background(), &roompb.GetRoomRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetRoom() missing ID code = %s, want InvalidArgument", status.Code(err))
	}
	service.err = &apperrors.Errors{Code: apperrors.NotFoundError, Message: "Resource not found"}
	_, err = server.GetRoom(context.Background(), &roompb.GetRoomRequest{RoomId: "missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetRoom() missing room code = %s, want NotFound", status.Code(err))
	}

	service.err = errors.New("database credentials leaked")
	_, err = server.GetRoom(context.Background(), &roompb.GetRoomRequest{RoomId: "room-id"})
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal server error" {
		t.Fatalf("GetRoom() returned unexpected internal error: %v", err)
	}
}

func roomServerWithToken(t *testing.T, service *roomServiceStub, userID, role string) (*RoomServer, context.Context) {
	t.Helper()
	secret := []byte("test-secret")
	token, err := security.CreateAccessToken(userID, role, secret)
	if err != nil {
		t.Fatalf("CreateAccessToken() error = %v", err)
	}
	ctx := contextWithAuthorization("Bearer " + token)
	return NewRoomServer(service, secret), ctx
}
