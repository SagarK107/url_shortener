package main

import (
	"errors"
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
		got     string
		want    string
		wantErr error
	}{
		{"bare domain gets https", "go.dev", "https://go.dev", nil},
		{"https unchanged", "https://go.dev", "https://go.dev", nil},
		{"http not upgraded", "http://go.dev", "http://go.dev", nil},
		{"empty urls rejected", "", "", ErrNoEmptyURLs},

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
			got, err := normalizeURL(test.got)
			want := test.want

			if err != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("Normalization failed for string:%s, Error:%s", got, err.Error())
			}

			if err == nil && test.wantErr != nil {
				t.Errorf("Expected error %s", test.wantErr)
			}

			if got != want {
				t.Errorf("Got %s Expected %s", got, want)
			}

		})
	}

}
