package service

import (
	"path/filepath"
	"strings"
	"testing"

	"ilo-pana/internal/collection"
)

const petstoreSpec = `
openapi: 3.0.3
info:
  title: Petstore
servers:
  - url: https://api.example.com/v1
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          required: true
          schema: {type: string}
  /pets:
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        name: {type: string, default: rex}
`

func newOpenAPIImporter(t *testing.T, now Clock) *OpenAPIImporter {
	t.Helper()
	svc := NewCollectionService(
		collection.NewFileStorage(filepath.Join(t.TempDir(), "collections")),
		now,
	)
	return NewOpenAPIImporter(svc, now)
}

func TestOpenAPIImporter_Import(t *testing.T) {
	importer := newOpenAPIImporter(t, fixedClock(clockT1))

	n, err := importer.Import(petstoreSpec, "petstore")
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if n != 2 {
		t.Errorf("imported = %d, want 2", n)
	}

	// The importer goes through CollectionService.LoadOrCreate/Save, so
	// the collection timestamps are stamped by the injected clock.
	c, err := importer.collections.Get("petstore")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !c.Created.Equal(clockT1) || !c.Updated.Equal(clockT1) {
		t.Errorf("collection stamps = %v/%v, want both %v", c.Created, c.Updated, clockT1)
	}

	var get, post *collection.SavedRequest
	for i := range c.Requests {
		switch c.Requests[i].Method {
		case "GET":
			get = &c.Requests[i]
		case "POST":
			post = &c.Requests[i]
		}
	}
	if get == nil || get.URL != "https://api.example.com/v1/pets/{{petId}}" {
		t.Errorf("unexpected imported GET request: %+v", get)
	}
	if get.Variables["petId"] != "" {
		t.Errorf("petId variable not registered: %+v", get.Variables)
	}
	if post == nil || !strings.Contains(post.Body, `"name": "rex"`) {
		t.Errorf("POST body should include generated example: %s", post)
	}
	if !post.Created.Equal(clockT1) || !post.Updated.Equal(clockT1) {
		t.Errorf("request stamps = %v/%v, want %v", post.Created, post.Updated, clockT1)
	}
}

func TestOpenAPIImporter_TitleFallback(t *testing.T) {
	importer := newOpenAPIImporter(t, fixedClock(clockT1))
	if _, err := importer.Import(petstoreSpec, ""); err != nil {
		t.Fatalf("Import() without collection name error = %v", err)
	}
	if _, err := importer.collections.Get("Petstore"); err != nil {
		t.Errorf("collection should be named after spec title: %v", err)
	}
}

func TestOpenAPIImporter_InvalidSpec(t *testing.T) {
	importer := newOpenAPIImporter(t, fixedClock(clockT1))
	if _, err := importer.Import("not: a: spec", "x"); err == nil {
		t.Error("Import() of invalid spec should error")
	}
}

func TestOpenAPIImporter_AppendsToExistingCollection(t *testing.T) {
	importer := newOpenAPIImporter(t, fixedClock(clockT1))
	if err := importer.collections.SaveRequest("petstore", collection.SavedRequest{
		Name:   "existing",
		Method: "PUT",
		URL:    "https://api.example.com/v1/existing",
	}); err != nil {
		t.Fatalf("SaveRequest() error = %v", err)
	}

	if _, err := importer.Import(petstoreSpec, "petstore"); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	c, err := importer.collections.Get("petstore")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(c.Requests) != 3 {
		t.Errorf("requests = %d, want 3 (existing + 2 imported)", len(c.Requests))
	}
	if !c.Created.Equal(clockT1) {
		t.Errorf("Created changed on import into existing collection: %v, want %v", c.Created, clockT1)
	}
}
