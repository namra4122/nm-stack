package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/namra4122/nm-multiplexer/internal/telemetry"
)

var tracer = telemetry.Tracer("nmux/server")

// recorder captures the status code and a bounded copy of the response body.
type recorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if room := telemetry.MaxPayload + 1 - r.body.Len(); room > 0 {
		r.body.Write(b[:min(len(b), room)])
	}
	return r.ResponseWriter.Write(b)
}

// capture is the middleware layer: it opens its own span (a child of the
// otelhttp server span), records the request and response payloads, and turns
// panics into recorded exceptions plus a 500 instead of a dropped connection.
func capture(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), "middleware.capture")
		defer span.End()

		span.SetAttributes(
			attribute.String("request.method", r.Method),
			attribute.String("request.path", r.URL.Path),
			attribute.String("request.query", r.URL.RawQuery),
			attribute.String("request.traceparent", r.Header.Get("Traceparent")),
		)

		if r.Body != nil {
			body, err := io.ReadAll(io.LimitReader(r.Body, telemetry.MaxPayload+1))
			telemetry.RecordError(span, err)
			telemetry.RecordPayload(span, "request.body", body)
			r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))
		}

		rec := &recorder{ResponseWriter: w}
		defer func() {
			if p := recover(); p != nil {
				telemetry.RecordError(span, fmt.Errorf("panic: %v", p))
				if rec.status == 0 {
					http.Error(rec, "internal server error", http.StatusInternalServerError)
				}
			}
			span.SetAttributes(attribute.Int("response.status", rec.status))
			telemetry.RecordPayload(span, "response.body", rec.body.Bytes())
			if rec.status >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rec.status))
			}
		}()

		next.ServeHTTP(rec, r.WithContext(ctx))
	})
}
