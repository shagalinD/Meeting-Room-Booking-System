CREATE TABLE booking_event_outbox (
  id UUID PRIMARY KEY,
  aggregate_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  published_at TIMESTAMPTZ,
  locked_by TEXT,
  locked_until TIMESTAMPTZ,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT
);

CREATE INDEX booking_event_outbox_pending_idx
  ON booking_event_outbox (created_at)
  WHERE published_at IS NULL;
