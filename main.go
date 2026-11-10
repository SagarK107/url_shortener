package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"slices"
)

var urls = make(map[string]string)

type urlBody struct {
	Url string
}

type shortenBody struct {
	Short_url    string `json:"short_url"`
	Original_url string `json:"original_url"`
}

func health(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status" : "UP"}`))
}

func sendURLForShortening(w http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodPost {

		http.Error(w, "Method must be POST", http.StatusMethodNotAllowed)
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

	if ok {
		fmt.Printf("URL already exists for this url: %s", existing_url)
		return
	}

	shortURL, err := generateShortURL(7)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	urls[normalizedURL] = shortURL

	fmt.Printf("Assigned url %s for original url %s", shortURL, urlBody.Url)

	response := shortenBody{
		Short_url:    shortURL,
		Original_url: normalizedURL,
	}
	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

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

	fmt.Println("Server listening on port 8080")

	err := http.ListenAndServe("localhost:8080", nil)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Error in server:", err)
	}

}
