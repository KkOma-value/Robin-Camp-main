package logging

import (
	"log/slog"
	"os"
)

// New returns a structured logger configured for production-friendly text output.
func New() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler)
}
