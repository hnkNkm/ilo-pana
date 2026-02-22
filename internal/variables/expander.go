// Package variables handles variable expansion for templates
package variables

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Expander handles variable expansion with multiple sources
type Expander struct {
	variables map[string]string
	useEnv    bool
}

// New creates a new variable expander
func New() *Expander {
	return &Expander{
		variables: make(map[string]string),
		useEnv:    true,
	}
}

// Set adds or updates a variable
func (e *Expander) Set(key, value string) {
	e.variables[key] = value
}

// SetAll adds multiple variables at once
func (e *Expander) SetAll(vars map[string]string) {
	for k, v := range vars {
		e.variables[k] = v
	}
}

// Get retrieves a variable value with fallback to environment
func (e *Expander) Get(key string) (string, bool) {
	// First check internal variables
	if val, ok := e.variables[key]; ok {
		return val, true
	}
	
	// Then check environment variables if enabled
	if e.useEnv {
		if val := os.Getenv(key); val != "" {
			return val, true
		}
	}
	
	return "", false
}

// Expand replaces all {{variable}} patterns in the input string
func (e *Expander) Expand(input string) string {
	// Pattern matches {{variable_name}} with optional spaces
	pattern := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	
	return pattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name from match
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		varName := submatches[1]
		if value, ok := e.Get(varName); ok {
			return value
		}
		
		// Return original if variable not found
		// This allows for undefined variables to pass through
		return match
	})
}

// ExpandWithWarnings expands variables and collects warnings for undefined variables
func (e *Expander) ExpandWithWarnings(input string) (string, []string) {
	var warnings []string
	pattern := regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}`)
	
	result := pattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := pattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		
		varName := submatches[1]
		if value, ok := e.Get(varName); ok {
			return value
		}
		
		// Collect warning for undefined variable
		warnings = append(warnings, fmt.Sprintf("undefined variable: %s", varName))
		return match
	})
	
	return result, warnings
}

// ExpandHeaders expands variables in header values
func (e *Expander) ExpandHeaders(headers map[string]string) map[string]string {
	expanded := make(map[string]string, len(headers))
	for key, value := range headers {
		expanded[key] = e.Expand(value)
	}
	return expanded
}

// ParseVariables parses key=value pairs from command line arguments
func ParseVariables(args []string) (map[string]string, error) {
	vars := make(map[string]string)
	
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable format %q, expected 'key=value'", arg)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		if key == "" {
			return nil, fmt.Errorf("empty variable key in %q", arg)
		}
		
		// Validate variable name
		if !isValidVariableName(key) {
			return nil, fmt.Errorf("invalid variable name %q: must start with letter or underscore and contain only letters, numbers, and underscores", key)
		}
		
		vars[key] = value
	}
	
	return vars, nil
}

// isValidVariableName checks if a variable name is valid
func isValidVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// Must start with letter or underscore
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}
	
	// Rest must be letters, numbers, or underscores
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	
	return true
}