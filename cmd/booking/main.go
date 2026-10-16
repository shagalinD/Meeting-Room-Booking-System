package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	bookingpb "github.com/shagalinD/booking-service-proto/gen/go"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/internal/messaging/kafka"
	"github.com/shdmitri/booking-service/internal/outbox"
	postgres "github.com/shdmitri/booking-service/internal/repository/postgres"
	bookingcache "github.com/shdmitri/booking-service/internal/repository/redis"
	"github.com/shdmitri/booking-service/internal/service"
	transportgrpc "github.com/shdmitri/booking-service/internal/transport/grpc"
	"github.com/shdmitri/booking-service/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	if err := run(); err != nil {
		slog.Error("booking service stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	log := logger.NewLogger(config.AppConfig.Server.LogLevel)
	dbpool, err := config.ConnectDB(
		&config.AppConfig.Postgres,
		config.AppConfig.Postgres.ConnectionString(),
	)
	if err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	defer dbpool.Close()

	redisClient, err := config.ConnectRedis(&config.AppConfig.Redis)
	if err != nil {
		return fmt.Errorf("connect to Redis: %w", err)
	}
	defer closeRedis(log, redisClient)

	kafkaConfig, err := kafka.ConfigFromEnv()
	if err != nil {
		return fmt.Errorf("load Kafka configuration: %w", err)
	}
	kafkaProducer, err := kafka.NewProducer(kafkaConfig.Brokers, kafkaConfig.Topic)
	if err != nil {
		return fmt.Errorf("create Kafka producer: %w", err)
	}
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Error("close Kafka producer", "error", err)
		}
	}()

	repository := postgres.NewBookingRepository(dbpool)
	cachedRepository := bookingcache.NewBookingCacheRepository(
		repository,
		redisClient,
		config.AppConfig.Redis.TimeToLive,
		log,
	)
	bookingService := &service.BookingService{Repo: cachedRepository}
	bookingServer := transportgrpc.NewBookingServer(
		bookingService,
		[]byte(config.AppConfig.Server.JWTAccessSecret),
	)

	addr := strings.TrimSpace(os.Getenv("BOOKING_GRPC_ADDR"))
	if addr == "" {
		addr = ":50051"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	defer listener.Close()

	grpcServer := grpc.NewServer()
	bookingpb.RegisterBookingServiceServer(grpcServer, bookingServer)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(
		"meeting_rooms.v1.BookingService",
		healthpb.HealthCheckResponse_SERVING,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	worker := &outbox.Worker{
		Repository: repository,
		Publisher:  kafkaProducer,
		Logger:     log,
	}
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- worker.Run(ctx)
	}()

	serveErr := make(chan error, 1)
	go func() {
		if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			serveErr <- err
		} else {
			serveErr <- nil
		}
	}()

	log.Info("booking gRPC service started", "address", listener.Addr().String())

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serveErr:
		stop()
		<-workerDone
		if err != nil {
			return fmt.Errorf("serve gRPC: %w", err)
		}
		return nil
	}

	stop()
	if err := <-workerDone; err != nil {
		log.Error("booking outbox worker stopped", "error", err)
	}
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	shutdownGracefully(grpcServer, config.AppConfig.Server.ShutdownTimeout)
	log.Info("booking gRPC service stopped")
	return nil
}

func shutdownGracefully(server *grpc.Server, timeout time.Duration) {
	stopped := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(stopped)
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-stopped:
	case <-timer.C:
		server.Stop()
		<-stopped
	}
}

func closeRedis(log *slog.Logger, client *redis.Client) {
	if err := client.Close(); err != nil {
		log.Error("close Redis client", "error", err)
	}
}
