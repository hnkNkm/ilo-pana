package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Storage defines the interface for environment persistence.
type Storage interface {
	Load(name string) (*Environment, error)
	Save(e *Environment) error
	Delete(name string) error
	List() ([]string, error)
}

// FileStorage implements file-based environment storage.
type FileStorage struct {
	dir string
}

// NewFileStorage creates a file-based storage backend. If dir is empty,
// the default directory (~/.ilo-pana/environments) is used.
func NewFileStorage(dir string) *FileStorage {
	if dir == "" {
		dir = defaultEnvironmentDir()
	}
	return &FileStorage{dir: dir}
}

// defaultEnvironmentDir returns the default environment directory.
func defaultEnvironmentDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ilo-pana/environments"
	}
	return filepath.Join(home, ".ilo-pana", "environments")
}

// Dir returns the storage directory (useful for tests and reporting).
func (fs *FileStorage) Dir() string {
	return fs.dir
}

// Load loads an environment from file.
func (fs *FileStorage) Load(name string) (*Environment, error) {
	filename := fs.environmentPath(name)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("environment %q not found", name)
		}
		return nil, fmt.Errorf("failed to open environment file: %w", err)
	}
	defer file.Close()

	var e Environment
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&e); err != nil {
		return nil, fmt.Errorf("failed to decode environment: %w", err)
	}
	return &e, nil
}

// Save writes an environment to file (atomic: temp file + rename).
func (fs *FileStorage) Save(e *Environment) error {
	if e.Name == "" {
		return fmt.Errorf("environment name is required")
	}
	if err := fs.ensureDir(); err != nil {
		return err
	}

	filename := fs.environmentPath(e.Name)
	tmpFile := filename + ".tmp"
	file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create environment file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(e); err != nil {
		file.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to encode environment: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close environment file: %w", err)
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to save environment file: %w", err)
	}
	return nil
}

// Delete removes an environment file.
func (fs *FileStorage) Delete(name string) error {
	filename := fs.environmentPath(name)
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("environment %q not found", name)
		}
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	return nil
}

// List returns the names of all environments.
func (fs *FileStorage) List() ([]string, error) {
	if err := fs.ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	return names, nil
}

// environmentPath returns the full path for an environment file.
func (fs *FileStorage) environmentPath(name string) string {
	return filepath.Join(fs.dir, sanitizeName(name)+".json")
}

// ensureDir ensures the storage directory exists.
func (fs *FileStorage) ensureDir() error {
	if err := os.MkdirAll(fs.dir, 0700); err != nil {
		return fmt.Errorf("failed to create environment directory: %w", err)
	}
	return nil
}

// sanitizeName removes path separators and dangerous characters from names.
func sanitizeName(name string) string {
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

// IsNotExist returns true if the error indicates an environment doesn't exist.
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}
