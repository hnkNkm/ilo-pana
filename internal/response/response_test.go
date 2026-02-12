package response

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		verbose bool
		want    string
	}{
		{
			name:    "authorization_header_masked",
			key:     "Authorization",
			value:   "Bearer secret-token-123456789",
			verbose: false,
			want:    "Bearer***MASKED***",
		},
		{
			name:    "authorization_header_verbose",
			key:     "Authorization",
			value:   "Bearer secret-token-123456789",
			verbose: true,
			want:    "Bearer secret-token-123456789",
		},
		{
			name:    "api_key_header_masked",
			key:     "X-API-Key",
			value:   "sk_live_123456789abcdef",
			verbose: false,
			want:    "sk_liv***MASKED***",
		},
		{
			name:    "short_sensitive_value",
			key:     "X-Auth-Token",
			value:   "short",
			verbose: false,
			want:    "***MASKED***",
		},
		{
			name:    "non_sensitive_header",
			key:     "Content-Type",
			value:   "application/json",
			verbose: false,
			want:    "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Verbose: tt.verbose}
			got := h.maskSensitiveValue(tt.key, tt.value)
			if got != tt.want {
				t.Errorf("maskSensitiveValue(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestPrettyPrintJSON(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		jsonStr   string
		wantError bool
		wantOut   string
	}{
		{
			name:      "valid_simple_json",
			prefix:    "",
			jsonStr:   `{"name":"test","id":1}`,
			wantError: false,
			wantOut: `{
  "id": 1,
  "name": "test"
}
`,
		},
		{
			name:      "valid_json_with_prefix",
			prefix:    "→ ",
			jsonStr:   `{"key":"value"}`,
			wantError: false,
			wantOut: `→ {
→   "key": "value"
→ }
`,
		},
		{
			name:      "invalid_json",
			prefix:    "",
			jsonStr:   `{invalid json}`,
			wantError: true,
			wantOut:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := PrettyPrintJSON(tt.prefix, tt.jsonStr)

			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)

			if tt.wantError {
				if err == nil {
					t.Errorf("PrettyPrintJSON() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("PrettyPrintJSON() error = %v, want nil", err)
				}
				if string(output) != tt.wantOut {
					t.Errorf("PrettyPrintJSON() output = %q, want %q", string(output), tt.wantOut)
				}
			}
		})
	}
}

func TestProcessResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		verbose  bool
		elapsed  time.Duration
	}{
		{
			name: "json_response",
			response: &http.Response{
				Status:     "200 OK",
				StatusCode: 200,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
					"X-API-Key":    []string{"secret-key"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"result":"success"}`)),
			},
			verbose: false,
			elapsed: 1500 * time.Millisecond,
		},
		{
			name: "empty_response",
			response: &http.Response{
				Status:     "204 No Content",
				StatusCode: 204,
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString("")),
			},
			verbose: false,
			elapsed: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			h := New(tt.verbose)
			err := h.Process(tt.response, tt.elapsed)

			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)

			if err != nil {
				t.Errorf("Process() error = %v, want nil", err)
			}

			outputStr := string(output)

			// Check for status line
			if !strings.Contains(outputStr, "HTTP/") {
				t.Error("Output should contain HTTP status line")
			}

			// Check for timing information
			if !strings.Contains(outputStr, "s)") {
				t.Error("Output should contain timing information")
			}
		})
	}
}

func TestPrintRequest(t *testing.T) {
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer secret"},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	PrintRequest("POST", "https://api.example.com/test", headers, `{"test":"data"}`, false)

	w.Close()
	os.Stdout = oldStdout
	output, _ := io.ReadAll(r)

	outputStr := string(output)

	// Check that output contains expected elements
	if !strings.Contains(outputStr, "POST https://api.example.com/test") {
		t.Error("Output should contain method and URL")
	}

	if !strings.Contains(outputStr, "Content-Type: application/json") {
		t.Error("Output should contain Content-Type header")
	}

	if !strings.Contains(outputStr, "Authorization: Bearer***MASKED***") {
		t.Error("Output should mask Authorization header")
	}

	if !strings.Contains(outputStr, `"test": "data"`) {
		t.Error("Output should contain prettified JSON body")
	}
}

func BenchmarkMaskSensitiveValue(b *testing.B) {
	testCases := []struct {
		key   string
		value string
	}{
		{"Authorization", "Bearer very-long-secret-token-123456789"},
		{"Content-Type", "application/json"},
		{"X-API-Key", "sk_live_abcdef123456"},
		{"Accept", "text/html"},
	}

	h := &Handler{Verbose: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = h.maskSensitiveValue(tc.key, tc.value)
		}
	}
}