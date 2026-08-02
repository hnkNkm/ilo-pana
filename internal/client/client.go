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

// executionResult carries the raw response and any non-fatal issues from the
// shared execution path.
type executionResult struct {
	req            *http.Request
	resp           *http.Response
	elapsed        time.Duration
	sessionSaveErr error
}

// prepareRequest builds the HTTP request and applies session headers.
func (c *Client) prepareRequest() (*http.Request, error) {
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
	return req, nil
}

// doRequest sends the request and processes the session response. A session
// save failure is reported on the result, never fatal.
func (c *Client) doRequest(req *http.Request) (*executionResult, error) {
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	elapsed := time.Since(startTime)

	var sessionSaveErr error
	if c.session != nil {
		c.session.ProcessResponse(resp)
		sessionSaveErr = c.session.Save()
	}

	return &executionResult{req: req, resp: resp, elapsed: elapsed, sessionSaveErr: sessionSaveErr}, nil
}

// Execute performs the HTTP request and handles the response
func (c *Client) Execute() error {
	req, err := c.prepareRequest()
	if err != nil {
		return err
	}

	// Print request details
	response.PrintRequest(c.config.Method, c.config.URL, req.Header, c.config.Data, c.config.Verbose)

	result, err := c.doRequest(req)
	if err != nil {
		return err
	}
	defer result.resp.Body.Close()

	if result.sessionSaveErr != nil {
		// Log warning but don't fail the request
		fmt.Printf("Warning: failed to save session: %v\n", result.sessionSaveErr)
	}

	// Process and print the response
	handler := response.New(c.config.Verbose)
	if err := handler.Process(result.resp, result.elapsed); err != nil {
		return fmt.Errorf("failed to process response: %w", err)
	}

	// Fail on HTTP error status codes when --fail is set
	if c.config.FailStatus && result.resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with HTTP status %d", result.resp.StatusCode)
	}

	return nil
}

// ExecuteForGUI performs the HTTP request and returns structured response data for GUI
func (c *Client) ExecuteForGUI() (*response.ResponseData, error) {
	req, err := c.prepareRequest()
	if err != nil {
		return nil, err
	}

	result, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer result.resp.Body.Close()

	handler := response.New(c.config.Verbose)
	data, err := handler.ProcessToStruct(result.resp, result.elapsed)
	if err != nil {
		return nil, err
	}
	if result.sessionSaveErr != nil {
		data.Warning = fmt.Sprintf("failed to save session: %v", result.sessionSaveErr)
	}
	return data, nil
}
