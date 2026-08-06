package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shdmitri/booking-service/internal/config"
	"github.com/shdmitri/booking-service/internal/domain"
)

func TestUserRepository_CreateAndGetByEmailIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	testDBName := getenv("POSTGRES_TEST_DB", "test_db")
	postgresDBName := getenv("POSTGRES_DB", "postgres")
	adminDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("POSTGRES_USER", "postgres"),
		getenv("POSTGRES_PASSWORD", "12345678"),
		getenv("POSTGRES_HOST", "localhost"),
		getenv("POSTGRES_PORT", "5434"),
		postgresDBName,
	)

	adminPool, err := config.ConnectDB(&config.PostgresConfig{
		Host:            getenv("POSTGRES_HOST", "localhost"),
		Port:            getenv("POSTGRES_PORT", "5434"),
		User:            getenv("POSTGRES_USER", "postgres"),
		Password:        getenv("POSTGRES_PASSWORD", "12345678"),
		Name:            postgresDBName,
		ConnectTimeout:  5 * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
	}, adminDSN)
	if err != nil {
		t.Fatalf("connect to postgres admin database: %v", err)
	}
	defer adminPool.Close()

	if _, err := adminPool.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", testDBName)); err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P04" {
			t.Fatalf("create test database: %v", err)
		}
	}

	testDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getenv("POSTGRES_USER", "postgres"),
		getenv("POSTGRES_PASSWORD", "12345678"),
		getenv("POSTGRES_HOST", "localhost"),
		getenv("POSTGRES_PORT", "5434"),
		testDBName,
	)

	pool, err := config.ConnectDB(&config.PostgresConfig{
		Host:            getenv("POSTGRES_HOST", "localhost"),
		Port:            getenv("POSTGRES_PORT", "5434"),
		User:            getenv("POSTGRES_USER", "postgres"),
		Password:        getenv("POSTGRES_PASSWORD", "12345678"),
		Name:            testDBName,
		ConnectTimeout:  5 * time.Second,
		MaxOpenConns:    5,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
	}, testDSN)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer pool.Close()

	t.Cleanup(func() {
		pool.Close()
		if _, err := adminPool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", testDBName)); err != nil {
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "55006" {
				t.Logf("cleanup test database: %v", err)
			}
		}
	})

	if _, err := pool.Exec(context.Background(), `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
				CREATE TYPE user_role AS ENUM ('admin', 'user');
			END IF;
		END$$;
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			role user_role NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ
		);
	`); err != nil {
		t.Fatalf("prepare users table: %v", err)
	}

	repo := NewUserRepository(pool)
	email := fmt.Sprintf("integration-%d@example.com", time.Now().UnixNano())
	user := domain.User{
		Email:        email,
		PasswordHash: "hashed-password",
		FirstName:    "Integration",
		LastName:     "Test",
	}

	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	got, err := repo.GetByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("read created user: %v", err)
	}

	if got.ID == "" {
		t.Fatalf("expected generated user id")
	}
	if got.Email != email {
		t.Fatalf("expected email %q, got %q", email, got.Email)
	}
	if got.FirstName != user.FirstName {
		t.Fatalf("expected first name %q, got %q", user.FirstName, got.FirstName)
	}
	if got.LastName != user.LastName {
		t.Fatalf("expected last name %q, got %q", user.LastName, got.LastName)
	}
	if got.PasswordHash != user.PasswordHash {
		t.Fatalf("expected password hash %q, got %q", user.PasswordHash, got.PasswordHash)
	}

	if _, err := pool.Exec(context.Background(), `DELETE FROM users WHERE email = $1`, email); err != nil {
		t.Fatalf("cleanup created user: %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
