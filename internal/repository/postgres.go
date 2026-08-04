package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(dsn string) (*pgxpool.Pool, error) {
	dbpool, err := pgxpool.New(context.Background(), dsn)

	if err != nil {
		return nil, err
	}

	contextdb, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if err := dbpool.Ping(contextdb); err != nil {
		return nil, fmt.Errorf("timeout on pinging postgres: %w", err)
	}

	return dbpool, nil
}