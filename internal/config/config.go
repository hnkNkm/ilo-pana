// Package config handles command-line flag parsing and configuration
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all configuration for an HTTP request
type Config struct {
	Method  string
	URL     string
	Headers map[string]string
	Data    string
	Timeout time.Duration
	Verbose bool
}

// headerList implements flag.Value to accumulate multiple -H flag values
type headerList []string

// String returns the string representation of the header list
func (h *headerList) String() string {
	return strings.Join(*h, ", ")
}

// Set appends a new header to the list
func (h *headerList) Set(value string) error {
	if value == "" {
		return errors.New("header value cannot be empty")
	}
	*h = append(*h, value)
	return nil
}

// Parse parses command-line flags and returns a Config
func Parse() (*Config, error) {
	var (
		method  = flag.String("X", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)")
		headers headerList
		data    = flag.String("d", "", "Request body data")
		timeout = flag.Duration("timeout", 30*time.Second, "Request timeout")
		verbose = flag.Bool("v", false, "Verbose output (show all headers without masking)")
	)

	flag.Var(&headers, "H", "HTTP headers (format: 'Key: Value'), can be specified multiple times")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] URL\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A simple and secure API testing tool\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Simple GET request\n")
		fmt.Fprintf(os.Stderr, "  %s https://api.example.com/users\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # POST request with JSON data\n")
		fmt.Fprintf(os.Stderr, "  %s -X POST -d '{\"name\":\"test\"}' https://api.example.com/users\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Multiple headers\n")
		fmt.Fprintf(os.Stderr, "  %s -H 'Authorization: Bearer token' -H 'Accept: application/json' https://api.example.com/protected\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Custom timeout\n")
		fmt.Fprintf(os.Stderr, "  %s -timeout 10s https://slow-api.example.com/endpoint\n", os.Args[0])
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return nil, errors.New("URL is required")
	}

	targetURL := args[0]

	parsedHeaders, err := parseHeaders(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}

	upperMethod := strings.ToUpper(*method)

	return &Config{
		Method:  upperMethod,
		URL:     targetURL,
		Headers: parsedHeaders,
		Data:    *data,
		Timeout: *timeout,
		Verbose: *verbose,
	}, nil
}

// parseHeaders converts a slice of header strings into a map
func parseHeaders(headerStrings []string) (map[string]string, error) {
	headers := make(map[string]string)

	for _, headerString := range headerStrings {
		colonIndex := strings.Index(headerString, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("invalid header format %q, expected 'Key: Value'", headerString)
		}

		key := strings.TrimSpace(headerString[:colonIndex])
		value := strings.TrimSpace(headerString[colonIndex+1:])

		if key == "" {
			return nil, fmt.Errorf("empty header key in %q", headerString)
		}

		if _, exists := headers[key]; exists {
			fmt.Fprintf(os.Stderr, "Warning: Duplicate header %q, using latest value\n", key)
		}

		headers[key] = value
	}

	return headers, nil
}