// Command client is the nmux client. Today it has two modes:
//
//   - shell mode (default): starts bash on a local PTY and bridges your
//     terminal to it. This is the "local PTY bridge" from lesson 0001; the
//     server does not take part yet.
//   - probe mode (-probe): sends traced HTTP requests to the server over its
//     Unix socket, so we can check the client→server path end to end.
//
// The byte path in shell mode is:
//
//	keyboard → your terminal (raw) → this process → PTY master → line discipline → PTY slave → bash
//
// and output takes the same path back. Later, everything from the PTY master
// rightwards moves into the server, so the shell outlives the client.
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

// socketPath is the Unix socket the server listens on. A Unix socket is a
// file on disk rather than a TCP port, so only local processes can connect.
const socketPath = "./server/server.sock"

var tracer = telemetry.Tracer("nmux/client")

// runShell runs bash on a new PTY and copies bytes between your terminal and
// the PTY master until bash exits. It returns when the shell's output ends.
func runShell(ctx context.Context) (err error) {
	// One span covers the whole shell session. The deferred func records any
	// returned error on the span before ending it.
	ctx, span := tracer.Start(ctx, "client.shell")
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	// Start bash on a PTY. pty.Start opens a master/slave pair, makes the
	// slave bash's stdin, stdout, stderr and controlling terminal, then
	// starts bash. We get back the master (nmux): writing to it is "typing"
	// into bash, reading from it is bash's output. Because bash sees a real
	// terminal on the slave, it runs interactively (prompt, readline, job
	// control).
	cmd := exec.Command("bash")
	nmux, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	// Closing the master hangs up the slave, so bash gets SIGHUP if it is
	// still running when we return.
	defer nmux.Close()
	span.AddEvent("pty.started", trace.WithAttributes(
		attribute.String("cmd", cmd.Path),
		attribute.Int("pid", cmd.Process.Pid),
	))

	// Keep the PTY size the same as your terminal window. The kernel stores
	// a size on each TTY; when we change the PTY's size the kernel sends
	// SIGWINCH to bash's foreground job, which then redraws for the new size.
	// We get SIGWINCH ourselves when your terminal window is resized.
	// Buffer of 1: signal.Notify drops signals instead of blocking, and one
	// pending resize is enough because each one reads the latest size.
	sizeCh := make(chan os.Signal, 1)
	signal.Notify(sizeCh, syscall.SIGWINCH)

	// This goroutine applies each resize to the PTY. It stops when sizeCh is
	// closed (see the defer below).
	go func() {
		for range sizeCh {
			// SPLIT: InheritSize = GetsizeFull(os.Stdin) + Setsize(nmux).
			// Once the client has the window's size (GetsizeFull, client
			// side), it has to send it to the server, where Setsize(...)
			// is applied to the master.
			if err := pty.InheritSize(os.Stdin, nmux); err != nil {
				// A failed resize is not fatal: the shell keeps working at
				// the old size, so record it and wait for the next signal.
				telemetry.RecordError(span, fmt.Errorf("resizing pty: %w", err))
				continue
			}
			rows, cols, _ := pty.Getsize(nmux)
			span.AddEvent("pty.resized", trace.WithAttributes(
				attribute.Int("rows", rows), attribute.Int("cols", cols),
			))
		}
	}()

	// Send one fake SIGWINCH so the PTY gets the correct size at startup.
	// Without it bash would start at the PTY default size until your first
	// window resize.
	sizeCh <- syscall.SIGWINCH
	// Stop signal delivery first, then close the channel. The order matters:
	// closing first could let the runtime send on a closed channel and panic.
	defer func() {
		signal.Stop(sizeCh)
		close(sizeCh)
	}()

	// Put your real terminal into raw mode. In the default (cooked) mode your
	// terminal's line discipline would echo keys, buffer whole lines and turn
	// ^C into SIGINT for *this* process. In raw mode every byte comes to us
	// unchanged, and we forward it so the PTY's line discipline handles it
	// for bash instead. MakeRaw returns the old settings; the defer restores
	// them so your terminal works normally after we exit.
	prevState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), prevState)
	span.AddEvent("terminal.raw")

	// Input direction: keyboard → PTY master. This runs in its own goroutine
	// because io.Copy blocks, and we need the output copy below to run at the
	// same time. It is not waited on: os.Stdin.Read cannot be interrupted, so
	// the goroutine simply dies with the process.
	// Keystroke contents are deliberately not traced: they include anything
	// typed into the shell, passwords too.
	go func() {
		_, _ = io.Copy(nmux, os.Stdin)
	}()

	// Output direction: PTY master → your terminal. This blocks on the main
	// goroutine, so runShell lasts as long as bash. When bash exits the slave
	// closes, reading the master returns an error (EIO on Linux), and io.Copy
	// returns. That error is expected, so it is ignored.
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

	// net/http only knows how to dial TCP. A custom DialContext ignores the
	// host:port from the URL and always connects to the Unix socket, so we
	// can still use normal HTTP requests against "http://unix/...".
	client := http.Client{
		Timeout: 5 * time.Second,
		Transport: otelhttp.NewTransport(&http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", socketPath)
			},
		}),
	}

	// Stop at the first failing path; the error is returned to main.
	for _, path := range paths {
		if err := request(ctx, &client, path); err != nil {
			return err
		}
	}
	return nil
}

// request sends one GET for path, prints "path status body", and returns an
// error for transport failures or any 4xx/5xx status.
func request(ctx context.Context, client *http.Client, path string) (err error) {
	ctx, span := tracer.Start(ctx, "client.request "+path)
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	// The host "unix" is a placeholder: the custom dialer in probe ignores it.
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

	// Set up tracing first so both modes export spans to Jaeger.
	ctx := context.Background()
	shutdownTracing, err := telemetry.Init(ctx, "nmux-client")
	if err != nil {
		log.Fatal(err)
	}

	if *probeMode {
		// Any extra arguments are paths to request; default to two checks.
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

	// Flush buffered spans before exiting. This uses a fresh context with its
	// own timeout so a slow or missing Jaeger cannot hang the exit.
	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if serr := shutdownTracing(flushCtx); serr != nil {
		log.Printf("tracing shutdown: %v", serr)
	}
	// log.Fatal calls os.Exit, which skips deferred calls. That is why the
	// tracing flush happens above, before this point.
	if err != nil {
		log.Fatal(err)
	}
}
