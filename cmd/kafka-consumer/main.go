package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	kafkaclient "github.com/shdmitri/booking-service/internal/messaging/kafka"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Kafka consumer stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := kafkaclient.ConfigFromEnv()
	if err != nil {
		return fmt.Errorf("load Kafka configuration: %w", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := kafkaclient.RunConsumer(ctx, config.Brokers, config.Topic, config.ConsumerGroup, logger); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("run Kafka consumer: %w", err)
	}
	return nil
}
