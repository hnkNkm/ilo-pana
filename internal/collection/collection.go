// Package collection provides named request storage for the GUI.
// A Collection is a group of SavedRequests persisted as a JSON file,
// so it can be versioned with Git like Postman/Bruno collections.
package collection

import (
	"encoding/json"
	"fmt"
	"time"
)

// SavedRequest is a named, reusable request.
type SavedRequest struct {
	Name      string            `json:"name"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
	Created   time.Time         `json:"created"`
	Updated   time.Time         `json:"updated"`
}

// Collection is a named group of saved requests.
type Collection struct {
	Name     string         `json:"name"`
	Requests []SavedRequest `json:"requests"`
	Created  time.Time      `json:"created"`
	Updated  time.Time      `json:"updated"`
}

// Upsert adds or replaces a request with the same name.
func (c *Collection) Upsert(req SavedRequest) {
	for i, existing := range c.Requests {
		if existing.Name == req.Name {
			req.Created = existing.Created
			c.Requests[i] = req
			return
		}
	}
	c.Requests = append(c.Requests, req)
}

// Find returns a request by name and whether it exists.
func (c *Collection) Find(name string) (SavedRequest, bool) {
	for _, existing := range c.Requests {
		if existing.Name == name {
			return existing, true
		}
	}
	return SavedRequest{}, false
}

// Remove deletes a request by name. Returns true if it was found.
func (c *Collection) Remove(name string) bool {
	for i, existing := range c.Requests {
		if existing.Name == name {
			c.Requests = append(c.Requests[:i], c.Requests[i+1:]...)
			return true
		}
	}
	return false
}

// MarshalJSON serializes the collection to indented JSON.
func (c *Collection) MarshalJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to encode collection: %w", err)
	}
	return string(data), nil
}

// UnmarshalJSON parses a collection from a JSON string.
func UnmarshalJSON(data string) (*Collection, error) {
	var c Collection
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("invalid collection JSON: %w", err)
	}
	return &c, nil
}
