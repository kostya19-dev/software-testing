package gui

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const paymentTimeout = 40 * time.Second

type paymentBackend struct {
	api *backendAPI
}

func (c *paymentBackend) Create(ctx context.Context, dto usecase.CreatePaymentDTO) (usecase.PaymentResponseDTO, error) {
	var result usecase.PaymentResponseDTO
	err := c.api.do(ctx, http.MethodPost, "/api/payments", nil, map[string]any{"appointment_id": dto.AppointmentID}, &result)
	return result, err
}

func (c *paymentBackend) GetByID(ctx context.Context, id uuid.UUID) (usecase.PaymentResponseDTO, error) {
	var result usecase.PaymentResponseDTO
	err := c.api.do(ctx, http.MethodGet, "/api/payments/"+id.String(), nil, nil, &result)
	return result, err
}

func (c *paymentBackend) GetByAppointment(ctx context.Context, appointmentID uuid.UUID) ([]usecase.PaymentResponseDTO, error) {
	var result []usecase.PaymentResponseDTO
	query := url.Values{"appointment_id": {appointmentID.String()}}
	err := c.api.do(ctx, http.MethodGet, "/api/payments", query, nil, &result)
	return result, err
}

func (c *paymentBackend) Refresh(ctx context.Context, id uuid.UUID) (usecase.PaymentResponseDTO, error) {
	var result usecase.PaymentResponseDTO
	err := c.api.do(ctx, http.MethodPost, "/api/payments/"+id.String()+"/refresh", nil, nil, &result)
	return result, err
}

type PaymentView struct {
	ID                 string
	AppointmentID      string
	Amount             string
	Description        string
	Status             string
	Badge              string
	Provider           string
	ProviderPaymentID  string
	ConfirmationURL    string
	CancellationReason string
	Test               bool
	Pending            bool
}

func (s *Server) handlePaymentCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodPost) {
		return
	}
	appointmentID, err := parseUUID(r.FormValue("id"), "id")
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	payment, err := s.services.Payments.Create(r.Context(), usecase.CreatePaymentDTO{AppointmentID: appointmentID})
	if err != nil {
		s.redirect(w, r, "/appointments", "", err)
		return
	}
	s.logger.Info("redirect to payment provider", "payment_id", payment.ID, "provider", payment.Provider)
	if payment.Status == domain.PaymentPending && payment.ConfirmationURL != "" {
		http.Redirect(w, r, payment.ConfirmationURL, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/payments?payment_id="+payment.ID.String(), http.StatusSeeOther)
}

func (s *Server) handlePayment(w http.ResponseWriter, r *http.Request) {
	if !s.requireMethod(w, r, http.MethodGet) {
		return
	}
	vm := s.baseView(r, "payments", "Оплата")
	id, err := parseUUID(r.URL.Query().Get("payment_id"), "payment_id")
	if err != nil {
		vm.Error = userError(err)
		s.render(w, "payment", vm)
		return
	}
	payment, err := s.services.Payments.Refresh(r.Context(), id)
	if err != nil {
		vm.Error = userError(err)
		if payment, err = s.services.Payments.GetByID(r.Context(), id); err != nil {
			s.render(w, "payment", vm)
			return
		}
	}
	vm.Payment = toPaymentView(payment)
	s.render(w, "payment", vm)
}

func toPaymentView(p usecase.PaymentResponseDTO) *PaymentView {
	badges := map[domain.PaymentStatus]string{
		domain.PaymentSucceeded: "accent",
		domain.PaymentCanceled:  "danger",
	}
	return &PaymentView{
		ID:                 p.ID.String(),
		AppointmentID:      p.AppointmentID.String(),
		Amount:             p.Amount + " " + p.Currency,
		Description:        p.Description,
		Status:             string(p.Status),
		Badge:              badges[p.Status],
		Provider:           p.Provider,
		ProviderPaymentID:  p.ProviderPaymentID,
		ConfirmationURL:    p.ConfirmationURL,
		CancellationReason: p.CancellationReason,
		Test:               p.Test,
		Pending:            p.Status == domain.PaymentPending,
	}
}

const paymentTemplate = `
{{define "payment"}}
{{template "head" .}}
{{template "nav" .}}
<section class="page-hero">
  <div>
    <p class="eyebrow">Платёжный сервис ЮKassa</p>
    <h1>Оплата приёма</h1>
    <p class="lead">Статус запрашивается у провайдера при каждом открытии страницы.</p>
  </div>
</section>
{{template "flash" .}}
{{with .Payment}}
<div class="layout">
  <section class="panel">
    <h2>{{.Amount}} <span class="badge {{.Badge}}">{{.Status}}</span></h2>
    <div class="table-wrap">
    <table>
      <tbody>
        <tr><th>Назначение</th><td>{{.Description}}</td></tr>
        <tr><th>Провайдер</th><td>{{.Provider}}{{if .Test}} (тестовый режим){{end}}</td></tr>
        <tr><th>ID у провайдера</th><td class="id">{{.ProviderPaymentID}}</td></tr>
        <tr><th>ID платежа</th><td class="id">{{.ID}}</td></tr>
        <tr><th>Запись</th><td class="id">{{.AppointmentID}}</td></tr>
        {{if .CancellationReason}}<tr><th>Причина отмены</th><td>{{.CancellationReason}}</td></tr>{{end}}
      </tbody>
    </table>
    </div>
  </section>
  <aside class="stack">
    <section class="panel">
      {{if .Pending}}
      <h2>Ожидает оплаты</h2>
      <p><a href="{{.ConfirmationURL}}">Перейти на страницу оплаты</a></p>
      {{end}}
      <form method="get" action="/payments">
        <input type="hidden" name="payment_id" value="{{.ID}}">
        <button class="secondary" type="submit">Обновить статус</button>
      </form>
      <p><a href="/appointments">К записям</a></p>
    </section>
  </aside>
</div>
{{end}}
{{template "footer" .}}
{{end}}
`
