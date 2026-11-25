package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"strings"
)

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
