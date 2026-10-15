package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/shdmitri/booking-service/internal/api/dto"
	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type AuthHandler struct {
	S      domain.AuthService
	Logger *slog.Logger
}

// ErrorResponse is the JSON representation returned for API errors.
type ErrorResponse struct {
	Error string `json:"error"`
}

func containsIgnoreCase(S, sub string) bool {
	return strings.Contains(strings.ToLower(S), strings.ToLower(sub))
}

// Register creates a user account and returns access and refresh tokens.
// @Summary Register a user
// @Description Creates a user account and returns signed access and refresh tokens.
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "User registration details"
// @Success 201 {object} dto.RegisterResponse
// @Failure 400 {object} ErrorResponse "Invalid request body or registration details"
// @Failure 409 {object} ErrorResponse "An account with this email already exists"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var user dto.RegisterRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.ValidationError, Message: "invalid request body"})
		return
	}

	// obtain tokens by logging in immediately
	accessToken, refreshToken, err := h.S.Register(r.Context(), user.Email, user.Password, user.FirstName, user.LastName, "user")
	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(dto.RegisterResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Login authenticates a user and returns access and refresh tokens.
// @Summary Log in
// @Description Authenticates a user with an email address and password.
// @Tags authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login credentials"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var user dto.LoginRequest
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeErrorResponse(w, h.Logger, &apperrors.Errors{Code: apperrors.ValidationError, Message: "invalid request body"})
		return
	}
	accessToken, refreshToken, err := h.S.Login(r.Context(), user.Email, user.Password)
	if err != nil {
		writeErrorResponse(w, h.Logger, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// RefreshTokens issues a new token pair from a valid refresh token.
func (h *AuthHandler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	accessToken, refreshToken, err := h.S.RefreshTokens(r.Context(), req.RefreshToken)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	_ = json.NewEncoder(w).Encode(dto.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}
