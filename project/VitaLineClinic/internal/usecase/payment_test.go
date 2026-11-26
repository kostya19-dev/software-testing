package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

type fakeGateway struct {
	created  []GatewayPaymentRequest
	status   domain.PaymentStatus
	reason   string
	err      error
	statuses int
}

func (g *fakeGateway) Name() string { return "fake" }

func (g *fakeGateway) CreatePayment(_ context.Context, rq GatewayPaymentRequest) (GatewayPayment, error) {
	if g.err != nil {
		return GatewayPayment{}, g.err
	}
	g.created = append(g.created, rq)
	return GatewayPayment{ID: "pay-" + rq.IdempotenceKey, Status: domain.PaymentPending, Amount: rq.Amount, ConfirmationURL: "https://checkout.test/" + rq.IdempotenceKey, Test: true}, nil
}

func (g *fakeGateway) GetPayment(_ context.Context, id string) (GatewayPayment, error) {
	g.statuses++
	if g.err != nil {
		return GatewayPayment{}, g.err
	}
	return GatewayPayment{ID: id, Status: g.status, CancellationReason: g.reason}, nil
}

type memoryPayments map[uuid.UUID]*domain.Payment

func (m memoryPayments) Save(_ context.Context, p *domain.Payment) error {
	if err := p.Validate(); err != nil {
		return err
	}
	m[p.ID] = p
	return nil
}

func (m memoryPayments) Update(ctx context.Context, p *domain.Payment) error { return m.Save(ctx, p) }

func (m memoryPayments) FindByID(_ context.Context, id uuid.UUID) (*domain.Payment, error) {
	if p, ok := m[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m memoryPayments) FindByAppointment(_ context.Context, id uuid.UUID) ([]*domain.Payment, error) {
	var result []*domain.Payment
	for _, p := range m {
		if p.AppointmentID == id {
			result = append(result, p)
		}
	}
	return result, nil
}

func newPaymentFixture(t *testing.T) (*PaymentService, *fakeGateway, *domain.Appointment) {
	t.Helper()
	apps := newMockAppRepo()
	appointment := &domain.Appointment{ID: uuid.New(), PatientID: uuid.New(), DoctorID: uuid.New(), Status: domain.StatusCreated, Date: time.Now().Add(48 * time.Hour)}
	apps.appointments[appointment.ID] = appointment
	gateway := &fakeGateway{status: domain.PaymentPending}
	price := domain.Money{Kopecks: 150000, Currency: "RUB"}
	service := NewPaymentService(memoryPayments{}, apps, gateway, PaymentSettings{Price: price, ReturnURL: "http://clinic.test/payments"})
	return service, gateway, appointment
}

func TestPaymentCreateSendsPriceAndIdempotenceKey(t *testing.T) {
	service, gateway, appointment := newPaymentFixture(t)

	payment, err := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if payment.Status != domain.PaymentPending || payment.Amount != "1500.00" || payment.ConfirmationURL == "" {
		t.Fatalf("unexpected payment %+v", payment)
	}
	sent := gateway.created[0]
	if sent.IdempotenceKey != payment.ID.String() {
		t.Errorf("idempotence key %q, want payment id %s", sent.IdempotenceKey, payment.ID)
	}
	if sent.Metadata["appointment_id"] != appointment.ID.String() || sent.ReturnURL != "http://clinic.test/payments?payment_id="+payment.ID.String() {
		t.Errorf("unexpected gateway request %+v", sent)
	}
}

func TestPaymentCreateReusesPendingPayment(t *testing.T) {
	service, gateway, appointment := newPaymentFixture(t)
	first, _ := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID})

	second, err := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if second.ID != first.ID || len(gateway.created) != 1 {
		t.Fatalf("pending payment must be reused: first %s second %s, provider calls %d", first.ID, second.ID, len(gateway.created))
	}
}

func TestPaymentRefreshUpdatesStatusAndForbidsSecondPayment(t *testing.T) {
	service, gateway, appointment := newPaymentFixture(t)
	created, _ := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID})

	gateway.status = domain.PaymentSucceeded
	refreshed, err := service.Refresh(context.Background(), created.ID)
	if err != nil || refreshed.Status != domain.PaymentSucceeded {
		t.Fatalf("refresh: %+v %v", refreshed, err)
	}
	if _, err := service.Refresh(context.Background(), created.ID); err != nil || gateway.statuses != 1 {
		t.Fatalf("final payment must not be requested again: calls %d, err %v", gateway.statuses, err)
	}
	if _, err := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID}); !errors.Is(err, domain.ErrAlreadyPaid) {
		t.Fatalf("second payment: got %v, want ErrAlreadyPaid", err)
	}
}

func TestPaymentCreateRejectsCancelledAppointment(t *testing.T) {
	service, gateway, appointment := newPaymentFixture(t)
	appointment.Status = domain.StatusCancelled

	if _, err := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID}); !errors.Is(err, domain.ErrInvalidStatus) {
		t.Fatalf("got %v, want ErrInvalidStatus", err)
	}
	if len(gateway.created) != 0 {
		t.Fatal("provider must not be called")
	}
}

func TestPaymentProviderFailureIsNotStored(t *testing.T) {
	service, gateway, appointment := newPaymentFixture(t)
	gateway.err = domain.ErrPaymentProvider

	if _, err := service.Create(context.Background(), CreatePaymentDTO{AppointmentID: appointment.ID}); !errors.Is(err, domain.ErrPaymentProvider) {
		t.Fatalf("got %v, want ErrPaymentProvider", err)
	}
	list, _ := service.GetByAppointment(context.Background(), appointment.ID)
	if len(list) != 0 {
		t.Fatalf("payment stored after provider failure: %+v", list)
	}
}

func TestParseMoney(t *testing.T) {
	cases := map[string]int64{"1500.00": 150000, "99.9": 9990, "7": 700, "0.05": 5}
	for input, want := range cases {
		m, err := domain.ParseMoney(input, "RUB")
		if err != nil || m.Kopecks != want {
			t.Errorf("ParseMoney(%q) = %d, %v; want %d", input, m.Kopecks, err, want)
		}
	}
	for _, bad := range []string{"", "1.234", "-5.00", "abc", ".50"} {
		if _, err := domain.ParseMoney(bad, "RUB"); err == nil {
			t.Errorf("ParseMoney(%q) must fail", bad)
		}
	}
	if got := (domain.Money{Kopecks: 150005}).Value(); got != "1500.05" {
		t.Errorf("Value() = %q", got)
	}
}
