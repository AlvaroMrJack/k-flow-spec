package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the default slog logger for CLI usage.
func Setup() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
