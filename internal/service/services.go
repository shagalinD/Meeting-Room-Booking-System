package service

import (
	"errors"

	apperrors "github.com/shdmitri/booking-service/pkg/errors"
)

type Services struct {
	AuthService *AuthService
}

func serviceError(err error, message string) error {
	if errors.Is(err, &apperrors.Errors{}) {
		return err 
	} 

	return &apperrors.Errors{
		Message: message,
		Code: apperrors.ValidationError,
		Err: err,
	}
}

