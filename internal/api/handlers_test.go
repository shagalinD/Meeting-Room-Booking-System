package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shdmitri/booking-service/internal/api/middleware"
	"github.com/shdmitri/booking-service/internal/domain"
	"github.com/shdmitri/booking-service/pkg/security"
)

type authServiceStub struct {
	registerArgs []string
	loginArgs    []string
	access       string
	refresh      string
	err          error
}

func (s *authServiceStub) Register(_ context.Context, email, password, firstName, lastName, role string) (string, string, error) {
	s.registerArgs = []string{email, password, firstName, lastName, role}
	return s.access, s.refresh, s.err
}

func (s *authServiceStub) Login(_ context.Context, email, password string) (string, string, error) {
	s.loginArgs = []string{email, password}
	return s.access, s.refresh, s.err
}

func (s *authServiceStub) RefreshTokens(context.Context, string) (string, string, error) {
	return s.access, s.refresh, s.err
}

type roomServiceStub struct {
	created *domain.Room
	filter  domain.RoomFilter
	id      string
	rooms   []*domain.Room
	err     error
	calls   int
}

func (s *roomServiceStub) Create(room *domain.Room) (string, error) {
	s.calls++
	s.created = room
	return s.id, s.err
}

func (*roomServiceStub) GetByID(string) (*domain.Room, error) {
	panic("unexpected GetByID call")
}

func (s *roomServiceStub) List(filter domain.RoomFilter) ([]*domain.Room, error) {
	s.calls++
	s.filter = filter
	return s.rooms, s.err
}

func (*roomServiceStub) Update(*domain.Room) error { panic("unexpected Update call") }
func (*roomServiceStub) Delete(string) error       { panic("unexpected Delete call") }

type bookingServiceStub struct {
	created  *domain.Booking
	userID   string
	id       string
	bookings []*domain.Booking
	err      error
}

func (s *bookingServiceStub) Create(_ context.Context, booking *domain.Booking) (string, error) {
	s.created = booking
	return s.id, s.err
}

func (s *bookingServiceStub) GetByUserId(_ context.Context, userID string) ([]*domain.Booking, error) {
	s.userID = userID
	return s.bookings, s.err
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestAuthHandlerRegisterAndLogin(t *testing.T) {
	service := &authServiceStub{access: "access-token", refresh: "refresh-token"}
	handler := &AuthHandler{S: service, Logger: testLogger()}

	t.Run("register creates user role and returns tokens", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(
			`{"email":"ada@example.com","password":"password123","first_name":"Ada","last_name":"Lovelace"}`,
		))
		response := httptest.NewRecorder()
		handler.Register(response, request)

		if response.Code != http.StatusCreated || response.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("status=%d Content-Type=%q", response.Code, response.Header().Get("Content-Type"))
		}
		if strings.Join(service.registerArgs, "|") != "ada@example.com|password123|Ada|Lovelace|user" {
			t.Fatalf("Register() args = %v", service.registerArgs)
		}
		var body struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.AccessToken != service.access || body.RefreshToken != service.refresh {
			t.Fatalf("response tokens = (%q, %q)", body.AccessToken, body.RefreshToken)
		}
	})

	t.Run("invalid JSON is rejected", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.Register(response, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader("{")))
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid request body") {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
	})

	t.Run("service error is translated to an HTTP error", func(t *testing.T) {
		service.err = errors.New("service failure")
		response := httptest.NewRecorder()
		handler.Login(response, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(
			`{"email":"ada@example.com","password":"password123"}`,
		)))
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "internal server error") {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
		service.err = nil
	})

	t.Run("login returns tokens", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.Login(response, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(
			`{"email":"ada@example.com","password":"password123"}`,
		)))
		if response.Code != http.StatusOK || strings.Join(service.loginArgs, "|") != "ada@example.com|password123" {
			t.Fatalf("status=%d args=%v", response.Code, service.loginArgs)
		}
	})
}

func TestRoomHandlerListRooms(t *testing.T) {
	service := &roomServiceStub{rooms: []*domain.Room{{ID: "room-1", Name: "Board room", Description: "Large room", Capacity: 8}}}
	handler := &RoomHandler{Service: service, Logger: testLogger()}

	request := httptest.NewRequest(http.MethodGet, "/rooms/list?min_capacity=2&max_capacity=10", nil)
	response := httptest.NewRecorder()
	handler.ListRooms(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d Content-Type=%q", response.Code, response.Header().Get("Content-Type"))
	}
	if service.filter != (domain.RoomFilter{MinCapacity: 2, MaxCapacity: 10}) {
		t.Fatalf("service filter = %+v", service.filter)
	}
	var body struct {
		Rooms []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Capacity    int    `json:"capacity"`
		} `json:"rooms"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Rooms) != 1 || body.Rooms[0].ID != "room-1" || body.Rooms[0].Capacity != 8 {
		t.Fatalf("rooms response = %+v", body.Rooms)
	}
}

func TestRoomHandlerListRoomsRejectsInvalidCapacity(t *testing.T) {
	service := &roomServiceStub{}
	handler := &RoomHandler{Service: service, Logger: testLogger()}
	response := httptest.NewRecorder()
	handler.ListRooms(response, httptest.NewRequest(http.MethodGet, "/rooms/list?max_capacity=many", nil))

	if response.Code != http.StatusUnauthorized || service.calls != 0 {
		t.Fatalf("status=%d service calls=%d, want 401 and no service call", response.Code, service.calls)
	}
}

func TestRoomHandlerCreateRoomRequiresAdminRole(t *testing.T) {
	service := &roomServiceStub{id: "room-1"}
	handler := &RoomHandler{Service: service, Logger: testLogger()}

	t.Run("non-admin is forbidden", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := withRole(httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(
			`{"name":"Board room","description":"Large room","capacity":8}`,
		)), "user")
		handler.CreateRoom(response, request)
		if response.Code != http.StatusUnauthorized || service.calls != 0 {
			t.Fatalf("status=%d service calls=%d", response.Code, service.calls)
		}
	})

	t.Run("admin creates room", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := withRole(httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(
			`{"name":"Board room","description":"Large room","capacity":8}`,
		)), "admin")
		handler.CreateRoom(response, request)
		if response.Code != http.StatusCreated || service.calls != 1 {
			t.Fatalf("status=%d service calls=%d", response.Code, service.calls)
		}
		if service.created == nil || service.created.Name != "Board room" || service.created.Capacity != 8 {
			t.Fatalf("created room = %+v", service.created)
		}
	})

	t.Run("invalid JSON returns a documented error", func(t *testing.T) {
		response := httptest.NewRecorder()
		request := withRole(httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader("{")), "admin")
		handler.CreateRoom(response, request)
		if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/json" {
			t.Fatalf("status=%d Content-Type=%q", response.Code, response.Header().Get("Content-Type"))
		}
	})
}

func TestBookingHandlerCreateUsesAuthenticatedUser(t *testing.T) {
	start := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	service := &bookingServiceStub{id: "booking-1"}
	handler := &BookingHandler{Service: service, Logger: testLogger()}
	body := `{"room_id":"room-1","title":"Planning","start_time":"2026-10-15T10:00:00Z","end_time":"2026-10-15T11:00:00Z"}`
	request := withUserID(httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(body)), "user-1")
	response := httptest.NewRecorder()
	handler.Create(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" || service.created == nil {
		t.Fatalf("status=%d Content-Type=%q created=%+v body=%q", response.Code, response.Header().Get("Content-Type"), service.created, response.Body.String())
	}
	if service.created.UserId != "user-1" || service.created.RoomId != "room-1" || !service.created.StartTime.Equal(start) {
		t.Fatalf("created booking = %+v", service.created)
	}
	var result map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result["booking_id"] != "booking-1" {
		t.Fatalf("response=%v decode error=%v", result, err)
	}

	badRequest := withUserID(httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader("{")), "user-1")
	badResponse := httptest.NewRecorder()
	handler.Create(badResponse, badRequest)
	if badResponse.Code != http.StatusBadRequest || badResponse.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("invalid body status=%d Content-Type=%q", badResponse.Code, badResponse.Header().Get("Content-Type"))
	}
}

func TestBookingHandlerGetByUserIDUsesAuthenticatedUser(t *testing.T) {
	service := &bookingServiceStub{bookings: []*domain.Booking{{Id: "booking-1", UserId: "user-1", RoomId: "room-1", Title: "Planning"}}}
	handler := &BookingHandler{Service: service, Logger: testLogger()}
	request := withUserID(httptest.NewRequest(http.MethodGet, "/bookings/get_by_user_id", nil), "user-1")
	response := httptest.NewRecorder()
	handler.GetByUserId(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" || service.userID != "user-1" {
		t.Fatalf("status=%d Content-Type=%q service user ID=%q", response.Code, response.Header().Get("Content-Type"), service.userID)
	}
	var bookings []map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &bookings); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(bookings) != 1 || bookings[0]["id"] != "booking-1" || bookings[0]["room_id"] != "room-1" {
		t.Fatalf("bookings response = %v", bookings)
	}
}

func withRole(request *http.Request, role string) *http.Request {
	return withClaims(request, "user-1", role)
}

func withUserID(request *http.Request, userID string) *http.Request {
	return withClaims(request, userID, "user")
}

func withClaims(request *http.Request, userID, role string) *http.Request {
	const secret = "api-test-secret"
	token, err := security.CreateAccessToken(userID, role, []byte(secret))
	if err != nil {
		panic(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	(&middleware.AuthMiddleware{Logger: testLogger(), JWTAccessSecret: []byte(secret)}).Auth(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*request = *r
	})).ServeHTTP(recorder, request)
	return request
}
