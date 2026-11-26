package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"vitaline-clinic/internal/logging"
	"vitaline-clinic/internal/tracing"
	"vitaline-clinic/internal/usecase"
)

const (
	bodyPreview = 1024

	readHeaderTimeout = 10 * time.Second
)

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Services struct {
	Patients     usecase.IPatientService
	Appointments usecase.IAppointmentService
	Schedule     usecase.IScheduleService
	Visits       usecase.IVisitService
	Emergencies  usecase.IEmergencyService
	Payments     usecase.IPaymentService
}

type Server struct {
	httpServer *http.Server
	health     HealthChecker
	services   Services
	logger     *slog.Logger
	appName    string
}

func NewServer(address string, appName string, health HealthChecker, services Services, logger *slog.Logger) *Server {
	srv := &Server{
		health:   health,
		services: services,
		logger:   logger,
		appName:  appName,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.handleHealth)
	mux.HandleFunc("/info", srv.handleInfo)
	mux.HandleFunc("/api/patients", srv.handlePatients)
	mux.HandleFunc("/api/patients/", srv.handlePatientByID)
	mux.HandleFunc("/api/appointments", srv.handleAppointments)
	mux.HandleFunc("/api/appointments/", srv.handleAppointmentAction)
	mux.HandleFunc("/api/schedule", srv.handleSchedule)
	mux.HandleFunc("/api/schedule/", srv.handleScheduleByID)
	mux.HandleFunc("/api/schedule/free", srv.handleFreeSchedule)
	mux.HandleFunc("/api/visits", srv.handleVisits)
	mux.HandleFunc("/api/visits/", srv.handleVisitAction)
	mux.HandleFunc("/api/emergencies", srv.handleEmergencies)
	mux.HandleFunc("/api/reschedule-offers", srv.handleRescheduleOffers)
	mux.HandleFunc("/api/reschedule-offers/", srv.handleRescheduleOfferAction)
	mux.HandleFunc("/api/payments", srv.handlePayments)
	mux.HandleFunc("/api/payments/", srv.handlePaymentByID)

	srv.httpServer = &http.Server{
		Addr:              address,
		Handler:           srv.logRequests(mux),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return srv
}

func (s *Server) ListenAndServe() error {
	s.logger.Info("http server starting", "address", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Wrap(middleware func(http.Handler) http.Handler) {
	s.httpServer.Handler = middleware(s.httpServer.Handler)
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	response := map[string]string{
		"status": "ok",
		"app":    s.appName,
	}

	if err := s.health.Ping(ctx); err != nil {
		s.logger.Error("health check failed", "error", err)
		response["status"] = "degraded"
		response["database"] = "down"
		writeJSON(w, http.StatusServiceUnavailable, response)
		return
	}

	response["database"] = "up"
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app":         s.appName,
		"component":   "clinic-api",
		"description": "backend component for VitaLine Clinic",
	})
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		detailed := s.logger.Enabled(r.Context(), slog.LevelDebug)
		var request *bodyRecorder
		if detailed {
			request = &bodyRecorder{ReadCloser: r.Body}
			r.Body = request
			recorder.body = &bytes.Buffer{}
		}

		next.ServeHTTP(recorder, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", time.Since(start),
		}
		if traceID := tracing.TraceID(r.Context()); traceID != "" {
			attrs = append(attrs, "trace_id", traceID)
		}
		s.logger.InfoContext(r.Context(), "http request handled", attrs...)
		if detailed {
			s.logger.DebugContext(r.Context(), "http request details",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_bytes", request.size,
				"request_body", logging.Truncate(request.body.String(), bodyPreview),
				"response_bytes", recorder.size,
				"response_body", logging.Truncate(recorder.body.String(), bodyPreview),
			)
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
	body   *bytes.Buffer
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	n, err := r.ResponseWriter.Write(data)
	r.size += n
	if r.body != nil && r.body.Len() < bodyPreview {
		r.body.Write(data[:min(n, bodyPreview-r.body.Len())])
	}
	return n, err
}

type bodyRecorder struct {
	io.ReadCloser
	size int
	body bytes.Buffer
}

func (b *bodyRecorder) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	b.size += n
	if b.body.Len() < bodyPreview {
		b.body.Write(data[:min(n, bodyPreview-b.body.Len())])
	}
	return n, err
}
