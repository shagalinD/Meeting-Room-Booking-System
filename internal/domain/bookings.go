package domain

import (
	"context"
	"time"
)

const (
	BookingStatusPending = "pending"
	BookingStatusConfirmed = "confirmed"
	BookingStatusCanceled = "canceled"
	BookingStatusFinished = "finished"
)

type Booking struct {
	Id string
	RoomId string
	UserId string 
	Title string 
	StartTime time.Time
	EndTime time.Time 
	Status string 
}

type BookingRepository interface {
	Create(context.Context, *Booking) (string, error)
	GetByUserId(context.Context, string) ([]*Booking, error)
}

type BookingService interface {
	Create(context.Context, *Booking) (string, error)
	GetByUserId(context.Context, string) ([]*Booking, error)
}