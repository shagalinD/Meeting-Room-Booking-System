package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/shdmitri/booking-service/internal/domain"
)

type repositoryStub struct {
	events     []domain.OutboxEvent
	published  []string
	released   []string
	claimError error
	markError  error
	releaseErr error
	workerID   string
}

func (r *repositoryStub) ClaimOutboxEvents(_ context.Context, workerID string, _ int, _ time.Duration) ([]domain.OutboxEvent, error) {
	r.workerID = workerID
	return r.events, r.claimError
}

func (r *repositoryStub) MarkOutboxEventPublished(_ context.Context, eventID, workerID string) error {
	r.published = append(r.published, eventID)
	r.workerID = workerID
	return r.markError
}

func (r *repositoryStub) ReleaseOutboxEvent(_ context.Context, eventID, workerID string, _ error) error {
	r.released = append(r.released, eventID)
	r.workerID = workerID
	return r.releaseErr
}

type publisherStub struct {
	published []domain.OutboxEvent
	err       error
}

func (p *publisherStub) Publish(_ context.Context, event domain.OutboxEvent) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, event)
	return nil
}

func TestWorkerPublishesAndMarksEvents(t *testing.T) {
	repository := &repositoryStub{events: []domain.OutboxEvent{{ID: "event-1", EventType: "booking.created"}}}
	publisher := &publisherStub{}
	worker := &Worker{
		Repository: repository,
		Publisher:  publisher,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if err := worker.publishBatch(context.Background(), "worker-1", 10, time.Minute); err != nil {
		t.Fatalf("publishBatch() error = %v", err)
	}
	if len(publisher.published) != 1 || publisher.published[0].ID != "event-1" {
		t.Fatalf("published events = %+v", publisher.published)
	}
	if len(repository.published) != 1 || repository.published[0] != "event-1" {
		t.Fatalf("marked events = %v", repository.published)
	}
}

func TestWorkerReleasesEventAfterPublishFailure(t *testing.T) {
	publishErr := errors.New("broker unavailable")
	repository := &repositoryStub{events: []domain.OutboxEvent{{ID: "event-1"}, {ID: "event-2"}}}
	publisher := &publisherStub{err: publishErr}
	worker := &Worker{
		Repository: repository,
		Publisher:  publisher,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	if err := worker.publishBatch(context.Background(), "worker-1", 10, time.Minute); err != nil {
		t.Fatalf("publishBatch() error = %v", err)
	}
	if len(repository.released) != 2 || repository.released[0] != "event-1" || repository.released[1] != "event-2" {
		t.Fatalf("released events = %v", repository.released)
	}
	if len(repository.published) != 0 {
		t.Fatalf("successfully published events = %v", repository.published)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("broker accepted events = %+v", publisher.published)
	}
}

func TestWorkerRequiresDependencies(t *testing.T) {
	worker := &Worker{}
	if err := worker.Run(context.Background()); err == nil {
		t.Fatal("Run() expected a configuration error")
	}
}
