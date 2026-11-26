package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/namra4122/nm-multiplexer/internal/telemetry"
)

const socketPath = "./server/server.sock"

// responseJSON serializes data, records the serialized output on the handler
// span from ctx, and writes it.
func responseJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	_, span := tracer.Start(ctx, "encode.json")
	defer span.End()

	body, err := json.Marshal(data)
	if err != nil {
		telemetry.RecordError(span, err)
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}
	telemetry.RecordPayload(span, "output", body)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		telemetry.RecordError(span, err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.health")
	defer span.End()

	responseJSON(ctx, w, http.StatusOK, map[string]string{"status": "ok"})
}

func readMemory(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handler.memory")
	defer span.End()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	responseJSON(ctx, w, http.StatusOK, map[string]int{
		"status": http.StatusOK,
		"memory": int(m.Alloc),
	})
}

// newHandler builds the layered stack: otelhttp (server span, extracts the
// client's traceparent) -> capture middleware -> route handlers.
func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthCheck)
	mux.HandleFunc("/memory", readMemory)
	return otelhttp.NewHandler(capture(mux), "nmux-server",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return "server " + r.Method + " " + r.URL.Path
		}),
	)
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTracing, err := telemetry.Init(ctx, "nmux-server")
	if err != nil {
		return err
	}
	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(flushCtx); err != nil {
			log.Printf("tracing shutdown: %v", err)
		}
	}()

	os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	srv := &http.Server{Handler: newHandler()}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", socketPath)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
