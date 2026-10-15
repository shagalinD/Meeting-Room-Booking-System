package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/shdmitri/booking-service/docs"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/pkg/logger"
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
	// Config
	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	// Logger
	logger := logger.NewLogger(config.AppConfig.Server.LogLevel)
	logger.Info("Connecting to PostgreSQL database...")

	// DBs connection
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		config.AppConfig.Postgres.Host,
		config.AppConfig.Postgres.Port,
		config.AppConfig.Postgres.Name,
		config.AppConfig.Postgres.User,
		config.AppConfig.Postgres.Password,
	)

	dbpool, err := config.ConnectDB(&config.AppConfig.Postgres, dsn)

	if err != nil {
		logger.Error(err.Error())
		panic(err)
	} else {
		logger.Info("Successfully connected to PostgreSQL database!")
	}

	logger.Info("Connecting to Redis database...")
	redisClient, err := config.ConnectRedis(&config.AppConfig.Redis)

	if err != nil {
		panic(err)
	}
	logger.Info("Successfully connected to Redis database!")

	// Starting server
	logger.Info("Starting server on port " + config.AppConfig.Server.Port)

	mx := http.NewServeMux()
	handlers := configHandlers(dbpool, logger, redisClient)
	middlewares := configMiddleware(logger)
	assignHandlers(mx, handlers, middlewares)

	srv := &http.Server{
		Addr:              config.AppConfig.Server.Port,
		Handler:           mx,
		ReadTimeout:       config.AppConfig.Server.ReadTimeout,
		ReadHeaderTimeout: config.AppConfig.Server.ReadHeaderTimeout,
		WriteTimeout:      config.AppConfig.Server.WriteTimeout,
		IdleTimeout:       config.AppConfig.Server.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска: %v", err)
		} else {
		}
	}()
	logger.Info("Server successfully started: https://localhost" + "/health")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Получен сигнал для завершения! Закрываем сервер...")

	ctx, cancel := context.WithTimeout(context.Background(), config.AppConfig.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Ошибка при graceful shutdown:", err)
	}
	logger.Info("Сервер остановлен")
}
