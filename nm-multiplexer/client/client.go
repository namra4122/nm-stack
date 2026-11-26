package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/term"

	"github.com/namra4122/nm-multiplexer/internal/telemetry"
)

const socketPath = "./server/server.sock"

var tracer = telemetry.Tracer("nmux/client")

func runShell(ctx context.Context) (err error) {
	ctx, span := tracer.Start(ctx, "client.shell")
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	// Starting the PTY shell
	cmd := exec.Command("bash")
	nmux, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer nmux.Close()
	span.AddEvent("pty.started", trace.WithAttributes(
		attribute.String("cmd", cmd.Path),
		attribute.Int("pid", cmd.Process.Pid),
	))

	// syncing the emulator size with pty-shell
	sizeCh := make(chan os.Signal, 1)
	signal.Notify(sizeCh, syscall.SIGWINCH)

	go func() {
		for range sizeCh {
			if err := pty.InheritSize(os.Stdin, nmux); err != nil {
				telemetry.RecordError(span, fmt.Errorf("resizing pty: %w", err))
				continue
			}
			rows, cols, _ := pty.Getsize(nmux)
			span.AddEvent("pty.resized", trace.WithAttributes(
				attribute.Int("rows", rows), attribute.Int("cols", cols),
			))
		}
	}()

	sizeCh <- syscall.SIGWINCH
	defer func() {
		signal.Stop(sizeCh)
		close(sizeCh)
	}()

	// terminal backup
	prevState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), prevState)
	span.AddEvent("terminal.raw")

	// stream recording. Keystroke contents are deliberately not traced:
	// they include anything typed into the shell, passwords too.
	go func() {
		_, _ = io.Copy(nmux, os.Stdin)
	}()

	// TODO: check the cmd, if it is "hello" print "world". Otherwise run everything as normal

	n, _ := io.Copy(os.Stdout, nmux)
	span.SetAttributes(attribute.Int64("pty.output.bytes", n))
	return nil
}

// probe sends traced requests to the server over the Unix socket. otelhttp's
// transport creates the client span and injects the traceparent header, so
// the server's spans join this trace.
func probe(ctx context.Context, paths []string) (err error) {
	ctx, span := tracer.Start(ctx, "client.probe")
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: otelhttp.NewTransport(&http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		}),
	}

	for _, path := range paths {
		if err := request(ctx, &client, path); err != nil {
			return err
		}
	}
	return nil
}

func request(ctx context.Context, client *http.Client, path string) (err error) {
	ctx, span := tracer.Start(ctx, "client.request "+path)
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}
	span.SetAttributes(attribute.String("request.method", req.Method), attribute.String("request.url", req.URL.String()))
	telemetry.RecordPayload(span, "request.body", nil)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	span.SetAttributes(attribute.Int("response.status", res.StatusCode))
	telemetry.RecordPayload(span, "response.body", body)
	fmt.Printf("%s %d %s\n", path, res.StatusCode, bytes.TrimSpace(body))

	if res.StatusCode >= 400 {
		return fmt.Errorf("%s: unexpected status %d", path, res.StatusCode)
	}
	return nil
}

func main() {
	probeMode := flag.Bool("probe", false, "send traced requests to the server instead of starting a shell")
	flag.Parse()

	ctx := context.Background()
	shutdownTracing, err := telemetry.Init(ctx, "nmux-client")
	if err != nil {
		log.Fatal(err)
	}

	if *probeMode {
		paths := flag.Args()
		if len(paths) == 0 {
			paths = []string{"/health", "/memory"}
		}
		err = probe(ctx, paths)
	} else {
		// Exporter errors would print over the raw-mode terminal; drop them.
		otel.SetErrorHandler(otel.ErrorHandlerFunc(func(error) {}))
		err = runShell(ctx)
	}

	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if serr := shutdownTracing(flushCtx); serr != nil {
		log.Printf("tracing shutdown: %v", serr)
	}
	if err != nil {
		log.Fatal(err)
	}
}
