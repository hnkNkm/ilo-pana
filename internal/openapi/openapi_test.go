package openapi

import (
	"strings"
	"testing"
)

const v3YAML = `
openapi: 3.0.3
info:
  title: Petstore
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
          example: 20
        - name: sort
          in: query
          schema:
            type: string
        - name: X-Trace
          in: header
          schema:
            type: string
          example: abc
      responses:
        '200':
          description: ok
    post:
      operationId: createPet
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: created
  /pets/{petId}:
    get:
      summary: Find pet by ID
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: ok
components:
  schemas:
    Pet:
      type: object
      required: [name]
      properties:
        id:
          type: integer
        name:
          type: string
          default: rex
        tags:
          type: array
          items:
            type: string
`

const v3JSON = `{
  "openapi": "3.1.0",
  "info": { "title": "Echo", "version": "1.0.0" },
  "paths": {
    "/ping": {
      "get": {
        "operationId": "ping",
        "requestBody": {
          "content": { "application/json": { "schema": { "type": "object", "properties": { "echo": { "type": "string", "format": "date-time" } } } } }
        }
      }
    }
  }
}`

const v2YAML = `
swagger: '2.0'
info:
  title: Old API
  version: 1.0.0
schemes: [https]
host: old.example.com
basePath: /v2
paths:
  /users:
    post:
      operationId: createUser
      parameters:
        - name: user
          in: body
          schema:
            $ref: '#/definitions/User'
definitions:
  User:
    type: object
    properties:
      email:
        type: string
        format: email
      age:
        type: integer
`

func TestParseV3YAML(t *testing.T) {
	doc, err := Parse([]byte(v3YAML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if doc.Title != "Petstore" {
		t.Errorf("Title = %q, want Petstore", doc.Title)
	}
	if len(doc.Endpoints) != 3 {
		t.Fatalf("got %d endpoints, want 3", len(doc.Endpoints))
	}

	list := doc.Endpoints[0]
	if list.Name != "listPets" || list.Method != "GET" {
		t.Errorf("unexpected first endpoint: %+v", list)
	}
	if list.URL != "https://api.example.com/v1/pets?limit=20&sort=" {
		t.Errorf("URL = %q", list.URL)
	}
	if list.Headers["X-Trace"] != "abc" {
		t.Errorf("header X-Trace = %q, want abc", list.Headers["X-Trace"])
	}
	if list.Body != "" {
		t.Errorf("GET should have no body, got %q", list.Body)
	}

	create := doc.Endpoints[1]
	if !strings.Contains(create.Body, `"name": "rex"`) {
		t.Errorf("createPet body should contain default name: %s", create.Body)
	}
	if !strings.Contains(create.Body, `"tags"`) || !strings.Contains(create.Body, `"string"`) {
		t.Errorf("createPet body should contain array example: %s", create.Body)
	}

	byID := doc.Endpoints[2]
	if byID.Name != "Find pet by ID" {
		t.Errorf("Name = %q, want summary fallback", byID.Name)
	}
	if byID.URL != "https://api.example.com/v1/pets/{{petId}}" {
		t.Errorf("URL = %q", byID.URL)
	}
	if _, ok := byID.Variables["petId"]; !ok {
		t.Errorf("petId should be registered as a variable")
	}
}

func TestParseV3JSON(t *testing.T) {
	doc, err := Parse([]byte(v3JSON))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(doc.Endpoints) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(doc.Endpoints))
	}
	ping := doc.Endpoints[0]
	if ping.URL != "/ping" {
		t.Errorf("URL = %q, want /ping (no servers)", ping.URL)
	}
	if !strings.Contains(ping.Body, `"2024-01-01T00:00:00Z"`) {
		t.Errorf("date-time format not generated: %s", ping.Body)
	}
}

func TestParseSwagger2(t *testing.T) {
	doc, err := Parse([]byte(v2YAML))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	create := doc.Endpoints[0]
	if create.URL != "https://old.example.com/v2/users" {
		t.Errorf("URL = %q, want https://old.example.com/v2/users", create.URL)
	}
	if !strings.Contains(create.Body, `"email": "user@example.com"`) {
		t.Errorf("email format not generated: %s", create.Body)
	}
	if !strings.Contains(create.Body, `"age": 0`) {
		t.Errorf("integer example not generated: %s", create.Body)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("foo: bar")); err == nil {
		t.Error("expected error for non-OpenAPI YAML")
	}
	if _, err := Parse([]byte("{{{")); err == nil {
		t.Error("expected error for invalid YAML")
	}
	if _, err := Parse([]byte("")); err == nil {
		t.Error("expected error for empty input")
	}
}

func TestExampleFromSchema(t *testing.T) {
	refs := map[string]any{
		"Node": map[string]any{
			"type":       "object",
			"properties": map[string]any{"child": map[string]any{"$ref": "#/components/schemas/Node"}},
		},
	}
	// Recursive schema must terminate.
	v := exampleFromSchema(refs["Node"].(map[string]any), refs, 0)
	if _, ok := v.(map[string]any); !ok {
		t.Errorf("recursive schema should yield an object, got %#v", v)
	}

	enum := exampleFromSchema(map[string]any{"type": "string", "enum": []any{"a", "b"}}, nil, 0)
	if enum != "a" {
		t.Errorf("enum example = %#v, want a", enum)
	}

	flag := exampleFromSchema(map[string]any{"type": "boolean"}, nil, 0)
	if flag != false {
		t.Errorf("boolean example = %#v, want false", flag)
	}
}

func TestExampleFromSchemaAllOfMergesAllMembers(t *testing.T) {
	refs := map[string]any{
		"Base": map[string]any{
			"type":       "object",
			"properties": map[string]any{"id": map[string]any{"type": "integer"}},
		},
	}
	schema := map[string]any{
		"allOf": []any{
			map[string]any{"$ref": "#/components/schemas/Base"},
			map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
			},
			map[string]any{
				"allOf": []any{
					map[string]any{"properties": map[string]any{"nested": map[string]any{"type": "boolean"}}},
				},
			},
		},
	}
	v := exampleFromSchema(schema, refs, 0).(map[string]any)
	for _, want := range []string{"id", "name", "nested"} {
		if _, ok := v[want]; !ok {
			t.Errorf("allOf merge missing property %q, got %#v", want, v)
		}
	}
}

func TestUniqueNames(t *testing.T) {
	used := map[string]int{}
	if n := uniqueName("dup", used); n != "dup" {
		t.Errorf("first = %q", n)
	}
	if n := uniqueName("dup", used); n != "dup (2)" {
		t.Errorf("second = %q", n)
	}
	if n := uniqueName("dup", used); n != "dup (3)" {
		t.Errorf("third = %q", n)
	}
}

func TestParsePathLevelParams(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: T}
paths:
  /items:
    parameters:
      - name: apiKey
        in: query
        schema: {type: string}
        example: secret
    get:
      operationId: getItems
`
	doc, err := Parse([]byte(spec))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !strings.Contains(doc.Endpoints[0].URL, "apiKey=secret") {
		t.Errorf("path-level param not applied: %q", doc.Endpoints[0].URL)
	}
}

func TestParseIsDeterministic(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: T}
paths:
  /z:
    get: {operationId: op}
    post: {operationId: op}
  /a:
    get: {operationId: op}
  /m:
    get:
      operationId: list
      parameters:
        - {name: b, in: query, example: "50%"}
        - {name: a, in: query, example: "x+y #"}
        - {name: c, in: query, example: "hello world"}
`
	first, err := Parse([]byte(spec))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	second, err := Parse([]byte(spec))
	if err != nil {
		t.Fatalf("second Parse failed: %v", err)
	}

	if len(first.Endpoints) != 4 {
		t.Fatalf("got %d endpoints, want 4", len(first.Endpoints))
	}
	// Paths and operations must come out in stable order: sorted path keys,
	// then methods in methodKeys order (get before post).
	gotNames := []string{}
	for _, ep := range first.Endpoints {
		gotNames = append(gotNames, ep.Name)
	}
	wantNames := []string{"op", "list", "op (2)", "op (3)"}
	for i, want := range wantNames {
		if gotNames[i] != want {
			t.Errorf("endpoint %d name = %q, want %q (order must be stable)", i, gotNames[i], want)
		}
	}

	for i := range first.Endpoints {
		a, b := first.Endpoints[i], second.Endpoints[i]
		if a.URL != b.URL || a.Name != b.Name || a.Body != b.Body {
			t.Errorf("run mismatch at %d: %+v vs %+v", i, a, b)
		}
	}

	// Reserved characters must be encoded so decodeURIComponent survives.
	url := first.Endpoints[1].URL
	if !strings.Contains(url, "a=x%2By%20%23") {
		t.Errorf("URL = %q, want %% -encoded '+'/space/'#'", url)
	}
	if !strings.Contains(url, "b=50%25") {
		t.Errorf("URL = %q, want %% -encoded '%%'", url)
	}
}
