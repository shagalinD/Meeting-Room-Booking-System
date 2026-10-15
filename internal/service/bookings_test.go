package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shdmitri/booking-service/internal/domain"
)

type bookingRepositoryStub struct {
	createdID string
	created   *domain.Booking
	userID    string
	bookings  []*domain.Booking
	err       error
	createCalls int
	getCalls    int
}

func (r *bookingRepositoryStub) Create(_ context.Context, booking *domain.Booking) (string, error) {
	r.createCalls++
	r.created = booking
	return r.createdID, r.err
}

func (r *bookingRepositoryStub) GetByUserId(_ context.Context, userID string) ([]*domain.Booking, error) {
	r.getCalls++
	r.userID = userID
	return r.bookings, r.err
}

func TestBookingServiceCreateValidation(t *testing.T) {
	start := time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)
	valid := &domain.Booking{RoomId: "room-1", UserId: "user-1", StartTime: start, EndTime: start.Add(time.Hour)}
	tests := []struct {
		name    string
		booking *domain.Booking
		wantErr bool
	}{
		{name: "valid booking", booking: valid},
		{name: "missing start time", booking: &domain.Booking{RoomId: "room-1", UserId: "user-1", EndTime: start}, wantErr: true},
		{name: "missing end time", booking: &domain.Booking{RoomId: "room-1", UserId: "user-1", StartTime: start}, wantErr: true},
		{name: "missing room ID", booking: &domain.Booking{UserId: "user-1", StartTime: start, EndTime: start.Add(time.Hour)}, wantErr: true},
		{name: "missing user ID", booking: &domain.Booking{RoomId: "room-1", StartTime: start, EndTime: start.Add(time.Hour)}, wantErr: true},
		{name: "start after end", booking: &domain.Booking{RoomId: "room-1", UserId: "user-1", StartTime: start.Add(time.Hour), EndTime: start}, wantErr: true},
		{name: "equal start and end", booking: &domain.Booking{RoomId: "room-1", UserId: "user-1", StartTime: start, EndTime: start}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &bookingRepositoryStub{createdID: "booking-1"}
			id, err := (&BookingService{Repo: repo}).Create(context.Background(), tt.booking)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if repo.createCalls != 0 {
					t.Fatalf("repository called %d times for invalid booking", repo.createCalls)
				}
				return
			}
			if id != "booking-1" || repo.created != tt.booking || repo.createCalls != 1 {
				t.Fatalf("Create() id=%q booking=%p calls=%d", id, repo.created, repo.createCalls)
			}
		})
	}
}

func TestBookingServicePropagatesRepositoryResults(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("repository failed")

	t.Run("create error", func(t *testing.T) {
		start := time.Now()
		booking := &domain.Booking{RoomId: "room-1", UserId: "user-1", StartTime: start, EndTime: start.Add(time.Hour)}
		repo := &bookingRepositoryStub{err: wantErr}
		id, err := (&BookingService{Repo: repo}).Create(ctx, booking)
		if id != "" || !errors.Is(err, wantErr) || repo.createCalls != 1 {
			t.Fatalf("Create() id=%q error=%v calls=%d", id, err, repo.createCalls)
		}
	})

	t.Run("get bookings result and error", func(t *testing.T) {
		wantBookings := []*domain.Booking{{Id: "booking-1", UserId: "user-1"}}
		repo := &bookingRepositoryStub{bookings: wantBookings}
		bookings, err := (&BookingService{Repo: repo}).GetByUserId(ctx, "user-1")
		if err != nil || repo.userID != "user-1" || repo.getCalls != 1 || len(bookings) != 1 || bookings[0] != wantBookings[0] {
			t.Fatalf("GetByUserId() bookings=%v error=%v repo=%+v", bookings, err, repo)
		}

		repo.err = wantErr
		bookings, err = (&BookingService{Repo: repo}).GetByUserId(ctx, "user-2")
		if !errors.Is(err, wantErr) || bookings != nil || repo.userID != "user-2" || repo.getCalls != 2 {
			t.Fatalf("GetByUserId() bookings=%v error=%v repo=%+v", bookings, err, repo)
		}
	})
}
