package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/shdmitri/booking-service/internal/api"
	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/internal/gateway"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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

func configHandlers(
	authConn grpc.ClientConnInterface,
	roomConn grpc.ClientConnInterface,
	bookingConn grpc.ClientConnInterface,
	logger *slog.Logger,
) api.Handlers {
	authHandler := &api.AuthHandler{
		S:      gateway.NewAuthClient(authConn, 0),
		Logger: logger,
	}

	roomHandler := &api.RoomHandler{
		Service: gateway.NewRoomClient(roomConn, 0),
		Logger:  logger,
	}
	bookingHandler := &api.BookingHandler{
		Service: gateway.NewBookingClient(bookingConn, 0),
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
		JWTRefreshSecret: []byte(config.AppConfig.Server.JWTRefreshSecret),
	}

	return &middleware.Middlewares{
		AuthMiddleware: authMiddleware,
	}
}

// readyHandler checks whether every upstream gRPC service is serving.
// @Summary Readiness check
// @Description Returns success only when Auth, Room, and Booking services are serving.
// @Tags system
// @Produce plain
// @Success 204 "Gateway and upstream services are ready"
// @Failure 503 "One or more upstream services are unavailable"
// @Router /ready [get]
func readyHandler(healthClients []healthpb.HealthClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, client := range healthClients {
			ctx, cancel := context.WithTimeout(r.Context(), time.Second)
			response, err := client.Check(ctx, &healthpb.HealthCheckRequest{})
			cancel()
			if err != nil || response.GetStatus() != healthpb.HealthCheckResponse_SERVING {
				http.Error(w, "upstream services are not ready", http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func assignHandlers(mx *http.ServeMux, h api.Handlers, middleware *middleware.Middlewares, healthClients []healthpb.HealthClient) {
	mx.HandleFunc("GET /health", healthCheck)
	mx.Handle("GET /ready", readyHandler(healthClients))
	mx.HandleFunc("POST /auth/register", h.AuthHandler.Register)
	mx.HandleFunc("POST /auth/login", h.AuthHandler.Login)
	mx.HandleFunc("POST /auth/refresh", h.AuthHandler.RefreshTokens)

	mx.Handle("POST /rooms/create", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.RoomHandler.CreateRoom),
	))
	mx.HandleFunc("GET /rooms/list", h.RoomHandler.ListRooms)
	mx.HandleFunc("GET /rooms/{id}", h.RoomHandler.GetRoom)

	mx.Handle("POST /bookings/create", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.BookingHandler.Create),
	))
	mx.Handle("GET /bookings/get_by_user_id", middleware.AuthMiddleware.Auth(
		http.HandlerFunc(h.BookingHandler.GetByUserId),
	))
	mx.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))
}
