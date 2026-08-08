package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shdmitri/booking-service/internal/api"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/internal/repository"
	"github.com/shdmitri/booking-service/internal/service"
	"github.com/shdmitri/booking-service/pkg/logger"
)

func configHandlers(dbpool *pgxpool.Pool, logger *slog.Logger) api.Handlers {
	userRepo := repository.NewUserRepository(dbpool)
	authService := &service.AuthService{
		Repo:      userRepo,
		JWTSecret: []byte(config.AppConfig.Server.JWTSecret),
	}
	authHandler := &api.AuthHandler{
		S:      authService,
		Logger: logger,
	}

	return api.Handlers{
		AuthHandler: authHandler,
	}
}


func assignHandlers(mx *http.ServeMux, h *api.AuthHandler) {
	mx.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is up and running"))
	})
	mx.HandleFunc("POST /auth/register", h.Register)
	mx.HandleFunc("POST /auth/login", h.Login)
}

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
		panic(err)
	} else {
		logger.Info("Successfully connected to PostgreSQL database!")
	}

	logger.Info("Connecting to Redis database...")
	_, err = config.ConnectRedis(&config.AppConfig.Redis)

	if err != nil {
		panic(err)
	}
	logger.Info("Successfully connected to Redis database!")

	// Starting server
	logger.Info("Starting server on port " + config.AppConfig.Server.Port)

	mx := http.NewServeMux()
	handlers := configHandlers(dbpool, logger)
	assignHandlers(mx, handlers.AuthHandler)

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
	logger.Info("Server successfully started: http://localhost" + config.AppConfig.Server.Port + "/health")

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
