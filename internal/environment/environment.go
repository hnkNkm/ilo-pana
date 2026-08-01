// Package environment provides named variable sets ("environments")
// for the GUI, e.g. dev/staging/prod. An environment is a persisted
// set of key-value variables that are merged into {{VAR}} expansion
// when a request is sent.
package environment

import (
	"encoding/json"
	"fmt"
	"time"
)

// Environment is a named set of variables.
type Environment struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	Created   time.Time         `json:"created"`
	Updated   time.Time         `json:"updated"`
}

// MarshalJSON serializes the environment to indented JSON.
func (e *Environment) MarshalJSON() (string, error) {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode environment: %w", err)
	}
	return string(data), nil
}

// UnmarshalJSON parses an environment from a JSON string.
func UnmarshalJSON(data string) (*Environment, error) {
	var e Environment
	if err := json.Unmarshal([]byte(data), &e); err != nil {
		return nil, fmt.Errorf("invalid environment JSON: %w", err)
	}
	return &e, nil
}
