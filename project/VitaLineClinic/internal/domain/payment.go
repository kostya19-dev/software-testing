package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus mirrors the payment lifecycle of the external payment provider.
type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "PENDING"
	PaymentWaitingForCapture PaymentStatus = "WAITING_FOR_CAPTURE"
	PaymentSucceeded         PaymentStatus = "SUCCEEDED"
	PaymentCanceled          PaymentStatus = "CANCELED"
)

// ErrPaymentProvider means the external payment service failed or rejected the call.
var ErrPaymentProvider = errors.New("платёжный сервис вернул ошибку")

// ErrAlreadyPaid is returned when an appointment already has a successful payment.
var ErrAlreadyPaid = errors.New("приём уже оплачен")

// Payment is a clinic-side record of a payment made through the external provider.
type Payment struct {
	ID                 uuid.UUID
	AppointmentID      uuid.UUID
	Amount             Money
	Description        string
	Status             PaymentStatus
	Provider           string
	ProviderPaymentID  string
	ConfirmationURL    string
	CancellationReason string
	Test               bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (p *Payment) Validate() error {
	if p.AppointmentID == uuid.Nil {
		return NewValidationError("AppointmentID", "не может быть пустым")
	}
	if p.Amount.Kopecks <= 0 {
		return NewValidationError("Amount", "сумма должна быть больше нуля")
	}
	if strings.TrimSpace(p.ProviderPaymentID) == "" {
		return NewValidationError("ProviderPaymentID", "не может быть пустым")
	}
	if !IsValidPaymentStatus(p.Status) {
		return NewValidationError("Status", "недопустимый статус: "+string(p.Status))
	}
	return nil
}

// Final reports whether the provider will not change the payment status anymore.
func (p *Payment) Final() bool {
	return p.Status == PaymentSucceeded || p.Status == PaymentCanceled
}

func IsValidPaymentStatus(s PaymentStatus) bool {
	switch s {
	case PaymentPending, PaymentWaitingForCapture, PaymentSucceeded, PaymentCanceled:
		return true
	}
	return false
}

// Money is an amount in minor units (kopecks) with an ISO-4217 currency code.
type Money struct {
	Kopecks  int64
	Currency string
}

// ParseMoney parses a decimal amount such as "1500.00" or "99.9".
func ParseMoney(value, currency string) (Money, error) {
	whole, fraction, _ := strings.Cut(strings.TrimSpace(value), ".")
	if whole == "" || len(fraction) > 2 {
		return Money{}, NewValidationError("amount", "ожидается сумма вида 1500.00")
	}
	fraction += strings.Repeat("0", 2-len(fraction))
	rubles, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || rubles < 0 {
		return Money{}, NewValidationError("amount", "ожидается сумма вида 1500.00")
	}
	kopecks, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil || kopecks < 0 {
		return Money{}, NewValidationError("amount", "ожидается сумма вида 1500.00")
	}
	if strings.TrimSpace(currency) == "" {
		return Money{}, NewValidationError("currency", "не может быть пустой")
	}
	return Money{Kopecks: rubles*100 + kopecks, Currency: currency}, nil
}

// Value formats the amount as the provider expects it: "1500.00".
func (m Money) Value() string {
	return fmt.Sprintf("%d.%02d", m.Kopecks/100, m.Kopecks%100)
}
