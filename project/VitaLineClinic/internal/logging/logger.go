package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"vitaline-clinic/internal/appconfig"
)

type Instance struct {
	Logger *slog.Logger
	file   *os.File
}

func New(cfg appconfig.LoggingConfig, debug bool) (*Instance, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	if debug {
		level = slog.LevelDebug
	}

	if err := os.MkdirAll(filepath.Dir(cfg.File), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	writer := io.MultiWriter(os.Stdout, file)
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level:     level,
		AddSource: debug,
	})

	return &Instance{
		Logger: slog.New(handler),
		file:   file,
	}, nil
}

func (i *Instance) Close() error {
	if i == nil || i.file == nil {
		return nil
	}
	return i.file.Close()
}

func parseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", value)
	}
}
