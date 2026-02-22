package session

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage defines the interface for session persistence
type Storage interface {
	Load(name string) (*Session, error)
	Save(name string, session *Session) error
	Delete(name string) error
	List() ([]string, error)
}

// FileStorage implements file-based session storage
type FileStorage struct {
	dir string
}

// NewFileStorage creates a new file-based storage backend
func NewFileStorage(dir string) *FileStorage {
	if dir == "" {
		dir = defaultSessionDir()
	}
	return &FileStorage{dir: dir}
}

// defaultSessionDir returns the default session directory
func defaultSessionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".ilo-pana/sessions"
	}
	return filepath.Join(home, ".ilo-pana", "sessions")
}

// Load loads a session from file
func (fs *FileStorage) Load(name string) (*Session, error) {
	filename := fs.sessionPath(name)
	
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session %q not found", name)
		}
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()
	
	var session Session
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session: %w", err)
	}
	
	return &session, nil
}

// Save saves a session to file
func (fs *FileStorage) Save(name string, session *Session) error {
	// Ensure directory exists
	if err := fs.ensureDir(); err != nil {
		return err
	}
	
	filename := fs.sessionPath(name)
	
	// Write to temporary file first
	tmpFile := filename + ".tmp"
	file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(session); err != nil {
		file.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to encode session: %w", err)
	}
	
	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close session file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to save session file: %w", err)
	}
	
	return nil
}

// Delete removes a session file
func (fs *FileStorage) Delete(name string) error {
	filename := fs.sessionPath(name)
	
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session %q not found", name)
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	return nil
}

// List returns all available session names
func (fs *FileStorage) List() ([]string, error) {
	// Ensure directory exists
	if err := fs.ensureDir(); err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	
	var sessions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			sessionName := strings.TrimSuffix(name, ".json")
			sessions = append(sessions, sessionName)
		}
	}
	
	return sessions, nil
}

// sessionPath returns the full path for a session file
func (fs *FileStorage) sessionPath(name string) string {
	// Sanitize session name
	name = sanitizeSessionName(name)
	return filepath.Join(fs.dir, name+".json")
}

// ensureDir ensures the storage directory exists
func (fs *FileStorage) ensureDir() error {
	if err := os.MkdirAll(fs.dir, 0700); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}
	return nil
}

// sanitizeSessionName removes potentially dangerous characters from session names
func sanitizeSessionName(name string) string {
	// Replace path separators and other dangerous characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		"..", "_",
		"~", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// IsNotExist returns true if the error indicates a session doesn't exist
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if it's an os.IsNotExist error
	if os.IsNotExist(err) {
		return true
	}
	
	// Check if error message contains "not found"
	return strings.Contains(err.Error(), "not found")
}

// ShowSession displays session information
func ShowSession(w io.Writer, session *Session, verbose bool) {
	fmt.Fprintf(w, "Session: %s\n", session.Name)
	fmt.Fprintf(w, "Created: %s\n", session.Created.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated: %s\n", session.Updated.Format("2006-01-02 15:04:05"))
	
	if len(session.Cookies) > 0 {
		fmt.Fprintf(w, "\nCookies (%d):\n", len(session.Cookies))
		for _, cookie := range session.Cookies {
			if verbose {
				fmt.Fprintf(w, "  %s=%s (Domain: %s, Path: %s)\n", 
					cookie.Name, cookie.Value, cookie.Domain, cookie.Path)
			} else {
				// Mask cookie values for security
				maskedValue := maskValue(cookie.Value)
				fmt.Fprintf(w, "  %s=%s (Domain: %s)\n", 
					cookie.Name, maskedValue, cookie.Domain)
			}
		}
	}
	
	if len(session.Headers) > 0 {
		fmt.Fprintf(w, "\nHeaders (%d):\n", len(session.Headers))
		for key, value := range session.Headers {
			if verbose || !isSensitiveHeader(key) {
				fmt.Fprintf(w, "  %s: %s\n", key, value)
			} else {
				fmt.Fprintf(w, "  %s: %s\n", key, maskValue(value))
			}
		}
	}
	
	if len(session.Variables) > 0 {
		fmt.Fprintf(w, "\nVariables (%d):\n", len(session.Variables))
		for key, value := range session.Variables {
			if verbose || !isSensitiveVariable(key) {
				fmt.Fprintf(w, "  %s = %s\n", key, value)
			} else {
				fmt.Fprintf(w, "  %s = %s\n", key, maskValue(value))
			}
		}
	}
}

// maskValue masks sensitive values
func maskValue(value string) string {
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****"
}

// isSensitiveHeader checks if a header contains sensitive data
func isSensitiveHeader(key string) bool {
	lower := strings.ToLower(key)
	sensitiveHeaders := []string{
		"authorization",
		"x-api-key",
		"x-auth-token",
		"x-access-token",
		"x-secret-key",
	}
	
	for _, sensitive := range sensitiveHeaders {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}

// isSensitiveVariable checks if a variable contains sensitive data
func isSensitiveVariable(key string) bool {
	lower := strings.ToLower(key)
	sensitiveVars := []string{
		"token",
		"secret",
		"password",
		"key",
		"credential",
	}
	
	for _, sensitive := range sensitiveVars {
		if strings.Contains(lower, sensitive) {
			return true
		}
	}
	return false
}