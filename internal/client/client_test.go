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

func TestExecuteFailStatus(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		failStatus  bool
		wantError   bool
		errorString string
	}{
		{
			name:        "fail_flag_with_404",
			statusCode:  http.StatusNotFound,
			failStatus:  true,
			wantError:   true,
			errorString: "HTTP status 404",
		},
		{
			name:        "fail_flag_with_500",
			statusCode:  http.StatusInternalServerError,
			failStatus:  true,
			wantError:   true,
			errorString: "HTTP status 500",
		},
		{
			name:       "fail_flag_with_200",
			statusCode: http.StatusOK,
			failStatus: true,
			wantError:  false,
		},
		{
			name:       "no_fail_flag_with_404",
			statusCode: http.StatusNotFound,
			failStatus: false,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"error":"test"}`))
			}))
			defer server.Close()

			cfg := &config.Config{
				Method:     "GET",
				URL:        server.URL,
				Headers:    map[string]string{},
				Timeout:    5 * time.Second,
				FailStatus: tt.failStatus,
			}

			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			rOut, wOut, _ := os.Pipe()
			rErr, wErr, _ := os.Pipe()
			os.Stdout = wOut
			os.Stderr = wErr

			client, err := New(cfg)
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
			} else if err != nil {
				t.Errorf("Execute() error = %v, want nil", err)
			}
		})
	}
}

func TestClientExecuteForGUI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Authorization", "Bearer secret-token")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":1}`)
	}))
	defer server.Close()

	cfg := &config.Config{
		Method:  "POST",
		URL:     server.URL,
		Data:    `{"name":"x"}`,
		Headers: map[string]string{},
		Timeout: 5 * time.Second,
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	data, err := client.ExecuteForGUI()
	if err != nil {
		t.Fatalf("ExecuteForGUI() error = %v", err)
	}
	if data.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", data.StatusCode, http.StatusCreated)
	}
	if data.Body != "{\n  \"id\": 1\n}" {
		t.Errorf("Body = %q, want pretty-printed JSON", data.Body)
	}
	if got := data.Headers["Authorization"]; !strings.Contains(got, "MASKED") {
		t.Errorf("Authorization header should be masked, got %q", got)
	}
	if data.ElapsedMs < 0 {
		t.Errorf("ElapsedMs = %d, want >= 0", data.ElapsedMs)
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