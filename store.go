package main

import (
	"fmt"
	"log/slog"
	"sync"
)

const codeLength = 7
const maxCodeAttempts = 20

type MemoryStore struct {
	lock         sync.RWMutex
	codeToURL    map[string]string
	urlToCode    map[string]string
	generateCode func(int) (string, error)
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		codeToURL:    make(map[string]string),
		urlToCode:    make(map[string]string),
		generateCode: generateShortURL,
	}
}

func (s *MemoryStore) GetOrCreate(url string) (code string, existed bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	existingCode, ok := s.urlToCode[url]

	if ok {
		slog.Info("url already existing in code", "existing_url", existingCode)
		return existingCode, true, nil
	} else {
		//Check for collision

		for range maxCodeAttempts {

			code, err = s.generateCode(codeLength)

			if err != nil {
				slog.Error("URL could not be shortened", "err", err)
				return "", false, fmt.Errorf("URL could not be shortened")
			}

			// Free code: save it and return straight from inside the loop.
			if _, taken := s.codeToURL[code]; !taken {
				s.urlToCode[url] = code
				s.codeToURL[code] = url
				return code, false, nil
			}
		}

		// Reaching here means every attempt collided.
		slog.Error("Could not generate URL due to collisions")
		return "", false, ErrNoFreeCode
	}
}

func (s *MemoryStore) Lookup(code string) (string, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	url, ok := s.codeToURL[code]
	if ok {
		return url, true
	}

	return "", false

}
