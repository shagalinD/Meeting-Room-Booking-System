package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

func MapError(err error) error {
	if err == nil {
		return nil
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch pgErr.Code {
		case "23505":
			return &apperrors.Errors{
				Err:     err,
				Code:    apperrors.InternalServerError,
				Message: "Conflict with existing data",
			}
		case "23503":
			return &apperrors.Errors{
				Err:     err,
				Code:    apperrors.InternalServerError,
				Message: "Referenced record does not exist",
			}
		}
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return &apperrors.Errors{
			Err:     err,
			Code:    apperrors.NotFoundError,
			Message: "Resource not found",
		}
	}

	return &apperrors.Errors{
		Err:     err,
		Code:    apperrors.InternalServerError,
		Message: "Internal server error",
	}
}
