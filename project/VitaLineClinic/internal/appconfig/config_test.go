package appconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vitaline-clinic/internal/domain"
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

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
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

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, tc := range []struct {
		name      string
		got, want any
	}{
		{"app name", cfg.App.Name, "VitaLine Clinic"},
		{"api address", cfg.API.Address, ":8080"},
		{"gui address", cfg.GUI.Address, ":8090"},
		{"gui api base url", cfg.GUI.APIBaseURL, "http://localhost:8080"},
		{"database driver", cfg.Database.Driver, "mongodb"},
		{"database url", cfg.Database.URL, "mongodb://localhost:27017"},
		{"database name", cfg.Database.Name, "vitaline_clinic"},
		{"log level", cfg.Logging.Level, "info"},
		{"lead time", cfg.Business.AppointmentMinLeadTimeMinutes, 30},
		{"tracing sample ratio", cfg.Tracing.SampleRatio, 1.0},
	} {
		if tc.got != tc.want {
			t.Errorf("unexpected default %s: %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestDefaultDatabaseURLDependsOnDriver(t *testing.T) {
	cfg := Config{Database: DatabaseConfig{Driver: "postgres"}}
	cfg.applyDefaults()
	if !strings.HasPrefix(cfg.Database.URL, "postgres://") {
		t.Fatalf("expected postgres default url, got %s", cfg.Database.URL)
	}
}

func TestValidateReportsFirstInvalidField(t *testing.T) {
	valid := func() Config {
		cfg := Config{}
		cfg.applyDefaults()
		return cfg
	}
	for _, tc := range []struct {
		field  string
		mutate func(*Config)
	}{
		{"app.name", func(c *Config) { c.App.Name = " " }},
		{"database.driver", func(c *Config) { c.Database.Driver = "" }},
		{"database.driver", func(c *Config) { c.Database.Driver = "sqlite" }},
		{"database.url", func(c *Config) { c.Database.URL = "" }},
		{"database.name", func(c *Config) { c.Database.Name = "" }},
		{"logging.level", func(c *Config) { c.Logging.Level = "" }},
		{"logging.file", func(c *Config) { c.Logging.File = "" }},
		{"api.address", func(c *Config) { c.API.Address = "" }},
		{"gui.address", func(c *Config) { c.GUI.Address = "" }},
		{"gui.api_base_url", func(c *Config) { c.GUI.APIBaseURL = "" }},
		{"business.appointment_min_lead_time_minutes", func(c *Config) { c.Business.AppointmentMinLeadTimeMinutes = -1 }},
		{"tracing.endpoint", func(c *Config) { c.Tracing.Enabled, c.Tracing.Endpoint = true, "" }},
		{"tracing.sample_ratio", func(c *Config) { c.Tracing.SampleRatio = 1.5 }},
	} {
		cfg := valid()
		tc.mutate(&cfg)
		err := cfg.Validate()
		var validation *domain.ValidationError
		if !errors.As(err, &validation) || validation.Field != tc.field {
			t.Errorf("expected validation error for %s, got %v", tc.field, err)
		}
	}

	cfg := valid()
	cfg.Database.Driver = ""
	if err := cfg.ValidateWithOptions(ValidateOptions{}); err != nil {
		t.Errorf("database must not be checked without RequireDatabase: %v", err)
	}
	if err := valid().Validate(); err != nil {
		t.Errorf("default config must be valid: %v", err)
	}
}
