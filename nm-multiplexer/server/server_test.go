package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var spans = tracetest.NewSpanRecorder()

// The global provider can only be delegated once, so install it for all tests.
func TestMain(m *testing.M) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spans)))
	otel.SetTextMapPropagator(propagation.TraceContext{})
	os.Exit(m.Run())
}

func attr(s sdktrace.ReadOnlySpan, key string) string {
	for _, kv := range s.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.Emit()
		}
	}
	return ""
}

func TestTraceSpansClientMiddlewareServer(t *testing.T) {
	srv := httptest.NewServer(newHandler())
	defer srv.Close()

	ctx, root := otel.Tracer("test").Start(context.Background(), "client.probe")
	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/health", nil)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(res.Body)
	res.Body.Close()
	root.End()

	byName := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range spans.Ended() {
		if s.SpanContext().TraceID() != root.SpanContext().TraceID() {
			continue
		}
		byName[s.Name()] = s
	}
	for _, want := range []string{"client.probe", "server GET /health", "middleware.capture", "handler.health", "encode.json"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("span %q missing from trace; got %v", want, keys(byName))
		}
	}
	if got := attr(byName["middleware.capture"], "response.body"); !strings.Contains(got, `"ok"`) {
		t.Errorf("middleware response.body = %q, want JSON with ok", got)
	}
	if got := attr(byName["encode.json"], "output"); got != `{"status":"ok"}` {
		t.Errorf("encode.json output = %q", got)
	}
}

func TestCaptureRecordsPanicAsException(t *testing.T) {
	h := capture(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/explode", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var span sdktrace.ReadOnlySpan
	for _, s := range spans.Ended() {
		if s.Name() == "middleware.capture" && attr(s, "request.path") == "/explode" {
			span = s
		}
	}
	if span == nil {
		t.Fatal("no middleware span for /explode")
	}
	if span.Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error", span.Status().Code)
	}
	if len(span.Events()) == 0 || span.Events()[0].Name != "exception" {
		t.Errorf("want exception event, got %v", span.Events())
	}
}

func keys(m map[string]sdktrace.ReadOnlySpan) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
