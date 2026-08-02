package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ilo-pana/internal/collection"
)

// failingCollectionStorage implements collection.Storage and fails every call.
type failingCollectionStorage struct{ err error }

func (f *failingCollectionStorage) Load(name string) (*collection.Collection, error) {
	return nil, f.err
}
func (f *failingCollectionStorage) Save(c *collection.Collection) error { return f.err }
func (f *failingCollectionStorage) Delete(name string) error           { return f.err }
func (f *failingCollectionStorage) List() ([]string, error)            { return nil, f.err }

func newCollectionService(t *testing.T, now Clock) *CollectionService {
	t.Helper()
	return NewCollectionService(
		collection.NewFileStorage(filepath.Join(t.TempDir(), "collections")),
		now,
	)
}

func TestCollectionService_StampsTimestamps(t *testing.T) {
	svc := newCollectionService(t, fixedClock(clockT1))

	req := collection.SavedRequest{
		Name:   "get-pikachu",
		Method: "GET",
		URL:    "https://pokeapi.co/api/v2/pokemon/pikachu",
	}
	if err := svc.SaveRequest("pokemon", req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	c, err := svc.Get("pokemon")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !c.Created.Equal(clockT1) || !c.Updated.Equal(clockT1) {
		t.Errorf("collection stamps = %v/%v, want both %v", c.Created, c.Updated, clockT1)
	}
	got, ok := c.Find("get-pikachu")
	if !ok {
		t.Fatal("request not found")
	}
	if !got.Updated.Equal(clockT1) {
		t.Errorf("request Updated = %v, want %v", got.Updated, clockT1)
	}
	if !got.Created.IsZero() {
		t.Errorf("new request Created = %v, want zero (preserved on upsert)", got.Created)
	}
}

func TestCollectionService_UpsertPreservesCreated(t *testing.T) {
	svc := newCollectionService(t, fixedClock(clockT1))
	req := collection.SavedRequest{Name: "get-pikachu", Method: "GET", URL: "u1"}
	if err := svc.SaveRequest("pokemon", req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	// Advance the clock and replace the request.
	svc.now = fixedClock(clockT2)
	req2 := collection.SavedRequest{Name: "get-pikachu", Method: "POST", URL: "u2"}
	if err := svc.SaveRequest("pokemon", req2); err != nil {
		t.Fatalf("SaveRequest() upsert error = %v", err)
	}

	c, err := svc.Get("pokemon")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !c.Created.Equal(clockT1) {
		t.Errorf("collection Created changed on upsert: %v, want %v", c.Created, clockT1)
	}
	if !c.Updated.Equal(clockT2) {
		t.Errorf("collection Updated = %v, want %v", c.Updated, clockT2)
	}
	got, _ := c.Find("get-pikachu")
	if !got.Created.IsZero() {
		t.Errorf("request Created not preserved: %v, want %v (original was zero)", got.Created, time.Time{})
	}
	if !got.Updated.Equal(clockT2) || got.Method != "POST" {
		t.Errorf("request not replaced: updated=%v method=%s", got.Updated, got.Method)
	}
}

func TestCollectionService_LoadOrCreateSharedByWriters(t *testing.T) {
	// SaveRequest and the OpenAPI importer must share one load-or-create
	// path, so a collection created through one writer is never
	// re-created (or its Created reset) by the other.
	svc := newCollectionService(t, fixedClock(clockT1))
	importer := NewOpenAPIImporter(svc, fixedClock(clockT1))

	// Writer 1: the importer creates the collection.
	if _, err := importer.Import(petstoreSpec, "pokemon"); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	c, err := svc.Get("pokemon")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !c.Created.Equal(clockT1) {
		t.Fatalf("Created = %v, want %v", c.Created, clockT1)
	}

	// Writer 2: SaveRequest must load the existing collection.
	svc.now = fixedClock(clockT2)
	if err := svc.SaveRequest("pokemon", collection.SavedRequest{Name: "r1", Method: "GET", URL: "u"}); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	c, err = svc.Get("pokemon")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !c.Created.Equal(clockT1) {
		t.Errorf("SaveRequest re-created collection: Created = %v, want %v", c.Created, clockT1)
	}
	if !c.Updated.Equal(clockT2) {
		t.Errorf("Updated = %v, want %v", c.Updated, clockT2)
	}
	if len(c.Requests) != 3 {
		t.Errorf("requests = %d, want 3 (2 imported + 1 saved)", len(c.Requests))
	}
}

func TestCollectionService_Validation(t *testing.T) {
	svc := newCollectionService(t, fixedClock(clockT1))
	req := collection.SavedRequest{Name: "r1", Method: "GET", URL: "u"}

	if err := svc.SaveRequest("", req); err == nil || !strings.Contains(err.Error(), "collection name is required") {
		t.Errorf("empty collection name error = %v", err)
	}
	if err := svc.SaveRequest("pokemon", collection.SavedRequest{}); err == nil || !strings.Contains(err.Error(), "request name is required") {
		t.Errorf("empty request name error = %v", err)
	}
	if err := svc.DeleteRequest("missing", "r1"); err == nil {
		t.Error("DeleteRequest() on missing collection should error")
	}
	if err := svc.DeleteCollection("missing"); err == nil {
		t.Error("DeleteCollection() on missing collection should error")
	}
}

func TestCollectionService_DeleteRequest(t *testing.T) {
	svc := newCollectionService(t, fixedClock(clockT1))
	req := collection.SavedRequest{Name: "r1", Method: "GET", URL: "u"}
	req2 := collection.SavedRequest{Name: "r2", Method: "GET", URL: "u2"}
	if err := svc.SaveRequest("pokemon", req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}
	if err := svc.SaveRequest("pokemon", req2); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	if err := svc.DeleteRequest("pokemon", "r1"); err != nil {
		t.Fatalf("DeleteRequest() error = %v", err)
	}
	c, _ := svc.Get("pokemon")
	if _, ok := c.Find("r1"); ok {
		t.Error("r1 should be deleted")
	}
	if !c.Updated.Equal(clockT1) {
		t.Errorf("Updated after delete = %v, want %v", c.Updated, clockT1)
	}
	if err := svc.DeleteRequest("pokemon", "nope"); err == nil || !strings.Contains(err.Error(), `request "nope" not found`) {
		t.Errorf("DeleteRequest() missing error = %v", err)
	}
}

func TestCollectionService_ExportImport(t *testing.T) {
	svc := newCollectionService(t, fixedClock(clockT1))
	req := collection.SavedRequest{Name: "r1", Method: "GET", URL: "u"}
	if err := svc.SaveRequest("pokemon", req); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	exported, err := svc.Export("pokemon")
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if err := svc.DeleteCollection("pokemon"); err != nil {
		t.Fatalf("DeleteCollection() error = %v", err)
	}
	if err := svc.Import(exported); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	c, err := svc.Get("pokemon")
	if err != nil {
		t.Fatalf("Get() after import error = %v", err)
	}
	if len(c.Requests) != 1 {
		t.Errorf("requests after import = %d, want 1", len(c.Requests))
	}
	if !c.Created.Equal(clockT1) {
		t.Errorf("Created after import = %v, want %v", c.Created, clockT1)
	}
	if err := svc.Import("{bad json"); err == nil {
		t.Error("Import() of invalid JSON should error")
	}
	if err := svc.Import(`{"requests": []}`); err == nil {
		t.Error("Import() without name should error")
	}
}

func TestCollectionService_StorageErrors(t *testing.T) {
	boom := errors.New("disk on fire")
	svc := NewCollectionService(&failingCollectionStorage{err: boom}, fixedClock(clockT1))

	if err := svc.SaveRequest("pokemon", collection.SavedRequest{Name: "r1", Method: "GET", URL: "u"}); !errors.Is(err, boom) {
		t.Errorf("SaveRequest() error = %v, want %v", err, boom)
	}
	if _, err := svc.Get("pokemon"); !errors.Is(err, boom) {
		t.Errorf("Get() error = %v, want %v", err, boom)
	}
	if _, err := svc.List(); !errors.Is(err, boom) {
		t.Errorf("List() error = %v, want %v", err, boom)
	}
	if err := svc.DeleteCollection("pokemon"); !errors.Is(err, boom) {
		t.Errorf("DeleteCollection() error = %v, want %v", err, boom)
	}
	if err := svc.Import(`{"name": "x"}`); !errors.Is(err, boom) {
		t.Errorf("Import() error = %v, want %v", err, boom)
	}
}
