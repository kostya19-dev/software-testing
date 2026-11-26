package usecase

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
)

// PaymentGateway is the contract of the external payment service as the clinic sees it.
// It is implemented by the YooKassa client; the same client talks either to the real
// API or to the mock server, depending on the configuration.
type PaymentGateway interface {
	Name() string
	CreatePayment(ctx context.Context, request GatewayPaymentRequest) (GatewayPayment, error)
	GetPayment(ctx context.Context, providerPaymentID string) (GatewayPayment, error)
}

type GatewayPaymentRequest struct {
	IdempotenceKey string
	Amount         domain.Money
	Description    string
	ReturnURL      string
	Metadata       map[string]string
}

type GatewayPayment struct {
	ID                 string
	Status             domain.PaymentStatus
	Amount             domain.Money
	ConfirmationURL    string
	CancellationReason string
	Test               bool
}

type PaymentRepository interface {
	Save(ctx context.Context, payment *domain.Payment) error
	Update(ctx context.Context, payment *domain.Payment) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	FindByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]*domain.Payment, error)
}

type CreatePaymentDTO struct {
	AppointmentID uuid.UUID
}

type PaymentResponseDTO struct {
	ID                 uuid.UUID            `json:"id"`
	AppointmentID      uuid.UUID            `json:"appointment_id"`
	Amount             string               `json:"amount"`
	Currency           string               `json:"currency"`
	Description        string               `json:"description"`
	Status             domain.PaymentStatus `json:"status"`
	Provider           string               `json:"provider"`
	ProviderPaymentID  string               `json:"provider_payment_id"`
	ConfirmationURL    string               `json:"confirmation_url,omitempty"`
	CancellationReason string               `json:"cancellation_reason,omitempty"`
	Test               bool                 `json:"test"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type IPaymentService interface {
	Create(ctx context.Context, dto CreatePaymentDTO) (PaymentResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (PaymentResponseDTO, error)
	GetByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]PaymentResponseDTO, error)
	Refresh(ctx context.Context, id uuid.UUID) (PaymentResponseDTO, error)
}

type PaymentSettings struct {
	Price     domain.Money
	ReturnURL string
}

type PaymentService struct {
	payments     PaymentRepository
	appointments AppointmentRepository
	gateway      PaymentGateway
	settings     PaymentSettings
	now          func() time.Time
}

func NewPaymentService(payments PaymentRepository, appointments AppointmentRepository, gateway PaymentGateway, settings PaymentSettings) *PaymentService {
	return &PaymentService{
		payments:     payments,
		appointments: appointments,
		gateway:      gateway,
		settings:     settings,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

// Create starts the payment of an appointment. A pending payment is reused, so a
// repeated click on "pay" does not charge the patient twice.
func (s *PaymentService) Create(ctx context.Context, dto CreatePaymentDTO) (PaymentResponseDTO, error) {
	if dto.AppointmentID == uuid.Nil {
		return PaymentResponseDTO{}, domain.NewValidationError("appointment_id", "не может быть пустым")
	}
	appointment, err := s.appointments.FindByID(ctx, dto.AppointmentID)
	if err != nil {
		return PaymentResponseDTO{}, err
	}
	if appointment.Status != domain.StatusCreated {
		return PaymentResponseDTO{}, domain.ErrInvalidStatus
	}
	existing, err := s.payments.FindByAppointment(ctx, appointment.ID)
	if err != nil {
		return PaymentResponseDTO{}, err
	}
	if reused, ok, err := reusablePayment(existing); ok || err != nil {
		return reused, err
	}

	id := uuid.New()
	description := "Оплата приёма " + appointment.Date.UTC().Format("02.01.2006 15:04") + " UTC"
	remote, err := s.gateway.CreatePayment(ctx, GatewayPaymentRequest{
		IdempotenceKey: id.String(),
		Amount:         s.settings.Price,
		Description:    description,
		ReturnURL:      returnURL(s.settings.ReturnURL, id),
		Metadata: map[string]string{
			"payment_id":     id.String(),
			"appointment_id": appointment.ID.String(),
		},
	})
	if err != nil {
		return PaymentResponseDTO{}, err
	}

	now := s.now()
	payment := &domain.Payment{
		ID:                 id,
		AppointmentID:      appointment.ID,
		Amount:             s.settings.Price,
		Description:        description,
		Status:             remote.Status,
		Provider:           s.gateway.Name(),
		ProviderPaymentID:  remote.ID,
		ConfirmationURL:    remote.ConfirmationURL,
		CancellationReason: remote.CancellationReason,
		Test:               remote.Test,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	// The provider has already created the payment: store it even if the caller gave up
	// waiting, otherwise the clinic would lose track of a payment the patient can pay.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), saveTimeout)
	defer cancel()
	if err := s.payments.Save(saveCtx, payment); err != nil {
		return PaymentResponseDTO{}, err
	}
	return toPaymentResponse(payment), nil
}

const saveTimeout = 5 * time.Second

// returnURL adds the payment id, so the page the provider returns the payer to
// knows which payment to check.
func returnURL(base string, id uuid.UUID) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := u.Query()
	query.Set("payment_id", id.String())
	u.RawQuery = query.Encode()
	return u.String()
}

func reusablePayment(payments []*domain.Payment) (PaymentResponseDTO, bool, error) {
	for _, p := range payments {
		switch p.Status {
		case domain.PaymentSucceeded:
			return PaymentResponseDTO{}, true, domain.ErrAlreadyPaid
		case domain.PaymentPending, domain.PaymentWaitingForCapture:
			return toPaymentResponse(p), true, nil
		}
	}
	return PaymentResponseDTO{}, false, nil
}

func (s *PaymentService) GetByID(ctx context.Context, id uuid.UUID) (PaymentResponseDTO, error) {
	payment, err := s.payments.FindByID(ctx, id)
	if err != nil {
		return PaymentResponseDTO{}, err
	}
	return toPaymentResponse(payment), nil
}

func (s *PaymentService) GetByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]PaymentResponseDTO, error) {
	if appointmentID == uuid.Nil {
		return nil, domain.NewValidationError("appointment_id", "не может быть пустым")
	}
	payments, err := s.payments.FindByAppointment(ctx, appointmentID)
	if err != nil {
		return nil, err
	}
	result := make([]PaymentResponseDTO, 0, len(payments))
	for _, p := range payments {
		result = append(result, toPaymentResponse(p))
	}
	return result, nil
}

// Refresh asks the provider for the current status of a payment that is not final yet.
func (s *PaymentService) Refresh(ctx context.Context, id uuid.UUID) (PaymentResponseDTO, error) {
	payment, err := s.payments.FindByID(ctx, id)
	if err != nil {
		return PaymentResponseDTO{}, err
	}
	if payment.Final() {
		return toPaymentResponse(payment), nil
	}
	remote, err := s.gateway.GetPayment(ctx, payment.ProviderPaymentID)
	if err != nil {
		return PaymentResponseDTO{}, err
	}
	if remote.Status == payment.Status {
		return toPaymentResponse(payment), nil
	}
	payment.Status = remote.Status
	payment.CancellationReason = remote.CancellationReason
	payment.UpdatedAt = s.now()
	if err := s.payments.Update(ctx, payment); err != nil {
		return PaymentResponseDTO{}, err
	}
	return toPaymentResponse(payment), nil
}

func toPaymentResponse(p *domain.Payment) PaymentResponseDTO {
	return PaymentResponseDTO{
		ID:                 p.ID,
		AppointmentID:      p.AppointmentID,
		Amount:             p.Amount.Value(),
		Currency:           p.Amount.Currency,
		Description:        p.Description,
		Status:             p.Status,
		Provider:           p.Provider,
		ProviderPaymentID:  p.ProviderPaymentID,
		ConfirmationURL:    p.ConfirmationURL,
		CancellationReason: p.CancellationReason,
		Test:               p.Test,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
