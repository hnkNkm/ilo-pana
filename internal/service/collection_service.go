package service

import (
	"fmt"

	"ilo-pana/internal/collection"
)

// CollectionService manages named collections of saved requests.
type CollectionService struct {
	storage collection.Storage
	now     Clock
}

// NewCollectionService creates a service backed by the given storage.
func NewCollectionService(storage collection.Storage, now Clock) *CollectionService {
	return &CollectionService{storage: storage, now: now}
}

// SaveRequest upserts a request into the named collection, creating it if needed.
func (s *CollectionService) SaveRequest(collectionName string, req collection.SavedRequest) error {
	if collectionName == "" {
		return fmt.Errorf("collection name is required")
	}
	if req.Name == "" {
		return fmt.Errorf("request name is required")
	}
	c, err := s.LoadOrCreate(collectionName)
	if err != nil {
		return err
	}
	now := s.now()
	req.Updated = now
	c.Upsert(req)
	return s.Save(c)
}

// LoadOrCreate loads a collection or creates one (stamped with the current
// time) when it does not exist yet. It is shared by all write paths so the
// load-or-create + timestamp logic lives in exactly one place.
func (s *CollectionService) LoadOrCreate(name string) (*collection.Collection, error) {
	c, err := s.storage.Load(name)
	if err != nil {
		if !collection.IsNotExist(err) {
			return nil, err
		}
		c = &collection.Collection{Name: name, Created: s.now()}
	}
	if c.Created.IsZero() {
		c.Created = s.now()
	}
	return c, nil
}

// Save stamps the collection with the current time and persists it.
func (s *CollectionService) Save(c *collection.Collection) error {
	c.Updated = s.now()
	return s.storage.Save(c)
}

// List returns the names of all collections.
func (s *CollectionService) List() ([]string, error) {
	return s.storage.List()
}

// Get returns a collection by name.
func (s *CollectionService) Get(name string) (*collection.Collection, error) {
	return s.storage.Load(name)
}

// DeleteRequest removes a request from a collection.
func (s *CollectionService) DeleteRequest(collectionName, requestName string) error {
	c, err := s.storage.Load(collectionName)
	if err != nil {
		return err
	}
	if !c.Remove(requestName) {
		return fmt.Errorf("request %q not found in collection %q", requestName, collectionName)
	}
	return s.Save(c)
}

// DeleteCollection removes a collection entirely.
func (s *CollectionService) DeleteCollection(name string) error {
	return s.storage.Delete(name)
}

// Export returns a collection as a JSON string (for copy/paste sharing).
func (s *CollectionService) Export(name string) (string, error) {
	c, err := s.storage.Load(name)
	if err != nil {
		return "", err
	}
	return c.MarshalJSON()
}

// Import parses a JSON collection and saves it, replacing any existing one.
func (s *CollectionService) Import(data string) error {
	c, err := collection.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	if c.Name == "" {
		return fmt.Errorf("collection JSON must include a name")
	}
	return s.storage.Save(c)
}
