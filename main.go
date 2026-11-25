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

	store := NewMemoryStore()
	srv := NewServer(store)
	http.HandleFunc("/health", health)
	http.HandleFunc("/shorten", srv.shorten)
	http.HandleFunc("/{shortenedURL}", srv.redirect)

	addr := "localhost:" + port

	fmt.Printf("Server listening on %s (public URL %s)\n", addr, baseURL)

	err := http.ListenAndServe(addr, nil)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
