// Package request handles HTTP request building and validation
package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"ilo-pana/internal/config"
)

// ValidateMethods defines all valid HTTP methods
var ValidMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"PATCH":   true,
	"HEAD":    true,
	"OPTIONS": true,
	"TRACE":   true,
	"CONNECT": true,
}

// Build creates an HTTP request from the configuration
func Build(cfg *config.Config) (*http.Request, error) {
	if err := ValidateURL(cfg.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if err := ValidateMethod(cfg.Method); err != nil {
		return nil, err
	}

	var body io.Reader
	var bodyFormat string
	switch cfg.BodyFormat {
	case "multipart":
		body, bodyFormat = buildMultipartBody(cfg)
	case "urlencoded":
		body, bodyFormat = buildURLEncodedBody(cfg)
	default:
		if cfg.Data != "" {
			body = bytes.NewBufferString(cfg.Data)
		}
	}

	req, err := http.NewRequest(cfg.Method, cfg.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range cfg.Headers {
		req.Header.Set(key, value)
	}

	// Auto-set Content-Type for JSON data if not already set
	if cfg.Data != "" && req.Header.Get("Content-Type") == "" {
		if IsJSON(cfg.Data) {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Form bodies require an exact Content-Type (multipart includes the boundary)
	if bodyFormat != "" {
		req.Header.Set("Content-Type", bodyFormat)
	}

	return req, nil
}

// ValidateURL performs security and format validation on the URL
func ValidateURL(targetURL string) error {
	if targetURL == "" {
		return errors.New("URL cannot be empty")
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q, only http and https are allowed", u.Scheme)
	}

	if u.Host == "" {
		return errors.New("URL must include a host")
	}

	// Warn about localhost connections
	lowerHost := strings.ToLower(u.Hostname())
	blockedHosts := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1"}
	for _, blocked := range blockedHosts {
		if lowerHost == blocked {
			fmt.Fprintf(os.Stderr, "Warning: Connecting to local address %q\n", lowerHost)
			break
		}
	}

	return nil
}

// ValidateMethod checks if the HTTP method is valid
func ValidateMethod(method string) error {
	if !ValidMethods[method] {
		return fmt.Errorf("invalid HTTP method %q", method)
	}
	return nil
}

// IsJSON attempts to detect if a string is JSON
func IsJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

// buildMultipartBody encodes form fields as multipart/form-data and returns
// the body along with the Content-Type header value (including boundary).
func buildMultipartBody(cfg *config.Config) (io.Reader, string) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	for _, f := range cfg.FormFields {
		if f.Key == "" {
			continue
		}
		if f.IsFile {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", fmt.Sprintf(
				`form-data; name="%s"; filename="%s"`,
				escapeQuotes(f.Key), escapeQuotes(filepath.Base(f.FileName))))
			contentType := f.ContentType
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			h.Set("Content-Type", contentType)
			part, err := writer.CreatePart(h)
			if err != nil {
				continue
			}
			part.Write(f.FileContent)
		} else {
			writer.WriteField(f.Key, f.Value)
		}
	}
	writer.Close()
	return &buf, writer.FormDataContentType()
}

// buildURLEncodedBody encodes form fields as application/x-www-form-urlencoded.
func buildURLEncodedBody(cfg *config.Config) (io.Reader, string) {
	form := url.Values{}
	for _, f := range cfg.FormFields {
		if f.Key != "" && !f.IsFile {
			form.Add(f.Key, f.Value)
		}
	}
	return strings.NewReader(form.Encode()), "application/x-www-form-urlencoded"
}

// escapeQuotes escapes quotes and newlines in multipart header values.
func escapeQuotes(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r\n", " ")
	return s
}