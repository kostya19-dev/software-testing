// Package yookassa is the clinic's client of the YooKassa payment API v3.
//
// Contract: https://yookassa.ru/developers/api (copy of the used subset:
// lab8/contract/yookassa-v3.yaml). The client does not know whether it talks to
// the real API or to the mock server: the base URL and credentials come from the
// project configuration (payments.mode = real | mock).
package yookassa

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const (
	defaultAttempts = 3
	defaultBackoff  = 300 * time.Millisecond
	maxBodyBytes    = 1 << 20
)

type Config struct {
	Name      string
	BaseURL   string
	ShopID    string
	SecretKey string
	Timeout   time.Duration
	Attempts  int
	Backoff   time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

var _ usecase.PaymentGateway = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	if _, err := url.ParseRequestURI(cfg.BaseURL); err != nil {
		return nil, fmt.Errorf("yookassa base url: %w", err)
	}
	if cfg.ShopID == "" || cfg.SecretKey == "" {
		return nil, errors.New("yookassa: shop id and secret key are required")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Attempts <= 0 {
		cfg.Attempts = defaultAttempts
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = defaultBackoff
	}
	if cfg.Name == "" {
		cfg.Name = "yookassa"
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}, nil
}

func (c *Client) Name() string { return c.cfg.Name }

// Wire format of the API.

type amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

type confirmation struct {
	Type            string `json:"type"`
	ReturnURL       string `json:"return_url,omitempty"`
	ConfirmationURL string `json:"confirmation_url,omitempty"`
}

type createPaymentRequest struct {
	Amount       amount            `json:"amount"`
	Capture      bool              `json:"capture"`
	Confirmation confirmation      `json:"confirmation"`
	Description  string            `json:"description,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type payment struct {
	ID                  string        `json:"id"`
	Status              string        `json:"status"`
	Paid                bool          `json:"paid"`
	Amount              amount        `json:"amount"`
	Confirmation        *confirmation `json:"confirmation,omitempty"`
	Test                bool          `json:"test"`
	CancellationDetails *struct {
		Party  string `json:"party"`
		Reason string `json:"reason"`
	} `json:"cancellation_details,omitempty"`
}

type apiError struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Parameter   string `json:"parameter"`
	RetryAfter  int    `json:"retry_after"`
}

// Error is returned for every unsuccessful exchange; it wraps domain.ErrPaymentProvider.
type Error struct {
	Status      int
	Code        string
	Description string
	Parameter   string
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("yookassa: HTTP %d %s", e.Status, e.Code)
	if e.Description != "" {
		msg += ": " + e.Description
	}
	if e.Parameter != "" {
		msg += " (" + e.Parameter + ")"
	}
	return msg
}

func (e *Error) Unwrap() error { return domain.ErrPaymentProvider }

func (c *Client) CreatePayment(ctx context.Context, rq usecase.GatewayPaymentRequest) (usecase.GatewayPayment, error) {
	body, err := json.Marshal(createPaymentRequest{
		Amount:       amount{Value: rq.Amount.Value(), Currency: rq.Amount.Currency},
		Capture:      true,
		Confirmation: confirmation{Type: "redirect", ReturnURL: rq.ReturnURL},
		Description:  rq.Description,
		Metadata:     rq.Metadata,
	})
	if err != nil {
		return usecase.GatewayPayment{}, err
	}
	var p payment
	if err := c.call(ctx, http.MethodPost, "/payments", rq.IdempotenceKey, body, &p); err != nil {
		return usecase.GatewayPayment{}, err
	}
	return p.toGateway()
}

func (c *Client) GetPayment(ctx context.Context, providerPaymentID string) (usecase.GatewayPayment, error) {
	var p payment
	if err := c.call(ctx, http.MethodGet, "/payments/"+url.PathEscape(providerPaymentID), "", nil, &p); err != nil {
		return usecase.GatewayPayment{}, err
	}
	return p.toGateway()
}

// call performs the request and repeats it with the same Idempotence-Key while the
// provider answers "result unknown" (202, 429, 5xx) or the network fails, as the
// YooKassa documentation prescribes.
func (c *Client) call(ctx context.Context, method, path, key string, body []byte, out any) error {
	var lastErr error
	for attempt := 1; attempt <= c.cfg.Attempts; attempt++ {
		wait, err := c.exchange(ctx, method, path, key, body, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if wait < 0 || attempt == c.cfg.Attempts {
			break
		}
		if wait == 0 {
			wait = c.cfg.Backoff * time.Duration(attempt)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", domain.ErrPaymentProvider, ctx.Err())
		case <-time.After(wait):
		}
	}
	return lastErr
}

// exchange returns the pause before a retry, or -1 when retrying makes no sense.
func (c *Client) exchange(ctx context.Context, method, path, key string, body []byte, out any) (time.Duration, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return -1, err
	}
	request.SetBasicAuth(c.cfg.ShopID, c.cfg.SecretKey)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		request.Header.Set("Idempotence-Key", key)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", domain.ErrPaymentProvider, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
	if err != nil {
		return 0, fmt.Errorf("%w: read response: %w", domain.ErrPaymentProvider, err)
	}

	if response.StatusCode == http.StatusOK {
		if err := json.Unmarshal(raw, out); err != nil {
			return -1, fmt.Errorf("%w: decode response: %w", domain.ErrPaymentProvider, err)
		}
		return 0, nil
	}
	var e apiError
	_ = json.Unmarshal(raw, &e)
	if e.Code == "" {
		e.Code = http.StatusText(response.StatusCode)
	}
	providerErr := &Error{Status: response.StatusCode, Code: e.Code, Description: e.Description, Parameter: e.Parameter}
	return retryPause(response, e.RetryAfter), providerErr
}

// retryPause: 202 carries retry_after in milliseconds in the body, 429/5xx may carry
// the standard Retry-After header in seconds; other statuses are not retried.
func retryPause(response *http.Response, retryAfterMs int) time.Duration {
	status := response.StatusCode
	if status != http.StatusAccepted && status != http.StatusTooManyRequests && status < http.StatusInternalServerError {
		return -1
	}
	if retryAfterMs > 0 {
		return time.Duration(retryAfterMs) * time.Millisecond
	}
	if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 0
}

var statuses = map[string]domain.PaymentStatus{
	"pending":             domain.PaymentPending,
	"waiting_for_capture": domain.PaymentWaitingForCapture,
	"succeeded":           domain.PaymentSucceeded,
	"canceled":            domain.PaymentCanceled,
}

func (p payment) toGateway() (usecase.GatewayPayment, error) {
	status, ok := statuses[p.Status]
	if !ok || p.ID == "" {
		return usecase.GatewayPayment{}, fmt.Errorf("%w: unexpected payment %q with status %q", domain.ErrPaymentProvider, p.ID, p.Status)
	}
	money, err := domain.ParseMoney(p.Amount.Value, p.Amount.Currency)
	if err != nil {
		return usecase.GatewayPayment{}, fmt.Errorf("%w: amount: %w", domain.ErrPaymentProvider, err)
	}
	result := usecase.GatewayPayment{ID: p.ID, Status: status, Amount: money, Test: p.Test}
	if p.Confirmation != nil {
		result.ConfirmationURL = p.Confirmation.ConfirmationURL
	}
	if p.CancellationDetails != nil {
		result.CancellationReason = p.CancellationDetails.Reason
	}
	return result, nil
}
