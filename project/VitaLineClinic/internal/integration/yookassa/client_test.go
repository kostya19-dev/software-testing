package yookassa

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

type recorded struct {
	method, path, key, user, pass string
	body                          map[string]any
}

type provider struct {
	mu        sync.Mutex
	requests  []recorded
	responses []func(w http.ResponseWriter)
}

func (p *provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	user, pass, _ := r.BasicAuth()
	rec := recorded{method: r.Method, path: r.URL.Path, key: r.Header.Get("Idempotence-Key"), user: user, pass: pass}
	_ = json.NewDecoder(r.Body).Decode(&rec.body)
	p.requests = append(p.requests, rec)
	next := p.responses[0]
	if len(p.responses) > 1 {
		p.responses = p.responses[1:]
	}
	next(w)
}

func reply(status int, body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

const pendingPayment = `{"id":"2f1a-pay","status":"pending","paid":false,"amount":{"value":"1500.00","currency":"RUB"},
"confirmation":{"type":"redirect","confirmation_url":"https://yoomoney.ru/checkout/payments/v2/contract?orderId=2f1a-pay"},"test":true}`

func newClient(t *testing.T, p *provider) *Client {
	t.Helper()
	server := httptest.NewServer(p)
	t.Cleanup(server.Close)
	client, err := New(Config{BaseURL: server.URL + "/v3", ShopID: "100500", SecretKey: "test_secret", Backoff: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func paymentRequest() usecase.GatewayPaymentRequest {
	return usecase.GatewayPaymentRequest{
		IdempotenceKey: "key-1",
		Amount:         domain.Money{Kopecks: 150000, Currency: "RUB"},
		Description:    "Оплата приёма",
		ReturnURL:      "http://clinic.test/payments",
		Metadata:       map[string]string{"appointment_id": "a-1"},
	}
}

func TestCreatePaymentFollowsContract(t *testing.T) {
	p := &provider{responses: []func(http.ResponseWriter){reply(200, pendingPayment)}}
	client := newClient(t, p)

	got, err := client.CreatePayment(context.Background(), paymentRequest())
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got.ID != "2f1a-pay" || got.Status != domain.PaymentPending || !got.Test || got.Amount.Kopecks != 150000 {
		t.Fatalf("unexpected payment %+v", got)
	}
	rq := p.requests[0]
	expectEqual(t, "method", rq.method, http.MethodPost)
	expectEqual(t, "path", rq.path, "/v3/payments")
	expectEqual(t, "Idempotence-Key", rq.key, "key-1")
	expectEqual(t, "shop id", rq.user, "100500")
	expectEqual(t, "secret key", rq.pass, "test_secret")

	amount := rq.body["amount"].(map[string]any)
	confirmation := rq.body["confirmation"].(map[string]any)
	expectEqual(t, "amount.value", amount["value"], "1500.00")
	expectEqual(t, "amount.currency", amount["currency"], "RUB")
	expectEqual(t, "capture", rq.body["capture"], true)
	expectEqual(t, "confirmation.type", confirmation["type"], "redirect")
	expectEqual(t, "confirmation.return_url", confirmation["return_url"], "http://clinic.test/payments")
}

func expectEqual(t *testing.T, field string, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", field, got, want)
	}
}

func TestCreatePaymentRetriesWithSameKey(t *testing.T) {
	p := &provider{responses: []func(http.ResponseWriter){
		reply(500, `{"type":"error","code":"internal_server_error"}`),
		reply(202, `{"type":"error","code":"processing","retry_after":1}`),
		reply(200, pendingPayment),
	}}
	client := newClient(t, p)

	if _, err := client.CreatePayment(context.Background(), paymentRequest()); err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(p.requests) != 3 {
		t.Fatalf("requests: %d, want 3", len(p.requests))
	}
	for _, rq := range p.requests {
		if rq.key != "key-1" {
			t.Fatalf("retry changed the idempotence key: %q", rq.key)
		}
	}
}

func TestCreatePaymentDoesNotRetryInvalidRequest(t *testing.T) {
	p := &provider{responses: []func(http.ResponseWriter){
		reply(400, `{"type":"error","id":"e1","code":"invalid_request","description":"Invalid amount","parameter":"amount.value"}`),
	}}
	client := newClient(t, p)

	_, err := client.CreatePayment(context.Background(), paymentRequest())
	var providerErr *Error
	if !errors.As(err, &providerErr) || !errors.Is(err, domain.ErrPaymentProvider) {
		t.Fatalf("got %v, want *Error wrapping ErrPaymentProvider", err)
	}
	if providerErr.Code != "invalid_request" || providerErr.Parameter != "amount.value" || len(p.requests) != 1 {
		t.Fatalf("unexpected error %+v after %d requests", providerErr, len(p.requests))
	}
}

func TestGetPaymentMapsCancellation(t *testing.T) {
	p := &provider{responses: []func(http.ResponseWriter){reply(200,
		`{"id":"2f1a-pay","status":"canceled","paid":false,"amount":{"value":"1500.00","currency":"RUB"},
		"cancellation_details":{"party":"payment_network","reason":"insufficient_funds"},"test":true}`)}}
	client := newClient(t, p)

	got, err := client.GetPayment(context.Background(), "2f1a-pay")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.PaymentCanceled || got.CancellationReason != "insufficient_funds" {
		t.Fatalf("unexpected payment %+v", got)
	}
	if p.requests[0].method != http.MethodGet || p.requests[0].path != "/v3/payments/2f1a-pay" {
		t.Fatalf("unexpected request %+v", p.requests[0])
	}
}

func TestUnknownStatusIsProviderError(t *testing.T) {
	p := &provider{responses: []func(http.ResponseWriter){reply(200, `{"id":"x","status":"weird","amount":{"value":"1.00","currency":"RUB"}}`)}}
	client := newClient(t, p)

	if _, err := client.GetPayment(context.Background(), "x"); !errors.Is(err, domain.ErrPaymentProvider) {
		t.Fatalf("got %v, want ErrPaymentProvider", err)
	}
}
