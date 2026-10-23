package integration_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	storage "vitaline-clinic/internal/storage/mongodb"
)

func TestPatientRepositoryLifecycle(t *testing.T) {
	store, ctx := database(t)
	repo := storage.NewPatientRepository(store.Database())
	p := patient()

	must(t, repo.Save(ctx, p))
	saved, err := repo.FindByID(ctx, p.ID)
	must(t, err)
	if saved.FullName != p.FullName || saved.Email != p.Email || !saved.BirthDate.Equal(p.BirthDate) {
		t.Fatalf("patient changed after saving: %+v", saved)
	}

	duplicate := *p
	duplicate.ID = uuid.New()
	if err := repo.Save(ctx, &duplicate); err == nil {
		t.Fatal("duplicate email must be rejected")
	}
	found, err := repo.FindByEmail(ctx, p.Email)
	must(t, err)
	if found.ID != p.ID {
		t.Fatal("duplicate replaced the original patient")
	}

	p.Phone = "+79007654321"
	must(t, repo.Update(ctx, p))
	saved, err = repo.FindByID(ctx, p.ID)
	must(t, err)
	if saved.Phone != p.Phone {
		t.Fatal("new phone was not saved")
	}

	must(t, repo.Delete(ctx, p.ID))
	_, err = repo.FindByID(ctx, p.ID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted patient: expected not found, got %v", err)
	}
}
