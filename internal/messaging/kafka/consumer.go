package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func RunConsumer(ctx context.Context, brokers []string, topic, group string, logger *slog.Logger) error {
	if len(brokers) == 0 {
		return errors.New("at least one Kafka broker is required")
	}
	if topic == "" || group == "" {
		return errors.New("Kafka topic and consumer group are required")
	}
	if logger == nil {
		return errors.New("Kafka consumer logger is required")
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        group,
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        time.Second,
		CommitInterval: 0,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Error("close Kafka consumer", "error", err)
		}
	}()

	logger.Info("Kafka event consumer started", "topic", topic, "group", group)
	for {
		message, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			logger.Error("fetch Kafka event", "error", err)
			if err := waitForRetry(ctx); err != nil {
				return nil
			}
			continue
		}

		var event struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message.Value, &event); err != nil {
			logger.Error("decode Kafka event", "key", string(message.Key), "offset", message.Offset, "error", err)
		} else if event.Type == "" {
			logger.Error("Kafka event has no type", "key", string(message.Key), "offset", message.Offset)
		}
		logger.Info(
			"Kafka event received",
			"key", string(message.Key),
			"event_type", event.Type,
			"partition", message.Partition,
			"offset", message.Offset,
			"size_bytes", len(message.Value),
		)
		for {
			if err := reader.CommitMessages(ctx, message); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				logger.Error("commit Kafka event", "key", string(message.Key), "offset", message.Offset, "error", err)
				if err := waitForRetry(ctx); err != nil {
					return nil
				}
				continue
			}
			break
		}
	}
}

func waitForRetry(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
