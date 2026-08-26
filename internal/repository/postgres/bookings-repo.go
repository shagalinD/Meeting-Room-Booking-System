package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shdmitri/booking-service/internal/db"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

var _ domain.BookingRepository = (*BookingRepository)(nil)

type BookingRepository struct {
	db   *db.Queries
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{
		db:   db.New(pool),
		pool: pool,
	}
}

func (br *BookingRepository) GetByUserId(ctx context.Context, userId string) ([]*domain.Booking, error) {
	pgUUID, err := ParseUUID(userId)
	if err != nil {
		return nil, err
	}

	conn, err := br.pool.Acquire(ctx)
	if err != nil {
		return nil, MapError(err)
	}

	queries := db.New(conn)
	bookings, err := queries.GetByUserId(ctx, pgUUID)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.Booking, len(bookings))
	for i, b := range bookings {
		result[i] = &domain.Booking{
			Id:        b.ID.String(),
			RoomId:    b.RoomID.String(),
			UserId:    b.UserID.String(),
			Title:     b.Title.String,
			StartTime: b.StartTime.Time,
			EndTime:   b.EndTime.Time,
			Status:    string(b.Status),
		}
	}

	return result, nil
}

func (br *BookingRepository) Create(ctx context.Context, booking *domain.Booking) (string, error) {
	tx, err := br.pool.Begin(ctx)
	defer tx.Rollback(ctx)

	if err != nil {
		return "", MapError(err)
	}
	params, err := toRepositoryParams(booking)
	query := br.db.WithTx(tx)

	if _, err := query.SelectRoomForUpdate(ctx, params.RoomID); 
	err != nil {
		return "", MapError(err)
	}

	if err != nil {
		return "", MapError(err)
	}
	
	found, err := getIntersections(ctx, br, params.RoomID, params.StartTime, params.EndTime)

	if err != nil {
		return "", MapError(err) 
	}

	if found {
		return "", &apperrors.Errors{
				Err:     err,
				Code:    apperrors.ValidationError,
				Message: "Conflict with existing data",
		}
	}

	booking_id, err := query.CreateBooking(ctx, params)
	
	if err != nil {
		return "", err 
	} 

	if err := tx.Commit(ctx); err != nil {
		return "", MapError(err)
	}
	return booking_id.String(), nil
}

func getIntersections(ctx context.Context, br *BookingRepository, roomId pgtype.UUID, startTime, endTime pgtype.Timestamptz) (bool, error) {
	_, err := br.db.GetIntersections(ctx, db.GetIntersectionsParams{
		RoomID: roomId,
		Tstzrange: startTime,
		Tstzrange_2: endTime,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return true, err 
	}

	return true, nil
}

func toRepositoryParams(booking *domain.Booking) (db.CreateBookingParams, error) {
	params := db.CreateBookingParams{}

	if err := params.UserID.Scan(booking.UserId); err != nil {
		return db.CreateBookingParams{}, err
	}
	if err := params.RoomID.Scan(booking.RoomId); err != nil {
		return db.CreateBookingParams{}, err
	}
	if err := params.StartTime.Scan(booking.StartTime); err != nil {
		return db.CreateBookingParams{}, err
	}
	if err := params.EndTime.Scan(booking.EndTime); err != nil {
		return db.CreateBookingParams{}, err
	}
	if err := params.Title.Scan(booking.Title); err != nil {
		return db.CreateBookingParams{}, err
	}
	if err := params.Status.Scan(booking.Status); err != nil {
		return db.CreateBookingParams{}, err
	}

	return params, nil
}

func StringToText(input string) (pgtype.Text, error){
	var text pgtype.Text 
	if err := text.Scan(input); err != nil {
		return pgtype.Text{}, err 
	} else {
		return text, nil
	}
}

func ParseUUID(id string) (pgtype.UUID, error) {
    var pgUUID pgtype.UUID
    if err := pgUUID.Scan(id); err != nil {
        return pgtype.UUID{}, err
    }

    return pgUUID, nil
}


