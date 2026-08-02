package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ilo-pana/internal/assertion"
	"ilo-pana/internal/client"
	"ilo-pana/internal/collection"
	"ilo-pana/internal/config"
	"ilo-pana/internal/curl"
	"ilo-pana/internal/environment"
	"ilo-pana/internal/openapi"
	"ilo-pana/internal/request"
	"ilo-pana/internal/response"
	"ilo-pana/internal/variables"
)

// App struct
type App struct {
	ctx          context.Context
	collections  collection.Storage
	environments environment.Storage
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		collections:  collection.NewFileStorage(""),
		environments: environment.NewFileStorage(""),
	}
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
	Environment string // Name of the selected environment, "" for none
}

// ExecuteRequest executes an HTTP request and returns structured response data for GUI
func (a *App) ExecuteRequest(params RequestParams) (*response.ResponseData, error) {
	method := strings.ToUpper(params.Method)

	// Merge environment variables with request-level variables
	// (request-level variables take precedence)
	vars := make(map[string]string)
	if params.Environment != "" {
		env, err := a.environments.Load(params.Environment)
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
	if len(vars) > 0 {
		expander := variables.New()
		expander.SetAll(vars)
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

// SaveRequest upserts a request into the named collection, creating it if needed.
func (a *App) SaveRequest(collectionName string, req collection.SavedRequest) error {
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if req.Name == "" {
		return fmt.Errorf("request name is required")
	}

	c, err := a.collections.Load(collectionName)
	if err != nil {
		if !collection.IsNotExist(err) {
			return err
		}
		c = &collection.Collection{
			Name:    collectionName,
			Created: time.Now(),
		}
	}
	now := time.Now()
	if c.Created.IsZero() {
		c.Created = now
	}
	req.Updated = now
	c.Updated = now
	c.Upsert(req)
	return a.collections.Save(c)
}

// ListCollections returns the names of all saved collections.
func (a *App) ListCollections() ([]string, error) {
	return a.collections.List()
}

// GetCollection returns a collection by name.
func (a *App) GetCollection(name string) (*collection.Collection, error) {
	return a.collections.Load(name)
}

// DeleteRequest removes a request from a collection.
func (a *App) DeleteRequest(collectionName, requestName string) error {
	c, err := a.collections.Load(collectionName)
	if err != nil {
		return err
	}
	if !c.Remove(requestName) {
		return fmt.Errorf("request %q not found in collection %q", requestName, collectionName)
	}
	c.Updated = time.Now()
	return a.collections.Save(c)
}

// DeleteCollection removes a collection entirely.
func (a *App) DeleteCollection(name string) error {
	return a.collections.Delete(name)
}

// ExportCollection returns a collection as a JSON string (for copy/paste sharing).
func (a *App) ExportCollection(name string) (string, error) {
	c, err := a.collections.Load(name)
	if err != nil {
		return "", err
	}
	return c.MarshalJSON()
}

// ImportCollection parses a JSON collection and saves it, replacing any existing one.
func (a *App) ImportCollection(json string) error {
	c, err := collection.UnmarshalJSON(json)
	if err != nil {
		return err
	}
	if c.Name == "" {
		return fmt.Errorf("collection JSON must include a name")
	}
	return a.collections.Save(c)
}

// SaveEnvironment upserts a named environment with the given variables.
func (a *App) SaveEnvironment(name string, vars map[string]string) error {
	if name == "" {
		return fmt.Errorf("environment name is required")
	}

	e, err := a.environments.Load(name)
	if err != nil {
		if !environment.IsNotExist(err) {
			return err
		}
		e = &environment.Environment{
			Name:    name,
			Created: time.Now(),
		}
	}
	now := time.Now()
	if e.Created.IsZero() {
		e.Created = now
	}
	if vars == nil {
		vars = make(map[string]string)
	}
	e.Variables = vars
	e.Updated = now
	return a.environments.Save(e)
}

// ListEnvironments returns the names of all saved environments.
func (a *App) ListEnvironments() ([]string, error) {
	return a.environments.List()
}

// GetEnvironment returns an environment by name.
func (a *App) GetEnvironment(name string) (*environment.Environment, error) {
	return a.environments.Load(name)
}

// DeleteEnvironment removes an environment entirely.
func (a *App) DeleteEnvironment(name string) error {
	return a.environments.Delete(name)
}

// ImportOpenAPI parses an OpenAPI (YAML/JSON) spec and imports its
// operations into the named collection (default: the spec's title),
// returning the number of imported endpoints.
func (a *App) ImportOpenAPI(content, collectionName string) (int, error) {
	doc, err := openapi.Parse([]byte(content))
	if err != nil {
		return 0, err
	}
	if collectionName == "" {
		collectionName = doc.Title
	}
	if collectionName == "" {
		return 0, fmt.Errorf("collection name is required")
	}

	c, err := a.collections.Load(collectionName)
	if err != nil {
		if !collection.IsNotExist(err) {
			return 0, err
		}
		c = &collection.Collection{
			Name:    collectionName,
			Created: time.Now(),
		}
	}
	if c.Created.IsZero() {
		c.Created = time.Now()
	}
	now := time.Now()
	for _, ep := range doc.Endpoints {
		c.Upsert(collection.SavedRequest{
			Name:      ep.Name,
			Method:    ep.Method,
			URL:       ep.URL,
			Headers:   ep.Headers,
			Body:      ep.Body,
			Variables: ep.Variables,
			Created:   now,
			Updated:   now,
		})
	}
	c.Updated = now
	if err := a.collections.Save(c); err != nil {
		return 0, err
	}
	return len(doc.Endpoints), nil
}

// EvaluateAssertions runs assertion rules against a response and returns
// the pass/fail results in rule order.
func (a *App) EvaluateAssertions(data *response.ResponseData, rules []assertion.Rule) []assertion.Result {
	return assertion.Evaluate(data, rules)
}

// CurlParams holds the current request state for curl generation.
type CurlParams struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// GenerateCurl renders the current request as a curl command.
func (a *App) GenerateCurl(params CurlParams) (string, error) {
	return curl.Generate(curl.Request{
		Method:  params.Method,
		URL:     params.URL,
		Headers: params.Headers,
		Body:    params.Body,
	})
}

// ImportCurl parses a curl command and returns the reconstructed request.
func (a *App) ImportCurl(command string) (*curl.Request, error) {
	req, err := curl.Parse(command)
	return &req, err
}
