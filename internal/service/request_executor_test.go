package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestExecutor(t *testing.T) (*RequestExecutor, *EnvironmentService) {
	t.Helper()
	envs := newEnvironmentService(t, fixedClock(clockT1))
	return NewRequestExecutor(envs), envs
}

func TestRequestExecutor_BasicExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "yes"})
	}))
	defer server.Close()

	ex, _ := newTestExecutor(t)
	result, err := ex.Execute(RequestParams{Method: "get", URL: server.URL, TimeoutMs: 5000})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if !strings.Contains(result.Body, `"ok": "yes"`) {
		t.Errorf("Body = %q, want ok:yes", result.Body)
	}
}

func TestRequestExecutor_EnvironmentMergeAndOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"path":    r.URL.Path,
			"headers": r.Header,
		})
	}))
	defer server.Close()

	ex, envs := newTestExecutor(t)
	if err := envs.Save("dev", map[string]string{
		"BASE": server.URL,
		"ID":   "7",
		"MODE": "env",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	result, err := ex.Execute(RequestParams{
		Method:      "GET",
		URL:         "{{BASE}}/env/{{ID}}",
		Headers:     map[string]string{"X-Mode": "{{MODE}}"},
		TimeoutMs:   5000,
		Environment: "dev",
		Variables:   map[string]string{"MODE": "request"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var resp struct {
		Path    string              `json:"path"`
		Headers map[string][]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(result.Body), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Path != "/env/7" {
		t.Errorf("path = %q, want /env/7", resp.Path)
	}
	if got := resp.Headers["X-Mode"][0]; got != "request" {
		t.Errorf("X-Mode = %q, want request (request variable should win)", got)
	}
}

func TestRequestExecutor_MissingEnvironment(t *testing.T) {
	ex, _ := newTestExecutor(t)
	_, err := ex.Execute(RequestParams{
		Method:      "GET",
		URL:         "http://example.com",
		TimeoutMs:   5000,
		Environment: "does-not-exist",
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want error for missing environment")
	}
	if !strings.Contains(err.Error(), `failed to load environment "does-not-exist"`) {
		t.Errorf("error = %v, want wrapped load failure", err)
	}
}

func TestRequestExecutor_EnvironmentLoadFailure(t *testing.T) {
	boom := errors.New("disk on fire")
	envs := NewEnvironmentService(&failingEnvironmentStorage{err: boom}, fixedClock(clockT1))
	ex := NewRequestExecutor(envs)

	_, err := ex.Execute(RequestParams{
		Method:      "GET",
		URL:         "http://example.com",
		TimeoutMs:   5000,
		Environment: "dev",
	})
	if !errors.Is(err, boom) {
		t.Errorf("Execute() error = %v, want %v", err, boom)
	}
}

func TestRequestExecutor_InvalidURL(t *testing.T) {
	ex, _ := newTestExecutor(t)
	_, err := ex.Execute(RequestParams{Method: "GET", URL: "not-a-url", TimeoutMs: 5000})
	if err == nil || !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("Execute() error = %v, want invalid URL", err)
	}
}
