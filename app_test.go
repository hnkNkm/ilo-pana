package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"ilo-pana/internal/collection"
	"ilo-pana/internal/config"
	"ilo-pana/internal/environment"
	"ilo-pana/internal/service"
)

func TestExecuteRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"method":  r.Method,
			"path":    r.URL.Path,
			"headers": r.Header,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	app := newTestApp(t)

	t.Run("basic_get", func(t *testing.T) {
		result, err := app.ExecuteRequest(service.RequestParams{
			Method: "get",
			URL:    server.URL + "/json",
			TimeoutMs: 5000,
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() error = %v", err)
		}
		if result.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", result.StatusCode)
		}
		if len(result.Body) == 0 {
			t.Error("Body is empty")
		}
	})

	t.Run("variable_expansion", func(t *testing.T) {
		result, err := app.ExecuteRequest(service.RequestParams{
			Method: "GET",
			URL:    "{{BASE}}/users/{{ID}}",
			Headers: map[string]string{"X-Token": "token-{{TOKEN}}"},
			TimeoutMs: 5000,
			Variables: map[string]string{
				"BASE":  server.URL,
				"ID":    "42",
				"TOKEN": "abc",
			},
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() error = %v", err)
		}
		var resp struct {
			Path    string              `json:"path"`
			Headers map[string][]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Path != "/users/42" {
			t.Errorf("path = %q, want /users/42", resp.Path)
		}
		if got := resp.Headers["X-Token"][0]; got != "token-abc" {
			t.Errorf("X-Token = %q, want token-abc", got)
		}
	})

	t.Run("environment_variable_expansion", func(t *testing.T) {
		if err := app.SaveEnvironment("dev", map[string]string{
			"BASE": server.URL,
			"ID":   "7",
		}); err != nil {
			t.Fatalf("SaveEnvironment() error = %v", err)
		}

		result, err := app.ExecuteRequest(service.RequestParams{
			Method:      "GET",
			URL:         "{{BASE}}/env/{{ID}}",
			TimeoutMs:   5000,
			Environment: "dev",
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() error = %v", err)
		}
		var resp struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if resp.Path != "/env/7" {
			t.Errorf("path = %q, want /env/7", resp.Path)
		}
	})

	t.Run("request_variables_override_environment", func(t *testing.T) {
		result, err := app.ExecuteRequest(service.RequestParams{
			Method:      "GET",
			URL:         "{{BASE}}/override",
			Headers:     map[string]string{"X-Mode": "{{MODE}}"},
			TimeoutMs:   5000,
			Environment: "dev",
			Variables:   map[string]string{"MODE": "request"},
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() error = %v", err)
		}
		var resp struct {
			Headers map[string][]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}
		if got := resp.Headers["X-Mode"][0]; got != "request" {
			t.Errorf("X-Mode = %q, want %q (request variable should win)", got, "request")
		}
	})

	t.Run("missing_environment_errors", func(t *testing.T) {
		_, err := app.ExecuteRequest(service.RequestParams{
			Method:      "GET",
			URL:         server.URL,
			TimeoutMs:   5000,
			Environment: "does-not-exist",
		})
		if err == nil {
			t.Error("ExecuteRequest() error = nil, want error for missing environment")
		}
	})

	t.Run("invalid_url", func(t *testing.T) {
		_, err := app.ExecuteRequest(service.RequestParams{
			Method: "GET",
			URL:    "not-a-url",
			TimeoutMs: 5000,
		})
		if err == nil {
			t.Error("ExecuteRequest() error = nil, want error for invalid URL")
		}
	})

	t.Run("timeout_clamped_to_max", func(t *testing.T) {
		// TimeoutMs above the 5-minute cap should be accepted (clamped server-side)
		result, err := app.ExecuteRequest(service.RequestParams{
			Method: "GET",
			URL:    server.URL,
			TimeoutMs: 999999999,
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() error = %v", err)
		}
		if result.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", result.StatusCode)
		}
	})

	t.Run("multipart_upload", func(t *testing.T) {
		result, err := app.ExecuteRequest(service.RequestParams{
			Method:     "POST",
			URL:        server.URL + "/multipart",
			BodyFormat: "multipart",
			FormFields: []config.FormField{
				{Key: "title", Value: "doc"},
				{Key: "file", IsFile: true, FileName: "a.txt", FileContent: []byte("contents"), ContentType: "text/plain"},
			},
			TimeoutMs: 5000,
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() multipart error = %v", err)
		}
		var resp struct {
			Headers map[string][]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		contentType := ""
		if v := resp.Headers["Content-Type"]; len(v) > 0 {
			contentType = v[0]
		}
		if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
			t.Errorf("Content-Type = %q, want multipart with boundary", contentType)
		}
	})

	t.Run("urlencoded_form", func(t *testing.T) {
		result, err := app.ExecuteRequest(service.RequestParams{
			Method:     "POST",
			URL:        server.URL + "/form",
			BodyFormat: "urlencoded",
			FormFields: []config.FormField{
				{Key: "name", Value: "alice"},
			},
			TimeoutMs: 5000,
		})
		if err != nil {
			t.Fatalf("ExecuteRequest() urlencoded error = %v", err)
		}
		var resp struct {
			Headers map[string][]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if v := resp.Headers["Content-Type"]; len(v) == 0 || v[0] != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %v, want urlencoded", v)
		}
	})
}

// newTestApp returns an App backed by temp-dir storages.
func newTestApp(t *testing.T) *App {
	t.Helper()
	collections := service.NewCollectionService(
		collection.NewFileStorage(filepath.Join(t.TempDir(), "collections")),
		service.RealClock(),
	)
	environments := service.NewEnvironmentService(
		environment.NewFileStorage(filepath.Join(t.TempDir(), "environments")),
		service.RealClock(),
	)
	return &App{
		executor:     service.NewRequestExecutor(environments),
		collections:  collections,
		environments: environments,
		openapi:      service.NewOpenAPIImporter(collections, service.RealClock()),
	}
}

func TestEnvironmentsCRUD(t *testing.T) {
	app := newTestApp(t)
	t.Run("save_and_list", func(t *testing.T) {
		if err := app.SaveEnvironment("dev", map[string]string{"BASE_URL": "http://localhost:8080"}); err != nil {
			t.Fatalf("SaveEnvironment() error = %v", err)
		}
		if err := app.SaveEnvironment("prod", map[string]string{"BASE_URL": "https://api.example.com"}); err != nil {
			t.Fatalf("SaveEnvironment() error = %v", err)
		}
		names, err := app.ListEnvironments()
		if err != nil {
			t.Fatalf("ListEnvironments() error = %v", err)
		}
		if len(names) != 2 {
			t.Errorf("ListEnvironments() = %v, want 2", names)
		}
	})

	t.Run("load_and_upsert_preserves_created", func(t *testing.T) {
		first, err := app.GetEnvironment("dev")
		if err != nil {
			t.Fatalf("GetEnvironment() error = %v", err)
		}
		created := first.Created

		if err := app.SaveEnvironment("dev", map[string]string{"BASE_URL": "http://localhost:9090"}); err != nil {
			t.Fatalf("SaveEnvironment() upsert error = %v", err)
		}
		second, err := app.GetEnvironment("dev")
		if err != nil {
			t.Fatalf("GetEnvironment() error = %v", err)
		}
		if !second.Created.Equal(created) {
			t.Errorf("Created changed on upsert: %v -> %v", created, second.Created)
		}
		if second.Variables["BASE_URL"] != "http://localhost:9090" {
			t.Errorf("variables not updated: %v", second.Variables)
		}
	})

	t.Run("validation", func(t *testing.T) {
		if err := app.SaveEnvironment("", map[string]string{}); err == nil {
			t.Error("expected error for empty name")
		}
		if err := app.DeleteEnvironment("missing"); err == nil {
			t.Error("expected error for missing environment")
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := app.DeleteEnvironment("prod"); err != nil {
			t.Fatalf("DeleteEnvironment() error = %v", err)
		}
		names, err := app.ListEnvironments()
		if err != nil {
			t.Fatalf("ListEnvironments() error = %v", err)
		}
		if len(names) != 1 || names[0] != "dev" {
			t.Errorf("ListEnvironments() = %v, want [dev]", names)
		}
	})
}

func TestCollectionsCRUD(t *testing.T) {
	app := newTestApp(t)

	req := collection.SavedRequest{
		Name:    "get-pikachu",
		Method:  "GET",
		URL:     "https://pokeapi.co/api/v2/pokemon/pikachu",
		Headers: map[string]string{"Accept": "application/json"},
	}

	t.Run("save_and_list", func(t *testing.T) {
		if err := app.SaveRequest("pokemon", req); err != nil {
			t.Fatalf("SaveRequest() error = %v", err)
		}
		names, err := app.ListCollections()
		if err != nil {
			t.Fatalf("ListCollections() error = %v", err)
		}
		if len(names) != 1 || names[0] != "pokemon" {
			t.Errorf("ListCollections() = %v, want [pokemon]", names)
		}
	})

	t.Run("load_and_upsert", func(t *testing.T) {
		req2 := req
		req2.Name = "get-raichu"
		req2.URL = "https://pokeapi.co/api/v2/pokemon/raichu"
		if err := app.SaveRequest("pokemon", req2); err != nil {
			t.Fatalf("SaveRequest() error = %v", err)
		}

		// Upsert same name: should replace, not duplicate
		req3 := req
		req3.Method = "POST"
		if err := app.SaveRequest("pokemon", req3); err != nil {
			t.Fatalf("SaveRequest() upsert error = %v", err)
		}

		c, err := app.GetCollection("pokemon")
		if err != nil {
			t.Fatalf("GetCollection() error = %v", err)
		}
		if len(c.Requests) != 2 {
			t.Errorf("expected 2 requests, got %d", len(c.Requests))
		}
		got, ok := c.Find("get-pikachu")
		if !ok {
			t.Fatal("expected to find get-pikachu")
		}
		if got.Method != "POST" {
			t.Errorf("upserted method = %q, want POST", got.Method)
		}
	})

	t.Run("validation", func(t *testing.T) {
		if err := app.SaveRequest("", req); err == nil {
			t.Error("SaveRequest() with empty collection name should error")
		}
		bad := req
		bad.Name = ""
		if err := app.SaveRequest("pokemon", bad); err == nil {
			t.Error("SaveRequest() with empty request name should error")
		}
	})

	t.Run("delete_request", func(t *testing.T) {
		if err := app.DeleteRequest("pokemon", "get-raichu"); err != nil {
			t.Fatalf("DeleteRequest() error = %v", err)
		}
		c, _ := app.GetCollection("pokemon")
		if _, ok := c.Find("get-raichu"); ok {
			t.Error("get-raichu should be deleted")
		}
		if err := app.DeleteRequest("pokemon", "nope"); err == nil {
			t.Error("DeleteRequest() of missing request should error")
		}
	})

	t.Run("export_import", func(t *testing.T) {
		exported, err := app.ExportCollection("pokemon")
		if err != nil {
			t.Fatalf("ExportCollection() error = %v", err)
		}
		if err := app.DeleteCollection("pokemon"); err != nil {
			t.Fatalf("DeleteCollection() error = %v", err)
		}
		if err := app.ImportCollection(exported); err != nil {
			t.Fatalf("ImportCollection() error = %v", err)
		}
		c, err := app.GetCollection("pokemon")
		if err != nil {
			t.Fatalf("GetCollection() after import error = %v", err)
		}
		if len(c.Requests) != 1 {
			t.Errorf("expected 1 request after import, got %d", len(c.Requests))
		}
		if err := app.ImportCollection("{bad json"); err == nil {
			t.Error("ImportCollection() of invalid JSON should error")
		}
	})
}

func TestImportOpenAPI(t *testing.T) {
	app := newTestApp(t)

	const spec = `
openapi: 3.0.3
info:
  title: Petstore
servers:
  - url: https://api.example.com/v1
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema: {type: string}
  /pets:
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string, default: rex}
`

	t.Run("import_into_named_collection", func(t *testing.T) {
		n, err := app.ImportOpenAPI(spec, "petstore")
		if err != nil {
			t.Fatalf("ImportOpenAPI() error = %v", err)
		}
		if n != 2 {
			t.Errorf("imported = %d, want 2", n)
		}
		c, err := app.GetCollection("petstore")
		if err != nil {
			t.Fatalf("GetCollection() error = %v", err)
		}
		if len(c.Requests) != 2 {
			t.Fatalf("requests = %d, want 2", len(c.Requests))
		}
		var get, post *collection.SavedRequest
		for i := range c.Requests {
			switch c.Requests[i].Method {
			case "GET":
				get = &c.Requests[i]
			case "POST":
				post = &c.Requests[i]
			}
		}
		if get == nil || get.URL != "https://api.example.com/v1/pets/{{petId}}" {
			t.Errorf("unexpected imported GET request: %+v", get)
		}
		if get.Variables["petId"] != "" {
			t.Errorf("petId variable not registered: %+v", get.Variables)
		}
		if post == nil || !strings.Contains(post.Body, `"name": "rex"`) {
			t.Errorf("POST body should include generated example: %s", post)
		}
	})

	t.Run("import_without_name_uses_title", func(t *testing.T) {
		if _, err := app.ImportOpenAPI(spec, ""); err != nil {
			t.Fatalf("ImportOpenAPI() without collection name error = %v", err)
		}
		if _, err := app.GetCollection("Petstore"); err != nil {
			t.Errorf("collection should be named after spec title: %v", err)
		}
	})

	t.Run("invalid_spec_errors", func(t *testing.T) {
		if _, err := app.ImportOpenAPI("not: a: spec", "x"); err == nil {
			t.Error("ImportOpenAPI() of invalid spec should error")
		}
	})
}
