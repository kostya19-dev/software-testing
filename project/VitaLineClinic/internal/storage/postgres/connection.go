package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"vitaline-clinic/internal/domain"
)

const (
	driverName         = "pgx"
	defaultPingTimeout = 5 * time.Second
)

func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, domain.NewValidationError("DATABASE_URL", "не должен быть пустым")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

func OpenFromEnv(ctx context.Context) (*sql.DB, error) {
	return Open(ctx, os.Getenv("DATABASE_URL"))
}
