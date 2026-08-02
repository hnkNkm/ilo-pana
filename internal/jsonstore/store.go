// Package jsonstore provides a generic JSON file store with atomic writes,
// sentinel errors, and shared name sanitization. It backs both collection and
// environment persistence.
package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is the sentinel error for missing stored objects. Wrap it with
// %w and test with errors.Is.
var ErrNotFound = errors.New("not found")

// Store persists typed objects as JSON files in a single directory.
type Store[T any] struct {
	dir  string
	kind string // noun for error messages, e.g. "collection"
}

// New creates a store rooted at dir. kind is used in error messages.
func New[T any](dir, kind string) *Store[T] {
	return &Store[T]{dir: dir, kind: kind}
}

// Dir returns the storage directory.
func (s *Store[T]) Dir() string {
	return s.dir
}

// Load loads the object stored under name. It wraps ErrNotFound when the file
// does not exist.
func (s *Store[T]) Load(name string) (*T, error) {
	filename := s.path(name)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s %q: %w", s.kind, name, ErrNotFound)
		}
		return nil, fmt.Errorf("failed to open %s file: %w", s.kind, err)
	}
	defer file.Close()

	var obj T
	if err := json.NewDecoder(file).Decode(&obj); err != nil {
		return nil, fmt.Errorf("failed to decode %s: %w", s.kind, err)
	}
	return &obj, nil
}

// Save writes obj under name atomically (temp file + rename). Files are
// created with 0600 and the directory with 0700.
func (s *Store[T]) Save(name string, obj *T) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	filename := s.path(name)
	tmpFile := filename + ".tmp"
	file, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", s.kind, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(obj); err != nil {
		file.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to encode %s: %w", s.kind, err)
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close %s file: %w", s.kind, err)
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to save %s file: %w", s.kind, err)
	}
	return nil
}

// Delete removes the file stored under name. It wraps ErrNotFound when the
// file does not exist.
func (s *Store[T]) Delete(name string) error {
	filename := s.path(name)
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s %q: %w", s.kind, name, ErrNotFound)
		}
		return fmt.Errorf("failed to delete %s: %w", s.kind, err)
	}
	return nil
}

// List returns the base names of all stored objects, in directory order.
func (s *Store[T]) List() ([]string, error) {
	if err := s.ensureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list %ss: %w", s.kind, err)
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

// path returns the full path for an object file. Sanitized names that collide
// (e.g. "a/b" and "a:b" both become "a_b") silently share one file; callers
// should validate names if that matters to them.
func (s *Store[T]) path(name string) string {
	return filepath.Join(s.dir, SanitizeName(name)+".json")
}

// ensureDir ensures the storage directory exists.
func (s *Store[T]) ensureDir() error {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", s.kind, err)
	}
	return nil
}

// SanitizeName removes path separators and dangerous characters from names.
func SanitizeName(name string) string {
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

// IsNotFound reports whether err is (or wraps) ErrNotFound or is an
// os.IsNotExist error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotFound) || os.IsNotExist(err)
}
