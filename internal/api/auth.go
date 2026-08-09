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
	S domain.AuthService
	Logger *slog.Logger
}

type errorResponse struct {
	Error string `json:"error"`
}



func containsIgnoreCase(S, sub string) bool {
	return strings.Contains(strings.ToLower(S), strings.ToLower(sub))
}

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
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	})
}