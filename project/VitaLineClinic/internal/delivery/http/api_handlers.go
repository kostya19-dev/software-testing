package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

const (
	dateOnlyLayout      = "2006-01-02"
	apiOperationTimeout = 1500 * time.Millisecond
)

type createPatientRequest struct {
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	BirthDate time.Time `json:"birth_date"`
}

type createAppointmentRequest struct {
	PatientID uuid.UUID `json:"patient_id"`
	DoctorID  uuid.UUID `json:"doctor_id"`
	Date      time.Time `json:"date"`
}

type createScheduleSlotRequest struct {
	DoctorID  uuid.UUID `json:"doctor_id"`
	Date      time.Time `json:"date"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Cabinet   string    `json:"cabinet"`
}

type createVisitRequest struct {
	AppointmentID uuid.UUID `json:"appointment_id"`
	Complaints    string    `json:"complaints"`
}

type completeVisitRequest struct {
	Diagnosis       string `json:"diagnosis"`
	Recommendations string `json:"recommendations"`
}

func (s *Server) handlePatients(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Patients == nil {
		writeAPIError(w, domain.NewValidationError("patients", "service is not configured"))
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		patients, err := s.services.Patients.GetAll(ctx)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, patients)
	case nethttp.MethodPost:
		var req createPatientRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, err)
			return
		}
		patient, err := s.services.Patients.Register(ctx, usecase.CreatePatientDTO{
			FullName:  req.FullName,
			Phone:     req.Phone,
			Email:     req.Email,
			BirthDate: req.BirthDate,
		})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, patient)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handlePatientByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, err := parsePathID(r.URL.Path, "/api/patients/")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		patient, err := s.services.Patients.GetByID(ctx, id)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, patient)
	case nethttp.MethodDelete:
		if err := s.services.Patients.Delete(ctx, id); err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, map[string]string{"status": "deleted"})
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodDelete)
	}
}

func (s *Server) handleAppointments(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Appointments == nil {
		writeAPIError(w, domain.NewValidationError("appointments", "service is not configured"))
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		patientID := strings.TrimSpace(r.URL.Query().Get("patient_id"))
		if patientID != "" {
			id, err := uuid.Parse(patientID)
			if err != nil {
				writeAPIError(w, fmt.Errorf("parse patient_id: %w", err))
				return
			}
			appointments, err := s.services.Appointments.GetByPatient(ctx, id)
			if err != nil {
				writeAPIError(w, err)
				return
			}
			writeJSON(w, nethttp.StatusOK, appointments)
			return
		}

		appointments, err := s.services.Appointments.GetAll(ctx)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, appointments)
	case nethttp.MethodPost:
		var req createAppointmentRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, err)
			return
		}
		appointment, err := s.services.Appointments.Create(ctx, usecase.CreateAppointmentDTO{
			PatientID: req.PatientID,
			DoctorID:  req.DoctorID,
			Date:      req.Date,
		})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, appointment)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handleAppointmentAction(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, action, err := parsePathIDAction(r.URL.Path, "/api/appointments/")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if action != "cancel" {
		writeAPIError(w, domain.NewValidationError("action", "unknown appointment action"))
		return
	}
	if r.Method != nethttp.MethodPost {
		methodNotAllowed(w, nethttp.MethodPost)
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()
	if err := s.services.Appointments.Cancel(ctx, id); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) handleSchedule(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Schedule == nil {
		writeAPIError(w, domain.NewValidationError("schedule", "service is not configured"))
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		doctorID, err := parseUUIDQuery(r, "doctor_id")
		if err != nil {
			writeAPIError(w, err)
			return
		}
		slots, err := s.services.Schedule.GetByDoctor(ctx, doctorID)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, slots)
	case nethttp.MethodPost:
		var req createScheduleSlotRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, err)
			return
		}
		slot, err := s.services.Schedule.CreateSlot(ctx, usecase.CreateScheduleSlotDTO{
			DoctorID:  req.DoctorID,
			Date:      req.Date,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
			Cabinet:   req.Cabinet,
		})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, slot)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handleFreeSchedule(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		methodNotAllowed(w, nethttp.MethodGet)
		return
	}
	doctorID, err := parseUUIDQuery(r, "doctor_id")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	date, err := parseDateQuery(r, "date")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()
	slots, err := s.services.Schedule.GetFreeSlots(ctx, doctorID, date)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, slots)
}

func (s *Server) handleScheduleByID(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, err := parsePathID(r.URL.Path, "/api/schedule/")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if r.Method != nethttp.MethodDelete {
		methodNotAllowed(w, nethttp.MethodDelete)
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()
	if err := s.services.Schedule.DeleteSlot(ctx, id); err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleVisits(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Visits == nil {
		writeAPIError(w, domain.NewValidationError("visits", "service is not configured"))
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		patientID, err := parseUUIDQuery(r, "patient_id")
		if err != nil {
			writeAPIError(w, err)
			return
		}
		visits, err := s.services.Visits.GetByPatient(ctx, patientID)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, visits)
	case nethttp.MethodPost:
		var req createVisitRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, err)
			return
		}
		visit, err := s.services.Visits.Create(ctx, usecase.CreateVisitDTO{
			AppointmentID: req.AppointmentID,
			Complaints:    req.Complaints,
		})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, visit)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handleVisitAction(w nethttp.ResponseWriter, r *nethttp.Request) {
	id, action, err := parsePathIDAction(r.URL.Path, "/api/visits/")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if action != "complete" {
		writeAPIError(w, domain.NewValidationError("action", "unknown visit action"))
		return
	}
	if r.Method != nethttp.MethodPost {
		methodNotAllowed(w, nethttp.MethodPost)
		return
	}

	var req completeVisitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, err)
		return
	}
	ctx, cancel := apiContext(r)
	defer cancel()
	visit, err := s.services.Visits.Complete(ctx, id, usecase.UpdateVisitDTO{
		Diagnosis:       req.Diagnosis,
		Recommendations: req.Recommendations,
	})
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, visit)
}

func apiContext(r *nethttp.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), apiOperationTimeout)
}

func decodeJSON(r *nethttp.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

func writeAPIError(w nethttp.ResponseWriter, err error) {
	status := nethttp.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrValidation):
		status = nethttp.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		status = nethttp.StatusNotFound
	case errors.Is(err, domain.ErrSlotBusy), errors.Is(err, domain.ErrInvalidStatus),
		errors.Is(err, domain.ErrPastDate), errors.Is(err, domain.ErrAppointmentLeadTime):
		status = nethttp.StatusConflict
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w nethttp.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeJSON(w, nethttp.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func parsePathID(path, prefix string) (uuid.UUID, error) {
	raw := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if raw == "" || strings.Contains(raw, "/") {
		return uuid.Nil, domain.NewValidationError("id", "invalid path id")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse id: %w", err)
	}
	return id, nil
}

func parsePathIDAction(path, prefix string) (uuid.UUID, string, error) {
	raw := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return uuid.Nil, "", domain.NewValidationError("path", "invalid action path")
	}
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("parse id: %w", err)
	}
	return id, parts[1], nil
}

func parseUUIDQuery(r *nethttp.Request, name string) (uuid.UUID, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return uuid.Nil, domain.NewValidationError(name, "не может быть пустым")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse %s: %w", name, err)
	}
	return id, nil
}

func parseDateQuery(r *nethttp.Request, name string) (time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return time.Time{}, domain.NewValidationError(name, "не может быть пустым")
	}
	if date, err := time.Parse(dateOnlyLayout, raw); err == nil {
		return date, nil
	}
	date, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s: %w", name, err)
	}
	return date, nil
}
