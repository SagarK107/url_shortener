package main

// Settings that change between environments. In Phase 3 these become
// environment variables (PORT, BASE_URL) with these values as defaults.
const (
	// port is where this process listens for connections.
	port = "8080"

	// baseURL is the public origin users type to reach the service:
	// scheme + host (+ port when it isn't the default). No trailing slash.
	baseURL = "http://localhost:8080"
)
