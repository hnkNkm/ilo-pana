package session

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"time"
	
	"golang.org/x/net/publicsuffix"
)

// Manager handles session lifecycle and cookie management
type Manager struct {
	mu        sync.RWMutex
	name      string
	storage   Storage
	jar       http.CookieJar
	headers   map[string]string
	variables map[string]string
	options   *Options
	tracker   *CookieTracker
}

// NewManager creates a new session manager
func NewManager(name string, opts *Options) (*Manager, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	
	// Create cookie jar with public suffix list
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}
	
	// Create storage backend
	storage := NewFileStorage(opts.Dir)
	
	m := &Manager{
		name:      name,
		storage:   storage,
		jar:       jar,
		headers:   make(map[string]string),
		variables: make(map[string]string),
		options:   opts,
		tracker:   NewCookieTracker(),
	}
	
	// Try to load existing session
	if err := m.Load(); err != nil {
		// If session doesn't exist and CreateNew is false, return error
		if !opts.CreateNew && IsNotExist(err) {
			return nil, fmt.Errorf("session %q does not exist", name)
		}
		// Otherwise create new session
		if opts.CreateNew {
			m.initNewSession()
		}
	}
	
	return m, nil
}

// DefaultOptions returns default session options
func DefaultOptions() *Options {
	return &Options{
		Dir:         defaultSessionDir(),
		SaveHeaders: false,
		CreateNew:   false,
	}
}

// Load loads session from storage
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, err := m.storage.Load(m.name)
	if err != nil {
		return err
	}
	
	// Restore cookies to jar
	for _, sc := range session.Cookies {
		cookie := &http.Cookie{
			Name:     sc.Name,
			Value:    sc.Value,
			Domain:   sc.Domain,
			Path:     sc.Path,
			Expires:  sc.Expires,
			Secure:   sc.Secure,
			HttpOnly: sc.HttpOnly,
		}
		
		// Parse SameSite
		switch sc.SameSite {
		case "Strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "Lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "None":
			cookie.SameSite = http.SameSiteNoneMode
		default:
			cookie.SameSite = http.SameSiteDefaultMode
		}
		
		// Add cookie to jar for the appropriate URL
		u := &url.URL{
			Scheme: "https",
			Host:   sc.Domain,
			Path:   sc.Path,
		}
		m.jar.SetCookies(u, []*http.Cookie{cookie})
	}
	
	// Restore headers and variables
	if session.Headers != nil {
		m.headers = session.Headers
	}
	if session.Variables != nil {
		m.variables = session.Variables
	}
	
	return nil
}

// Save saves current session to storage
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session := &Session{
		Version:   "1.0",
		Name:      m.name,
		Updated:   time.Now(),
		Headers:   m.headers,
		Variables: m.variables,
	}
	
	// Check if this is a new session
	existing, err := m.storage.Load(m.name)
	if err == nil && existing != nil {
		session.Created = existing.Created
	} else {
		session.Created = time.Now()
	}
	
	// Extract cookies from all known domains
	// This is a simplified approach - in production we'd track domains
	session.Cookies = m.extractAllCookies()
	
	return m.storage.Save(m.name, session)
}

// extractAllCookies extracts cookies from the tracker
func (m *Manager) extractAllCookies() []*SerializedCookie {
	return m.tracker.GetAllCookies()
}

// GetCookieJar returns the HTTP cookie jar
func (m *Manager) GetCookieJar() http.CookieJar {
	return m.jar
}

// SetHeader sets a custom header for the session
func (m *Manager) SetHeader(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.headers[key] = value
}

// GetHeaders returns all session headers
func (m *Manager) GetHeaders() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to prevent external modification
	result := make(map[string]string, len(m.headers))
	for k, v := range m.headers {
		result[k] = v
	}
	return result
}

// SetVariable sets a session variable
func (m *Manager) SetVariable(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.variables[key] = value
}

// GetVariable retrieves a session variable
func (m *Manager) GetVariable(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.variables[key]
	return val, ok
}

// Clear removes all session data
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create new cookie jar
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return fmt.Errorf("failed to create new cookie jar: %w", err)
	}
	
	m.jar = jar
	m.headers = make(map[string]string)
	m.variables = make(map[string]string)
	
	// Delete from storage
	return m.storage.Delete(m.name)
}

// initNewSession initializes a new empty session
func (m *Manager) initNewSession() {
	m.headers = make(map[string]string)
	m.variables = make(map[string]string)
}

// ProcessResponse processes HTTP response to update session
func (m *Manager) ProcessResponse(resp *http.Response) {
	// The cookie jar automatically handles Set-Cookie headers
	// We also track them for serialization
	m.tracker.TrackResponse(resp)
	
	// Here we could also extract tokens from response if configured
	// For example, looking for common token patterns in JSON responses
}

// ApplyToRequest applies session data to an HTTP request
func (m *Manager) ApplyToRequest(req *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Apply custom headers
	for key, value := range m.headers {
		if req.Header.Get(key) == "" {
			req.Header.Set(key, value)
		}
	}
	
	// Cookies are automatically handled by the cookie jar in the HTTP client
}