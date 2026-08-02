package environment

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStorage(t *testing.T) *FileStorage {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "environments")
	return NewFileStorage(dir)
}

func TestSaveAndLoad(t *testing.T) {
	fs := newTestStorage(t)

	e := &Environment{
		Name:      "dev",
		Variables: map[string]string{"BASE_URL": "http://localhost:8080"},
	}
	if err := fs.Save(e); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := fs.Load("dev")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "dev" {
		t.Errorf("name mismatch: got %q", loaded.Name)
	}
	if loaded.Variables["BASE_URL"] != "http://localhost:8080" {
		t.Errorf("variable mismatch: got %q", loaded.Variables["BASE_URL"])
	}
}

func TestSaveRequiresName(t *testing.T) {
	fs := newTestStorage(t)
	if err := fs.Save(&Environment{Name: ""}); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestLoadNotFound(t *testing.T) {
	fs := newTestStorage(t)
	_, err := fs.Load("nope")
	if err == nil {
		t.Fatal("expected error for missing environment")
	}
	if !IsNotExist(err) {
		t.Fatalf("expected IsNotExist, got: %v", err)
	}
}

func TestList(t *testing.T) {
	fs := newTestStorage(t)

	for _, name := range []string{"dev", "staging", "prod"} {
		if err := fs.Save(&Environment{Name: name}); err != nil {
			t.Fatalf("Save(%q) failed: %v", name, err)
		}
	}

	names, err := fs.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 environments, got %d: %v", len(names), names)
	}
}

func TestListEmpty(t *testing.T) {
	fs := newTestStorage(t)
	names, err := fs.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 0 {
		t.Fatalf("expected empty list, got %v", names)
	}
}

func TestDelete(t *testing.T) {
	fs := newTestStorage(t)
	if err := fs.Save(&Environment{Name: "dev"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := fs.Delete("dev"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := fs.Load("dev"); !IsNotExist(err) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
	if err := fs.Delete("dev"); !IsNotExist(err) {
		t.Fatalf("expected not found on double delete, got: %v", err)
	}
}

func TestSanitizedName(t *testing.T) {
	fs := newTestStorage(t)
	dir := fs.Dir()

	if err := fs.Save(&Environment{Name: "../evil"}); err != nil {
		t.Fatalf("Save with dangerous name failed: %v", err)
	}
	// Must not escape the storage directory
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "evil.json")); err == nil {
		t.Fatal("file escaped storage directory")
	}
	names, err := fs.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 1 || names[0] != "__evil" {
		t.Fatalf("expected sanitized name %q, got %v", "__evil", names)
	}
}

func TestFilePermissions(t *testing.T) {
	fs := newTestStorage(t)
	if err := fs.Save(&Environment{Name: "dev", Variables: map[string]string{"SECRET": "hunter2"}}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	info, err := os.Stat(filepath.Join(fs.Dir(), "dev.json"))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}
