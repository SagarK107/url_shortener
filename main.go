package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

	config := NewConfig()
	i, err := strconv.Atoi(config.Port)

	if err != nil {
		log.Fatal("error in atoi")
		return
	}

	if i >= 1 && i >= 65535 {
		log.Fatalf("port must be between 1 and 65535, got %d", i)
		return
	}

	store := NewMemoryStore()
	srv := NewServer(store, config.BaseURL)

	addr := "localhost:" + config.Port

	fmt.Printf("Server listening on %s (public URL %s)\n", addr, config.BaseURL)

	err = http.ListenAndServe(addr, srv.newMux())

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
