package http

import (
	"bytes"
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"vitaline-clinic/internal/appconfig"
	"vitaline-clinic/internal/logging"
	"vitaline-clinic/internal/storage/memory"
	"vitaline-clinic/internal/tracing"
	"vitaline-clinic/internal/usecase"
)

type observability struct {
	name     string
	tracing  bool
	extended bool
}

var observabilityVariants = []observability{
	{name: "baseline"},
	{name: "tracing", tracing: true},
	{name: "logging", extended: true},
	{name: "tracing+logging", tracing: true, extended: true},
}

func BenchmarkObservabilityOverhead(b *testing.B) {
	for _, variant := range observabilityVariants {
		b.Run(variant.name, func(b *testing.B) {
			handler := newBenchmarkHandler(b, variant)
			doctor := uuid.New()
			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				visit(b, handler, doctor, i)
			}
		})
	}
}

func newBenchmarkHandler(b *testing.B, variant observability) nethttp.Handler {
	b.Helper()
	logFile, err := os.Create(filepath.Join(b.TempDir(), "clinic.log"))
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { _ = logFile.Close() })
	logger, err := logging.NewLogger(logFile, appconfig.LoggingConfig{Level: "info", Extended: variant.extended}, false)
	if err != nil {
		b.Fatal(err)
	}

	appointments := memory.NewAppointmentRepository()
	server := NewServer(":0", "bench", nil, Services{
		Patients:     usecase.NewPatientService(memory.NewPatientRepository()),
		Appointments: usecase.NewAppointmentService(appointments),
		Schedule:     usecase.NewScheduleService(memory.NewScheduleRepository()),
		Visits:       usecase.NewVisitService(memory.NewVisitRepository(), appointments),
	}, logger)

	if variant.tracing {
		tracer := tracing.New(discardExporter{}, appconfig.TracingConfig{ServiceName: "bench", SampleRatio: 1})
		b.Cleanup(func() { _ = tracer.Close() })
		server.Wrap(tracer.Handler)
	}
	return server.httpServer.Handler
}

func visit(b *testing.B, handler nethttp.Handler, doctor uuid.UUID, i int) {
	created := call(b, handler, nethttp.MethodPost, "/api/patients", map[string]any{
		"full_name":  "Нагрузочный Пациент",
		"phone":      "+79001234567",
		"email":      uuid.NewString() + "@example.com",
		"birth_date": time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
	}, nethttp.StatusCreated)
	var patient struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created, &patient); err != nil {
		b.Fatal(err)
	}
	call(b, handler, nethttp.MethodGet, "/api/patients/"+patient.ID, nil, nethttp.StatusOK)
	call(b, handler, nethttp.MethodPost, "/api/appointments", map[string]any{
		"patient_id": patient.ID,
		"doctor_id":  doctor,
		"date":       time.Now().UTC().Truncate(time.Hour).Add(48*time.Hour + time.Duration(i)*30*time.Minute),
	}, nethttp.StatusCreated)
	call(b, handler, nethttp.MethodGet, "/api/appointments?patient_id="+patient.ID, nil, nethttp.StatusOK)
}

func call(b *testing.B, handler nethttp.Handler, method, path string, payload any, want int) []byte {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			b.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != want {
		b.Fatalf("%s %s: status %d, want %d: %s", method, path, response.Code, want, response.Body.String())
	}
	return response.Body.Bytes()
}

type discardExporter struct{}

func (discardExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error { return nil }

func (discardExporter) Shutdown(context.Context) error { return nil }
