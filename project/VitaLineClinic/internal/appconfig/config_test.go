package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ConfigFileAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")

	content := `{
		"app": {"name": "Test App"},
		"api": {"address": ":9090"},
		"gui": {"address": ":9091", "api_base_url": "http://localhost:9090"},
		"database": {"driver": "postgres", "url": "postgres://from-file", "name": "from_file"},
		"logging": {"level": "warn", "file": "./logs/test.log"},
		"business": {"appointment_min_lead_time_minutes": 45}
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("VITALINE_DATABASE_DRIVER", "mongodb")
	t.Setenv("MONGODB_URI", "mongodb://from-env:27017")
	t.Setenv("MONGODB_DATABASE", "from_env")
	t.Setenv("VITALINE_APP_NAME", "Env App")
	t.Setenv("VITALINE_GUI_ADDRESS", ":9092")

	cfg, resolved, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if resolved != path {
		t.Fatalf("unexpected config path: %s", resolved)
	}
	if cfg.Database.URL != "mongodb://from-env:27017" {
		t.Fatalf("expected env override, got %s", cfg.Database.URL)
	}
	if cfg.Database.Driver != "mongodb" {
		t.Fatalf("expected driver env override, got %s", cfg.Database.Driver)
	}
	if cfg.Database.Name != "from_env" {
		t.Fatalf("expected database name env override, got %s", cfg.Database.Name)
	}
	if cfg.App.Name != "Env App" {
		t.Fatalf("expected app env override, got %s", cfg.App.Name)
	}
	if cfg.GUI.Address != ":9092" {
		t.Fatalf("expected gui env override, got %s", cfg.GUI.Address)
	}
	if cfg.Business.AppointmentMinLeadTimeMinutes != 45 {
		t.Fatalf("unexpected business parameter: %d", cfg.Business.AppointmentMinLeadTimeMinutes)
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.json")

	content := `{
		"logging": {"file": "./logs/test.log"}
	}`

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "VitaLine Clinic" {
		t.Fatalf("unexpected default app name: %s", cfg.App.Name)
	}
	if cfg.API.Address != ":8080" {
		t.Fatalf("unexpected default api address: %s", cfg.API.Address)
	}
	if cfg.GUI.Address != ":8090" {
		t.Fatalf("unexpected default gui address: %s", cfg.GUI.Address)
	}
	if cfg.GUI.APIBaseURL != "http://localhost:8080" {
		t.Fatalf("unexpected default gui api base url: %s", cfg.GUI.APIBaseURL)
	}
	if cfg.Database.Driver != "mongodb" {
		t.Fatalf("unexpected default database driver: %s", cfg.Database.Driver)
	}
	if cfg.Database.URL != "mongodb://localhost:27017" {
		t.Fatalf("unexpected default database url: %s", cfg.Database.URL)
	}
	if cfg.Database.Name != "vitaline_clinic" {
		t.Fatalf("unexpected default database name: %s", cfg.Database.Name)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("unexpected default log level: %s", cfg.Logging.Level)
	}
	if cfg.Business.AppointmentMinLeadTimeMinutes != 30 {
		t.Fatalf("unexpected default lead time: %d", cfg.Business.AppointmentMinLeadTimeMinutes)
	}
}
