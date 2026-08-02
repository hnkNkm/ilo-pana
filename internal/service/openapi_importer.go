package service

import (
	"fmt"

	"ilo-pana/internal/collection"
	"ilo-pana/internal/openapi"
)

// OpenAPIImporter parses OpenAPI (YAML/JSON) specs into collections.
type OpenAPIImporter struct {
	collections *CollectionService
	now         Clock
}

// NewOpenAPIImporter creates an importer that saves into the given
// collection service.
func NewOpenAPIImporter(collections *CollectionService, now Clock) *OpenAPIImporter {
	return &OpenAPIImporter{collections: collections, now: now}
}

// Import parses an OpenAPI spec and imports its operations into the named
// collection (default: the spec's title), returning the number of imported
// endpoints.
func (i *OpenAPIImporter) Import(content, collectionName string) (int, error) {
	doc, err := openapi.Parse([]byte(content))
	if err != nil {
		return 0, err
	}
	if collectionName == "" {
		collectionName = doc.Title
	}
	if collectionName == "" {
		return 0, fmt.Errorf("collection name is required")
	}

	c, err := i.collections.LoadOrCreate(collectionName)
	if err != nil {
		return 0, err
	}
	now := i.now()
	for _, ep := range doc.Endpoints {
		c.Upsert(collection.SavedRequest{
			Name:      ep.Name,
			Method:    ep.Method,
			URL:       ep.URL,
			Headers:   ep.Headers,
			Body:      ep.Body,
			Variables: ep.Variables,
			Created:   now,
			Updated:   now,
		})
	}
	if err := i.collections.Save(c); err != nil {
		return 0, err
	}
	return len(doc.Endpoints), nil
}
