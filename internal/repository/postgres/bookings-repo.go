package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	defer conn.Release()

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
	if err != nil {
		return "", MapError(err)
	}
	defer tx.Rollback(ctx)

	params, err := toRepositoryParams(booking)
	if err != nil {
		return "", MapError(err)
	}
	query := br.db.WithTx(tx)

	if _, err := query.SelectRoomForUpdate(ctx, params.RoomID); err != nil {
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

	if err := insertBookingCreatedEvent(ctx, tx, booking_id, booking); err != nil {
		return "", fmt.Errorf("insert booking.created outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", MapError(err)
	}
	return booking_id.String(), nil
}

type bookingCreatedEvent struct {
	ID            string                  `json:"id"`
	Type          string                  `json:"type"`
	SchemaVersion int                     `json:"schema_version"`
	OccurredAt    time.Time               `json:"occurred_at"`
	Data          bookingCreatedEventData `json:"data"`
}

type bookingCreatedEventData struct {
	BookingID string    `json:"booking_id"`
	UserID    string    `json:"user_id"`
	RoomID    string    `json:"room_id"`
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

func insertBookingCreatedEvent(ctx context.Context, tx pgx.Tx, bookingID pgtype.UUID, booking *domain.Booking) error {
	eventID := uuid.New()
	event := bookingCreatedEvent{
		ID:            eventID.String(),
		Type:          "booking.created",
		SchemaVersion: 1,
		OccurredAt:    time.Now().UTC(),
		Data: bookingCreatedEventData{
			BookingID: bookingID.String(),
			UserID:    booking.UserId,
			RoomID:    booking.RoomId,
			Title:     booking.Title,
			StartTime: booking.StartTime,
			EndTime:   booking.EndTime,
			Status:    booking.Status,
		},
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal booking.created event: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO booking_event_outbox (id, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4::jsonb)
	`, eventID, bookingID, event.Type, payload)
	if err != nil {
		return fmt.Errorf("store booking.created event: %w", err)
	}
	return nil
}

func (br *BookingRepository) ClaimOutboxEvents(ctx context.Context, workerID string, limit int, lease time.Duration) ([]domain.OutboxEvent, error) {
	if workerID == "" || limit <= 0 || lease <= 0 {
		return nil, errors.New("worker ID, positive batch size, and positive lease are required")
	}

	tx, err := br.pool.Begin(ctx)
	if err != nil {
		return nil, MapError(err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH claimable AS (
			SELECT id
			FROM booking_event_outbox
			WHERE published_at IS NULL
			  AND (locked_until IS NULL OR locked_until <= NOW())
			ORDER BY created_at, id
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE booking_event_outbox AS events
		SET locked_by = $1,
		    locked_until = NOW() + ($2 * INTERVAL '1 millisecond'),
		    attempts = attempts + 1
		FROM claimable
		WHERE events.id = claimable.id
		RETURNING events.id::text, events.aggregate_id::text, events.event_type, events.payload
	`, workerID, lease.Milliseconds(), limit)
	if err != nil {
		return nil, MapError(err)
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0)
	for rows.Next() {
		var event domain.OutboxEvent
		if err := rows.Scan(&event.ID, &event.Key, &event.EventType, &event.Payload); err != nil {
			return nil, MapError(err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, MapError(err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, MapError(err)
	}
	return events, nil
}

func (br *BookingRepository) MarkOutboxEventPublished(ctx context.Context, eventID, workerID string) error {
	tag, err := br.pool.Exec(ctx, `
		UPDATE booking_event_outbox
		SET published_at = NOW(), locked_by = NULL, locked_until = NULL, last_error = NULL
		WHERE id = $1 AND locked_by = $2 AND published_at IS NULL
	`, eventID, workerID)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("outbox event %s is no longer owned by worker %s", eventID, workerID)
	}
	return nil
}

func (br *BookingRepository) ReleaseOutboxEvent(ctx context.Context, eventID, workerID string, publishErr error) error {
	if publishErr == nil {
		return errors.New("publish error is required to release an outbox event")
	}
	message := publishErr.Error()
	if runeCount := len([]rune(message)); runeCount > 1024 {
		message = string([]rune(message)[:1024])
	}
	tag, err := br.pool.Exec(ctx, `
		UPDATE booking_event_outbox
		SET locked_by = NULL, locked_until = NULL, last_error = $3
		WHERE id = $1 AND locked_by = $2 AND published_at IS NULL
	`, eventID, workerID, message)
	if err != nil {
		return MapError(err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("outbox event %s is no longer owned by worker %s", eventID, workerID)
	}
	return nil
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
