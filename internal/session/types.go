// Package session handles HTTP session management including cookies and custom headers
package session

import (
	"time"
)

// Session represents a stored HTTP session with cookies and headers
type Session struct {
	Version   string              `json:"version"`
	Name      string              `json:"name"`
	Created   time.Time           `json:"created"`
	Updated   time.Time           `json:"updated"`
	Cookies   []*SerializedCookie `json:"cookies,omitempty"`
	Headers   map[string]string   `json:"headers,omitempty"`
	Variables map[string]string   `json:"variables,omitempty"`
}

// SerializedCookie represents a cookie that can be saved to JSON
type SerializedCookie struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"httpOnly,omitempty"`
	SameSite string    `json:"sameSite,omitempty"`
}

// Options configures session behavior
type Options struct {
	// Directory to store session files
	Dir string
	
	// Whether to save headers automatically
	SaveHeaders bool
	
	// Whether to create new session if not exists
	CreateNew bool
}