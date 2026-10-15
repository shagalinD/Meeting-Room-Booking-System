package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/shdmitri/booking-service/docs"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// @title Meeting Room Booking API
// @version 1.0
// @description HTTP API for authentication, meeting-room discovery, and booking management.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Use an access token to authorize requests to protected endpoints.
func main() {
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	logger := logger.NewLogger(config.AppConfig.Server.LogLevel)
	authConn, err := connectGRPCService("AUTH_GRPC_ADDR", "localhost:50053")
	if err != nil {
		logger.Error("failed to configure auth service connection", "error", err)
		return
	}
	defer authConn.Close()
	roomConn, err := connectGRPCService("ROOM_GRPC_ADDR", "localhost:50052")
	if err != nil {
		logger.Error("failed to configure room service connection", "error", err)
		return
	}
	defer roomConn.Close()
	bookingConn, err := connectGRPCService("BOOKING_GRPC_ADDR", "localhost:50051")
	if err != nil {
		logger.Error("failed to configure booking service connection", "error", err)
		return
	}
	defer bookingConn.Close()

	mx := http.NewServeMux()
	handlers := configHandlers(
		authConn,
		roomConn,
		bookingConn,
		logger,
	)
	middlewares := configMiddleware(logger)
	healthClients := []healthpb.HealthClient{
		healthpb.NewHealthClient(authConn),
		healthpb.NewHealthClient(roomConn),
		healthpb.NewHealthClient(bookingConn),
	}
	assignHandlers(mx, handlers, middlewares, healthClients)

	srv := &http.Server{
		Addr:              config.AppConfig.Server.Port,
		Handler:           mx,
		ReadTimeout:       config.AppConfig.Server.ReadTimeout,
		ReadHeaderTimeout: config.AppConfig.Server.ReadHeaderTimeout,
		WriteTimeout:      config.AppConfig.Server.WriteTimeout,
		IdleTimeout:       config.AppConfig.Server.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	logger.Info("API gateway started", "address", srv.Addr)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case sig := <-quit:
		logger.Info("shutting down API gateway", "signal", sig.String())
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("API gateway stopped unexpectedly", "error", err)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.AppConfig.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	logger.Info("API gateway stopped")
}

func connectGRPCService(envName, defaultAddress string) (*grpc.ClientConn, error) {
	address := os.Getenv(envName)
	if address == "" {
		address = defaultAddress
	}
	return grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		grpc.WithConnectParams(grpc.ConnectParams{MinConnectTimeout: time.Second}),
	)
}
