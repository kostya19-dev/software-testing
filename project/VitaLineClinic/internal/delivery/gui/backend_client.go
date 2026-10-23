package gui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const backendTimeout = 3 * time.Second

type backendAPI struct {
	baseURL string
	client  *nethttp.Client
	logger  *slog.Logger
}

type patientBackend struct {
	api *backendAPI
}

type appointmentBackend struct {
	api *backendAPI
}

type scheduleBackend struct {
	api *backendAPI
}

type visitBackend struct {
	api *backendAPI
}

type emergencyBackend struct {
	api *backendAPI
}

func NewBackendServices(baseURL string, logger *slog.Logger) Services {
	if logger == nil {
		logger = slog.Default()
	}
	api := &backendAPI{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &nethttp.Client{},
		logger:  logger,
	}
	return Services{
		Patients:     &patientBackend{api: api},
		Appointments: &appointmentBackend{api: api},
		Schedule:     &scheduleBackend{api: api},
		Visits:       &visitBackend{api: api},
		Emergencies:  &emergencyBackend{api: api},
	}
}

func (c *patientBackend) Register(ctx context.Context, dto usecase.CreatePatientDTO) (usecase.PatientResponseDTO, error) {
	var result usecase.PatientResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/patients", nil, map[string]any{
		"full_name":  dto.FullName,
		"phone":      dto.Phone,
		"email":      dto.Email,
		"birth_date": dto.BirthDate,
	}, &result)
	return result, err
}

func (c *patientBackend) GetByID(ctx context.Context, id uuid.UUID) (usecase.PatientResponseDTO, error) {
	var result usecase.PatientResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/patients/"+id.String(), nil, nil, &result)
	return result, err
}

func (c *patientBackend) GetAll(ctx context.Context) ([]usecase.PatientResponseDTO, error) {
	var result []usecase.PatientResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/patients", nil, nil, &result)
	return result, err
}

func (c *patientBackend) FindByEmail(ctx context.Context, email string) (usecase.PatientResponseDTO, error) {
	patients, err := c.GetAll(ctx)
	if err != nil {
		return usecase.PatientResponseDTO{}, err
	}
	for _, patient := range patients {
		if strings.EqualFold(strings.TrimSpace(patient.Email), strings.TrimSpace(email)) {
			return patient, nil
		}
	}
	return usecase.PatientResponseDTO{}, domain.ErrNotFound
}

func (c *patientBackend) Delete(ctx context.Context, id uuid.UUID) error {
	return c.api.do(ctx, nethttp.MethodDelete, "/api/patients/"+id.String(), nil, nil, nil)
}

func (c *appointmentBackend) Create(ctx context.Context, dto usecase.CreateAppointmentDTO) (usecase.AppointmentResponseDTO, error) {
	var result usecase.AppointmentResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/appointments", nil, map[string]any{
		"patient_id": dto.PatientID,
		"doctor_id":  dto.DoctorID,
		"date":       dto.Date,
	}, &result)
	return result, err
}

func (c *appointmentBackend) Cancel(ctx context.Context, appointmentID uuid.UUID) error {
	return c.api.do(ctx, nethttp.MethodPost, "/api/appointments/"+appointmentID.String()+"/cancel", nil, nil, nil)
}

func (c *appointmentBackend) GetByID(ctx context.Context, id uuid.UUID) (usecase.AppointmentResponseDTO, error) {
	appointments, err := c.GetAll(ctx)
	if err != nil {
		return usecase.AppointmentResponseDTO{}, err
	}
	for _, appointment := range appointments {
		if appointment.ID == id {
			return appointment, nil
		}
	}
	return usecase.AppointmentResponseDTO{}, domain.ErrNotFound
}

func (c *appointmentBackend) GetAll(ctx context.Context) ([]usecase.AppointmentResponseDTO, error) {
	var result []usecase.AppointmentResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/appointments", nil, nil, &result)
	return result, err
}

func (c *appointmentBackend) GetByPatient(ctx context.Context, patientID uuid.UUID) ([]usecase.AppointmentResponseDTO, error) {
	query := url.Values{"patient_id": []string{patientID.String()}}
	var result []usecase.AppointmentResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/appointments", query, nil, &result)
	return result, err
}

func (c *scheduleBackend) CreateSlot(ctx context.Context, dto usecase.CreateScheduleSlotDTO) (usecase.ScheduleSlotResponseDTO, error) {
	var result usecase.ScheduleSlotResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/schedule", nil, map[string]any{
		"doctor_id":  dto.DoctorID,
		"date":       dto.Date,
		"start_time": dto.StartTime,
		"end_time":   dto.EndTime,
		"cabinet":    dto.Cabinet,
	}, &result)
	return result, err
}

func (c *scheduleBackend) GetByDoctor(ctx context.Context, doctorID uuid.UUID) ([]usecase.ScheduleSlotResponseDTO, error) {
	query := url.Values{"doctor_id": []string{doctorID.String()}}
	var result []usecase.ScheduleSlotResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/schedule", query, nil, &result)
	return result, err
}

func (c *scheduleBackend) GetFreeSlots(ctx context.Context, doctorID uuid.UUID, date time.Time) ([]usecase.ScheduleSlotResponseDTO, error) {
	query := url.Values{
		"doctor_id": []string{doctorID.String()},
		"date":      []string{date.Format(dateLayout)},
	}
	var result []usecase.ScheduleSlotResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/schedule/free", query, nil, &result)
	return result, err
}

func (c *scheduleBackend) DeleteSlot(ctx context.Context, id uuid.UUID) error {
	return c.api.do(ctx, nethttp.MethodDelete, "/api/schedule/"+id.String(), nil, nil, nil)
}

func (c *visitBackend) Create(ctx context.Context, dto usecase.CreateVisitDTO) (usecase.VisitResponseDTO, error) {
	var result usecase.VisitResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/visits", nil, map[string]any{
		"appointment_id": dto.AppointmentID,
		"complaints":     dto.Complaints,
	}, &result)
	return result, err
}

func (c *visitBackend) Complete(ctx context.Context, visitID uuid.UUID, dto usecase.UpdateVisitDTO) (usecase.VisitResponseDTO, error) {
	var result usecase.VisitResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/visits/"+visitID.String()+"/complete", nil, map[string]any{
		"diagnosis":       dto.Diagnosis,
		"recommendations": dto.Recommendations,
	}, &result)
	return result, err
}

func (c *visitBackend) GetByAppointment(context.Context, uuid.UUID) (usecase.VisitResponseDTO, error) {
	return usecase.VisitResponseDTO{}, fmt.Errorf("%w: поиск приема по записи в GUI API не используется", domain.ErrNotFound)
}

func (c *visitBackend) GetByPatient(ctx context.Context, patientID uuid.UUID) ([]usecase.VisitResponseDTO, error) {
	query := url.Values{"patient_id": []string{patientID.String()}}
	var result []usecase.VisitResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/visits", query, nil, &result)
	return result, err
}

func (c *emergencyBackend) Create(ctx context.Context, dto usecase.CreateEmergencyRequestDTO) (usecase.EmergencyRequestResponseDTO, error) {
	var result usecase.EmergencyRequestResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/emergencies", nil, map[string]any{
		"patient_id":     dto.PatientID,
		"doctor_id":      dto.DoctorID,
		"requested_date": dto.RequestedDate,
		"reason":         dto.Reason,
	}, &result)
	return result, err
}

func (c *emergencyBackend) GetRequests(ctx context.Context) ([]usecase.EmergencyRequestResponseDTO, error) {
	var result []usecase.EmergencyRequestResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/emergencies", nil, nil, &result)
	return result, err
}

func (c *emergencyBackend) GetOffers(ctx context.Context) ([]usecase.RescheduleOfferResponseDTO, error) {
	var result []usecase.RescheduleOfferResponseDTO
	err := c.api.do(ctx, nethttp.MethodGet, "/api/reschedule-offers", nil, nil, &result)
	return result, err
}

func (c *emergencyBackend) AcceptOffer(ctx context.Context, offerID uuid.UUID) (usecase.RescheduleOfferResponseDTO, error) {
	var result usecase.RescheduleOfferResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/reschedule-offers/"+offerID.String()+"/accept", nil, nil, &result)
	return result, err
}

func (c *emergencyBackend) DeclineOffer(ctx context.Context, offerID uuid.UUID) (usecase.RescheduleOfferResponseDTO, error) {
	var result usecase.RescheduleOfferResponseDTO
	err := c.api.do(ctx, nethttp.MethodPost, "/api/reschedule-offers/"+offerID.String()+"/decline", nil, nil, &result)
	return result, err
}

func (c *backendAPI) do(ctx context.Context, method, path string, query url.Values, payload any, result any) error {
	if c.baseURL == "" {
		return errors.New("backend api url is empty")
	}

	requestCtx, cancel := context.WithTimeout(ctx, backendTimeout)
	defer cancel()

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode backend request: %w", err)
		}
		body = bytes.NewReader(raw)
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := nethttp.NewRequestWithContext(requestCtx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("create backend request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("backend request failed", "method", method, "path", path, "error", err)
		return fmt.Errorf("backend недоступен: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg := readBackendError(resp.Body)
		c.logger.Error("backend returned error", "method", method, "path", path, "status", resp.StatusCode, "error", msg)
		return fmt.Errorf("backend вернул ошибку %d: %s", resp.StatusCode, msg)
	}

	if result == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode backend response: %w", err)
	}
	return nil
}

func readBackendError(body io.Reader) string {
	var response struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(body).Decode(&response); err == nil && strings.TrimSpace(response.Error) != "" {
		return response.Error
	}
	return "unknown backend error"
}
