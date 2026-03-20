package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Setup creates a JSON logger writing to stdout with the given level.
func Setup(level string) *slog.Logger {
	return SetupWithWriter(level, os.Stdout)
}

// SetupWithWriter creates a JSON logger writing to the provided writer.
func SetupWithWriter(level string, w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: parseLevel(level),
	}))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
