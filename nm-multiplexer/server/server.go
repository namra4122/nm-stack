package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
)

func responeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("ERROR | %d | failed to encode response: %v ", status, err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	responeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
	log.Printf("GET | %d | /health", http.StatusOK)
}

func readMemory(w http.ResponseWriter, r *http.Request) {

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	responeJSON(w, http.StatusOK, map[string]int{
		"status": http.StatusOK,
		"memory": int(m.Alloc),
	})
	log.Printf("GET | %d | /memory", http.StatusOK)
}

func main() {
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/memory", readMemory)

	os.Remove("./server/server.sock")

	listener, err := net.Listen("unix", "./server/server.sock")
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	log.Println("Listening on ./server/server.go")

	if err := http.Serve(listener, nil); err != nil {
		log.Fatal(err)
	}
}
