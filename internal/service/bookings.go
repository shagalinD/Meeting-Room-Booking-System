package service

import (
	"context"
	"errors"

	"github.com/shdmitri/booking-service/internal/domain"
)

var _ domain.BookingService = (*BookingService)(nil)

type BookingService struct {
	Repo domain.BookingRepository
}

func (s *BookingService) GetByUserId(ctx context.Context, userId string) ([]*domain.Booking, error) {
	booking, err := s.Repo.GetByUserId(ctx, userId)

	if err != nil {
		return nil, err 
	}

	return booking, nil
}

func (s *BookingService) Create(ctx context.Context, booking *domain.Booking) (string, error) {
	if err := validateBooking(booking); err != nil {
		return "", err 
	}

	id, err := s.Repo.Create(ctx, booking)

	if err != nil {
		return "", err
	}

	return id, nil
}

func validateBooking(booking *domain.Booking) error {
	if booking.EndTime.IsZero() || booking.StartTime.IsZero() {
		return serviceError(errors.New("validationg error"), "Invalid time format")
	}
	if booking.RoomId == "" {
		return serviceError(errors.New("validation error"), "Room id is required")
	}
	if booking.UserId == "" {
		return serviceError(errors.New("validation error"), "Room id is required")
	}
	if booking.StartTime.After(booking.EndTime) {
		return serviceError(errors.New("validation error"), "Start time should be before end time of booking")
	}

	return nil
}

// type BookingService interface {
// 	Create(context.Context, Booking) (string, error)
// 	GetByUserId(context.Context, string) (Booking)
// }