package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Server struct {
	store *MemoryStore
}

func NewServer(store *MemoryStore) *Server {
	return &Server{
		store: store,
	}
}

func (srv *Server) shorten(w http.ResponseWriter, req *http.Request) {

	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	var urlBody urlBody

	err := json.NewDecoder(req.Body).Decode(&urlBody)

	if err != nil {
		slog.Error("Error Decoding JSON: Invalid JSON", "err", err)
		http.Error(w, "Error Decoding JSON: Invalid JSON sent.", http.StatusBadRequest)
		return
	}
	normalizedURL, err := normalizeURL(urlBody.Url)

	if err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}

	err = validateURL(normalizedURL)

	if err != nil {

		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code, isShortURLExisting, err := srv.store.GetOrCreate(normalizedURL)

	if err != nil {
		slog.Error("getOrCreate error", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", baseURL, code)
	response := shortenBody{
		ShortURL:    shortURL,
		OriginalURL: normalizedURL,
	}

	w.Header().Set("Content-Type", "application/json")
	if isShortURLExisting {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		// Headers are already sent, so we can't change the response.
		// Record it for ourselves instead.
		slog.Error("writing shorten response", "err", err)
	}
}

func (srv *Server) redirect(w http.ResponseWriter, req *http.Request) {

	shortenedURL := req.PathValue("shortenedURL")

	originalURL, ok := srv.store.Lookup(shortenedURL)

	if !ok {
		http.Error(w, "The requested URL does not exist", http.StatusNotFound)
		return

	}

	http.Redirect(w, req, originalURL, http.StatusTemporaryRedirect)
}

func (srv *Server) newMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("POST /shorten", srv.shorten)
	mux.HandleFunc("GET /{shortenedURL}", srv.redirect)
	return mux
}
