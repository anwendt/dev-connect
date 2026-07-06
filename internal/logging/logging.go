package logging

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
)

const redacted = "[redacted]"

// Config describes structured logger setup.
type Config struct {
	Format string
	Level  string
	Writer io.Writer
}

// New creates a metadata-only structured logger with redaction.
func New(config Config) (*slog.Logger, error) {
	writer := config.Writer
	if writer == nil {
		writer = os.Stderr
	}

	level, err := parseLevel(config.Level)
	if err != nil {
		return nil, err
	}

	options := &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactAttr,
	}

	switch config.Format {
	case "", "text":
		return slog.New(slog.NewTextHandler(writer, options)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(writer, options)), nil
	default:
		return nil, errors.New("unsupported log format")
	}
}

func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errors.New("unsupported log level")
	}
}

func redactAttr(_ []string, attr slog.Attr) slog.Attr {
	if isSensitiveKey(attr.Key) {
		return slog.String(attr.Key, redacted)
	}
	return attr
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(key)
	for _, fragment := range []string{
		"password",
		"privatekey",
		"private_key",
		"token",
		"bearer",
		"authorization",
		"credential",
		"kubeconfig",
		"secret",
	} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
