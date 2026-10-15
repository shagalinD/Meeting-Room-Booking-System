package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/shdmitri/booking-service/internal/api/dto"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type RoomService interface {
	Create(context.Context, *domain.Room) (string, error)
	List(context.Context, domain.RoomFilter) ([]*domain.Room, error)
	GetByID(context.Context, string) (*domain.Room, error)
}

type RoomHandler struct {
	Service RoomService
	Logger  *slog.Logger
}

// CreateRoom creates a room. Only users with the admin role are allowed.
// @Summary Create a room
// @Description Creates a room. Requires a valid access token belonging to an admin.
// @Tags rooms
// @Accept json
// @Produce json
// @Param request body dto.CreateRoomRequest true "Room details"
// @Security BearerAuth
// @Success 201 {object} dto.CreateRoomResponse
// @Failure 400 {object} ErrorResponse "Invalid request body or room details"
// @Failure 401 {object} ErrorResponse "Missing/invalid access token or user is not an admin"
// @Failure 409 {object} ErrorResponse "A room with conflicting details already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /rooms/create [post]
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var room dto.CreateRoomRequest

	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Err:     err,
			Code:    apperrors.ValidationError,
			Message: "invalid request body",
		})
		return
	}

	role, ok := middleware.RoleFromContext(r.Context())

	if !ok {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.InternalServerError, Message: "error on getting role from context"})
		return
	}

	if role != "admin" {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "unauthorized"})
		return
	}

	id, err := h.Service.Create(r.Context(), &domain.Room{
		Name:        room.Name,
		Description: room.Description,
		Capacity:    room.Capacity,
	})

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.CreateRoomResponse{
		ID: id,
	})
}

// ListRooms returns rooms filtered by their capacity.
// @Summary List rooms
// @Description Lists rooms whose capacity is within the inclusive minimum and maximum. Defaults are 0 and 10000.
// @Tags rooms
// @Produce json
// @Param min_capacity query int false "Minimum room capacity" default(0) minimum(0)
// @Param max_capacity query int false "Maximum room capacity" default(10000) minimum(0)
// @Success 200 {object} dto.ListRoomsResponse
// @Failure 400 {object} ErrorResponse "Capacity filter is invalid"
// @Failure 401 {object} ErrorResponse "Capacity query parameter is not a valid integer"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /rooms/list [get]
func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("max_capacity")
	if query == "" {
		query = "10000"
	}

	maxCapacity, err := strconv.Atoi(query)
	if err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "max capacity is in invalid format"})
		return
	}

	query = r.URL.Query().Get("min_capacity")
	if query == "" {
		query = "0"
	}

	minCapacity, err := strconv.Atoi(query)
	if err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.UnauthorizedError, Message: "min capacity is in invalid format"})
		return
	}

	rooms, err := h.Service.List(r.Context(), domain.RoomFilter{
		MinCapacity: minCapacity,
		MaxCapacity: maxCapacity,
	})

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ListRoomsResponse{
		Rooms: toDTORooms(rooms),
	})
}

// GetRoom returns a room by its ID.
// @Summary Get a room
// @Description Returns a meeting room by its UUID.
// @Tags rooms
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} dto.Room
// @Failure 400 {object} ErrorResponse "Invalid room ID"
// @Failure 404 {object} ErrorResponse "Room not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /rooms/{id} [get]
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	room, err := h.Service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dto.Room{
		ID:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		Capacity:    room.Capacity,
	})
}

func toDTORooms(rooms []*domain.Room) []dto.Room {
	dtoRooms := make([]dto.Room, len(rooms))
	for i, room := range rooms {
		dtoRooms[i] = dto.Room{
			ID:          room.ID,
			Name:        room.Name,
			Description: room.Description,
			Capacity:    room.Capacity,
		}
	}
	return dtoRooms
}
