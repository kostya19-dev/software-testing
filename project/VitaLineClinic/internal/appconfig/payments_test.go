package appconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vitaline-clinic/internal/domain"
)

func writeConfig(t *testing.T, payments string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.json")
	content := `{"database": {"driver": "mongodb", "url": "mongodb://db", "name": "clinic"}, "payments": ` + payments + `}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPaymentsModeSelectsEndpoint(t *testing.T) {
	path := writeConfig(t, `{"mode": "mock", "mock": {"base_url": "http://mock:8080/v3"}, "real": {"shop_id": "1", "secret_key": "s"}}`)

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := cfg.Payments.Endpoint().BaseURL; got != "http://mock:8080/v3" {
		t.Fatalf("mock endpoint = %q", got)
	}

	t.Setenv("VITALINE_PAYMENTS_MODE", "real")
	cfg, _, err = Load(path)
	if err != nil {
		t.Fatalf("load real: %v", err)
	}
	if got := cfg.Payments.Endpoint(); got.BaseURL != "https://api.yookassa.ru/v3" || got.ShopID != "1" {
		t.Fatalf("real endpoint = %+v", got)
	}
}

func TestRealPaymentsRequireCredentials(t *testing.T) {
	path := writeConfig(t, `{"mode": "real"}`)

	_, _, err := Load(path)
	if !errors.Is(err, domain.ErrValidation) || !strings.Contains(err.Error(), "payments.real.shop_id") {
		t.Fatalf("got %v, want validation error for payments.real.shop_id", err)
	}

	t.Setenv("YOOKASSA_SHOP_ID", "123456")
	t.Setenv("YOOKASSA_SECRET_KEY", "test_secret")
	if _, _, err := Load(path); err != nil {
		t.Fatalf("credentials from env: %v", err)
	}
}

func TestPaymentsDisabledByDefaultAndUnknownModeRejected(t *testing.T) {
	cfg, _, err := Load(writeConfig(t, `{}`))
	if err != nil || cfg.Payments.Mode != PaymentsDisabled {
		t.Fatalf("default mode = %q, err %v", cfg.Payments.Mode, err)
	}
	if _, _, err := Load(writeConfig(t, `{"mode": "sandbox"}`)); err == nil {
		t.Fatal("unknown mode must be rejected")
	}
}
