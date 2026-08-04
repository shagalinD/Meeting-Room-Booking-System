package logger

import (
	"log/slog"
	"os"
)

func NewLogger(logLevel string) *slog.Logger {
	var handler slog.Handler 

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level: slog.LevelInfo,
	}

	if logLevel == "prod" || logLevel == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}