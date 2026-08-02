package collection

import (
	"errors"
	"os"
	"path/filepath"

	"ilo-pana/internal/jsonstore"
)

// Storage defines the interface for collection persistence.
type Storage interface {
	Load(name string) (*Collection, error)
	Save(c *Collection) error
	Delete(name string) error
	List() ([]string, error)
}

// FileStorage implements file-based collection storage.
type FileStorage struct {
	store *jsonstore.Store[Collection]
}

// NewFileStorage creates a file-based storage backend. If dir is empty,
// the default directory (~/.ilo-pana/collections) is used.
func NewFileStorage(dir string) *FileStorage {
	if dir == "" {
		dir = defaultCollectionDir()
	}
	return &FileStorage{store: jsonstore.New[Collection](dir, "collection")}
}

// defaultCollectionDir returns the default collection directory.
func defaultCollectionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ilo-pana/collections"
	}
	return filepath.Join(home, ".ilo-pana", "collections")
}

// Dir returns the storage directory (useful for tests and reporting).
func (fs *FileStorage) Dir() string {
	return fs.store.Dir()
}

// Load loads a collection from file.
func (fs *FileStorage) Load(name string) (*Collection, error) {
	return fs.store.Load(name)
}

// Save writes a collection to file (atomic: temp file + rename).
func (fs *FileStorage) Save(c *Collection) error {
	if c.Name == "" {
		return errors.New("collection name is required")
	}
	return fs.store.Save(c.Name, c)
}

// Delete removes a collection file.
func (fs *FileStorage) Delete(name string) error {
	return fs.store.Delete(name)
}

// List returns the names of all collections.
func (fs *FileStorage) List() ([]string, error) {
	return fs.store.List()
}

// IsNotExist returns true if the error indicates a collection doesn't exist.
func IsNotExist(err error) bool {
	return jsonstore.IsNotFound(err)
}
