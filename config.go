package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds the settings that change between environments. Values come
// from environment variables; anything unset falls back to a default that
// works for local development.
type Config struct {
	Port    string // PORT: the port this process listens on
	Host    string // HOST: the interface to listen on; "0.0.0.0" when deployed
	BaseURL string // BASE_URL: public origin used in short URLs, no trailing slash
}

// Defaults for local development.
const (
	defaultPort    = "8080"
	defaultHost    = "localhost"
	defaultBaseURL = "http://localhost:8080"
)

// loadDotEnv copies a local .env file into the environment, if there is one.
// The file is a development convenience: deployed servers and CI have no
// .env, so a missing file is normal and not an error. Variables that are
// already set always win, because godotenv never overrides them.
func loadDotEnv() error {
	err := godotenv.Load()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// loadConfig reads and validates the configuration. It returns an error
// instead of exiting, so main decides what to do and tests can check it.
func loadConfig() (Config, error) {
	cfg := Config{
		Port:    getenv("PORT", defaultPort),
		Host:    getenv("HOST", defaultHost),
		BaseURL: strings.TrimRight(getenv("BASE_URL", defaultBaseURL), "/"),
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a number from 1 to 65535, got %q", cfg.Port)
	}

	u, err := url.Parse(cfg.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Config{}, fmt.Errorf("BASE_URL must be an http or https URL with a host, got %q", cfg.BaseURL)
	}

	return cfg, nil
}

// Addr is the address to listen on, such as "localhost:8080".
// net.JoinHostPort also handles IPv6 hosts, which need square brackets.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, c.Port)
}

// getenv returns the trimmed value of key, or fallback if it is unset or blank.
func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
