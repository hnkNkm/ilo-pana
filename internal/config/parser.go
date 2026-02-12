package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// HeaderParser safely parses header strings
type HeaderParser struct {
	sensitiveKeys map[string]bool
	maxHeaders    int
}

// NewHeaderParser creates a parser with security settings
func NewHeaderParser() *HeaderParser {
	return &HeaderParser{
		sensitiveKeys: map[string]bool{
			"authorization": true,
			"x-api-key":     true,
			"x-auth-token":  true,
			"cookie":        true,
			"set-cookie":    true,
		},
		maxHeaders: 100, // Prevent header bombing
	}
}

// Parse processes multiple header strings safely
func (p *HeaderParser) Parse(headers []string) (map[string]string, error) {
	if len(headers) > p.maxHeaders {
		return nil, fmt.Errorf("too many headers: maximum %d allowed", p.maxHeaders)
	}

	result := make(map[string]string, len(headers))
	headerRegex := regexp.MustCompile(`^([a-zA-Z0-9\-_]+):\s*(.+)$`)

	for _, header := range headers {
		if header == "" {
			continue
		}

		matches := headerRegex.FindStringSubmatch(header)
		if len(matches) != 3 {
			return nil, fmt.Errorf("invalid header format: %q (expected 'Key: Value')", header)
		}

		key := matches[1]
		value := matches[2]

		// Validate header name
		if err := p.validateHeaderName(key); err != nil {
			return nil, err
		}

		// Validate header value
		if err := p.validateHeaderValue(value); err != nil {
			return nil, fmt.Errorf("invalid value for header %q: %w", key, err)
		}

		// Check for duplicate headers
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("duplicate header: %q", key)
		}

		result[key] = value
	}

	return result, nil
}

// validateHeaderName checks header name validity
func (p *HeaderParser) validateHeaderName(name string) error {
	if len(name) == 0 || len(name) > 1024 {
		return fmt.Errorf("header name length must be between 1 and 1024 characters")
	}

	// Check for invalid characters
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_') {
			return fmt.Errorf("header name contains invalid character: %q", r)
		}
	}

	return nil
}

// validateHeaderValue checks header value validity
func (p *HeaderParser) validateHeaderValue(value string) error {
	if len(value) > 8192 {
		return fmt.Errorf("header value too long (max 8192 characters)")
	}

	// Check for control characters
	for _, r := range value {
		if r < 32 && r != '\t' {
			return fmt.Errorf("header value contains control character")
		}
	}

	return nil
}

// MaskSensitive returns headers with sensitive values masked
func (p *HeaderParser) MaskSensitive(headers map[string]string) map[string]string {
	masked := make(map[string]string, len(headers))

	for key, value := range headers {
		lowerKey := strings.ToLower(key)
		if p.sensitiveKeys[lowerKey] {
			// Mask all but first and last 4 chars if long enough
			if len(value) > 8 {
				masked[key] = value[:4] + "****" + value[len(value)-4:]
			} else {
				masked[key] = "****"
			}
		} else {
			masked[key] = value
		}
	}

	return masked
}

// URLValidator validates and sanitizes URLs
type URLValidator struct {
	allowedSchemes  map[string]bool
	blockedHosts    map[string]bool
	allowPrivateIPs bool
}

// NewURLValidator creates a secure URL validator
func NewURLValidator() *URLValidator {
	return &URLValidator{
		allowedSchemes: map[string]bool{
			"http":  true,
			"https": true,
		},
		blockedHosts: map[string]bool{
			"localhost": true,
			"127.0.0.1": true,
			"0.0.0.0":   true,
			"::1":       true,
		},
		allowPrivateIPs: false,
	}
}

// Validate checks URL for security issues
func (v *URLValidator) Validate(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if !v.allowedSchemes[parsedURL.Scheme] {
		return fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Check for SSRF attacks
	host := strings.ToLower(parsedURL.Hostname())
	if v.blockedHosts[host] && !v.allowPrivateIPs {
		return fmt.Errorf("access to host %q is blocked", host)
	}

	// Check for private IP ranges
	if !v.allowPrivateIPs && isPrivateIP(host) {
		return fmt.Errorf("access to private IP addresses is blocked")
	}

	// Check for suspicious patterns
	if strings.Contains(rawURL, "..") || strings.Contains(rawURL, "//") {
		return fmt.Errorf("URL contains suspicious patterns")
	}

	return nil
}

// isPrivateIP checks if an IP is in private range
func isPrivateIP(host string) bool {
	privateRanges := []string{
		"10.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
		"192.168.",
		"169.254.", // Link-local
		"fc00::",   // IPv6 private
		"fe80::",   // IPv6 link-local
	}

	for _, prefix := range privateRanges {
		if strings.HasPrefix(host, prefix) {
			return true
		}
	}

	return false
}
