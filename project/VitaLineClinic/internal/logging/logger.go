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
	// Log files are bind-mounted (docker-compose, lab4/lab6 stands) and read on the host.
	if err := os.MkdirAll(filepath.Dir(cfg.File), 0o755); err != nil { //nolint:gosec // G301: host must read logs
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) //nolint:gosec // G302: host must read logs
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	logger, err := NewLogger(io.MultiWriter(os.Stdout, file), cfg, debug)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &Instance{Logger: logger, file: file}, nil
}

func NewLogger(w io.Writer, cfg appconfig.LoggingConfig, debug bool) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	debug = debug || cfg.Extended
	if debug {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:     level,
		AddSource: debug,
	})), nil
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
