package grpc

import (
	"context"
	"strings"

	roompb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ roompb.RoomServiceServer = (*RoomServer)(nil)

type RoomServer struct {
	roompb.UnimplementedRoomServiceServer
	service   domain.RoomService
	jwtSecret []byte
}

func NewRoomServer(service domain.RoomService, jwtSecret []byte) *RoomServer {
	return &RoomServer{
		service:   service,
		jwtSecret: jwtSecret,
	}
}

func (s *RoomServer) CreateRoom(ctx context.Context, req *roompb.CreateRoomRequest) (*roompb.CreateRoomResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "room details are required")
	}

	claims, err := authenticatedClaims(ctx, s.jwtSecret)
	if err != nil {
		return nil, err
	}
	if claims.Role != "admin" {
		return nil, status.Error(codes.PermissionDenied, "administrator role is required")
	}

	id, err := s.service.Create(&domain.Room{
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Capacity:    int(req.GetCapacity()),
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	return &roompb.CreateRoomResponse{Id: id}, nil
}

func (s *RoomServer) ListRooms(_ context.Context, req *roompb.ListRoomsRequest) (*roompb.ListRoomsResponse, error) {
	minCapacity := 0
	maxCapacity := 10000
	if req != nil {
		minCapacity = int(req.GetMinCapacity())
		if req.GetMaxCapacity() != 0 {
			maxCapacity = int(req.GetMaxCapacity())
		}
	}
	if minCapacity < 0 || maxCapacity < 0 {
		return nil, status.Error(codes.InvalidArgument, "room capacity filters must not be negative")
	}

	rooms, err := s.service.List(domain.RoomFilter{
		MinCapacity: minCapacity,
		MaxCapacity: maxCapacity,
	})
	if err != nil {
		return nil, toStatusError(err)
	}

	response := &roompb.ListRoomsResponse{
		Rooms: make([]*roompb.Room, 0, len(rooms)),
	}
	for _, room := range rooms {
		if room == nil {
			return nil, status.Error(codes.Internal, "room service returned an invalid room")
		}
		response.Rooms = append(response.Rooms, roomToProto(room))
	}

	return response, nil
}

func (s *RoomServer) GetRoom(_ context.Context, req *roompb.GetRoomRequest) (*roompb.Room, error) {
	if req == nil || strings.TrimSpace(req.GetRoomId()) == "" {
		return nil, status.Error(codes.InvalidArgument, "room_id is required")
	}

	room, err := s.service.GetByID(req.GetRoomId())
	if err != nil {
		return nil, toStatusError(err)
	}
	if room == nil {
		return nil, status.Error(codes.Internal, "room service returned an invalid room")
	}

	return roomToProto(room), nil
}

func roomToProto(room *domain.Room) *roompb.Room {
	return &roompb.Room{
		Id:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		Capacity:    int32(room.Capacity),
	}
}
