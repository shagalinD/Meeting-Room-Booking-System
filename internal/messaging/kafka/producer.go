package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/shdmitri/booking-service/internal/domain"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string, topic string) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("at least one Kafka broker is required")
	}
	if topic == "" {
		return nil, errors.New("Kafka topic is required")
	}
	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
			Async:        false,
			WriteTimeout: 5 * time.Second,
		},
	}, nil
}

func (p *Producer) Publish(ctx context.Context, event domain.OutboxEvent) error {
	if p == nil || p.writer == nil {
		return errors.New("Kafka producer is not initialized")
	}
	if event.Key == "" || len(event.Payload) == 0 {
		return errors.New("Kafka event requires a key and payload")
	}
	if err := p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(event.Key),
		Value: event.Payload,
	}); err != nil {
		return fmt.Errorf("write event %s to Kafka: %w", event.ID, err)
	}
	return nil
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close Kafka producer: %w", err)
	}
	return nil
}
