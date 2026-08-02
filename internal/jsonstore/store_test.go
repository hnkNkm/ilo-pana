package jsonstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type item struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func newTestStore(t *testing.T) *Store[item] {
	t.Helper()
	return New[item](filepath.Join(t.TempDir(), "store"), "item")
}

func TestRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save("one", &item{Name: "one", Value: 1}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := s.Load("one")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Value != 1 {
		t.Errorf("Value = %d, want 1", loaded.Value)
	}

	names, err := s.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 1 || names[0] != "one" {
		t.Errorf("List = %v, want [one]", names)
	}
}

func TestErrNotFoundWrapped(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Load("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load error %v does not wrap ErrNotFound", err)
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound(%v) = false, want true", err)
	}

	if err := s.Delete("missing"); !IsNotFound(err) {
		t.Errorf("Delete error = %v, want not-found", err)
	}
	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete error %v does not wrap ErrNotFound", err)
	}
}

func TestAtomicWriteAndFileModes(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save("one", &item{Name: "one"}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	entries, err := os.ReadDir(s.Dir())
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, e := range entries {
		if e.Name() == "one.json.tmp" {
			t.Fatal("temp file left behind after Save")
		}
		if e.Name() != "one.json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			t.Fatalf("Info failed: %v", err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("file mode = %o, want 0600", got)
		}
	}

	dirInfo, err := os.Stat(s.Dir())
	if err != nil {
		t.Fatalf("Stat dir failed: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Errorf("dir mode = %o, want 0700", got)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"plain":     "plain",
		"a/b:c":     "a_b_c",
		"a..b":      "a_b",
		`we"ird*?`:  "we_ird__",
		"~tilde":    "_tilde",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveOverwrites(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save("one", &item{Name: "one", Value: 1}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := s.Save("one", &item{Name: "one", Value: 2}); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}
	loaded, err := s.Load("one")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Value != 2 {
		t.Errorf("Value = %d, want 2 (overwrite)", loaded.Value)
	}
}
