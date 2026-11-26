package main

import (
	"errors"
	"testing"
)

// When every generated code is already taken, GetOrCreate must give up
// with ErrNoFreeCode, not loop forever or overwrite an existing link.
func TestGetOrCreateGivesUpOnCollisions(t *testing.T) {
	store := NewMemoryStore()

	// Swap the real generator for a fake that ALWAYS returns the same code.
	// The int (the length) is ignored, so it has no name.
	store.generateCode = func(int) (string, error) {
		return "aaaaaaa", nil
	}

	// First URL: "aaaaaaa" is still free, so this must succeed.
	if _, _, err := store.GetOrCreate("https://go.dev"); err != nil {
		t.Fatalf("first URL: unexpected error %v", err)
	}

	// Second URL: every attempt returns "aaaaaaa", which is now taken.
	_, _, err := store.GetOrCreate("https://example.com")
	if !errors.Is(err, ErrNoFreeCode) {
		t.Fatalf("second URL: got error %v, want %v", err, ErrNoFreeCode)
	}

	// The failure must not have damaged anything: still exactly one link,
	// and "aaaaaaa" still points to the FIRST URL.
	if len(store.urlToCode) != 1 || store.codeToURL["aaaaaaa"] != "https://go.dev" {
		t.Errorf("store changed after failure: %v", store.codeToURL)
	}
}

func TestGetOrCreateRetriesOnCollisions(t *testing.T) {
	store := NewMemoryStore()
	calls := 0
	store.generateCode = func(int) (string, error) {
		calls += 1

		if calls < 4 {

			return "aaaaaaa", nil

		}
		return "bbbbbbb", nil

	}
	_, _, err := store.GetOrCreate("www.google.com")

	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	code, _, err := store.GetOrCreate("www.king.com")

	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if code != "bbbbbbb" {
		t.Errorf("Code expected bbbbbbb, got %s", code)
	}

	if calls != 4 {
		t.Errorf("incorrect calls made")
	}

}

func TestGeneratorFailureCase(t *testing.T) {
	store := NewMemoryStore()
	store.generateCode = func(i int) (string, error) {
		return "", errors.New("random source broken")
	}

	_, _, err := store.GetOrCreate("www.c.com")

	if err == nil {
		t.Fatalf("expected error but did not get error")
	}

	if len(store.codeToURL) != 0 || len(store.urlToCode) != 0 {
		t.Fatalf("expected store maps to be empty but they are not")
	}

}

// Boundary: a free code on the very LAST allowed attempt must still be used.
// An off-by-one in the loop limit would give up one attempt too early.
func TestGetOrCreateSucceedsOnLastAttempt(t *testing.T) {
	store := NewMemoryStore()
	calls := 0
	store.generateCode = func(int) (string, error) {
		calls++
		// Call 1 is the first URL. Calls 2..maxCodeAttempts are the second
		// URL's first maxCodeAttempts-1 attempts, which all collide.
		if calls <= maxCodeAttempts {
			return "aaaaaaa", nil
		}
		return "bbbbbbb", nil // the second URL's final attempt
	}

	if _, _, err := store.GetOrCreate("https://go.dev"); err != nil {
		t.Fatalf("first URL: unexpected error %v", err)
	}

	code, _, err := store.GetOrCreate("https://example.com")
	if err != nil {
		t.Fatalf("second URL: got error %v, want success on the last attempt", err)
	}
	if code != "bbbbbbb" {
		t.Errorf("got code %q, want %q", code, "bbbbbbb")
	}
	if want := maxCodeAttempts + 1; calls != want {
		t.Errorf("generator called %d times, want %d", calls, want)
	}
}
