package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ilo-pana/internal/client"
	"ilo-pana/internal/config"
	"ilo-pana/internal/request"
	"ilo-pana/internal/response"
	"ilo-pana/internal/variables"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// RequestParams holds parameters for ExecuteRequest from the frontend
type RequestParams struct {
	Method      string
	URL         string
	Body        string
	Headers     map[string]string
	TimeoutMs   int // Timeout in milliseconds, default 30000 (30 seconds)
	SessionName string
	SessionNew  bool
	Variables   map[string]string
}

// ExecuteRequest executes an HTTP request and returns structured response data for GUI
func (a *App) ExecuteRequest(params RequestParams) (*response.ResponseData, error) {
	method := strings.ToUpper(params.Method)

	// Expand {{VAR}} variables in URL, headers, and body before validation
	expandedURL := params.URL
	expandedHeaders := params.Headers
	expandedBody := params.Body
	if len(params.Variables) > 0 {
		expander := variables.New()
		expander.SetAll(params.Variables)
		expandedURL = expander.Expand(params.URL)
		expandedHeaders = expander.ExpandHeaders(params.Headers)
		expandedBody = expander.Expand(params.Body)
	}

	// Validate
	if err := request.ValidateURL(expandedURL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if err := request.ValidateMethod(method); err != nil {
		return nil, err
	}

	// Default timeout to 30 seconds if not specified
	timeout := 30 * time.Second
	if params.TimeoutMs > 0 {
		// Max timeout is 5 minutes
		if params.TimeoutMs > 300000 {
			params.TimeoutMs = 300000
		}
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}

	cfg := &config.Config{
		Method:         method,
		URL:            expandedURL,
		Data:           expandedBody,
		Headers:        expandedHeaders,
		Timeout:        timeout,
		Verbose:        false,
		SessionName:    params.SessionName,
		SessionNew:     params.SessionNew,
		SessionHeaders: false,
		Variables:      params.Variables,
	}
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}

	c, err := client.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return c.ExecuteForGUI()
}

// ExecuteRequestSimple is a simpler signature for Wails binding (method, url, body, headers)
func (a *App) ExecuteRequestSimple(method, url, body string, headers map[string]string) (*response.ResponseData, error) {
	return a.ExecuteRequest(RequestParams{
		Method:    method,
		URL:       url,
		Body:      body,
		Headers:   headers,
		TimeoutMs: 30000, // Default 30 seconds
	})
}

// ExecuteRequestWithTimeout allows frontend to specify a custom timeout
func (a *App) ExecuteRequestWithTimeout(method, url, body string, headers map[string]string, timeoutMs int) (*response.ResponseData, error) {
	return a.ExecuteRequest(RequestParams{
		Method:    method,
		URL:       url,
		Body:      body,
		Headers:   headers,
		TimeoutMs: timeoutMs,
	})
}
