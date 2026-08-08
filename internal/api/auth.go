package api

import (
	"encoding/json"
	"errors"
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

func writeErrorResponse(w http.ResponseWriter, logger *slog.Logger, err error) {
	var aerr *apperrors.Errors
	if errors.As(err, &aerr) {
		var status int
		switch aerr.Code {
		case apperrors.ValidationError:
			status = http.StatusBadRequest
		case apperrors.InvalidCredentialsError, apperrors.UnauthorizedError:
			status = http.StatusUnauthorized
		case apperrors.NotFoundError:
			status = http.StatusNotFound
		case apperrors.InternalServerError:
			// detect conflict-like message produced by repository mapping
			if aerr.Message != "" && (containsIgnoreCase(aerr.Message, "conflict") || containsIgnoreCase(aerr.Message, "already")) {
				status = http.StatusConflict
			} else {
				status = http.StatusInternalServerError
				logger.Error("Internal server error", "Message", aerr.Message, "Error", aerr.Err)
			}
		default:
			status = http.StatusInternalServerError
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: aerr.Message})
		logger.Debug("Error response sent", "Status", status, "Message", aerr.Message, "Error", aerr.Err)
		return
	}

	// fallback for unknown errors
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: "internal server error"})
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