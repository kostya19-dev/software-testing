package http

import (
	"context"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

type createPaymentRequest struct {
	AppointmentID uuid.UUID `json:"appointment_id"`
}

func (s *Server) handlePayments(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !s.paymentsEnabled(w) {
		return
	}
	ctx, cancel := paymentContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		appointmentID, err := parseUUIDQuery(r, "appointment_id")
		if err != nil {
			writeAPIError(w, err)
			return
		}
		payments, err := s.services.Payments.GetByAppointment(ctx, appointmentID)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, payments)
	case nethttp.MethodPost:
		var req createPaymentRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, domain.NewValidationError("body", err.Error()))
			return
		}
		payment, err := s.services.Payments.Create(ctx, usecase.CreatePaymentDTO{AppointmentID: req.AppointmentID})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, payment)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handlePaymentByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	if !s.paymentsEnabled(w) {
		return
	}
	if strings.HasSuffix(strings.TrimRight(r.URL.Path, "/"), "/refresh") {
		s.handlePaymentRefresh(w, r)
		return
	}
	ctx, cancel := paymentContext(r)
	defer cancel()

	id, err := parsePathID(r.URL.Path, "/api/payments/")
	if err != nil {
		writeAPIError(w, domain.NewValidationError("id", err.Error()))
		return
	}
	if r.Method != nethttp.MethodGet {
		methodNotAllowed(w, nethttp.MethodGet)
		return
	}
	payment, err := s.services.Payments.GetByID(ctx, id)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, payment)
}

func (s *Server) handlePaymentRefresh(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, action, err := parsePathIDAction(r.URL.Path, "/api/payments/")
	if err != nil || action != "refresh" {
		writeAPIError(w, domain.NewValidationError("path", "ожидается /api/payments/{id}/refresh"))
		return
	}
	if r.Method != nethttp.MethodPost {
		methodNotAllowed(w, nethttp.MethodPost)
		return
	}
	ctx, cancel := paymentContext(r)
	defer cancel()

	payment, err := s.services.Payments.Refresh(ctx, id)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, payment)
}

func (s *Server) paymentsEnabled(w nethttp.ResponseWriter) bool {
	if s.services.Payments == nil {
		writeJSON(w, nethttp.StatusServiceUnavailable, map[string]string{"error": "оплата отключена: payments.mode = disabled"})
		return false
	}
	return true
}

const paymentOperationTimeout = 20 * time.Second

func paymentContext(r *nethttp.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), paymentOperationTimeout)
}
