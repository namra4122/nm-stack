package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

func main() {
	conn, err := net.Dial("unix", "./server/server.sock")
	if err != nil {
		log.Fatal(err)
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return conn, nil
			},
		},
	}

	res, err := client.Get("http://unix/health")
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(body))
}
