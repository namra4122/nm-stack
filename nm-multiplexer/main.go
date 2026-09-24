package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/creack/pty"
)

func shell() error {
	c := exec.Command("bash")

	ptmx, err := pty.Start(c)
	if err != nil {
		return err
	}

	defer ptmx.Close()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)

	go func() {
		for sig := range ch {
			fmt.Fprintf(os.Stderr, "[DEBUG] signal: %v\n", sig)

			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()

	ch <- syscall.SIGWINCH

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}

func main() {
	if err := shell(); err != nil {
		log.Fatal(err)
	}
}
