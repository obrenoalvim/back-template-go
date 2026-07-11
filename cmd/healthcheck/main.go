// healthcheck is a tiny standalone binary for the Dockerfile's HEALTHCHECK — the
// distroless runtime image has no shell/curl/wget, so a plain HTTP GET replaces them.
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}
