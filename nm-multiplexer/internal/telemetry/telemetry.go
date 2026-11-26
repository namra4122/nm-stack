// Package telemetry wires OpenTelemetry tracing to an OTLP collector (Jaeger)
// and provides helpers for recording payloads and errors on spans.
//
// Try it (from the module root, Jaeger on localhost:4318):
//
//	go run ./server               # terminal 1
//	go run ./client -probe        # terminal 2; or: -probe /health
//	open http://localhost:16686   # Jaeger UI, service nmux-client
package telemetry

import (
	"context"
	"os"
	"time"
	"unicode/utf8"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// MaxPayload caps how many bytes of a payload are stored on a span.
const MaxPayload = 4096

// DefaultEndpoint is Jaeger's OTLP/HTTP receiver. Override with NMUX_OTLP_ENDPOINT.
const DefaultEndpoint = "localhost:4318"

// Init installs a global tracer provider and the W3C trace-context propagator.
// The returned shutdown func flushes buffered spans; call it before exiting.
func Init(ctx context.Context, service string) (func(context.Context) error, error) {
	endpoint := os.Getenv("NMUX_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // local Jaeger speaks plain HTTP
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(service),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown, nil
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer { return otel.Tracer(name) }

// RecordPayload stores a payload (truncated to MaxPayload) on the span under
// key, along with its original size, so data shapes show up in Jaeger.
func RecordPayload(span trace.Span, key string, b []byte) {
	size := len(b)
	truncated := size > MaxPayload
	if truncated {
		b = b[:MaxPayload]
	}
	val := string(b)
	if !utf8.ValidString(val) {
		val = "<binary>"
	}
	span.SetAttributes(
		attribute.String(key, val),
		attribute.Int(key+".size", size),
		attribute.Bool(key+".truncated", truncated),
	)
}

// RecordError adds an exception event (with stack trace) and marks the span
// failed. Nil errors are ignored so callers can use it unconditionally.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err, trace.WithStackTrace(true))
	span.SetStatus(codes.Error, err.Error())
}
