package outbox

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shdmitri/booking-service/internal/domain"
)

type Repository interface {
	ClaimOutboxEvents(context.Context, string, int, time.Duration) ([]domain.OutboxEvent, error)
	MarkOutboxEventPublished(context.Context, string, string) error
	ReleaseOutboxEvent(context.Context, string, string, error) error
}

type Publisher interface {
	Publish(context.Context, domain.OutboxEvent) error
}

type Worker struct {
	Repository   Repository
	Publisher    Publisher
	Logger       *slog.Logger
	PollInterval time.Duration
	Lease        time.Duration
	BatchSize    int
}

func (w *Worker) Run(ctx context.Context) error {
	if w.Repository == nil {
		return errors.New("outbox repository is required")
	}
	if w.Publisher == nil {
		return errors.New("outbox publisher is required")
	}
	if w.Logger == nil {
		return errors.New("outbox logger is required")
	}

	pollInterval := w.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	lease := w.Lease
	if lease <= 0 {
		lease = 30 * time.Second
	}
	batchSize := w.BatchSize
	if batchSize <= 0 {
		batchSize = 20
	}
	workerID := uuid.NewString()

	if err := w.publishBatch(ctx, workerID, batchSize, lease); err != nil {
		return err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.publishBatch(ctx, workerID, batchSize, lease); err != nil {
				return err
			}
		}
	}
}

func (w *Worker) publishBatch(ctx context.Context, workerID string, batchSize int, lease time.Duration) error {
	events, err := w.Repository.ClaimOutboxEvents(ctx, workerID, batchSize, lease)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		w.Logger.Error("claim booking outbox events", "error", err)
		return nil
	}

	for _, event := range events {
		if err := w.Publisher.Publish(ctx, event); err != nil {
			if releaseErr := w.Repository.ReleaseOutboxEvent(ctx, event.ID, workerID, err); releaseErr != nil {
				w.Logger.Error("release booking outbox event after publish failure", "event_id", event.ID, "error", releaseErr)
			}
			w.releaseEvents(ctx, events, workerID, event.ID, err)
			w.Logger.Error("publish booking outbox event", "event_id", event.ID, "event_type", event.EventType, "error", err)
			break
		}

		if err := w.Repository.MarkOutboxEventPublished(ctx, event.ID, workerID); err != nil {
			w.releaseEvents(ctx, events, workerID, event.ID, err)
			w.Logger.Error("mark booking outbox event as published", "event_id", event.ID, "error", err)
			break
		}
		w.Logger.Info("booking outbox event published", "event_id", event.ID, "event_type", event.EventType)
	}
	return nil
}

func (w *Worker) releaseEvents(ctx context.Context, events []domain.OutboxEvent, workerID, failedEventID string, cause error) {
	foundFailedEvent := false
	for _, event := range events {
		if event.ID == failedEventID {
			foundFailedEvent = true
			continue
		}
		if !foundFailedEvent {
			continue
		}
		if err := w.Repository.ReleaseOutboxEvent(ctx, event.ID, workerID, cause); err != nil {
			w.Logger.Error("release unprocessed booking outbox event", "event_id", event.ID, "error", err)
		}
	}
}
