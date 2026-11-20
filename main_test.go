package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// urlOfLength returns prefix followed by a path of 'a's, so the whole
// string is exactly n characters long. Used to test the length limit
// at and around its boundary.
func urlOfLength(prefix string, n int) string {
	return prefix + strings.Repeat("a", n-len(prefix))
}

func TestNormalizeURL(t *testing.T) {
	var normalizeTests = []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{"bare domain gets https", "go.dev", "https://go.dev", nil},
		{"https unchanged", "https://go.dev", "https://go.dev", nil},
		{"http not upgraded", "http://go.dev", "http://go.dev", nil},
		{"empty urls rejected", "", "", ErrNoEmptyURLs},
		{"normalizing error", "%zzzz", "", ErrProblemNormalizingURL},
		{"ftp check", "ftp://go.dev", "ftp://go.dev", nil},

		// Length limit: test exactly at, just over, and far over the limit.
		{"url exactly at max length accepted",
			urlOfLength("https://example.com/", maxURLLength),
			urlOfLength("https://example.com/", maxURLLength), nil},
		{"url one over max length rejected",
			urlOfLength("https://example.com/", maxURLLength+1), "", ErrURLTooLong},
		{"very long url rejected",
			urlOfLength("https://example.com/", 10*maxURLLength), "", ErrURLTooLong},
		{"bare domain at max length gets https",
			urlOfLength("example.com/", maxURLLength),
			"https://" + urlOfLength("example.com/", maxURLLength), nil},
	}
	for _, test := range normalizeTests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeURL(test.input)
			want := test.want

			if err != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("Normalization failed for string:%q, Error:%s", test.input, err.Error())
			}

			if err == nil && test.wantErr != nil {
				t.Fatalf("Expected error %s", test.wantErr)
			}

			if err != nil && errors.Is(err, test.wantErr) {
				return
			}
			if got != want {
				t.Errorf("Got %s Expected %s", got, want)
			}

		})
	}

}

func TestValidateURL(t *testing.T) {
	var validateURlTests = []struct {
		name    string
		input   string
		wantErr error
	}{
		{"https accepted", "https://go.dev", nil},
		{"http accepted", "http://go.dev", nil},
		{"ftp rejected", "ftp://go.dev", ErrSchemeMustBeHTTPOrHTTPS},
		{"javascript rejected", "javascript:alert(1)", ErrSchemeMustBeHTTPOrHTTPS},
		{"missing host", "https://", ErrMissingHost},
		{"protocol-relative after normalize", "https:////evil.com", ErrMissingHost},
	}

	for _, test := range validateURlTests {
		t.Run(test.name, func(t *testing.T) {
			err := validateURL(test.input)

			if !errors.Is(err, test.wantErr) {
				t.Errorf("Expected %v, Got :%v", test.wantErr, err)
			}
		})
	}

}

func TestGenerateShortURL(t *testing.T) {

	lengths := []int{1, 7, 12, 20}

	for _, val := range lengths {
		testName := fmt.Sprintf("Test for length %d", val)
		stringsMap := make(map[string]bool)

		t.Run(testName, func(t *testing.T) {

			for range 100 {

				got, err := generateShortURL(val)

				if err != nil {
					t.Fatalf("Got error %s", err)
				}

				if len(got) != val {
					t.Errorf("Expected length %d, got %d", val, len(got))
				}

				for _, ch := range got {
					if !strings.ContainsRune(charset, ch) {
						t.Errorf("code %q contains %q which is not in the charset", got, ch)
					}
				}

				stringsMap[got] = true

			}

			if len(stringsMap) < 5 {
				t.Errorf("Generator stuck on small number of calls")
			}

		})

	}

}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	health(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)

	if err != nil {
		t.Errorf("Expected error nil got %v", err)
	}

	if !strings.Contains(string(data), "UP") {
		t.Errorf("Expected UP to be in string, but not found")
	}

}

func TestShorten(t *testing.T) {
	payload := []byte(`{"Url": "www.evil.com"}`)
	req := httptest.NewRequest("POST", "/shorten", bytes.NewBuffer(payload))
	w := httptest.NewRecorder()
	sendURLForShortening(w, req)
	res := w.Result()
	defer res.Body.Close()

	var got shortenBody

	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

}

// resetStore replaces the global maps with empty ones, so URLs stored by
// one test can't change the outcome of another. Every handler test that
// touches the maps calls this first.
func resetStore(t *testing.T) {
	t.Helper()
	lock.Lock()
	defer lock.Unlock()
	urls = make(map[string]string)
	originalURLMap = make(map[string]string)
}

// postShorten sends {"Url": rawURL} to the shorten handler and returns the
// status code and decoded response. It stops the test on a non-success
// status, because the body would then be a plain-text error, not JSON.
func postShorten(t *testing.T, rawURL string) (int, shortenBody) {
	t.Helper()

	// json.Marshal builds the body from the real struct, so quoting and
	// escaping are always correct, even for very long or odd URLs.
	payload, err := json.Marshal(urlBody{Url: rawURL})
	if err != nil {
		t.Fatalf("building request body for %q: %v", rawURL, err)
	}

	req := httptest.NewRequest(http.MethodPost, "/shorten", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	sendURLForShortening(w, req)

	// Check the status before touching the body.
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("POST %q: got status %d, body %q", rawURL, w.Code, w.Body.String())
	}

	var got shortenBody
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("POST %q: decoding response: %v", rawURL, err)
	}
	return w.Code, got
}

// Shortening the same URL twice must reuse the first code: 201 the first
// time (created), 200 the second (already existed), same short_url both times.
func TestShortenSameURLTwice(t *testing.T) {
	resetStore(t)

	firstStatus, first := postShorten(t, "go.dev")
	secondStatus, second := postShorten(t, "go.dev")

	if firstStatus != http.StatusCreated {
		t.Errorf("first POST: got status %d, want %d", firstStatus, http.StatusCreated)
	}
	if secondStatus != http.StatusOK {
		t.Errorf("second POST: got status %d, want %d", secondStatus, http.StatusOK)
	}
	if first.ShortURL != second.ShortURL {
		t.Errorf("same URL got two codes: %q then %q", first.ShortURL, second.ShortURL)
	}
}

// Different URLs must each get their own code. If they shared one,
// the redirect would send one of them to the wrong site.
func TestShortenDifferentURLs(t *testing.T) {
	resetStore(t)

	statusA, a := postShorten(t, "go.dev")
	statusB, b := postShorten(t, "example.com")

	if statusA != http.StatusCreated || statusB != http.StatusCreated {
		t.Errorf("got statuses %d and %d, want %d for both", statusA, statusB, http.StatusCreated)
	}
	if a.ShortURL == b.ShortURL {
		t.Errorf("different URLs got the same code %q", a.ShortURL)
	}
}
