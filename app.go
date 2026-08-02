package main

import (
	"context"
	"fmt"

	"ilo-pana/internal/assertion"
	"ilo-pana/internal/collection"
	"ilo-pana/internal/curl"
	"ilo-pana/internal/environment"
	"ilo-pana/internal/response"
	"ilo-pana/internal/service"
)

// App is a thin Wails facade over the service layer (internal/service).
// It only adapts Wails DTOs and delegates to services; no domain logic
// lives here. All execution, persistence and import logic is unit-tested
// in internal/service without Wails.
type App struct {
	ctx          context.Context
	executor     *service.RequestExecutor
	collections  *service.CollectionService
	environments *service.EnvironmentService
	openapi      *service.OpenAPIImporter
}

// NewApp creates a new App application struct
func NewApp() *App {
	collections := service.NewCollectionService(collection.NewFileStorage(""), service.RealClock())
	environments := service.NewEnvironmentService(environment.NewFileStorage(""), service.RealClock())
	return &App{
		executor:     service.NewRequestExecutor(environments),
		collections:  collections,
		environments: environments,
		openapi:      service.NewOpenAPIImporter(collections, service.RealClock()),
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

// ExecuteRequest executes an HTTP request and returns structured response data for GUI
func (a *App) ExecuteRequest(params service.RequestParams) (*response.ResponseData, error) {
	return a.executor.Execute(params)
}

// ExecuteRequestSimple is a simpler signature for Wails binding (method, url, body, headers)
func (a *App) ExecuteRequestSimple(method, url, body string, headers map[string]string) (*response.ResponseData, error) {
	return a.ExecuteRequest(service.RequestParams{
		Method:    method,
		URL:       url,
		Body:      body,
		Headers:   headers,
		TimeoutMs: 30000, // Default 30 seconds
	})
}

// ExecuteRequestWithTimeout allows frontend to specify a custom timeout
func (a *App) ExecuteRequestWithTimeout(method, url, body string, headers map[string]string, timeoutMs int) (*response.ResponseData, error) {
	return a.ExecuteRequest(service.RequestParams{
		Method:    method,
		URL:       url,
		Body:      body,
		Headers:   headers,
		TimeoutMs: timeoutMs,
	})
}

// SaveRequest upserts a request into the named collection, creating it if needed.
func (a *App) SaveRequest(collectionName string, req collection.SavedRequest) error {
	return a.collections.SaveRequest(collectionName, req)
}

// ListCollections returns the names of all saved collections.
func (a *App) ListCollections() ([]string, error) {
	return a.collections.List()
}

// GetCollection returns a collection by name.
func (a *App) GetCollection(name string) (*collection.Collection, error) {
	return a.collections.Get(name)
}

// DeleteRequest removes a request from a collection.
func (a *App) DeleteRequest(collectionName, requestName string) error {
	return a.collections.DeleteRequest(collectionName, requestName)
}

// DeleteCollection removes a collection entirely.
func (a *App) DeleteCollection(name string) error {
	return a.collections.DeleteCollection(name)
}

// ExportCollection returns a collection as a JSON string (for copy/paste sharing).
func (a *App) ExportCollection(name string) (string, error) {
	return a.collections.Export(name)
}

// ImportCollection parses a JSON collection and saves it, replacing any existing one.
func (a *App) ImportCollection(data string) error {
	return a.collections.Import(data)
}

// SaveEnvironment upserts a named environment with the given variables.
func (a *App) SaveEnvironment(name string, vars map[string]string) error {
	return a.environments.Save(name, vars)
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
	return a.openapi.Import(content, collectionName)
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
