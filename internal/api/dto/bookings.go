package dto

import "time"

type (
	Booking struct {
		Id 				string 		`json:"id"`
		Title     string 		`json:"title"`
		StartTime time.Time `json:"start_time"`
		EndTime		time.Time `json:"end_time"`
		UserId 		string 		`json:"user_id"`
		RoomId		string 		`json:"room_id"`
		Status 		string 		`json:"status"`
	}
	
	CreateBookingRequest struct {
		Title     string 		`json:"title"`
		StartTime time.Time `json:"start_time"`
		EndTime		time.Time `json:"end_time"`
		RoomId		string 		`json:"room_id"`
	}
	CreateBookingResponse struct {
		BookingId string `json:"booking_id"`
	}

	GetByUserIdBookingResponse []Booking
)