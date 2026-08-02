package service

import (
	"fmt"
	"strings"
	"time"

	"ilo-pana/internal/client"
	"ilo-pana/internal/config"
	"ilo-pana/internal/request"
	"ilo-pana/internal/response"
	"ilo-pana/internal/variables"
)

// RequestParams holds parameters for executing an HTTP request.
type RequestParams struct {
	Method      string
	URL         string
	Body        string
	BodyFormat  config.BodyFormat // "", "raw", "multipart", "urlencoded"
	FormFields  []config.FormField
	Headers     map[string]string
	TimeoutMs   int // Timeout in milliseconds, default 30000 (30 seconds)
	SessionName string
	SessionNew  bool
	Variables   map[string]string
	Environment string // Name of the selected environment, "" for none
}

// RequestExecutor executes HTTP requests with environment and
// request-level variable expansion.
type RequestExecutor struct {
	environments *EnvironmentService
}

// NewRequestExecutor creates an executor backed by the given
// environment service.
func NewRequestExecutor(environments *EnvironmentService) *RequestExecutor {
	return &RequestExecutor{environments: environments}
}

// Execute executes the request described by params and returns structured
// response data for the GUI.
func (e *RequestExecutor) Execute(params RequestParams) (*response.ResponseData, error) {
	method := strings.ToUpper(params.Method)

	// Merge environment variables with request-level variables
	// (request-level variables take precedence)
	vars := make(map[string]string)
	if params.Environment != "" {
		env, err := e.environments.Load(params.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to load environment %q: %w", params.Environment, err)
		}
		for k, v := range env.Variables {
			vars[k] = v
		}
	}
	for k, v := range params.Variables {
		vars[k] = v
	}

	// Expand {{VAR}} variables in URL, headers, and body before validation
	expandedURL := params.URL
	expandedHeaders := params.Headers
	expandedBody := params.Body
	expandedFields := params.FormFields
	if len(vars) > 0 {
		expander := variables.New()
		expander.SetAll(vars)
		expandedURL = expander.Expand(params.URL)
		expandedHeaders = expander.ExpandHeaders(params.Headers)
		expandedBody = expander.Expand(params.Body)
		for i := range params.FormFields {
			params.FormFields[i].Value = expander.Expand(params.FormFields[i].Value)
			params.FormFields[i].FileName = expander.Expand(params.FormFields[i].FileName)
		}
		expandedFields = params.FormFields
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
		BodyFormat:     params.BodyFormat,
		FormFields:     expandedFields,
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
