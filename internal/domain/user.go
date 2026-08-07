package domain

import (
	"context"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role 				 string 
}

type UserCommand struct {
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role 				 string 
}

type UserRepository interface {
	Create(ctx context.Context, user *UserCommand) (string, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
}
