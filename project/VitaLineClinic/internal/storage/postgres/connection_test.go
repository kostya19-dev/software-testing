package postgres

import (
	"context"
	"errors"
	"testing"

	"vitaline-clinic/internal/domain"
)

func TestOpen_EmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := Open(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty DATABASE_URL")
	}

	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation error, got: %v", err)
	}
	if validationErr.Field != "DATABASE_URL" {
		t.Fatalf("expected field DATABASE_URL, got: %s", validationErr.Field)
	}
}

func TestOpenFromEnv_Empty(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	_, err := OpenFromEnv(context.Background())
	if err == nil {
		t.Fatal("expected error for empty DATABASE_URL")
	}
}
