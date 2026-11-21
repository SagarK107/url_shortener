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
	"sync"
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
		{"bare domain has no scheme", "go.dev", ErrSchemeMustBeHTTPOrHTTPS},
		{"invalid percent escape", "https://go.dev/%zz", ErrURLCouldNotBeParsed},
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
		t.Fatalf("Expected error nil got %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", res.StatusCode, http.StatusOK)
	}

	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("got Content-Type %q, want %q", got, "application/json")
	}

	if !strings.Contains(string(data), "UP") {
		t.Errorf("Expected UP in body, got %q", data)
	}

}

// TestShorten checks valid URLs end to end through the handler: each one is
// created (201), gets a short URL on our domain, and comes back normalized.
// That proves the handler really uses normalizeURL, not just that
// normalizeURL works on its own.
func TestShorten(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantOriginal string
	}{
		{"bare domain gets https", "go.dev", "https://go.dev"},
		{"https kept", "https://go.dev", "https://go.dev"},
		{"http not upgraded", "http://go.dev", "http://go.dev"},
		{"path and query kept", "https://go.dev/doc?x=1", "https://go.dev/doc?x=1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Reset per row: "go.dev" and "https://go.dev" normalize to the
			// same URL, so without this the second row would get 200, not 201.
			resetStore(t)

			status, got := postShorten(t, test.input)

			if status != http.StatusCreated {
				t.Errorf("POST %q: got status %d, want %d", test.input, status, http.StatusCreated)
			}
			if !strings.HasPrefix(got.ShortURL, baseURL+"/") {
				t.Errorf("POST %q: short_url %q does not start with %q", test.input, got.ShortURL, baseURL+"/")
			}
			if got.OriginalURL != test.wantOriginal {
				t.Errorf("POST %q: original_url %q, want %q", test.input, got.OriginalURL, test.wantOriginal)
			}
		})
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

// bodyFor builds a valid JSON request body for rawURL. It needs no
// *testing.T, so it is safe to call from any goroutine.
func bodyFor(rawURL string) string {
	payload, _ := json.Marshal(urlBody{Url: rawURL}) // a struct with one string field can't fail
	return string(payload)
}

// Bad requests must get the right status and must store nothing.
func TestShortenRejectsBadInput(t *testing.T) {
	tooLong := urlOfLength("https://example.com/", maxURLLength+1)

	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{"wrong method", http.MethodGet, "", http.StatusMethodNotAllowed},
		{"not json", http.MethodPost, "not json", http.StatusBadRequest},
		{"empty body", http.MethodPost, "", http.StatusBadRequest},
		{"empty url", http.MethodPost, bodyFor(""), http.StatusBadRequest},
		{"unsupported scheme", http.MethodPost, bodyFor("ftp://go.dev"), http.StatusBadRequest},
		{"javascript scheme", http.MethodPost, bodyFor("javascript:alert(1)"), http.StatusBadRequest},
		{"missing host", http.MethodPost, bodyFor("https://"), http.StatusBadRequest},
		{"protocol-relative", http.MethodPost, bodyFor("//evil.com"), http.StatusBadRequest},
		{"invalid escape", http.MethodPost, bodyFor("%zz"), http.StatusBadRequest},
		{"too long", http.MethodPost, bodyFor(tooLong), http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resetStore(t)

			req := httptest.NewRequest(test.method, "/shorten", strings.NewReader(test.body))
			w := httptest.NewRecorder()
			sendURLForShortening(w, req)

			if w.Code != test.wantStatus {
				t.Errorf("got status %d, want %d, body %q", w.Code, test.wantStatus, w.Body.String())
			}
			if len(urls) != 0 || len(originalURLMap) != 0 {
				t.Errorf("rejected request was stored: urls=%v codes=%v", urls, originalURLMap)
			}
		})
	}
}

// getRedirect calls the redirect handler directly. A direct call skips the
// router, so the path value the router would normally fill in is set here.
func getRedirect(method, code string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/"+code, nil)
	req.SetPathValue("shortenedURL", code)
	w := httptest.NewRecorder()
	redirectFromShortURL(w, req)
	return w
}

// codeFrom removes the base URL from a short URL, leaving only the code.
func codeFrom(t *testing.T, shortURL string) string {
	t.Helper()
	code, ok := strings.CutPrefix(shortURL, baseURL+"/")
	if !ok || code == "" {
		t.Fatalf("short URL %q has no code after %q", shortURL, baseURL+"/")
	}
	return code
}

func TestRedirect(t *testing.T) {
	resetStore(t)
	_, created := postShorten(t, "https://go.dev/doc")
	knownCode := codeFrom(t, created.ShortURL)

	tests := []struct {
		name         string
		method       string
		code         string
		wantStatus   int
		wantLocation string
	}{
		{"known code redirects", http.MethodGet, knownCode, http.StatusTemporaryRedirect, "https://go.dev/doc"},
		{"unknown code", http.MethodGet, "missing", http.StatusNotFound, ""},
		{"wrong method", http.MethodPost, knownCode, http.StatusMethodNotAllowed, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := getRedirect(test.method, test.code)

			if w.Code != test.wantStatus {
				t.Errorf("%s /%s: got status %d, want %d", test.method, test.code, w.Code, test.wantStatus)
			}
			if got := w.Header().Get("Location"); got != test.wantLocation {
				t.Errorf("%s /%s: got Location %q, want %q", test.method, test.code, got, test.wantLocation)
			}
		})
	}
}

// The whole product in one test: shorten a URL, then follow its code.
func TestShortenThenRedirect(t *testing.T) {
	resetStore(t)

	_, created := postShorten(t, "go.dev/learn")
	w := getRedirect(http.MethodGet, codeFrom(t, created.ShortURL))

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("got status %d, want %d", w.Code, http.StatusTemporaryRedirect)
	}
	if got, want := w.Header().Get("Location"), "https://go.dev/learn"; got != want {
		t.Errorf("got Location %q, want %q", got, want)
	}
}

// postConcurrently sends one request per input, all released at once.
// It uses t.Errorf rather than postShorten because t.Fatalf must only be
// called from the test's own goroutine.
func postConcurrently(t *testing.T, inputs []string) ([]int, []shortenBody) {
	t.Helper()

	statuses := make([]int, len(inputs))
	bodies := make([]shortenBody, len(inputs))
	start := make(chan struct{}) // closed to release every goroutine together

	var wg sync.WaitGroup
	for i, input := range inputs {
		wg.Go(func() {
			<-start
			req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(bodyFor(input)))
			w := httptest.NewRecorder()
			sendURLForShortening(w, req)

			// Each goroutine writes only its own index, so no lock is needed.
			statuses[i] = w.Code
			if err := json.NewDecoder(w.Body).Decode(&bodies[i]); err != nil {
				t.Errorf("POST %q: status %d, decoding response: %v", input, w.Code, err)
			}
		})
	}
	close(start)
	wg.Wait()
	return statuses, bodies
}

// 50 simultaneous requests for one URL must create exactly one code.
// This is the check-then-act race the single write lock prevents.
func TestConcurrentSameURL(t *testing.T) {
	resetStore(t)

	inputs := make([]string, 50)
	for i := range inputs {
		inputs[i] = "https://go.dev/same"
	}
	statuses, bodies := postConcurrently(t, inputs)

	created := 0
	for _, status := range statuses {
		if status == http.StatusCreated {
			created++
		}
	}
	if created != 1 {
		t.Errorf("got %d responses with 201, want exactly 1", created)
	}
	for _, body := range bodies {
		if body.ShortURL != bodies[0].ShortURL {
			t.Fatalf("same URL got different codes: %q and %q", bodies[0].ShortURL, body.ShortURL)
		}
	}
	if len(urls) != 1 || len(originalURLMap) != 1 {
		t.Errorf("stored %d URLs and %d codes, want 1 of each", len(urls), len(originalURLMap))
	}
}

// 50 simultaneous requests for different URLs must each get a working code.
func TestConcurrentDifferentURLs(t *testing.T) {
	resetStore(t)

	inputs := make([]string, 50)
	for i := range inputs {
		inputs[i] = fmt.Sprintf("https://go.dev/page/%d", i)
	}
	statuses, bodies := postConcurrently(t, inputs)

	seen := make(map[string]bool)
	for i, body := range bodies {
		if statuses[i] != http.StatusCreated {
			t.Errorf("POST %q: got status %d, want %d", inputs[i], statuses[i], http.StatusCreated)
			continue
		}
		if seen[body.ShortURL] {
			t.Errorf("duplicate short URL %q", body.ShortURL)
		}
		seen[body.ShortURL] = true

		w := getRedirect(http.MethodGet, codeFrom(t, body.ShortURL))
		if got := w.Header().Get("Location"); got != inputs[i] {
			t.Errorf("code for %q redirects to %q", inputs[i], got)
		}
	}
}
