package request

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"testing"
	"time"

	"ilo-pana/internal/config"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantError   bool
		errorString string
	}{
		{
			name:      "valid_https_url",
			url:       "https://api.example.com/v1/users",
			wantError: false,
		},
		{
			name:      "valid_http_url",
			url:       "http://api.example.com/v1/users",
			wantError: false,
		},
		{
			name:      "valid_url_with_port",
			url:       "https://api.example.com:8080/v1/users",
			wantError: false,
		},
		{
			name:        "empty_url",
			url:         "",
			wantError:   true,
			errorString: "URL cannot be empty",
		},
		{
			name:        "invalid_scheme_ftp",
			url:         "ftp://files.example.com/data",
			wantError:   true,
			errorString: "unsupported URL scheme",
		},
		{
			name:        "no_host",
			url:         "https:///path",
			wantError:   true,
			errorString: "URL must include a host",
		},
		{
			name:      "localhost_warning",
			url:       "http://localhost:8080/api",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			err := ValidateURL(tt.url)

			w.Close()
			os.Stderr = oldStderr

			output, _ := io.ReadAll(r)
			if strings.Contains(tt.url, "localhost") || strings.Contains(tt.url, "127.0.0.1") {
				if !strings.Contains(string(output), "Warning: Connecting to local address") {
					t.Errorf("Expected localhost warning for URL %q", tt.url)
				}
			}

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateURL(%q) error = nil, want error containing %q", tt.url, tt.errorString)
				} else if !strings.Contains(err.Error(), tt.errorString) {
					t.Errorf("ValidateURL(%q) error = %v, want error containing %q", tt.url, err, tt.errorString)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateURL(%q) error = %v, want nil", tt.url, err)
				}
			}
		})
	}
}

func TestValidateMethod(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		wantError bool
	}{
		{"valid_GET", "GET", false},
		{"valid_POST", "POST", false},
		{"valid_PUT", "PUT", false},
		{"valid_DELETE", "DELETE", false},
		{"valid_PATCH", "PATCH", false},
		{"invalid_INVALID", "INVALID", true},
		{"invalid_lowercase", "get", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMethod(tt.method)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateMethod(%q) error = nil, want error", tt.method)
				} else if !strings.Contains(err.Error(), "invalid HTTP method") {
					t.Errorf("ValidateMethod(%q) error = %v, want 'invalid HTTP method'", tt.method, err)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateMethod(%q) error = %v, want nil", tt.method, err)
				}
			}
		})
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid_object", `{"key": "value"}`, true},
		{"valid_array", `["item1", "item2"]`, true},
		{"valid_with_spaces", `  {"key": "value"}  `, true},
		{"valid_empty_object", `{}`, true},
		{"valid_empty_array", `[]`, true},
		{"invalid_plain_text", `plain text`, false},
		{"invalid_partial_json", `{"key": `, false},
		{"invalid_number", `123`, false},
		{"invalid_empty", ``, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsJSON(tt.input); got != tt.want {
				t.Errorf("IsJSON(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name      string
		config    *config.Config
		wantError bool
		checkFunc func(*testing.T, *config.Config)
	}{
		{
			name: "simple_get_request",
			config: &config.Config{
				Method:  "GET",
				URL:     "https://api.example.com/users",
				Headers: map[string]string{},
				Timeout: 30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "post_with_json_data",
			config: &config.Config{
				Method:  "POST",
				URL:     "https://api.example.com/users",
				Data:    `{"name":"test"}`,
				Headers: map[string]string{},
				Timeout: 30 * time.Second,
			},
			wantError: false,
		},
		{
			name: "invalid_url",
			config: &config.Config{
				Method:  "GET",
				URL:     "not-a-url",
				Headers: map[string]string{},
				Timeout: 30 * time.Second,
			},
			wantError: true,
		},
		{
			name: "invalid_method",
			config: &config.Config{
				Method:  "INVALID",
				URL:     "https://api.example.com",
				Headers: map[string]string{},
				Timeout: 30 * time.Second,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := Build(tt.config)
			if tt.wantError {
				if err == nil {
					t.Error("Build() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("Build() error = %v, want nil", err)
				}
				if req == nil {
					t.Error("Build() returned nil request")
				}
			}
		})
	}
}

func TestBuildMultipart(t *testing.T) {
	cfg := &config.Config{
		Method: "POST",
		URL:    "https://api.example.com/upload",
		Headers: map[string]string{
			"Content-Type": "application/json", // must be overridden by the boundary value
		},
		BodyFormat: "multipart",
		FormFields: []config.FormField{
			{Key: "title", Value: "my file"},
			{Key: "file", IsFile: true, FileName: "doc.txt", ContentType: "text/plain", FileContent: []byte("hello")},
			{Key: `weird"name`, Value: "x"},
		},
		Timeout: 30 * time.Second,
	}

	req, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("Content-Type = %q, want multipart with boundary", contentType)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	parsed, err := multipart.NewReader(bytes.NewReader(body), strings.TrimPrefix(contentType, "multipart/form-data; boundary=")).ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("failed to parse multipart body: %v\n%s", err, body)
	}
	if got := parsed.Value["title"][0]; got != "my file" {
		t.Errorf("title = %q, want my file", got)
	}
	if got := parsed.Value[`weird"name`][0]; got != "x" {
		t.Errorf("weird name field = %q, want x", got)
	}
	fileHeader := parsed.File["file"][0]
	if fileHeader.Filename != "doc.txt" {
		t.Errorf("filename = %q, want doc.txt", fileHeader.Filename)
	}
	f, err := fileHeader.Open()
	if err != nil {
		t.Fatalf("failed to open file part: %v", err)
	}
	defer f.Close()
	data, _ := io.ReadAll(f)
	if string(data) != "hello" {
		t.Errorf("file content = %q, want hello", data)
	}
}

func TestBuildURLEncoded(t *testing.T) {
	cfg := &config.Config{
		Method:     "POST",
		URL:        "https://api.example.com/form",
		Headers:    map[string]string{},
		BodyFormat: "urlencoded",
		FormFields: []config.FormField{
			{Key: "name", Value: "alice smith"},
			{Key: "age", Value: "30"},
			{Key: "skip", IsFile: true}, // files are ignored for urlencoded
		},
		Timeout: 30 * time.Second,
	}

	req, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", got)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	want := "age=30&name=alice+smith"
	if string(body) != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestBuildURLEncodedMultipartFileContentType(t *testing.T) {
	// Non-form bodies must keep the auto-set JSON Content-Type.
	cfg := &config.Config{
		Method:  "POST",
		URL:     "https://api.example.com/x",
		Data:    `{"a":1}`,
		Headers: map[string]string{},
		Timeout: 30 * time.Second,
	}
	req, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got := req.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
}

func BenchmarkValidateURL(b *testing.B) {
	urls := []string{
		"https://api.example.com/v1/users",
		"http://localhost:8080/test",
		"https://sub.domain.example.com:443/path/to/resource?query=value",
	}

	// Suppress stderr during benchmark
	oldStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldStderr }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_ = ValidateURL(url)
		}
	}
}

func TestBuildRejectsUnknownBodyFormat(t *testing.T) {
	cfg := &config.Config{
		Method:     "POST",
		URL:        "https://api.example.com/x",
		Headers:    map[string]string{},
		BodyFormat: config.BodyFormat("multi-part"),
		Timeout:    5 * time.Second,
	}
	_, err := Build(cfg)
	if err == nil {
		t.Fatal("Build() error = nil, want unknown body format error")
	}
	if !strings.Contains(err.Error(), "unknown body format") {
		t.Errorf("Build() error = %v, want unknown body format error", err)
	}
}

func TestEscapeQuotes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", `doc.txt`, `doc.txt`},
		{"quote", `a"b`, `a\"b`},
		{"crlf", "a\r\nb", `a  b`},
		{"lone cr", "a\rb", `a b`},
		{"lone lf", "a\nb", `a b`},
		{"backslash", `a\b`, `a\\b`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := escapeQuotes(tc.in); got != tc.want {
				t.Errorf("escapeQuotes(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func BenchmarkIsJSON(b *testing.B) {
	testStrings := []string{
		`{"key": "value", "nested": {"a": 1, "b": 2}}`,
		`plain text that is not JSON`,
		`[1, 2, 3, 4, 5]`,
		`<xml>not json</xml>`,
		`{}`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, s := range testStrings {
			_ = IsJSON(s)
		}
	}
}