package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shdmitri/booking-service/internal/db"
	"github.com/shdmitri/booking-service/internal/domain"
)

type UserRepository struct {
	db *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{db.New(pool)}
}

func (ur *UserRepository) Create(ctx context.Context, user domain.User) error {
	err := ur.db.CreateUser(ctx, db.CreateUserParams{
		Email: user.Email,
		PasswordHash: user.PasswordHash,
		FirstName: user.FirstName,
		LastName: user.LastName,
	})

	return err
}

func (ur *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	userRow, err := ur.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:             userRow.ID.String(),
		Email:          userRow.Email,
		PasswordHash: userRow.PasswordHash,
		FirstName:      userRow.FirstName,
		LastName:       userRow.LastName,
	}, nil
}