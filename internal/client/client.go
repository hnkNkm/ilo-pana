// Package client handles HTTP client creation and request execution
package client

import (
	"fmt"
	"net/http"
	"time"

	"ilo-pana/internal/config"
	"ilo-pana/internal/request"
	"ilo-pana/internal/response"
	"ilo-pana/internal/session"
)

// Client wraps HTTP client functionality
type Client struct {
	httpClient *http.Client
	config     *config.Config
	session    *session.Manager
}

// New creates a new HTTP client with the specified configuration
func New(cfg *config.Config) (*Client, error) {
	client := &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
	}
	
	// Initialize session if specified
	if cfg.SessionName != "" {
		opts := &session.Options{
			CreateNew:   cfg.SessionNew,
			SaveHeaders: cfg.SessionHeaders,
		}
		
		mgr, err := session.NewManager(cfg.SessionName, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize session: %w", err)
		}
		
		client.session = mgr
		// Set cookie jar in HTTP client
		client.httpClient.Jar = mgr.GetCookieJar()
	}
	
	return client, nil
}

// Execute performs the HTTP request and handles the response
func (c *Client) Execute() error {
	// Build the request
	req, err := request.Build(c.config)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	
	// Apply session headers if available
	if c.session != nil {
		c.session.ApplyToRequest(req)
		
		// Save custom headers to session if requested
		if c.config.SessionHeaders {
			for key, value := range c.config.Headers {
				c.session.SetHeader(key, value)
			}
		}
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
	
	// Process session response and save
	if c.session != nil {
		c.session.ProcessResponse(resp)
		if err := c.session.Save(); err != nil {
			// Log warning but don't fail the request
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}

	// Process and print the response
	handler := response.New(c.config.Verbose)
	if err := handler.Process(resp, elapsed); err != nil {
		return fmt.Errorf("failed to process response: %w", err)
	}

	// Fail on HTTP error status codes when --fail is set
	if c.config.FailStatus && resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with HTTP status %d", resp.StatusCode)
	}

	return nil
}

// ExecuteForGUI performs the HTTP request and returns structured response data for GUI
func (c *Client) ExecuteForGUI() (*response.ResponseData, error) {
	req, err := request.Build(c.config)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	if c.session != nil {
		c.session.ApplyToRequest(req)
		if c.config.SessionHeaders {
			for key, value := range c.config.Headers {
				c.session.SetHeader(key, value)
			}
		}
	}

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	elapsed := time.Since(startTime)

	if c.session != nil {
		c.session.ProcessResponse(resp)
		if err := c.session.Save(); err != nil {
			// Log warning but don't fail the request
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}
	}

	handler := response.New(c.config.Verbose)
	return handler.ProcessToStruct(resp, elapsed)
}
