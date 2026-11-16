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
	"strings"
	"sync"
)

var lock sync.RWMutex //Lock for the mapping urls
var urls = make(map[string]string)
var originalURLMap = make(map[string]string)

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

	newURL, isShortURLExisting, err := safeShorten(normalizedURL, &lock, urls, originalURLMap)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func safeShorten(normalizedURL string, lock *sync.RWMutex, urls map[string]string, originalURLMap map[string]string) (string, bool, error) {
	lock.Lock()
	defer lock.Unlock()
	existing_url, ok := urls[normalizedURL]

	if ok {
		slog.Info("URL already exists for this url", "existing_url", existing_url)
		existingURL := fmt.Sprintf("%s/%s", baseURL, existing_url)
		return existingURL, true, nil

	} else {

		//Check for collision
		var shortURL string
		var err error
		tries := 20
		for tries > 0 {

			shortURL, err = generateShortURL(7)

			if err != nil {
				slog.Error("URL could not be shortened", "err", err)
				return "", false, fmt.Errorf("URL could not be shortened")
			}

			_, ok = originalURLMap[shortURL]

			if !ok {
				break
			}

			tries -= 1

		}

		if tries <= 0 {
			slog.Error("Could not generate URL due to collisions")
			return "", false, fmt.Errorf("Could not shorten URL due to collision error")
		}

		urls[normalizedURL] = shortURL
		originalURLMap[shortURL] = normalizedURL

		newURL := fmt.Sprintf("%s/%s", baseURL, shortURL)

		return newURL, false, nil

	}

}

func redirectFromShortURL(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(w, "Method not Allowed", http.StatusMethodNotAllowed)
		return
	}

	shortenedURL := req.PathValue("shortenedURL")

	lock.RLock()
	original_url, ok := originalURLMap[shortenedURL]
	lock.RUnlock()

	if !ok {
		http.Error(w, "The requested URL does not exist", http.StatusNotFound)
		return

	}

	http.Redirect(w, req, original_url, http.StatusTemporaryRedirect)

}
func generateShortURL(length int) (string, error) {

	numChars := len(charset)
	var s strings.Builder
	s.Grow(length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(numChars)))

		if err != nil {
			return "", fmt.Errorf("error in function generateShortUrl: %w ", err)
		}

		char := charset[n.Int64()]
		s.WriteByte(char)

	}

	return s.String(), nil

}

func validateURL(urlString string) error {

	allowList := []string{"http", "https"}
	u, err := url.Parse(urlString)

	if err != nil {
		return ErrURLCouldNotBeParsed
	}

	if !slices.Contains(allowList, u.Scheme) {
		return ErrSchemeMustBeHTTPOrHTTPS
	}

	if u.Host == "" {
		return ErrMissingHost
	}

	return nil

}

func normalizeURL(urlString string) (string, error) {

	urlLength := len(urlString)

	if urlLength == 0 {
		return "", ErrNoEmptyURLs
	}

	if urlLength > maxURLLength {
		return "", fmt.Errorf("%w: %d", ErrURLTooLong, maxURLLength)
	}

	u, err := url.Parse(urlString)
	if err != nil {
		return "", ErrProblemNormalizingURL
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
