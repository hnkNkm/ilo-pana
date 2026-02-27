package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"ilo-pana/internal/config"
)

func TestClientExecute(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		serverHandler http.HandlerFunc
		wantError     bool
		errorString   string
	}{
		{
			name: "successful_get_request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET, got %s", r.Method)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"status":"ok"}`)
			},
			config: &config.Config{
				Method:  "GET",
				Headers: map[string]string{},
				Timeout: 5 * time.Second,
			},
			wantError: false,
		},
		{
			name: "successful_post_with_data",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST, got %s", r.Method)
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"test":"data"}` {
					t.Errorf("Unexpected body: %s", body)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type header")
				}
				w.WriteHeader(http.StatusCreated)
			},
			config: &config.Config{
				Method:  "POST",
				Data:    `{"test":"data"}`,
				Headers: map[string]string{},
				Timeout: 5 * time.Second,
			},
			wantError: false,
		},
		{
			name: "custom_headers",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Custom-Header") != "custom-value" {
					t.Errorf("Expected custom header")
				}
				w.WriteHeader(http.StatusOK)
			},
			config: &config.Config{
				Method: "GET",
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
				},
				Timeout: 5 * time.Second,
			},
			wantError: false,
		},
		{
			name: "timeout_error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(100 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			config: &config.Config{
				Method:  "GET",
				Headers: map[string]string{},
				Timeout: 10 * time.Millisecond,
			},
			wantError:   true,
			errorString: "request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			// Update config with test server URL
			tt.config.URL = server.URL

			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			client, err := New(tt.config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			err = client.Execute()

			// Restore stdout and stderr
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			io.ReadAll(rOut)
			io.ReadAll(rErr)

			if tt.wantError {
				if err == nil {
					t.Errorf("Execute() error = nil, want error containing %q", tt.errorString)
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tt.errorString)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestClientWithInvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		wantError   bool
		errorString string
	}{
		{
			name: "invalid_url",
			config: &config.Config{
				Method:  "GET",
				URL:     "not-a-url",
				Headers: map[string]string{},
				Timeout: 5 * time.Second,
			},
			wantError:   true,
			errorString: "invalid URL",
		},
		{
			name: "invalid_method",
			config: &config.Config{
				Method:  "INVALID",
				URL:     "https://example.com",
				Headers: map[string]string{},
				Timeout: 5 * time.Second,
			},
			wantError:   true,
			errorString: "invalid HTTP method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			client, err := New(tt.config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			err = client.Execute()

			// Restore stdout and stderr
			wOut.Close()
			wErr.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			io.ReadAll(rOut)
			io.ReadAll(rErr)

			if tt.wantError {
				if err == nil {
					t.Errorf("Execute() error = nil, want error")
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Execute() error = %v, want error containing %q", err, tt.errorString)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
			}
		})
	}
}

func BenchmarkClientExecute(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Timeout: 5 * time.Second,
	}

	// Suppress output during benchmark
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout, _ = os.Open(os.DevNull)
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, _ := New(cfg)
		_ = client.Execute()
	}
}