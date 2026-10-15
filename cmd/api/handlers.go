package main

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shdmitri/booking-service/internal/api"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/config"
	repository "github.com/shdmitri/booking-service/internal/repository/postgres"
	cache "github.com/shdmitri/booking-service/internal/repository/redis"
	"github.com/shdmitri/booking-service/internal/service"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// healthCheck returns a plain-text response when the API process is running.
// @Summary Health check
// @Description Returns a plain-text response when the API process is running.
// @Tags system
// @Produce plain
// @Success 200 {string} string "Server is up and running"
// @Router /health [get]
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is up and running"))
}

func configHandlers(dbpool *pgxpool.Pool, logger *slog.Logger, redisClient *redis.Client) api.Handlers {
	userRepo := repository.NewUserRepository(dbpool)
	authService := &service.AuthService{
		Repo:             userRepo,
		JWTAccessSecret:  []byte(config.AppConfig.Server.JWTAccessSecret),
		JWTRefreshSecret: []byte(config.AppConfig.Server.JWTRefreshSecret),
	}
	authHandler := &api.AuthHandler{
		S:      authService,
		Logger: logger,
	}

	roomRepo := repository.NewRoomsRepository(dbpool)
	roomService := &service.RoomService{
		Repo: roomRepo,
	}
	roomHandler := &api.RoomHandler{
		Service: roomService,
		Logger:  logger,
	}
	bookingRepo := repository.NewBookingRepository(dbpool)
	bookingCacheRepo := cache.NewBookingCacheRepository(bookingRepo, redisClient, config.AppConfig.Redis.TimeToLive, logger)
	bookingService := &service.BookingService{
		Repo: bookingCacheRepo,
	}
	bookingHandler := &api.BookingHandler{
		Service: bookingService,
		Logger:  logger,
	}

	return api.Handlers{
		AuthHandler:    authHandler,
		RoomHandler:    roomHandler,
		BookingHandler: bookingHandler,
	}
}

func configMiddleware(logger *slog.Logger) *middleware.Middlewares {
	authMiddleware := &middleware.AuthMiddleware{
		Logger:           logger,
		JWTAccessSecret:  []byte(config.AppConfig.Server.JWTAccessSecret),
		JWTRefreshSecret: []byte(config.AppConfig.Server.JWTAccessSecret),
	}

	return &middleware.Middlewares{
		AuthMiddleware: authMiddleware,
	}
}

func assignHandlers(mx *http.ServeMux, h api.Handlers, middleware *middleware.Middlewares) {
	mx.HandleFunc("GET /health", healthCheck)
	mx.HandleFunc("POST /auth/register", h.AuthHandler.Register)
	mx.HandleFunc("POST /auth/login", h.AuthHandler.Login)

	mx.Handle("POST /rooms/create", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.RoomHandler.CreateRoom),
	))
	mx.HandleFunc("GET /rooms/list", h.RoomHandler.ListRooms)

	mx.Handle("POST /bookings/create", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.BookingHandler.Create),
	))
	mx.Handle("GET /bookings/get_by_user_id", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.BookingHandler.GetByUserId),
	))
	mx.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
}
