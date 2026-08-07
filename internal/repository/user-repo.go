package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shdmitri/booking-service/internal/db"
	"github.com/shdmitri/booking-service/internal/domain"
)

type UserRepository struct {
	DB *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{db.New(pool)}
}

func ParseUserRole(role string) db.UserRole {
	switch role {
	case string(db.UserRoleAdmin):
		return db.UserRoleAdmin
	case string(db.UserRoleUser):
		return db.UserRoleUser
	default:
		return db.UserRoleUser
	}
}

func (ur *UserRepository) Create(ctx context.Context, user *domain.UserCommand) (string, error) {
	id, err := ur.DB.CreateUser(ctx, db.CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:        	ParseUserRole(user.Role),
	})

	return id.String(), MapError(err)
}

func (ur *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	userRow, err := ur.DB.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, MapError(err)
	}

	return &domain.User{
		ID:             userRow.ID.String(),
		Email:          userRow.Email,
		PasswordHash: userRow.PasswordHash,
		FirstName:      userRow.FirstName,
		LastName:       userRow.LastName,
	}, nil
}

func ParseUUID(id string) (pgtype.UUID, error) {
    var pgUUID pgtype.UUID
    if err := pgUUID.Scan(id); err != nil {
        return pgtype.UUID{}, err
    }

    return pgUUID, nil
}

func (ur *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	pgUUID, err := ParseUUID(id)

	if err != nil {
		return nil, MapError(err)
	}
	
	userRow, err := ur.DB.GetUserByID(ctx, pgUUID)
	if err != nil {
		return nil, MapError(err)
	}

	return &domain.User{
		ID:             userRow.ID.String(),
		Email:          userRow.Email,
		PasswordHash: userRow.PasswordHash,
		FirstName:      userRow.FirstName,
		LastName:       userRow.LastName,
	}, nil
}