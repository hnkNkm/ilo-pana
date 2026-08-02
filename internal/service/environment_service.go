package service

import (
	"fmt"

	"ilo-pana/internal/environment"
)

// EnvironmentService manages named variable sets ("environments"),
// e.g. dev/staging/prod for the GUI.
type EnvironmentService struct {
	storage environment.Storage
	now     Clock
}

// NewEnvironmentService creates a service backed by the given storage.
func NewEnvironmentService(storage environment.Storage, now Clock) *EnvironmentService {
	return &EnvironmentService{storage: storage, now: now}
}

// Save upserts a named environment with the given variables.
func (s *EnvironmentService) Save(name string, vars map[string]string) error {
	if name == "" {
		return fmt.Errorf("environment name is required")
	}

	e, err := s.storage.Load(name)
	if err != nil {
		if !environment.IsNotExist(err) {
			return err
		}
		e = &environment.Environment{Name: name, Created: s.now()}
	}
	now := s.now()
	if e.Created.IsZero() {
		e.Created = now
	}
	if vars == nil {
		vars = make(map[string]string)
	}
	e.Variables = vars
	e.Updated = now
	return s.storage.Save(e)
}

// List returns the names of all environments.
func (s *EnvironmentService) List() ([]string, error) {
	return s.storage.List()
}

// Load returns an environment by name.
func (s *EnvironmentService) Load(name string) (*environment.Environment, error) {
	return s.storage.Load(name)
}

// Delete removes an environment entirely.
func (s *EnvironmentService) Delete(name string) error {
	return s.storage.Delete(name)
}
