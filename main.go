package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"slices"
)

var urls = make(map[string]string)
var originalURLMap = make(map[string]string)

const maxURLLength = 2000 // length picked as 2000 for overly long urls

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

func sendURLForShortening(w http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {

		http.Error(w, "Method must be POST", http.StatusMethodNotAllowed)
		return
	}
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	var urlBody urlBody

	err := json.NewDecoder(req.Body).Decode(&urlBody)

	if err != nil {
		slog.Error("Error Decoding JSON: Invlaid JSON", "err", err)
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

	existing_url, ok := urls[normalizedURL]
	var shortURL string
	var isShortURLExisting bool
	if ok {
		fmt.Printf("URL already exists for this url: %s\n", existing_url)
		shortURL = existing_url
		isShortURLExisting = true

	} else {
		isShortURLExisting = false
		shortURL, err = generateShortURL(7)

		if err != nil {
			slog.Error("URL could not be shortened", "err", err)
			http.Error(w, "URL could not be shortened", http.StatusInternalServerError)
			return
		}

		urls[normalizedURL] = shortURL
		originalURLMap[shortURL] = normalizedURL

		fmt.Printf("Assigned url %s for original url %s\n", shortURL, urlBody.Url)
	}

	newURL := fmt.Sprintf("%s/%s", baseURL, shortURL)
	response := shortenBody{
		ShortURL:    newURL,
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

func redirectFromShortURL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not Allowed", http.StatusMethodNotAllowed)
		return
	}

	shortenedURL := req.PathValue("shortenedURL")

	original_url, ok := originalURLMap[shortenedURL]

	if !ok {
		http.Error(w, "The requested URL does not exist", http.StatusNotFound)
		return

	}

	http.Redirect(w, req, original_url, http.StatusTemporaryRedirect)

}
func generateShortURL(length int) (string, error) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(62))

		if err != nil {
			return "", fmt.Errorf("error in function generateShortUrl: %w ", err)
		}

		char := string(charset[n.Int64()])
		s = s + char

	}

	return s, nil

}

func validateURL(urlString string) error {

	allowList := []string{"http", "https"}
	u, err := url.Parse(urlString)

	if err != nil {
		return errors.New("URL could not be parsed")
	}

	if !slices.Contains(allowList, u.Scheme) {
		return errors.New("scheme must be http or https")
	}

	if u.Host == "" {
		return errors.New("missing host")
	}

	return nil

}

func normalizeURL(urlString string) (string, error) {

	urlLength := len(urlString)

	if urlLength == 0 {
		return "", errors.New("no empty URLS")
	}

	if urlLength > maxURLLength {
		return "", fmt.Errorf("URL exceeds maximum length: %d", maxURLLength)
	}

	u, err := url.Parse(urlString)
	if err != nil {
		return "", errors.New("problem normalizing URL")
	}

	if u.Scheme == "" {
		return "https://" + urlString, nil
	}

	return urlString, nil
}

func main() {
	http.HandleFunc("/health", health)
	http.HandleFunc("/shorten", sendURLForShortening)
	http.HandleFunc("/{shortenedURL}", redirectFromShortURL)

	addr := "localhost:" + port

	fmt.Printf("Server listening on %s (public URL %s)\n", addr, baseURL)

	err := http.ListenAndServe(addr, nil)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
