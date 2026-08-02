package environment

import (
	"errors"
	"os"
	"path/filepath"

	"ilo-pana/internal/jsonstore"
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
	store *jsonstore.Store[Environment]
}

// NewFileStorage creates a file-based storage backend. If dir is empty,
// the default directory (~/.ilo-pana/environments) is used.
func NewFileStorage(dir string) *FileStorage {
	if dir == "" {
		dir = defaultEnvironmentDir()
	}
	return &FileStorage{store: jsonstore.New[Environment](dir, "environment")}
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
	return fs.store.Dir()
}

// Load loads an environment from file.
func (fs *FileStorage) Load(name string) (*Environment, error) {
	return fs.store.Load(name)
}

// Save writes an environment to file (atomic: temp file + rename).
func (fs *FileStorage) Save(e *Environment) error {
	if e.Name == "" {
		return errors.New("environment name is required")
	}
	return fs.store.Save(e.Name, e)
}

// Delete removes an environment file.
func (fs *FileStorage) Delete(name string) error {
	return fs.store.Delete(name)
}

// List returns the names of all environments.
func (fs *FileStorage) List() ([]string, error) {
	return fs.store.List()
}

// IsNotExist returns true if the error indicates an environment doesn't exist.
func IsNotExist(err error) bool {
	return jsonstore.IsNotFound(err)
}
