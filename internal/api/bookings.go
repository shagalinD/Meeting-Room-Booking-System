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

// GetByUserId returns the authenticated user's bookings.
// @Summary List my bookings
// @Description Returns bookings belonging to the user identified by the access token.
// @Tags bookings
// @Produce json
// @Security BearerAuth
// @Success 200 {array} dto.Booking
// @Failure 401 {object} ErrorResponse "Missing or invalid access token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /bookings/get_by_user_id [get]
func (h *BookingHandler) GetByUserId(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Message: "error on getting user id",
			Code:    apperrors.InvalidCredentialsError,
			Err:     nil,
		})
		return
	}

	bookings, err := h.Service.GetByUserId(r.Context(), userId)

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookingsToDTO(bookings))
}

// Create creates a booking for the authenticated user.
// @Summary Create a booking
// @Description Creates a confirmed booking for the user identified by the access token. Times must be RFC 3339 timestamps and start must not be after end.
// @Tags bookings
// @Accept json
// @Produce json
// @Param request body dto.CreateBookingRequest true "Booking details"
// @Security BearerAuth
// @Success 200 {object} dto.CreateBookingResponse
// @Failure 400 {object} ErrorResponse "Invalid booking details"
// @Failure 401 {object} ErrorResponse "Missing or invalid access token"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /bookings/create [post]
func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	userId, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Message: "error on getting user id",
			Code:    apperrors.InvalidCredentialsError,
			Err:     nil,
		})
		return
	}

	var booking dto.CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&booking); err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{
			Err:     err,
			Code:    apperrors.ValidationError,
			Message: "invalid request body",
		})
		return
	}

	bookingId, err := h.Service.Create(r.Context(), &domain.Booking{
		RoomId:    booking.RoomId,
		UserId:    userId,
		StartTime: booking.StartTime,
		EndTime:   booking.EndTime,
		Title:     booking.Title,
		Status:    domain.BookingStatusConfirmed,
	})

	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.CreateBookingResponse{
		BookingId: bookingId,
	})
}

func bookingsToDTO(bookings []*domain.Booking) dto.GetByUserIdBookingResponse {
	result := make(dto.GetByUserIdBookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, dto.Booking{
			Id:        booking.Id,
			Title:     booking.Title,
			StartTime: booking.StartTime,
			EndTime:   booking.EndTime,
			UserId:    booking.UserId,
			RoomId:    booking.RoomId,
			Status:    booking.Status,
		})
	}

	return result
}
