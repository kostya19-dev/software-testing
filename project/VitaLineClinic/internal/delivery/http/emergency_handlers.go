package http

import (
	nethttp "net/http"
	"time"

	"github.com/google/uuid"

	"vitaline-clinic/internal/domain"
	"vitaline-clinic/internal/usecase"
)

type createEmergencyRequest struct {
	PatientID     uuid.UUID `json:"patient_id"`
	DoctorID      uuid.UUID `json:"doctor_id"`
	RequestedDate time.Time `json:"requested_date"`
	Reason        string    `json:"reason"`
}

func (s *Server) handleEmergencies(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Emergencies == nil {
		writeAPIError(w, domain.NewValidationError("emergencies", "service is not configured"))
		return
	}

	ctx, cancel := apiContext(r)
	defer cancel()

	switch r.Method {
	case nethttp.MethodGet:
		requests, err := s.services.Emergencies.GetRequests(ctx)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, requests)
	case nethttp.MethodPost:
		var req createEmergencyRequest
		if err := decodeJSON(r, &req); err != nil {
			writeAPIError(w, err)
			return
		}
		request, err := s.services.Emergencies.Create(ctx, usecase.CreateEmergencyRequestDTO{
			PatientID:     req.PatientID,
			DoctorID:      req.DoctorID,
			RequestedDate: req.RequestedDate,
			Reason:        req.Reason,
		})
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusCreated, request)
	default:
		methodNotAllowed(w, nethttp.MethodGet, nethttp.MethodPost)
	}
}

func (s *Server) handleRescheduleOffers(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Emergencies == nil {
		writeAPIError(w, domain.NewValidationError("emergencies", "service is not configured"))
		return
	}
	if r.Method != nethttp.MethodGet {
		methodNotAllowed(w, nethttp.MethodGet)
		return
	}

	ctx, cancel := apiContext(r)
	defer cancel()

	offers, err := s.services.Emergencies.GetOffers(ctx)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	writeJSON(w, nethttp.StatusOK, offers)
}

func (s *Server) handleRescheduleOfferAction(w nethttp.ResponseWriter, r *nethttp.Request) {
	if s.services.Emergencies == nil {
		writeAPIError(w, domain.NewValidationError("emergencies", "service is not configured"))
		return
	}

	id, action, err := parsePathIDAction(r.URL.Path, "/api/reschedule-offers/")
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if r.Method != nethttp.MethodPost {
		methodNotAllowed(w, nethttp.MethodPost)
		return
	}

	ctx, cancel := apiContext(r)
	defer cancel()

	switch action {
	case "accept":
		offer, err := s.services.Emergencies.AcceptOffer(ctx, id)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, offer)
	case "decline":
		offer, err := s.services.Emergencies.DeclineOffer(ctx, id)
		if err != nil {
			writeAPIError(w, err)
			return
		}
		writeJSON(w, nethttp.StatusOK, offer)
	default:
		writeAPIError(w, domain.NewValidationError("action", "unknown reschedule offer action"))
	}
}
