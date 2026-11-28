package main

import "testing"

func TestLoadConfig(t *testing.T) {
	defaults := Config{Port: defaultPort, Host: defaultHost, BaseURL: defaultBaseURL}

	tests := []struct {
		name    string
		env     map[string]string // variables not listed here are set to ""
		want    Config
		wantErr bool
	}{
		{"defaults when nothing is set", nil, defaults, false},
		{"all values overridden",
			map[string]string{"PORT": "9090", "HOST": "0.0.0.0", "BASE_URL": "https://sho.rt"},
			Config{Port: "9090", Host: "0.0.0.0", BaseURL: "https://sho.rt"}, false},
		{"trailing slash trimmed",
			map[string]string{"BASE_URL": "https://sho.rt/"},
			Config{Port: defaultPort, Host: defaultHost, BaseURL: "https://sho.rt"}, false},
		{"surrounding spaces trimmed",
			map[string]string{"PORT": " 9090 "},
			Config{Port: "9090", Host: defaultHost, BaseURL: defaultBaseURL}, false},

		// Port boundaries: the lowest and highest valid ports, and one past each.
		{"port 1 accepted", map[string]string{"PORT": "1"},
			Config{Port: "1", Host: defaultHost, BaseURL: defaultBaseURL}, false},
		{"port 65535 accepted", map[string]string{"PORT": "65535"},
			Config{Port: "65535", Host: defaultHost, BaseURL: defaultBaseURL}, false},
		{"port 0 rejected", map[string]string{"PORT": "0"}, Config{}, true},
		{"port 65536 rejected", map[string]string{"PORT": "65536"}, Config{}, true},
		{"negative port rejected", map[string]string{"PORT": "-5"}, Config{}, true},
		{"non-numeric port rejected", map[string]string{"PORT": "abc"}, Config{}, true},

		{"base URL with ftp scheme rejected", map[string]string{"BASE_URL": "ftp://sho.rt"}, Config{}, true},
		{"base URL without scheme rejected", map[string]string{"BASE_URL": "sho.rt"}, Config{}, true},
		{"base URL without host rejected", map[string]string{"BASE_URL": "https://"}, Config{}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set every variable, so the real environment (or a .env file
			// loaded earlier) can't leak into the test. t.Setenv restores
			// the original value when the subtest ends.
			for _, key := range []string{"PORT", "HOST", "BASE_URL"} {
				t.Setenv(key, test.env[key]) // a missing key gives ""
			}

			got, err := loadConfig()

			if test.wantErr {
				if err == nil {
					t.Fatalf("env %v: got config %+v, want an error", test.env, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("env %v: unexpected error: %v", test.env, err)
			}
			if got != test.want {
				t.Errorf("env %v: got %+v, want %+v", test.env, got, test.want)
			}
		})
	}
}

func TestConfigAddr(t *testing.T) {
	tests := []struct {
		host, port, want string
	}{
		{"localhost", "8080", "localhost:8080"},
		{"0.0.0.0", "8080", "0.0.0.0:8080"},
		{"::1", "8080", "[::1]:8080"}, // IPv6 needs brackets
	}
	for _, test := range tests {
		if got := (Config{Host: test.host, Port: test.port}).Addr(); got != test.want {
			t.Errorf("Addr() for host %q port %q = %q, want %q", test.host, test.port, got, test.want)
		}
	}
}
