package logging

import (
	"context"
	"log/slog"

	"go.mongodb.org/mongo-driver/event"
)

const commandPreview = 512

func CommandMonitor(logger *slog.Logger) *event.CommandMonitor {
	if !logger.Enabled(context.Background(), slog.LevelDebug) {
		return nil
	}
	return &event.CommandMonitor{
		Started: func(ctx context.Context, e *event.CommandStartedEvent) {
			logger.DebugContext(ctx, "mongo command started",
				"command", e.CommandName,
				"database", e.DatabaseName,
				"request_id", e.RequestID,
				"connection", e.ConnectionID,
				"body", Truncate(e.Command.String(), commandPreview),
			)
		},
		Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) {
			logger.DebugContext(ctx, "mongo command succeeded",
				"command", e.CommandName,
				"request_id", e.RequestID,
				"duration", e.Duration,
			)
		},
		Failed: func(ctx context.Context, e *event.CommandFailedEvent) {
			logger.WarnContext(ctx, "mongo command failed",
				"command", e.CommandName,
				"request_id", e.RequestID,
				"duration", e.Duration,
				"error", e.Failure,
			)
		},
	}
}

func Truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
