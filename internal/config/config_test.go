package config

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func TestHeaderList(t *testing.T) {
	tests := []struct {
		name        string
		initial     []string
		setValue    string
		wantError   bool
		wantLen     int
		wantString  string
		errorString string
	}{
		{
			name:       "add_single_header",
			initial:    []string{},
			setValue:   "Content-Type: application/json",
			wantError:  false,
			wantLen:    1,
			wantString: "Content-Type: application/json",
		},
		{
			name:       "add_to_existing",
			initial:    []string{"Authorization: Bearer token"},
			setValue:   "Accept: application/json",
			wantError:  false,
			wantLen:    2,
			wantString: "Authorization: Bearer token, Accept: application/json",
		},
		{
			name:        "empty_header_error",
			initial:     []string{},
			setValue:    "",
			wantError:   true,
			wantLen:     0,
			wantString:  "",
			errorString: "header value cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := headerList(tt.initial)
			err := h.Set(tt.setValue)

			if tt.wantError {
				if err == nil {
					t.Errorf("Set() error = nil, want error containing %q", tt.errorString)
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("Set() error = %v, want error containing %q", err, tt.errorString)
				}
			} else {
				if err != nil {
					t.Errorf("Set() error = %v, want nil", err)
				}
			}

			if len(h) != tt.wantLen {
				t.Errorf("len(headerList) = %d, want %d", len(h), tt.wantLen)
			}

			if got := h.String(); got != tt.wantString {
				t.Errorf("String() = %q, want %q", got, tt.wantString)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     []string
		want        map[string]string
		wantError   bool
		errorString string
	}{
		{
			name:    "single_header",
			headers: []string{"Content-Type: application/json"},
			want: map[string]string{
				"Content-Type": "application/json",
			},
			wantError: false,
		},
		{
			name: "multiple_headers",
			headers: []string{
				"Content-Type: application/json",
				"Authorization: Bearer token123",
				"X-Custom-Header: custom-value",
			},
			want: map[string]string{
				"Content-Type":    "application/json",
				"Authorization":   "Bearer token123",
				"X-Custom-Header": "custom-value",
			},
			wantError: false,
		},
		{
			name: "header_with_colon_in_value",
			headers: []string{
				"Time: 12:30:45",
				"URL: https://example.com",
			},
			want: map[string]string{
				"Time": "12:30:45",
				"URL":  "https://example.com",
			},
			wantError: false,
		},
		{
			name:        "invalid_header_no_colon",
			headers:     []string{"InvalidHeader"},
			want:        nil,
			wantError:   true,
			errorString: "invalid header format",
		},
		{
			name:        "empty_header_key",
			headers:     []string{": value"},
			want:        nil,
			wantError:   true,
			errorString: "empty header key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr to suppress warnings during test
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			got, err := parseHeaders(tt.headers)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr
			io.ReadAll(r)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseHeaders() error = nil, want error containing %q", tt.errorString)
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("parseHeaders() error = %v, want error containing %q", err, tt.errorString)
				}
				return
			}

			if err != nil {
				t.Errorf("parseHeaders() error = %v, want nil", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseHeaders() returned %d headers, want %d", len(got), len(tt.want))
				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseHeaders()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestParseIntegration(t *testing.T) {
	// Save original args and flags
	origArgs := os.Args
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
	}()

	tests := []struct {
		name      string
		args      []string
		wantError bool
	}{
		{
			name:      "valid_simple_request",
			args:      []string{"cmd", "https://api.example.com"},
			wantError: false,
		},
		{
			name:      "missing_url",
			args:      []string{"cmd"},
			wantError: true,
		},
		{
			name:      "with_method_and_headers",
			args:      []string{"cmd", "-X", "POST", "-H", "Content-Type: application/json", "https://api.example.com"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
			flag.CommandLine.SetOutput(io.Discard)

			os.Args = tt.args
			_, err := Parse()

			if tt.wantError {
				if err == nil {
					t.Errorf("Parse() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("Parse() error = %v, want nil", err)
				}
			}
		})
	}
}