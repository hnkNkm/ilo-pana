package service

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"ilo-pana/internal/environment"
)

// failingEnvironmentStorage implements environment.Storage and fails every call.
type failingEnvironmentStorage struct{ err error }

func (f *failingEnvironmentStorage) Load(name string) (*environment.Environment, error) {
	return nil, f.err
}
func (f *failingEnvironmentStorage) Save(e *environment.Environment) error { return f.err }
func (f *failingEnvironmentStorage) Delete(name string) error              { return f.err }
func (f *failingEnvironmentStorage) List() ([]string, error)               { return nil, f.err }

func newEnvironmentService(t *testing.T, now Clock) *EnvironmentService {
	t.Helper()
	return NewEnvironmentService(
		environment.NewFileStorage(filepath.Join(t.TempDir(), "environments")),
		now,
	)
}

func TestEnvironmentService_StampsAndUpserts(t *testing.T) {
	svc := newEnvironmentService(t, fixedClock(clockT1))

	if err := svc.Save("dev", map[string]string{"BASE": "http://localhost:8080"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	e, err := svc.Load("dev")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !e.Created.Equal(clockT1) || !e.Updated.Equal(clockT1) {
		t.Errorf("stamps = %v/%v, want both %v", e.Created, e.Updated, clockT1)
	}

	// Upsert: Created preserved, variables replaced, Updated advanced.
	svc.now = fixedClock(clockT2)
	if err := svc.Save("dev", map[string]string{"BASE": "http://localhost:9090"}); err != nil {
		t.Fatalf("Save() upsert error = %v", err)
	}
	e, err = svc.Load("dev")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !e.Created.Equal(clockT1) {
		t.Errorf("Created changed on upsert: %v, want %v", e.Created, clockT1)
	}
	if !e.Updated.Equal(clockT2) {
		t.Errorf("Updated = %v, want %v", e.Updated, clockT2)
	}
	if got := e.Variables["BASE"]; got != "http://localhost:9090" {
		t.Errorf("BASE = %q, want http://localhost:9090", got)
	}
}

func TestEnvironmentService_NilVarsNormalized(t *testing.T) {
	svc := newEnvironmentService(t, fixedClock(clockT1))
	if err := svc.Save("dev", nil); err != nil {
		t.Fatalf("Save(nil) error = %v", err)
	}
	e, err := svc.Load("dev")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if e.Variables == nil {
		t.Error("Variables should be an empty (non-nil) map")
	}
}

func TestEnvironmentService_Validation(t *testing.T) {
	svc := newEnvironmentService(t, fixedClock(clockT1))
	if err := svc.Save("", map[string]string{}); err == nil || !strings.Contains(err.Error(), "environment name is required") {
		t.Errorf("empty name error = %v", err)
	}
	if err := svc.Delete("missing"); err == nil {
		t.Error("Delete() on missing environment should error")
	}
}

func TestEnvironmentService_List(t *testing.T) {
	svc := newEnvironmentService(t, fixedClock(clockT1))
	if err := svc.Save("dev", map[string]string{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := svc.Save("prod", map[string]string{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	names, err := svc.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) != 2 {
		t.Errorf("List() = %v, want 2", names)
	}
	if err := svc.Delete("prod"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	names, _ = svc.List()
	if len(names) != 1 || names[0] != "dev" {
		t.Errorf("List() after delete = %v, want [dev]", names)
	}
}

func TestEnvironmentService_StorageErrors(t *testing.T) {
	boom := errors.New("disk on fire")
	svc := NewEnvironmentService(&failingEnvironmentStorage{err: boom}, fixedClock(clockT1))

	if err := svc.Save("dev", map[string]string{}); !errors.Is(err, boom) {
		t.Errorf("Save() error = %v, want %v", err, boom)
	}
	if _, err := svc.Load("dev"); !errors.Is(err, boom) {
		t.Errorf("Load() error = %v, want %v", err, boom)
	}
	if _, err := svc.List(); !errors.Is(err, boom) {
		t.Errorf("List() error = %v, want %v", err, boom)
	}
	if err := svc.Delete("dev"); !errors.Is(err, boom) {
		t.Errorf("Delete() error = %v, want %v", err, boom)
	}
}
