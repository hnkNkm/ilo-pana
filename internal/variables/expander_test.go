package variables

import (
	"os"
	"reflect"
	"testing"
)

func TestExpander_Expand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		variables map[string]string
		envVars   map[string]string
		want      string
	}{
		{
			name:      "simple variable expansion",
			input:     "Hello {{name}}!",
			variables: map[string]string{"name": "World"},
			want:      "Hello World!",
		},
		{
			name:      "multiple variables",
			input:     "{{greeting}} {{name}}!",
			variables: map[string]string{"greeting": "Hello", "name": "World"},
			want:      "Hello World!",
		},
		{
			name:      "variable with spaces",
			input:     "Hello {{ name }}!",
			variables: map[string]string{"name": "World"},
			want:      "Hello World!",
		},
		{
			name:      "undefined variable preserved",
			input:     "Hello {{undefined}}!",
			variables: map[string]string{},
			want:      "Hello {{undefined}}!",
		},
		{
			name:      "environment variable fallback",
			input:     "Path: {{TEST_PATH}}",
			variables: map[string]string{},
			envVars:   map[string]string{"TEST_PATH": "/usr/bin"},
			want:      "Path: /usr/bin",
		},
		{
			name:      "variable overrides environment",
			input:     "Value: {{TEST_VAR}}",
			variables: map[string]string{"TEST_VAR": "local"},
			envVars:   map[string]string{"TEST_VAR": "env"},
			want:      "Value: local",
		},
		{
			name:      "URL with variables",
			input:     "{{BASE_URL}}/api/{{VERSION}}/users",
			variables: map[string]string{"BASE_URL": "https://api.example.com", "VERSION": "v1"},
			want:      "https://api.example.com/api/v1/users",
		},
		{
			name:      "JSON body with variables",
			input:     `{"token": "{{API_TOKEN}}", "user": "{{USERNAME}}"}`,
			variables: map[string]string{"API_TOKEN": "abc123", "USERNAME": "john"},
			want:      `{"token": "abc123", "user": "john"}`,
		},
		{
			name:      "no variables",
			input:     "Just plain text",
			variables: map[string]string{},
			want:      "Just plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for test
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			expander := New()
			expander.SetAll(tt.variables)

			got := expander.Expand(tt.input)
			if got != tt.want {
				t.Errorf("Expand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpander_ExpandWithWarnings(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		variables    map[string]string
		wantResult   string
		wantWarnings []string
	}{
		{
			name:         "no undefined variables",
			input:        "Hello {{name}}!",
			variables:    map[string]string{"name": "World"},
			wantResult:   "Hello World!",
			wantWarnings: nil,
		},
		{
			name:         "one undefined variable",
			input:        "Hello {{undefined}}!",
			variables:    map[string]string{},
			wantResult:   "Hello {{undefined}}!",
			wantWarnings: []string{"undefined variable: undefined"},
		},
		{
			name:         "multiple undefined variables",
			input:        "{{var1}} and {{var2}}",
			variables:    map[string]string{},
			wantResult:   "{{var1}} and {{var2}}",
			wantWarnings: []string{"undefined variable: var1", "undefined variable: var2"},
		},
		{
			name:         "mixed defined and undefined",
			input:        "{{defined}} and {{undefined}}",
			variables:    map[string]string{"defined": "value"},
			wantResult:   "value and {{undefined}}",
			wantWarnings: []string{"undefined variable: undefined"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expander := New()
			expander.SetAll(tt.variables)

			gotResult, gotWarnings := expander.ExpandWithWarnings(tt.input)
			if gotResult != tt.wantResult {
				t.Errorf("ExpandWithWarnings() result = %v, want %v", gotResult, tt.wantResult)
			}

			if !reflect.DeepEqual(gotWarnings, tt.wantWarnings) {
				t.Errorf("ExpandWithWarnings() warnings = %v, want %v", gotWarnings, tt.wantWarnings)
			}
		})
	}
}

func TestExpander_ExpandHeaders(t *testing.T) {
	expander := New()
	expander.Set("TOKEN", "bearer-abc123")
	expander.Set("VERSION", "v2")

	headers := map[string]string{
		"Authorization": "Bearer {{TOKEN}}",
		"API-Version":   "{{VERSION}}",
		"Content-Type":  "application/json",
	}

	expected := map[string]string{
		"Authorization": "Bearer bearer-abc123",
		"API-Version":   "v2",
		"Content-Type":  "application/json",
	}

	result := expander.ExpandHeaders(headers)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExpandHeaders() = %v, want %v", result, expected)
	}
}

func TestParseVariables(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "single variable",
			args: []string{"KEY=value"},
			want: map[string]string{"KEY": "value"},
		},
		{
			name: "multiple variables",
			args: []string{"KEY1=value1", "KEY2=value2"},
			want: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name: "variable with equals in value",
			args: []string{"URL=https://example.com?param=value"},
			want: map[string]string{"URL": "https://example.com?param=value"},
		},
		{
			name: "variable with spaces",
			args: []string{"MESSAGE=Hello World"},
			want: map[string]string{"MESSAGE": "Hello World"},
		},
		{
			name:    "invalid format - no equals",
			args:    []string{"INVALID"},
			wantErr: true,
		},
		{
			name:    "invalid format - empty key",
			args:    []string{"=value"},
			wantErr: true,
		},
		{
			name:    "invalid variable name - starts with number",
			args:    []string{"1KEY=value"},
			wantErr: true,
		},
		{
			name:    "invalid variable name - special chars",
			args:    []string{"KEY-NAME=value"},
			wantErr: true,
		},
		{
			name: "valid variable names",
			args: []string{"_var=1", "VAR_2=2", "myVar3=3"},
			want: map[string]string{"_var": "1", "VAR_2": "2", "myVar3": "3"},
		},
		{
			name: "empty value allowed",
			args: []string{"EMPTY="},
			want: map[string]string{"EMPTY": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVariables(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidVariableName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"starts with letter", "variable", true},
		{"starts with uppercase", "Variable", true},
		{"starts with underscore", "_variable", true},
		{"contains numbers", "var123", true},
		{"contains underscores", "var_name_2", true},
		{"all caps", "API_KEY", true},
		{"starts with number", "1variable", false},
		{"contains hyphen", "var-name", false},
		{"contains space", "var name", false},
		{"contains dot", "var.name", false},
		{"empty string", "", false},
		{"single letter", "x", true},
		{"single underscore", "_", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidVariableName(tt.input); got != tt.want {
				t.Errorf("isValidVariableName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}