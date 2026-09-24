package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func runShell() error {

	// Starting the PTY shell
	fmt.Fprintln(os.Stderr, "Debug -> creating the bash")
	cmd := exec.Command("bash")
	fmt.Fprintln(os.Stderr, "Debug -> starting pty")

	nmux, err := pty.Start(cmd)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Debug -> pty started")
	defer nmux.Close()

	// syncing the emulator size with pty-shell
	sizeCh := make(chan os.Signal, 1)
	signal.Notify(sizeCh, syscall.SIGWINCH)

	go func() {
		for s := range sizeCh {
			fmt.Fprintf(os.Stderr, "Debug -> signal data: %v type:%T\n", s, s)

			if err := pty.InheritSize(os.Stdin, nmux); err != nil {
				log.Printf("Error -> resizing pty: %v\n", err)
			}
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
	fmt.Fprintf(os.Stderr, "Debug -> prev data: %v type:%T\n", prevState, prevState)

	// stream recording
	go func() {
		_, _ = io.Copy(nmux, os.Stdin)
	}()

	// TODO: check the cmd, if it is "hello" print "world". Otherwise run everything as normal

	_, _ = io.Copy(os.Stdout, nmux)
	return nil
}
func main() {
	if err := runShell(); err != nil {
		log.Fatal(err)
	}

	// conn, err := net.Dial("unix", "./server/server.sock")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// client := http.Client{
	// 	Transport: &http.Transport{
	// 		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
	// 			return conn, nil
	// 		},
	// 	},
	// }
	//
	// res, err := client.Get("http://unix/health")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// body, err := io.ReadAll(res.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// fmt.Println(string(body))
}
