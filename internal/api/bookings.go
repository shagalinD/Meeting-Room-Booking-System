package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/shdmitri/booking-service/internal/api/dto"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)


type BookingHandler struct {
	Service domain.BookingService
	Logger  *slog.Logger
}

func (h *BookingHandler) GetByUserId(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Message: "error on getting user id",
			Code: apperrors.InvalidCredentialsError,
			Err: nil,
		})
		return
	}

	bookings, err := h.Service.GetByUserId(r.Context(), userId)

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	json.NewEncoder(w).Encode(bookingsToDTO(bookings))
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Message: "error on getting user id",
			Code: apperrors.InvalidCredentialsError,
			Err: nil,
		})
		return 
	}

	var booking dto.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		writeErrorResponse(w, h.Logger, err)
		return 
	}

	bookingId, err := h.Service.Create(r.Context(), &domain.Booking{
		RoomId: booking.RoomId,
		UserId: userId,
		StartTime: booking.StartTime,
		EndTime: booking.EndTime,
		Title: booking.Title,
		Status: domain.BookingStatusConfirmed,
	})

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return 
	}

	json.NewEncoder(w).Encode(dto.CreateBookingResponse{
		BookingId: bookingId,
	})
}

func bookingsToDTO(bookings []*domain.Booking) dto.GetByUserIdBookingResponse {
	result := make(dto.GetByUserIdBookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, dto.Booking{
					Id: booking.Id, 				
					Title: booking.Title,
					StartTime: booking.StartTime,
					EndTime: booking.EndTime,
					UserId: booking.UserId,
					RoomId: booking.RoomId,
					Status: booking.Status,
		})
	}

	return result
}