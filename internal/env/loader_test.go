package env

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantKey   string
		wantValue string
		wantErr   bool
	}{
		{
			name:      "simple key value",
			line:      "KEY=value",
			wantKey:   "KEY",
			wantValue: "value",
		},
		{
			name:      "with spaces around equals",
			line:      "KEY = value",
			wantKey:   "KEY",
			wantValue: "value",
		},
		{
			name:      "double quoted value",
			line:      `KEY="hello world"`,
			wantKey:   "KEY",
			wantValue: "hello world",
		},
		{
			name:      "single quoted value",
			line:      `KEY='hello world'`,
			wantKey:   "KEY",
			wantValue: "hello world",
		},
		{
			name:      "value with equals sign",
			line:      "URL=https://example.com?key=value",
			wantKey:   "URL",
			wantValue: "https://example.com?key=value",
		},
		{
			name:      "export prefix",
			line:      "export KEY=value",
			wantKey:   "KEY",
			wantValue: "value",
		},
		{
			name:      "inline comment",
			line:      "KEY=value # this is a comment",
			wantKey:   "KEY",
			wantValue: "value",
		},
		{
			name:      "empty value",
			line:      "KEY=",
			wantKey:   "KEY",
			wantValue: "",
		},
		{
			name:      "comment line",
			line:      "# This is a comment",
			wantKey:   "",
			wantValue: "",
		},
		{
			name:      "no equals sign",
			line:      "JUST_KEY",
			wantKey:   "",
			wantValue: "",
		},
		{
			name:    "empty key",
			line:    "=value",
			wantErr: true,
		},
		{
			name:    "invalid key with hyphen",
			line:    "KEY-NAME=value",
			wantErr: true,
		},
		{
			name:      "value with spaces",
			line:      "MESSAGE=Hello World",
			wantKey:   "MESSAGE",
			wantValue: "Hello World",
		},
		{
			name:      "underscore in key",
			line:      "API_KEY=secret123",
			wantKey:   "API_KEY",
			wantValue: "secret123",
		},
		{
			name:      "number in key",
			line:      "VAR2=value",
			wantKey:   "VAR2",
			wantValue: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue, err := parseLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotKey != tt.wantKey {
				t.Errorf("parseLine() gotKey = %v, want %v", gotKey, tt.wantKey)
			}
			if gotValue != tt.wantValue {
				t.Errorf("parseLine() gotValue = %v, want %v", gotValue, tt.wantValue)
			}
		})
	}
}

func TestLoader_Load(t *testing.T) {
	// Create a temporary .env file for testing
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	
	content := `# Test environment file
BASE_URL=https://api.example.com
API_KEY=secret123
DEBUG=true

# Another comment
VERSION=v1.0.0
EMPTY_VALUE=

export EXPORTED_VAR=exported_value
QUOTED="value with spaces"
SINGLE_QUOTED='another value'

# Inline comment test
PORT=8080 # Default port
`
	
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test .env file: %v", err)
	}

	loader := NewLoader()
	if err := loader.Load(envFile); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	expected := map[string]string{
		"BASE_URL":      "https://api.example.com",
		"API_KEY":       "secret123",
		"DEBUG":         "true",
		"VERSION":       "v1.0.0",
		"EMPTY_VALUE":   "",
		"EXPORTED_VAR":  "exported_value",
		"QUOTED":        "value with spaces",
		"SINGLE_QUOTED": "another value",
		"PORT":          "8080",
	}

	got := loader.GetVariables()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetVariables() = %v, want %v", got, expected)
	}
}

func TestLoader_LoadNonExistent(t *testing.T) {
	loader := NewLoader()
	// Should not error on non-existent file
	err := loader.Load("/non/existent/file.env")
	if err != nil {
		t.Errorf("Load() with non-existent file should return nil, got %v", err)
	}

	// Variables should be empty
	if len(loader.GetVariables()) != 0 {
		t.Errorf("Expected no variables, got %v", loader.GetVariables())
	}
}

func TestLoader_Merge(t *testing.T) {
	loader := NewLoader()
	
	// Set initial variables
	loader.Merge(map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	})

	// Merge more variables, including overwrite
	loader.Merge(map[string]string{
		"KEY2": "new_value2",
		"KEY3": "value3",
	})

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "new_value2",
		"KEY3": "value3",
	}

	got := loader.GetVariables()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("After Merge() = %v, want %v", got, expected)
	}
}

func TestProcessValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple value", "value", "value"},
		{"trimmed spaces", "  value  ", "value"},
		{"double quoted", `"hello world"`, "hello world"},
		{"single quoted", `'hello world'`, "hello world"},
		{"inline comment", "value # comment", "value"},
		{"no comment in quotes", `"value # not comment"`, "value # not comment"},
		{"equals in value", "key=value", "key=value"},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processValue(tt.input); got != tt.want {
				t.Errorf("processValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsValidEnvName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple name", "KEY", true},
		{"with underscore", "API_KEY", true},
		{"with numbers", "KEY123", true},
		{"starts with underscore", "_KEY", true},
		{"lowercase", "key", true},
		{"mixed case", "MyKey", true},
		{"starts with number", "1KEY", false},
		{"with hyphen", "KEY-NAME", false},
		{"with space", "KEY NAME", false},
		{"with dot", "KEY.NAME", false},
		{"empty", "", false},
		{"special chars", "KEY$VAR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidEnvName(tt.input); got != tt.want {
				t.Errorf("isValidEnvName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create first env file
	env1 := filepath.Join(tmpDir, ".env")
	content1 := `KEY1=value1
KEY2=original`
	os.WriteFile(env1, []byte(content1), 0644)

	// Create second env file
	env2 := filepath.Join(tmpDir, ".env.local")
	content2 := `KEY2=overridden
KEY3=value3`
	os.WriteFile(env2, []byte(content2), 0644)

	// Load both files
	vars, err := LoadMultiple(env1, env2)
	if err != nil {
		t.Fatalf("LoadMultiple() error = %v", err)
	}

	expected := map[string]string{
		"KEY1": "value1",
		"KEY2": "overridden",  // Should be overridden by second file
		"KEY3": "value3",
	}

	if !reflect.DeepEqual(vars, expected) {
		t.Errorf("LoadMultiple() = %v, want %v", vars, expected)
	}

	// Test with non-existent file (should not error)
	vars, err = LoadMultiple(env1, "/non/existent.env", env2)
	if err != nil {
		t.Fatalf("LoadMultiple() with non-existent file error = %v", err)
	}

	if !reflect.DeepEqual(vars, expected) {
		t.Errorf("LoadMultiple() with non-existent = %v, want %v", vars, expected)
	}
}