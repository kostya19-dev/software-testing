package integration_test

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	storage "vitaline-clinic/internal/storage/mongodb"
)

func database(t *testing.T) (*storage.Store, context.Context) {
	t.Helper()
	if os.Getenv("LAB2_STAND") != "persistent" || os.Getenv("MONGODB_URI") != "mongodb://mongo:27017" || !regexp.MustCompile(`^lab2_[a-f0-9]{24}$`).MatchString(os.Getenv("MONGODB_DATABASE")) {
		t.Fatal("Use make test: a dedicated Lab 2 Docker stand is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	store, err := storage.Open(ctx, os.Getenv("MONGODB_URI"), os.Getenv("MONGODB_DATABASE"))
	must(t, err)
	t.Cleanup(func() { must(t, store.Close()) })
	return store, ctx
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func patient() *domain.Patient {
	id := uuid.New()
	return &domain.Patient{ID: id, FullName: "Тестовый Пациент", Phone: "+79001234567", Email: id.String() + "@example.test", BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC().Truncate(time.Millisecond)}
}

func tomorrow() time.Time { return time.Now().UTC().Truncate(24 * time.Hour).Add(34 * time.Hour) }
