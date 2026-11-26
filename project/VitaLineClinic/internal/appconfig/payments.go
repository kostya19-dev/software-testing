package appconfig

import (
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	PaymentsDisabled = "disabled"
	PaymentsMock     = "mock"
	PaymentsReal     = "real"
)

type PaymentsConfig struct {
	Mode             string           `json:"mode"`
	AppointmentPrice string           `json:"appointment_price"`
	Currency         string           `json:"currency"`
	ReturnURL        string           `json:"return_url"`
	TimeoutSeconds   int              `json:"timeout_seconds"`
	Mock             ProviderEndpoint `json:"mock"`
	Real             ProviderEndpoint `json:"real"`
}

type ProviderEndpoint struct {
	BaseURL   string `json:"base_url"`
	ShopID    string `json:"shop_id"`
	SecretKey string `json:"secret_key"`
}

func (p PaymentsConfig) Endpoint() ProviderEndpoint {
	if p.Mode == PaymentsReal {
		return p.Real
	}
	return p.Mock
}

func (p *PaymentsConfig) applyDefaults() {
	defaultString(&p.Mode, PaymentsDisabled)
	defaultString(&p.AppointmentPrice, "1500.00")
	defaultString(&p.Currency, "RUB")
	defaultString(&p.ReturnURL, "http://localhost:8090/payments")
	defaultString(&p.Mock.BaseURL, "http://localhost:9095/v3")
	defaultString(&p.Mock.ShopID, "100500")
	defaultString(&p.Mock.SecretKey, "test_mock_secret_key")
	defaultString(&p.Real.BaseURL, "https://api.yookassa.ru/v3")
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 10
	}
}

func (p *PaymentsConfig) applyEnvOverrides() error {
	overrideString(&p.Mode, os.Getenv("VITALINE_PAYMENTS_MODE"))
	overrideString(&p.AppointmentPrice, os.Getenv("VITALINE_APPOINTMENT_PRICE"))
	overrideString(&p.ReturnURL, os.Getenv("VITALINE_PAYMENTS_RETURN_URL"))
	overrideString(&p.Mock.BaseURL, os.Getenv("VITALINE_PAYMENTS_MOCK_URL"))
	overrideString(&p.Mock.ShopID, os.Getenv("VITALINE_PAYMENTS_MOCK_SHOP_ID"))
	overrideString(&p.Mock.SecretKey, os.Getenv("VITALINE_PAYMENTS_MOCK_SECRET_KEY"))
	overrideString(&p.Real.BaseURL, os.Getenv("VITALINE_PAYMENTS_REAL_URL"))
	overrideString(&p.Real.ShopID, os.Getenv("YOOKASSA_SHOP_ID"))
	overrideString(&p.Real.SecretKey, os.Getenv("YOOKASSA_SECRET_KEY"))
	if v := strings.TrimSpace(os.Getenv("VITALINE_PAYMENTS_TIMEOUT_SECONDS")); v != "" {
		seconds, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		p.TimeoutSeconds = seconds
	}
	return nil
}

func (p PaymentsConfig) rules() []rule {
	mode := p.Mode
	rules := []rule{
		{mode != PaymentsDisabled && mode != PaymentsMock && mode != PaymentsReal, "payments.mode", "поддерживаются mock, real и disabled"},
	}
	if mode == PaymentsDisabled {
		return rules
	}
	endpoint := p.Endpoint()
	field := "payments." + mode
	return append(rules,
		rule{!isHTTPURL(endpoint.BaseURL), field + ".base_url", "ожидается http(s) URL"},
		rule{blank(endpoint.ShopID), field + ".shop_id", msgEmpty + " (для real — YOOKASSA_SHOP_ID)"},
		rule{blank(endpoint.SecretKey), field + ".secret_key", msgEmpty + " (для real — YOOKASSA_SECRET_KEY)"},
		rule{!isHTTPURL(p.ReturnURL), "payments.return_url", "ожидается http(s) URL"},
		rule{blank(p.AppointmentPrice), "payments.appointment_price", msgEmpty},
		rule{p.TimeoutSeconds <= 0, "payments.timeout_seconds", "должен быть больше нуля"},
	)
}

func isHTTPURL(value string) bool {
	u, err := url.Parse(strings.TrimSpace(value))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
