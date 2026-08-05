package api

import (
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type AppHandler struct {
	DB *pgx.Conn
	Logger *slog.Logger
}

func NewAppHandler(db *pgx.Conn, logger *slog.Logger) *AppHandler {
	return &AppHandler{db, logger}
}
