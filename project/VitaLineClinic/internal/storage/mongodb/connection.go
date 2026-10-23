package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"vitaline-clinic/internal/domain"
)

const defaultPingTimeout = 5 * time.Second

type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

func Open(ctx context.Context, uri, databaseName string) (*Store, error) {
	if strings.TrimSpace(uri) == "" {
		return nil, domain.NewValidationError("DATABASE_URL", "не должен быть пустым")
	}
	if strings.TrimSpace(databaseName) == "" {
		return nil, domain.NewValidationError("database.name", "не должен быть пустым")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect mongodb: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, defaultPingTimeout)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &Store{
		client: client,
		db:     client.Database(databaseName),
	}, nil
}

func (s *Store) Database() *mongo.Database {
	return s.db
}

func (s *Store) Ping(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return s.client.Ping(ctx, nil)
}

func (s *Store) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultPingTimeout)
	defer cancel()
	return s.client.Disconnect(ctx)
}
