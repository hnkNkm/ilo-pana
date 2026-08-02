// Package config handles command-line flag parsing and configuration
package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
	
	"ilo-pana/internal/env"
	"ilo-pana/internal/variables"
)

// FormField is a single entry of a form body (multipart or urlencoded).
type FormField struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsFile      bool   `json:"isFile"`
	FileName    string `json:"fileName"`
	FileContent []byte `json:"fileContent"`
	ContentType string `json:"contentType"`
}

// Config holds all configuration for an HTTP request
type Config struct {
	Method         string
	URL            string
	Headers        map[string]string
	Data           string
	BodyFormat     string      // "", "raw", "multipart", "urlencoded"
	FormFields     []FormField // used when BodyFormat is multipart or urlencoded
	Timeout        time.Duration
	Verbose        bool
	Variables      map[string]string
	EnvFile        string
	SessionName    string
	SessionNew     bool
	SessionHeaders bool
	FailStatus     bool
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

// varList implements flag.Value to accumulate multiple -var flag values
type varList []string

// String returns the string representation of the variable list
func (v *varList) String() string {
	return strings.Join(*v, ", ")
}

// Set appends a new variable to the list
func (v *varList) Set(value string) error {
	if value == "" {
		return errors.New("variable value cannot be empty")
	}
	*v = append(*v, value)
	return nil
}

// Parse parses command-line flags and returns a Config
func Parse() (*Config, error) {
	var (
		method         = flag.String("X", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)")
		headers        headerList
		vars           varList
		data           = flag.String("d", "", "Request body data")
		timeout        = flag.Duration("timeout", 30*time.Second, "Request timeout")
		verbose        = flag.Bool("v", false, "Verbose output (show all headers without masking)")
		envFile        = flag.String("env-file", "", "Path to environment file (default: ./.env if exists)")
		sessionName    = flag.String("session", "", "Session name to use for cookies and headers")
		sessionNew     = flag.Bool("session-new", false, "Create new session (overwrites existing)")
		sessionHeaders = flag.Bool("session-save-headers", false, "Save custom headers to session")
		failStatus     = flag.Bool("fail", false, "Exit with a non-zero status code if the HTTP response is 4xx or 5xx (like curl --fail)")
	)

	flag.Var(&headers, "H", "HTTP headers (format: 'Key: Value'), can be specified multiple times")
	flag.Var(&vars, "var", "Variables (format: 'key=value'), can be specified multiple times")
	
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
		fmt.Fprintf(os.Stderr, "  # With variables\n")
		fmt.Fprintf(os.Stderr, "  %s -var BASE_URL=https://api.example.com -var TOKEN=abc123 '{{BASE_URL}}/users'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Using environment file\n")
		fmt.Fprintf(os.Stderr, "  %s --env-file .env.dev '{{BASE_URL}}/api/{{VERSION}}/users'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Using sessions\n")
		fmt.Fprintf(os.Stderr, "  %s --session dev --session-new https://api.example.com/login\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Custom timeout\n")
		fmt.Fprintf(os.Stderr, "  %s -timeout 10s https://slow-api.example.com/endpoint\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Fail on HTTP error status\n")
		fmt.Fprintf(os.Stderr, "  %s --fail https://api.example.com/endpoint\n", os.Args[0])
	}

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return nil, errors.New("URL is required")
	}

	targetURL := args[0]

	// Load environment variables
	envLoader := env.NewLoader()
	
	// Load from env file (either specified or default .env)
	envFilePath := *envFile
	if envFilePath == "" {
		// Try to load default .env if it exists
		envLoader.LoadDefault()
	} else {
		if err := envLoader.Load(envFilePath); err != nil {
			return nil, fmt.Errorf("failed to load env file: %w", err)
		}
	}
	
	// Parse command-line variables
	cliVars, err := variables.ParseVariables(vars)
	if err != nil {
		return nil, fmt.Errorf("failed to parse variables: %w", err)
	}
	
	// Combine variables (CLI overrides env file)
	allVars := envLoader.GetVariables()
	for k, v := range cliVars {
		allVars[k] = v
	}

	// Create variable expander
	expander := variables.New()
	expander.SetAll(allVars)
	
	// Expand variables in URL
	expandedURL, warnings := expander.ExpandWithWarnings(targetURL)
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	// Parse and expand headers
	parsedHeaders, err := parseHeaders(headers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}
	expandedHeaders := expander.ExpandHeaders(parsedHeaders)
	
	// Expand variables in data
	expandedData := expander.Expand(*data)

	upperMethod := strings.ToUpper(*method)

	return &Config{
		Method:         upperMethod,
		URL:            expandedURL,
		Headers:        expandedHeaders,
		Data:           expandedData,
		Timeout:        *timeout,
		Verbose:        *verbose,
		Variables:      allVars,
		EnvFile:        envFilePath,
		SessionName:    *sessionName,
		SessionNew:     *sessionNew,
		SessionHeaders: *sessionHeaders,
		FailStatus:     *failStatus,
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