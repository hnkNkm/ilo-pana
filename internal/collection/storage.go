package collection

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	dir string
}

// NewFileStorage creates a file-based storage backend. If dir is empty,
// the default directory (~/.ilo-pana/collections) is used.
func NewFileStorage(dir string) *FileStorage {
	if dir == "" {
		dir = defaultCollectionDir()
	}
	return &FileStorage{dir: dir}
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
	return fs.dir
}

// Load loads a collection from file.
func (fs *FileStorage) Load(name string) (*Collection, error) {
	filename := fs.collectionPath(name)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("collection %q not found", name)
		}
		return nil, fmt.Errorf("failed to open collection file: %w", err)
	}
	defer file.Close()

	var c Collection
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&c); err != nil {
		return nil, fmt.Errorf("failed to decode collection: %w", err)
	}
	return &c, nil
}

// Save writes a collection to file (atomic: temp file + rename).
func (fs *FileStorage) Save(c *Collection) error {
	if c.Name == "" {
		return fmt.Errorf("collection name is required")
	}
	if err := fs.ensureDir(); err != nil {
		return err
	}

	filename := fs.collectionPath(c.Name)
	tmpFile := filename + ".tmp"
	file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create collection file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(c); err != nil {
		file.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to encode collection: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close collection file: %w", err)
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to save collection file: %w", err)
	}
	return nil
}

// Delete removes a collection file.
func (fs *FileStorage) Delete(name string) error {
	filename := fs.collectionPath(name)
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("collection %q not found", name)
		}
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

// List returns the names of all collections.
func (fs *FileStorage) List() ([]string, error) {
	if err := fs.ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
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

// collectionPath returns the full path for a collection file.
func (fs *FileStorage) collectionPath(name string) string {
	return filepath.Join(fs.dir, sanitizeName(name)+".json")
}

// ensureDir ensures the storage directory exists.
func (fs *FileStorage) ensureDir() error {
	if err := os.MkdirAll(fs.dir, 0700); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
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

// IsNotExist returns true if the error indicates a collection doesn't exist.
func IsNotExist(err error) bool {
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	return strings.Contains(err.Error(), "not found")
}
