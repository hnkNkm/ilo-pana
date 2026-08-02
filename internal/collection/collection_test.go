package collection

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "collections")
	return dir
}

func testRequest(name string) SavedRequest {
	return SavedRequest{
		Name:    name,
		Method:  "GET",
		URL:     "https://example.com/" + name,
		Headers: map[string]string{"Accept": "application/json"},
		Updated: time.Now(),
	}
}

func TestUpsertAddsAndReplaces(t *testing.T) {
	c := &Collection{Name: "api"}
	c.Upsert(testRequest("one"))
	c.Upsert(testRequest("two"))
	c.Upsert(testRequest("one"))

	if len(c.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(c.Requests))
	}
	if _, ok := c.Find("one"); !ok {
		t.Fatal("expected to find request 'one'")
	}
	if _, ok := c.Find("missing"); ok {
		t.Fatal("did not expect to find request 'missing'")
	}
}

func TestRemove(t *testing.T) {
	c := &Collection{Name: "api"}
	c.Upsert(testRequest("one"))

	if !c.Remove("one") {
		t.Fatal("expected Remove to return true for existing request")
	}
	if len(c.Requests) != 0 {
		t.Fatalf("expected 0 requests, got %d", len(c.Requests))
	}
	if c.Remove("one") {
		t.Fatal("expected Remove to return false for missing request")
	}
}

func TestUpsertPreservesCreated(t *testing.T) {
	c := &Collection{Name: "api"}
	req := testRequest("one")
	req.Created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c.Upsert(req)

	req.Updated = time.Now()
	c.Upsert(req)

	got, _ := c.Find("one")
	if !got.Created.Equal(req.Created) {
		t.Fatalf("expected Created to be preserved, got %v", got.Created)
	}
}

func TestFileStorageRoundTrip(t *testing.T) {
	fs := NewFileStorage(tempDir(t))
	c := &Collection{Name: "api", Created: time.Now()}
	c.Upsert(testRequest("one"))
	c.Upsert(testRequest("two"))

	if err := fs.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := fs.Load("api")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Name != "api" {
		t.Fatalf("expected name 'api', got %q", loaded.Name)
	}
	if len(loaded.Requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(loaded.Requests))
	}

	names, err := fs.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 1 || names[0] != "api" {
		t.Fatalf("expected [api], got %v", names)
	}
}

func TestFileStorageDelete(t *testing.T) {
	fs := NewFileStorage(tempDir(t))
	c := &Collection{Name: "api"}
	if err := fs.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := fs.Delete("api"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := fs.Load("api"); !IsNotExist(err) {
		t.Fatalf("expected not-exist error after delete, got %v", err)
	}
}

func TestFileStorageSanitizesNames(t *testing.T) {
	fs := NewFileStorage(tempDir(t))
	c := &Collection{Name: "a/b:c"}
	if err := fs.Save(c); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	files, err := os.ReadDir(fs.Dir())
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name() != "a_b_c.json" {
		t.Fatalf("expected sanitized filename a_b_c.json, got %s", files[0].Name())
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	c := &Collection{Name: "api", Created: time.Now()}
	c.Upsert(testRequest("one"))

	json, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	back, err := UnmarshalJSON(json)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}
	if back.Name != "api" {
		t.Fatalf("expected name 'api', got %q", back.Name)
	}
	if len(back.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(back.Requests))
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	if _, err := UnmarshalJSON("{not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
