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

	fmt.Printf("decoded: %q\n", urlBody.Url)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	normalizedURL, err := normalizeURL(urlBody.Url)
	if !validateURL(normalizedURL) {

		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			return "", fmt.Errorf("Error in function generateShortUrl: %w ", err)
		}

		char := string(charset[n.Int64()])
		s = s + char

	}

	return s, nil

}

func validateURL(urlString string) bool {
	allowList := []string{"http", "https", ""}
	u, err := url.Parse(urlString)
	if u != nil {
		fmt.Println("Scheme", u.Scheme)
	}

	return err == nil && slices.Contains(allowList, u.Scheme) && u.Host != ""

}

func normalizeURL(urlString string) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", errors.New("Problem normalizing URL")
	}

	if u.Scheme == "" {
		return "https://" + urlString, nil
	}

	return urlString, nil
}

// func retrieveShortURL(shortened_url string) string {

// }

func main() {
	http.HandleFunc("/health", health)
	http.HandleFunc("/shorten", sendURLForShortening)
	http.HandleFunc("/{shortenedURL}", redirectFromShortURL)

	// "localhost:8080" means "port 8080 on every network interface". The public
	// address (baseURL) is a separate setting used only to build links.
	addr := "localhost:" + port

	fmt.Printf("Server listening on %s (public URL %s)\n", addr, baseURL)

	err := http.ListenAndServe(addr, nil)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
