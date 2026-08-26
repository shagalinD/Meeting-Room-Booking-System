package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/shdmitri/booking-service/internal/api/dto"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type RoomHandler struct {
	Service domain.RoomService
	Logger *slog.Logger
}

func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var room dto.CreateRoomRequest
	
	if err := json.NewDecoder(r.Body).Decode(&room); err != nil {
		w.WriteHeader(http.StatusBadRequest)
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

	id, err := h.Service.Create(&domain.Room{
		Name: room.Name,
		Description: room.Description,
		Capacity: room.Capacity,
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

	rooms, err := h.Service.List(domain.RoomFilter{
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