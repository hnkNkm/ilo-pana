// Package config handles command-line flag parsing and configuration
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// BodyFormat identifies how the request body is encoded.
type BodyFormat string

// Supported body formats. The zero value ("") behaves like raw.
const (
	BodyFormatRaw        BodyFormat = "raw"
	BodyFormatMultipart  BodyFormat = "multipart"
	BodyFormatURLEncoded BodyFormat = "urlencoded"
)

// Valid reports whether f is a known body format. The empty string is treated
// as raw and therefore valid.
func (f BodyFormat) Valid() bool {
	switch f {
	case "", BodyFormatRaw, BodyFormatMultipart, BodyFormatURLEncoded:
		return true
	}
	return false
}

// Config holds all configuration for an HTTP request
type Config struct {
	Method         string
	URL            string
	Headers        map[string]string
	Data           string
	BodyFormat     BodyFormat  // "", "raw", "multipart", "urlencoded"
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

// ErrUsage marks errors caused by invalid command-line arguments. The flag
// package has already printed the error and usage text to the stderr writer
// passed to ParseWith, so callers typically exit without further output.
var ErrUsage = errors.New("invalid command-line arguments")

// cliFlags groups the flag values of a single ParseWith call. A fresh set is
// built per call so the parser is re-entrant and safe for parallel use.
type cliFlags struct {
	fs             *flag.FlagSet
	method         *string
	headers        headerList
	vars           varList
	data           *string
	timeout        *time.Duration
	verbose        *bool
	envFile        *string
	sessionName    *string
	sessionNew     *bool
	sessionHeaders *bool
	failStatus     *bool
}

// ParseWith parses command-line flags from args and returns a Config.
// Diagnostics (usage, warnings, flag errors) are written to stderr.
// The function is re-entrant: it can be called multiple times from the same
// process without interfering with the global flag.CommandLine.
func ParseWith(args []string, stderr io.Writer) (*Config, error) {
	flags := newFlags(stderr)
	if err := flags.fs.Parse(args); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUsage, err)
	}

	rest := flags.fs.Args()
	if len(rest) < 1 {
		return nil, errors.New("URL is required")
	}

	targetURL := rest[0]

	// Load environment variables
	envLoader := env.NewLoader()

	// Load from env file (either specified or default .env)
	envFilePath := *flags.envFile
	if envFilePath == "" {
		// Try to load default .env if it exists; a broken .env is a warning,
		// not a hard error, since the file is optional.
		if err := envLoader.LoadDefault(); err != nil {
			fmt.Fprintf(stderr, "Warning: failed to load .env: %v\n", err)
		}
	} else {
		if err := envLoader.Load(envFilePath); err != nil {
			return nil, fmt.Errorf("failed to load env file: %w", err)
		}
	}

	// Parse command-line variables
	cliVars, err := variables.ParseVariables(flags.vars)
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
		fmt.Fprintf(stderr, "Warning: %s\n", warning)
	}

	// Parse and expand headers
	parsedHeaders, err := parseHeaders(flags.headers, stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse headers: %w", err)
	}
	expandedHeaders := expander.ExpandHeaders(parsedHeaders)

	// Expand variables in data
	expandedData := expander.Expand(*flags.data)

	upperMethod := strings.ToUpper(*flags.method)

	return &Config{
		Method:         upperMethod,
		URL:            expandedURL,
		Headers:        expandedHeaders,
		Data:           expandedData,
		Timeout:        *flags.timeout,
		Verbose:        *flags.verbose,
		Variables:      allVars,
		EnvFile:        envFilePath,
		SessionName:    *flags.sessionName,
		SessionNew:     *flags.sessionNew,
		SessionHeaders: *flags.sessionHeaders,
		FailStatus:     *flags.failStatus,
	}, nil
}

// Usage writes the CLI usage message to w.
func Usage(w io.Writer) {
	newFlags(w).fs.Usage()
}

// newFlags builds a FlagSet with all CLI flags registered. It uses a local
// FlagSet so the parser has no dependency on the process-global
// flag.CommandLine and can run repeatedly in one process.
func newFlags(output io.Writer) *cliFlags {
	fs := flag.NewFlagSet("ilo-pana", flag.ContinueOnError)
	fs.SetOutput(output)
	f := &cliFlags{
		fs:          fs,
		method:      fs.String("X", "GET", "HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)"),
		data:        fs.String("d", "", "Request body data"),
		timeout:     fs.Duration("timeout", 30*time.Second, "Request timeout"),
		verbose:     fs.Bool("v", false, "Verbose output (show all headers without masking)"),
		envFile:     fs.String("env-file", "", "Path to environment file (default: ./.env if exists)"),
		sessionName: fs.String("session", "", "Session name to use for cookies and headers"),
		sessionNew:  fs.Bool("session-new", false, "Create new session (overwrites existing)"),
		sessionHeaders: fs.Bool("session-save-headers", false, "Save custom headers to session"),
		failStatus:  fs.Bool("fail", false, "Exit with a non-zero status code if the HTTP response is 4xx or 5xx (like curl --fail)"),
	}
	fs.Var(&f.headers, "H", "HTTP headers (format: 'Key: Value'), can be specified multiple times")
	fs.Var(&f.vars, "var", "Variables (format: 'key=value'), can be specified multiple times")
	fs.Usage = func() {
		fmt.Fprintf(output, "Usage: %s [OPTIONS] URL\n\n", fs.Name())
		fmt.Fprintf(output, "A simple and secure API testing tool\n\n")
		fmt.Fprintf(output, "Options:\n")
		fs.PrintDefaults()
		fmt.Fprintf(output, "\nExamples:\n")
		fmt.Fprintf(output, "  # Simple GET request\n")
		fmt.Fprintf(output, "  ilo-pana https://api.example.com/users\n\n")
		fmt.Fprintf(output, "  # POST request with JSON data\n")
		fmt.Fprintf(output, "  ilo-pana -X POST -d '{\"name\":\"test\"}' https://api.example.com/users\n\n")
		fmt.Fprintf(output, "  # Multiple headers\n")
		fmt.Fprintf(output, "  ilo-pana -H 'Authorization: Bearer token' -H 'Accept: application/json' https://api.example.com/protected\n\n")
		fmt.Fprintf(output, "  # With variables\n")
		fmt.Fprintf(output, "  ilo-pana -var BASE_URL=https://api.example.com -var TOKEN=abc123 '{{BASE_URL}}/users'\n\n")
		fmt.Fprintf(output, "  # Using environment file\n")
		fmt.Fprintf(output, "  ilo-pana --env-file .env.dev '{{BASE_URL}}/api/{{VERSION}}/users'\n\n")
		fmt.Fprintf(output, "  # Using sessions\n")
		fmt.Fprintf(output, "  ilo-pana --session dev --session-new https://api.example.com/login\n\n")
		fmt.Fprintf(output, "  # Custom timeout\n")
		fmt.Fprintf(output, "  ilo-pana -timeout 10s https://slow-api.example.com/endpoint\n")
		fmt.Fprintf(output, "  # Fail on HTTP error status\n")
		fmt.Fprintf(output, "  ilo-pana --fail https://api.example.com/endpoint\n")
	}
	return f
}

// parseHeaders converts a slice of header strings into a map
func parseHeaders(headerStrings []string, stderr io.Writer) (map[string]string, error) {
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
			fmt.Fprintf(stderr, "Warning: Duplicate header %q, using latest value\n", key)
		}

		headers[key] = value
	}

	return headers, nil
}