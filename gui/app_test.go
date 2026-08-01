package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	app := &App{}

	t.Run("basic_get", func(t *testing.T) {
		result, err := app.ExecuteRequest(RequestParams{
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
		result, err := app.ExecuteRequest(RequestParams{
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

	t.Run("invalid_url", func(t *testing.T) {
		_, err := app.ExecuteRequest(RequestParams{
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
		result, err := app.ExecuteRequest(RequestParams{
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
}
