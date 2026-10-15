package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shdmitri/booking-service/internal/api"
	"github.com/shdmitri/booking-service/internal/api/middleware"
)

func TestSwaggerEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	authMiddleware := &middleware.AuthMiddleware{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	assignHandlers(mux, api.Handlers{}, &middleware.Middlewares{AuthMiddleware: authMiddleware}, nil)

	uiResponse := httptest.NewRecorder()
	mux.ServeHTTP(uiResponse, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if uiResponse.Code != http.StatusOK || !strings.Contains(uiResponse.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("Swagger UI status=%d Content-Type=%q", uiResponse.Code, uiResponse.Header().Get("Content-Type"))
	}

	specResponse := httptest.NewRecorder()
	mux.ServeHTTP(specResponse, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if specResponse.Code != http.StatusOK || !strings.Contains(specResponse.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("Swagger spec status=%d Content-Type=%q", specResponse.Code, specResponse.Header().Get("Content-Type"))
	}

	var spec struct {
		Info struct {
			Title string `json:"title"`
		} `json:"info"`
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(specResponse.Body.Bytes(), &spec); err != nil {
		t.Fatalf("decode Swagger spec: %v", err)
	}
	if spec.Info.Title != "Meeting Room Booking API" {
		t.Fatalf("Swagger title = %q", spec.Info.Title)
	}
	for _, path := range []string{
		"/health",
		"/ready",
		"/auth/register",
		"/auth/login",
		"/auth/refresh",
		"/rooms/create",
		"/rooms/list",
		"/rooms/{id}",
		"/bookings/create",
		"/bookings/get_by_user_id",
	} {
		if _, ok := spec.Paths[path]; !ok {
			t.Errorf("Swagger spec is missing route %q", path)
		}
	}
}
