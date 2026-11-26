package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"vitaline-clinic/internal/usecase"
)

var (
	_ usecase.AppointmentRepository = (*AppointmentRepository)(nil)
	_ usecase.VisitRepository       = (*VisitRepository)(nil)
	_ usecase.PatientRepository     = (*PatientRepository)(nil)
	_ usecase.ScheduleRepository    = (*ScheduleRepository)(nil)
	_ usecase.EmergencyRepository   = (*EmergencyRepository)(nil)
)

func integrationDSN() string {
	if dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL")); dsn != "" {
		return dsn
	}
	return strings.TrimSpace(os.Getenv("DATABASE_URL"))
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dsn := integrationDSN()
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL to run PostgreSQL integration tests")
	}

	db, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	applySchema(t, db)
	resetTables(t, db)
	return db
}

func applySchema(t *testing.T, db *sql.DB) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve current test file path")
	}

	schemaPath := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../migrations/schema.sql"))
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema file: %v", err)
	}

	statements := strings.Split(string(raw), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply schema statement %q: %v", stmt, err)
		}
	}
}

func resetTables(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.ExecContext(ctx, `TRUNCATE TABLE reschedule_offers, emergency_requests, visits, schedule_slots, appointments, patients`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
