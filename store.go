package main

import (
	"fmt"
	"log/slog"
	"sync"
)

type MemoryStore struct {
	lock      sync.RWMutex
	codeToURL map[string]string
	urlToCode map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		codeToURL: make(map[string]string),
		urlToCode: make(map[string]string),
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

		tries := 20
		for tries > 0 {

			code, err = generateShortURL(7)

			if err != nil {
				slog.Error("URL could not be shortened", "err", err)
				return "", false, fmt.Errorf("URL could not be shortened")
			}

			_, ok = s.codeToURL[code]

			if !ok {
				break
			}

			tries -= 1

		}

		if tries <= 0 {
			slog.Error("Could not generate URL due to collisions")
			return "", false, fmt.Errorf("Could not shorten URL due to collision error")
		}

		s.urlToCode[url] = code
		s.codeToURL[code] = url

		return code, false, nil
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
