package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/shdmitri/booking-service/internal/domain"
	apperrors "github.com/shdmitri/booking-service/pkg/errors"
	"github.com/shdmitri/booking-service/pkg/security"
)
var _ domain.AuthService = (*AuthService)(nil)


var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type AuthService struct {
	Repo domain.UserRepository
	JWTAccessSecret []byte
	JWTRefreshSecret []byte
}

func (s *AuthService) Register(ctx context.Context, email, password, firstName, lastName, role string) (string, string, error) {
	if err := validateRegisterInput(email, password, firstName, lastName, role); err != nil {
		return "", "", err
	}

	passwordHash, err := security.HashPassword(password)

	if err != nil {
		return "", "", &apperrors.Errors{
			Err: err,
			Code: apperrors.InternalServerError,
			Message: "error on hashing password",
		}
	}
	var userId string

	if userId, err = s.Repo.Create(ctx, &domain.UserCommand{
		Email: email,
		PasswordHash: passwordHash,
		FirstName: firstName,
		LastName: lastName,
		Role: role,
	}); err != nil {
		if _, ok := errors.AsType[*apperrors.Errors](err); ok {
			return "", "", err
		}

		return "", "", &apperrors.Errors{
			Err: err,
			Code: apperrors.InternalServerError,
			Message: "error on creating user",
		}
	}

	return GetTokens(userId, role, s.JWTAccessSecret, s.JWTRefreshSecret)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.Repo.GetByEmail(ctx, email)
	if err != nil {
		if _, ok := errors.AsType[*apperrors.Errors](err); ok {
			return "", "", err
		} else {
			return "", "", &apperrors.Errors{
				Err: err,
				Code: apperrors.InternalServerError,
				Message: "error on getting user by email",
			}
		}
	}

	if ok, err := security.VerifyPassword(password, user.PasswordHash); err != nil {
		return "", "", &apperrors.Errors{
				Err: err, 
				Code: apperrors.InternalServerError, 
				Message: "error on verifying password",
		}
	} else if !ok {
		return "", "", &apperrors.Errors{
			Err: nil, 
			Code: apperrors.InvalidCredentialsError, 
			Message: "Invalid credentials",
		}
	}

	return GetTokens(user.ID, user.Role, s.JWTAccessSecret, s.JWTRefreshSecret)
}

func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	parsedToken, err := security.ParseToken(refreshToken, s.JWTRefreshSecret)

	if err != nil {
		return "", "", &apperrors.Errors{
			Err: err,
			Code: apperrors.UnauthorizedError,
			Message: "invalid or expired token",
		}
	}

	accessToken, newRefreshToken, err := GetTokens(parsedToken.UserID, parsedToken.Role, s.JWTAccessSecret, s.JWTRefreshSecret)
	if err != nil {
		return "", "", &apperrors.Errors{
			Err: err,
			Code: apperrors.InternalServerError,
			Message: "error on generating new tokens",
		}
	}

	return accessToken, newRefreshToken, nil
}

func GetTokens(userId string, role string, accessSecret []byte, refreshSecret []byte) (string, string, error) {
	accessToken, err := security.CreateAccessToken(userId, role, accessSecret)
	if err != nil {
		return "", "", &apperrors.Errors{
			Err: err, 
			Code: apperrors.InternalServerError, 
			Message: "error on creating access token",
		}
	}

	refreshToken, err := security.CreateRefreshToken(userId, role, refreshSecret)
	if err != nil {
		return "", "", &apperrors.Errors{
			Err: err, 
			Code: apperrors.InternalServerError, 
			Message: "error on creating refresh token",
		}
	}

	return accessToken, refreshToken, nil
}

func validateRegisterInput(email, password, firstName, lastName, role string) error {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	role = strings.TrimSpace(role)

	if email == "" {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "email is required",
		}
	}

	if !emailPattern.MatchString(email) {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "email must be a valid address",
		}
	}

	if len(password) < 8 {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "password must contain at least 8 characters",
		}
	}

	if firstName == "" {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "first name is required",
		}
	}

	if lastName == "" {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "last name is required",
		}
	}

	if role == "" {
		return &apperrors.Errors{
			Code:    apperrors.ValidationError,
			Message: "role is required",
		}
	}

	return nil
}
