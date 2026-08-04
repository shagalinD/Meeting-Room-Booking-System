package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/internal/repository"
	"github.com/shdmitri/booking-service/pkg/logger"
)

func assignUsers(mx *http.ServeMux) {
	// mx.Handle("/users/", http.StripPrefix("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request	) {

	// })))
}

func assignAuth(mx *http.ServeMux) {
	mx.Handle("/users/", http.StripPrefix("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request	) {

	})))
}

func assignHandlers(mx *http.ServeMux) {
	mx.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Server is up and running"))
	})
	assignUsers(mx)
	assignAuth(mx)
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

	_, err := repository.ConnectDB(dsn)
	
	if err != nil {
		panic(err)
	} else {
		logger.Info("Successfully connected to PostgreSQL database!")
	}

	logger.Info("Connecting to Redis database...")
	_, err = repository.ConnectRedis(&config.AppConfig.Redis)

	if err != nil {
		panic(err)
	}	
	logger.Info("Successfully connected to Redis database!")

	// Starting server
	logger.Info("Starting server on port" + config.AppConfig.Server.Port)

	mx := http.NewServeMux()
	assignHandlers(mx)

	srv := &http.Server{
		Addr:    config.AppConfig.Server.Port,
		Handler: mx,
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Ошибка при graceful shutdown:", err)
	}
	logger.Info("Сервер остановлен")
}