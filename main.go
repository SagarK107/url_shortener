package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

const maxURLLength = 2000 // length picked as 2000 for overly long urls

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

type urlBody struct {
	Url string
}

type shortenBody struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func health(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status" : "UP"}`))
}

func main() {

	// A local .env file is optional; real environment variables always win.
	if err := loadDotEnv(); err != nil {
		log.Fatalf("reading .env file: %v", err)
	}

	// Fail fast: refuse to start with bad settings rather than serve broken links.
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	store := NewMemoryStore()
	srv := NewServer(store, config.BaseURL)

	addr := config.Addr()

	fmt.Printf("Server listening on %s (public URL %s)\n", addr, config.BaseURL)

	err = http.ListenAndServe(addr, srv.newMux())

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
