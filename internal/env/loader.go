// Package env handles environment file loading and parsing
package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles loading environment variables from files
type Loader struct {
	variables map[string]string
}

// NewLoader creates a new environment loader
func NewLoader() *Loader {
	return &Loader{
		variables: make(map[string]string),
	}
}

// Load reads and parses an environment file
func (l *Loader) Load(filename string) error {
	// Expand ~ to home directory if needed
	if strings.HasPrefix(filename, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		filename = filepath.Join(home, filename[2:])
	}

	file, err := os.Open(filename)
	if err != nil {
		// It's okay if .env file doesn't exist
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Skip empty lines and comments
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, err := parseLine(line)
		if err != nil {
			return fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}

		if key != "" {
			l.variables[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %w", err)
	}

	return nil
}

// LoadDefault attempts to load .env from the current directory
func (l *Loader) LoadDefault() error {
	return l.Load(".env")
}

// GetVariables returns all loaded variables
func (l *Loader) GetVariables() map[string]string {
	// Return a copy to prevent external modification
	result := make(map[string]string, len(l.variables))
	for k, v := range l.variables {
		result[k] = v
	}
	return result
}

// Get retrieves a single variable
func (l *Loader) Get(key string) (string, bool) {
	value, ok := l.variables[key]
	return value, ok
}

// Merge adds variables from another map, overwriting existing ones
func (l *Loader) Merge(vars map[string]string) {
	for k, v := range vars {
		l.variables[k] = v
	}
}

// parseLine parses a single line from an env file
func parseLine(line string) (string, string, error) {
	// Handle export prefix (common in .env files)
	line = strings.TrimPrefix(line, "export ")
	
	// Find the equals sign
	equalIndex := strings.Index(line, "=")
	if equalIndex == -1 {
		// No equals sign, might be just a variable declaration
		return "", "", nil
	}

	key := strings.TrimSpace(line[:equalIndex])
	value := line[equalIndex+1:]

	// Validate key
	if key == "" {
		return "", "", fmt.Errorf("empty key")
	}

	// Check for valid variable name
	if !isValidEnvName(key) {
		return "", "", fmt.Errorf("invalid variable name: %s", key)
	}

	// Process value
	value = processValue(value)

	return key, value, nil
}

// processValue handles quoted values and escape sequences
func processValue(value string) string {
	value = strings.TrimSpace(value)
	
	// Check if value is quoted
	isQuoted := false

	// Handle quoted values
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
		   (value[0] == '\'' && value[len(value)-1] == '\'') {
			isQuoted = true
			quoteChar := value[0]
			// Remove quotes
			value = value[1 : len(value)-1]

			// Handle basic escape sequences in double quotes
			if quoteChar == '"' {
				value = strings.ReplaceAll(value, `\n`, "\n")
				value = strings.ReplaceAll(value, `\t`, "\t")
				value = strings.ReplaceAll(value, `\r`, "\r")
				value = strings.ReplaceAll(value, `\\`, "\\")
				value = strings.ReplaceAll(value, `\"`, "\"")
			}
		}
	}

	// Remove inline comments only if value is not quoted
	if !isQuoted {
		if commentIndex := strings.Index(value, " #"); commentIndex != -1 {
			value = value[:commentIndex]
		}
		value = strings.TrimSpace(value)
	}

	return value
}

// isValidEnvName checks if an environment variable name is valid
func isValidEnvName(name string) bool {
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

// LoadMultiple loads variables from multiple files in order
func LoadMultiple(filenames ...string) (map[string]string, error) {
	loader := NewLoader()
	
	for _, filename := range filenames {
		if err := loader.Load(filename); err != nil {
			// Skip if file doesn't exist
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load %s: %w", filename, err)
			}
		}
	}
	
	return loader.GetVariables(), nil
}