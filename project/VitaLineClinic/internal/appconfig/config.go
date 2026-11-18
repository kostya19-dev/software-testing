package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"vitaline-clinic/internal/domain"
)

type Config struct {
	App      AppConfig      `json:"app"`
	API      APIConfig      `json:"api"`
	GUI      GUIConfig      `json:"gui"`
	Database DatabaseConfig `json:"database"`
	Logging  LoggingConfig  `json:"logging"`
	Business BusinessConfig `json:"business"`
}

type AppConfig struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

type APIConfig struct {
	Address string `json:"address"`
}

type GUIConfig struct {
	Address    string `json:"address"`
	APIBaseURL string `json:"api_base_url"`
}

type DatabaseConfig struct {
	Driver string `json:"driver"`
	URL    string `json:"url"`
	Name   string `json:"name"`
}

type LoggingConfig struct {
	Level string `json:"level"`
	File  string `json:"file"`
}

type BusinessConfig struct {
	AppointmentMinLeadTimeMinutes int `json:"appointment_min_lead_time_minutes"`
}

var defaultSearchPaths = []string{
	"./config/app.json",
	"../config/app.json",
}

type LoadOptions struct {
	RequireDatabase bool
}

type ValidateOptions struct {
	RequireDatabase bool
}

func Load(path string) (Config, string, error) {
	return LoadWithOptions(path, LoadOptions{RequireDatabase: true})
}

func LoadWithOptions(path string, opts LoadOptions) (Config, string, error) {
	resolved, err := resolvePath(path)
	if err != nil {
		return Config{}, "", err
	}

	raw, err := os.ReadFile(resolved)
	if err != nil {
		return Config{}, "", fmt.Errorf("read config file: %w", err)
	}

	cfg := Config{}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, "", fmt.Errorf("decode config file: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.applyEnvOverrides(); err != nil {
		return Config{}, "", err
	}
	if err := cfg.ValidateWithOptions(ValidateOptions{RequireDatabase: opts.RequireDatabase}); err != nil {
		return Config{}, "", err
	}

	return cfg, resolved, nil
}

func (c Config) Validate() error {
	return c.ValidateWithOptions(ValidateOptions{RequireDatabase: true})
}

func (c Config) ValidateWithOptions(opts ValidateOptions) error {
	if strings.TrimSpace(c.App.Name) == "" {
		return domain.NewValidationError("app.name", "не может быть пустым")
	}
	if opts.RequireDatabase {
		driver := strings.TrimSpace(c.Database.Driver)
		if driver == "" {
			return domain.NewValidationError("database.driver", "не может быть пустым")
		}
		if driver != "postgres" && driver != "mongodb" {
			return domain.NewValidationError("database.driver", "поддерживаются только postgres и mongodb")
		}
		if strings.TrimSpace(c.Database.URL) == "" {
			return domain.NewValidationError("database.url", "не может быть пустым")
		}
		if driver == "mongodb" && strings.TrimSpace(c.Database.Name) == "" {
			return domain.NewValidationError("database.name", "не может быть пустым для mongodb")
		}
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		return domain.NewValidationError("logging.level", "не может быть пустым")
	}
	if strings.TrimSpace(c.Logging.File) == "" {
		return domain.NewValidationError("logging.file", "не может быть пустым")
	}
	if strings.TrimSpace(c.API.Address) == "" {
		return domain.NewValidationError("api.address", "не может быть пустым")
	}
	if strings.TrimSpace(c.GUI.Address) == "" {
		return domain.NewValidationError("gui.address", "не может быть пустым")
	}
	if strings.TrimSpace(c.GUI.APIBaseURL) == "" {
		return domain.NewValidationError("gui.api_base_url", "не может быть пустым")
	}
	if c.Business.AppointmentMinLeadTimeMinutes < 0 {
		return domain.NewValidationError("business.appointment_min_lead_time_minutes", "не может быть отрицательным")
	}
	return nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.App.Name) == "" {
		c.App.Name = "VitaLine Clinic"
	}
	if strings.TrimSpace(c.App.Environment) == "" {
		c.App.Environment = "local"
	}
	if strings.TrimSpace(c.API.Address) == "" {
		c.API.Address = ":8080"
	}
	if strings.TrimSpace(c.GUI.Address) == "" {
		c.GUI.Address = ":8090"
	}
	if strings.TrimSpace(c.GUI.APIBaseURL) == "" {
		c.GUI.APIBaseURL = "http://localhost:8080"
	}
	if strings.TrimSpace(c.Database.Driver) == "" {
		c.Database.Driver = "mongodb"
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		if c.Database.Driver == "postgres" {
			c.Database.URL = "postgres://vitaline_app:vitaline_app@localhost:5432/vitaline_clinic?sslmode=disable"
		} else {
			c.Database.URL = "mongodb://localhost:27017"
		}
	}
	if strings.TrimSpace(c.Database.Name) == "" {
		c.Database.Name = "vitaline_clinic"
	}
	if strings.TrimSpace(c.Logging.Level) == "" {
		c.Logging.Level = "info"
	}
	if strings.TrimSpace(c.Logging.File) == "" {
		c.Logging.File = "./logs/clinic.log"
	}
	if c.Business.AppointmentMinLeadTimeMinutes == 0 {
		c.Business.AppointmentMinLeadTimeMinutes = 30
	}
}

func (c *Config) applyEnvOverrides() error {
	overrideString(&c.App.Name, os.Getenv("VITALINE_APP_NAME"))
	overrideString(&c.Database.Driver, os.Getenv("VITALINE_DATABASE_DRIVER"))
	overrideString(&c.Database.URL, os.Getenv("DATABASE_URL"))
	overrideString(&c.Database.URL, os.Getenv("VITALINE_DATABASE_URL"))
	overrideString(&c.Database.URL, os.Getenv("MONGODB_URI"))
	overrideString(&c.Database.Name, os.Getenv("VITALINE_DATABASE_NAME"))
	overrideString(&c.Database.Name, os.Getenv("MONGODB_DATABASE"))
	overrideString(&c.API.Address, os.Getenv("VITALINE_HTTP_ADDRESS"))
	overrideString(&c.GUI.Address, os.Getenv("VITALINE_GUI_ADDRESS"))
	overrideString(&c.GUI.APIBaseURL, os.Getenv("VITALINE_GUI_API_BASE_URL"))
	overrideString(&c.Logging.Level, os.Getenv("VITALINE_LOG_LEVEL"))
	overrideString(&c.Logging.File, os.Getenv("VITALINE_LOG_FILE"))
	overrideString(&c.App.Environment, os.Getenv("VITALINE_ENV"))

	if v := strings.TrimSpace(os.Getenv("VITALINE_APPOINTMENT_MIN_LEAD_TIME_MINUTES")); v != "" {
		minutes, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("parse VITALINE_APPOINTMENT_MIN_LEAD_TIME_MINUTES: %w", err)
		}
		c.Business.AppointmentMinLeadTimeMinutes = minutes
	}

	return nil
}

func resolvePath(path string) (string, error) {
	candidates := defaultSearchPaths
	if strings.TrimSpace(path) != "" {
		candidates = []string{path}
	}

	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(candidate), nil
		}
	}

	if strings.TrimSpace(path) != "" {
		return "", fmt.Errorf("config file not found: %s", path)
	}
	return "", fmt.Errorf("config file not found in default locations: %s", strings.Join(defaultSearchPaths, ", "))
}

func overrideString(dst *string, value string) {
	if strings.TrimSpace(value) != "" {
		*dst = strings.TrimSpace(value)
	}
}
