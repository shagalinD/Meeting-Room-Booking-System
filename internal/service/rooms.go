package service

import (
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

// RoomService interface {
// 	Create(room *Room) error
// 	List(filter RoomFilter) ([]*Room, error)
// 	Update(room *Room) error
// 	Delete(id string) error
// }

var _ domain.RoomService = (*RoomService)(nil)


type RoomService struct {
	Repo domain.RoomRepository
}

func (s *RoomService) Create(room *domain.Room) (string, error) {
	if room.Name == "" {
		return "", &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room name is required",
		}
	}

	if room.Capacity <= 0 {
		return "", &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room capacity must be greater than zero",
		}
	}

	return s.Repo.Create(room)
}

func (s *RoomService) List(filter domain.RoomFilter) ([]*domain.Room, error) {
	if filter.MinCapacity < 0 || filter.MaxCapacity < 0 {
		return nil, &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "invalid room capacity filter",
		}
	}

	return s.Repo.List(filter)
}

func (s *RoomService) Update(room *domain.Room) error {
	if room.ID == "" {
		return &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room ID is required for update",
		}
	}

	if room.Name == "" {
		return &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room name is required for update",
		}
	}

	if room.Capacity <= 0 {
		return &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room capacity must be greater than zero",
		}
	}

	return s.Repo.Update(room)
}

func (s *RoomService) Delete(id string) error {
	if id == "" {
		return &apperrors.Errors{
			Err:     nil,
			Code:    apperrors.ValidationError,
			Message: "room ID is required for deletion",
		}
	}

	return s.Repo.Delete(id)
}