// Package client handles HTTP client creation and request execution
package client

import (
	"fmt"
	"net/http"
	"time"

	"ilo-pana/internal/config"
	"ilo-pana/internal/request"
	"ilo-pana/internal/response"
)

// Client wraps HTTP client functionality
type Client struct {
	httpClient *http.Client
	config     *config.Config
}

// New creates a new HTTP client with the specified configuration
func New(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}
}

// Execute performs the HTTP request and handles the response
func (c *Client) Execute() error {
	// Build the request
	req, err := request.Build(c.config)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}

	// Print request details
	response.PrintRequest(c.config.Method, c.config.URL, req.Header, c.config.Data, c.config.Verbose)

	// Execute the request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	// Process and print the response
	handler := response.New(c.config.Verbose)
	if err := handler.Process(resp, elapsed); err != nil {
		return fmt.Errorf("failed to process response: %w", err)
	}

	return nil
}
