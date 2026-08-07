package domain

import "context"

type AuthService interface {
	Register(ctx context.Context, email, password, firstName, lastName, role string) (string, string, error)
	Login(ctx context.Context, email, password string) (string, string, error)
}
