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
	Tracing  TracingConfig  `json:"tracing"`
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
	Level    string `json:"level"`
	File     string `json:"file"`
	Extended bool   `json:"extended"`
}

type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	Endpoint    string  `json:"endpoint"`
	ServiceName string  `json:"service_name"`
	SampleRatio float64 `json:"sample_ratio"`
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
	if err := cfg.ValidateWithOptions(ValidateOptions(opts)); err != nil {
		return Config{}, "", err
	}

	return cfg, resolved, nil
}

func (c Config) Validate() error {
	return c.ValidateWithOptions(ValidateOptions{RequireDatabase: true})
}

type rule struct {
	failed  bool
	field   string
	message string
}

const msgEmpty = "не может быть пустым"

func blank(value string) bool { return strings.TrimSpace(value) == "" }

func (c Config) ValidateWithOptions(opts ValidateOptions) error {
	rules := []rule{{blank(c.App.Name), "app.name", msgEmpty}}
	if opts.RequireDatabase {
		rules = append(rules, c.Database.rules()...)
	}
	rules = append(rules,
		rule{blank(c.Logging.Level), "logging.level", msgEmpty},
		rule{blank(c.Logging.File), "logging.file", msgEmpty},
		rule{blank(c.API.Address), "api.address", msgEmpty},
		rule{blank(c.GUI.Address), "gui.address", msgEmpty},
		rule{blank(c.GUI.APIBaseURL), "gui.api_base_url", msgEmpty},
		rule{c.Business.AppointmentMinLeadTimeMinutes < 0, "business.appointment_min_lead_time_minutes", "не может быть отрицательным"},
		rule{c.Tracing.Enabled && blank(c.Tracing.Endpoint), "tracing.endpoint", "не может быть пустым при включённой трассировке"},
		rule{c.Tracing.SampleRatio < 0 || c.Tracing.SampleRatio > 1, "tracing.sample_ratio", "должен быть от 0 до 1"},
	)
	return firstFailure(rules)
}

func (d DatabaseConfig) rules() []rule {
	driver := strings.TrimSpace(d.Driver)
	return []rule{
		{driver == "", "database.driver", msgEmpty},
		{driver != "postgres" && driver != "mongodb", "database.driver", "поддерживаются только postgres и mongodb"},
		{blank(d.URL), "database.url", msgEmpty},
		{driver == "mongodb" && blank(d.Name), "database.name", "не может быть пустым для mongodb"},
	}
}

func firstFailure(rules []rule) error {
	for _, r := range rules {
		if r.failed {
			return domain.NewValidationError(r.field, r.message)
		}
	}
	return nil
}

func (c *Config) applyDefaults() {
	defaultString(&c.App.Name, "VitaLine Clinic")
	defaultString(&c.App.Environment, "local")
	defaultString(&c.API.Address, ":8080")
	defaultString(&c.GUI.Address, ":8090")
	defaultString(&c.GUI.APIBaseURL, "http://localhost:8080")
	defaultString(&c.Database.Driver, "mongodb")
	defaultString(&c.Database.URL, defaultDatabaseURL(c.Database.Driver))
	defaultString(&c.Database.Name, "vitaline_clinic")
	defaultString(&c.Logging.Level, "info")
	defaultString(&c.Logging.File, "./logs/clinic.log")
	defaultString(&c.Tracing.Endpoint, "localhost:4318")
	defaultString(&c.Tracing.ServiceName, "vitaline-clinic-api")
	if c.Business.AppointmentMinLeadTimeMinutes == 0 {
		c.Business.AppointmentMinLeadTimeMinutes = 30
	}
	if c.Tracing.SampleRatio == 0 {
		c.Tracing.SampleRatio = 1
	}
}

func defaultDatabaseURL(driver string) string {
	if driver == "postgres" {
		return "postgres://vitaline_app:vitaline_app@localhost:5432/vitaline_clinic?sslmode=disable"
	}
	return "mongodb://localhost:27017"
}

func defaultString(dst *string, value string) {
	if blank(*dst) {
		*dst = value
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
	overrideString(&c.Tracing.Endpoint, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	overrideString(&c.Tracing.Endpoint, os.Getenv("VITALINE_TRACING_ENDPOINT"))
	overrideString(&c.Tracing.ServiceName, os.Getenv("VITALINE_TRACING_SERVICE"))

	if err := overrideBool(&c.Logging.Extended, "VITALINE_LOG_EXTENDED"); err != nil {
		return err
	}
	if err := overrideBool(&c.Tracing.Enabled, "VITALINE_TRACING_ENABLED"); err != nil {
		return err
	}
	if v := strings.TrimSpace(os.Getenv("VITALINE_TRACING_SAMPLE_RATIO")); v != "" {
		ratio, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("parse VITALINE_TRACING_SAMPLE_RATIO: %w", err)
		}
		c.Tracing.SampleRatio = ratio
	}

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

func overrideBool(dst *bool, name string) error {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	*dst = parsed
	return nil
}

func overrideString(dst *string, value string) {
	if strings.TrimSpace(value) != "" {
		*dst = strings.TrimSpace(value)
	}
}
