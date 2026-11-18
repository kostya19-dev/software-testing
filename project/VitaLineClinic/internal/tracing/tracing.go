package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"vitaline-clinic/internal/appconfig"
)

const shutdownTimeout = 5 * time.Second

type Tracer struct {
	provider *sdktrace.TracerProvider
}

func Start(ctx context.Context, cfg appconfig.TracingConfig) (*Tracer, error) {
	if !cfg.Enabled {
		return &Tracer{}, nil
	}
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "http://"), "https://")),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}
	return New(exporter, cfg), nil
}

func New(exporter sdktrace.SpanExporter, cfg appconfig.TracingConfig) *Tracer {
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", cfg.ServiceName))),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return &Tracer{provider: provider}
}

func (t *Tracer) Enabled() bool {
	return t != nil && t.provider != nil
}

func (t *Tracer) Handler(next http.Handler) http.Handler {
	if !t.Enabled() {
		return next
	}
	return otelhttp.NewHandler(next, "clinic-api",
		otelhttp.WithTracerProvider(t.provider),
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + Route(r.URL.Path)
		}),
	)
}

func (t *Tracer) CommandMonitor() *event.CommandMonitor {
	if !t.Enabled() {
		return nil
	}
	return otelmongo.NewMonitor(otelmongo.WithTracerProvider(t.provider))
}

func (t *Tracer) Close() error {
	if !t.Enabled() {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return t.provider.Shutdown(ctx)
}

func TraceID(ctx context.Context) string {
	span := trace.SpanContextFromContext(ctx)
	if !span.HasTraceID() {
		return ""
	}
	return span.TraceID().String()
}

func Route(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if _, err := uuid.Parse(segment); err == nil {
			segments[i] = "{id}"
		}
	}
	return strings.Join(segments, "/")
}
