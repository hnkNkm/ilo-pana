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
	Method  string
	URL     string
	Body    string
	Headers map[string]string
}

// ExecuteRequest executes an HTTP request and returns structured response data for GUI
func (a *App) ExecuteRequest(params RequestParams) (*response.ResponseData, error) {
	// Validate
	if err := request.ValidateURL(params.URL); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	method := strings.ToUpper(params.Method)
	if err := request.ValidateMethod(method); err != nil {
		return nil, err
	}

	cfg := &config.Config{
		Method:  method,
		URL:     params.URL,
		Data:    params.Body,
		Headers: params.Headers,
		Timeout: 30 * time.Second,
		Verbose: false,
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
		Method:  method,
		URL:     url,
		Body:    body,
		Headers: headers,
	})
}
