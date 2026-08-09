package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type Middlewares struct {
	AuthMiddleware *AuthMiddleware
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
			if aerr.Message != "" && (strings.Contains(aerr.Message, "conflict") || strings.Contains(aerr.Message, "already")) {
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

