// Package openapi parses OpenAPI 3.x / Swagger 2.0 specifications
// and converts operations into importable request endpoints.
package openapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Endpoint is a single operation converted from a spec.
type Endpoint struct {
	Name        string
	Method      string
	URL         string
	Headers     map[string]string
	QueryParams map[string]string
	Variables   map[string]string
	Body        string
}

// Document is a parsed OpenAPI specification.
type Document struct {
	Title     string
	Endpoints []Endpoint
}

var pathVarPattern = regexp.MustCompile(`\{([^}]+)\}`)

var methodKeys = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

// Parse parses an OpenAPI 3.x or Swagger 2.0 spec (YAML or JSON).
func Parse(data []byte) (*Document, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse spec: %w", err)
	}
	if raw == nil {
		return nil, errors.New("spec is empty")
	}

	version := str(raw["openapi"])
	isV2 := str(raw["swagger"]) == "2.0"
	if version == "" && !isV2 {
		return nil, errors.New("not an OpenAPI spec (missing openapi or swagger version)")
	}

	doc := &Document{}
	if info, ok := raw["info"].(map[string]any); ok {
		doc.Title = str(info["title"])
	}

	base := serverBaseURL(raw, isV2)
	refs := refsOf(raw, isV2)

	usedNames := map[string]int{}
	if paths, ok := raw["paths"].(map[string]any); ok {
		doc.Endpoints = parsePaths(paths, base, refs, isV2, usedNames)
	}

	if doc.Title == "" {
		doc.Title = "imported"
	}
	return doc, nil
}

func parsePaths(paths map[string]any, base string, refs map[string]any, isV2 bool, usedNames map[string]int) []Endpoint {
	var endpoints []Endpoint
	for path, item := range paths {
		pathItem, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pathParams := paramList(pathItem["parameters"])
		for _, method := range methodKeys {
			op, ok := pathItem[method].(map[string]any)
			if !ok {
				continue
			}
			endpoints = append(endpoints, parseOperation(path, method, op, base, refs, isV2, pathParams, usedNames))
		}
	}
	return endpoints
}

func parseOperation(path, method string, op map[string]any, base string, refs map[string]any, isV2 bool, pathParams []map[string]any, usedNames map[string]int) Endpoint {
	ep := Endpoint{
		Method:      strings.ToUpper(method),
		Headers:     map[string]string{},
		QueryParams: map[string]string{},
		Variables:   map[string]string{},
	}

	basePath := path
	if base != "" {
		basePath = strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
	}
	ep.URL = pathVarPattern.ReplaceAllString(basePath, "{{$1}}")
	for _, match := range pathVarPattern.FindAllStringSubmatch(path, -1) {
		ep.Variables[match[1]] = ""
	}

	params := append(append([]map[string]any{}, pathParams...), paramList(op["parameters"])...)
	for _, p := range params {
		name := str(p["name"])
		if name == "" {
			continue
		}
		value := paramValue(p)
		switch str(p["in"]) {
		case "query":
			ep.QueryParams[name] = value
		case "header":
			ep.Headers[name] = value
		case "path":
			ep.Variables[name] = value
		}
	}

	// Query params are appended to the URL so the GUI's Params tab
	// picks them up when the request is loaded.
	if len(ep.QueryParams) > 0 {
		sep := "?"
		for k, v := range ep.QueryParams {
			ep.URL += sep + k + "=" + escapeQueryValue(v)
			sep = "&"
		}
	}

	if schema := requestBodySchema(op, isV2); schema != nil {
		if example := exampleFromSchema(schema, refs, 0); example != nil {
			if body, err := json.MarshalIndent(example, "", "  "); err == nil {
				ep.Body = string(body)
			}
		}
	}

	ep.Name = uniqueName(operationName(op, method, path), usedNames)
	return ep
}

func operationName(op map[string]any, method, path string) string {
	if id := str(op["operationId"]); id != "" {
		return id
	}
	if summary := str(op["summary"]); summary != "" {
		return summary
	}
	return strings.ToUpper(method) + " " + path
}

func uniqueName(base string, used map[string]int) string {
	n := used[base]
	used[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s (%d)", base, n+1)
}

func serverBaseURL(raw map[string]any, isV2 bool) string {
	if isV2 {
		var host, basePath string
		if servers, ok := raw["schemes"].([]any); ok && len(servers) > 0 {
			scheme := str(servers[0])
			host = str(raw["host"])
			if host != "" {
				return scheme + "://" + host + str(raw["basePath"])
			}
			_ = basePath
		}
		return str(raw["basePath"])
	}
	if servers, ok := raw["servers"].([]any); ok && len(servers) > 0 {
		if first, ok := servers[0].(map[string]any); ok {
			return str(first["url"])
		}
	}
	return ""
}

func refsOf(raw map[string]any, isV2 bool) map[string]any {
	key := "components"
	if isV2 {
		key = "definitions"
	}
	if comp, ok := raw[key].(map[string]any); ok {
		if schemas, ok := comp["schemas"].(map[string]any); ok {
			return schemas
		}
		return comp
	}
	return nil
}

func requestBodySchema(op map[string]any, isV2 bool) map[string]any {
	if isV2 {
		for _, p := range paramList(op["parameters"]) {
			if str(p["in"]) == "body" {
				if s, ok := p["schema"].(map[string]any); ok {
					return s
				}
			}
		}
		return nil
	}
	body, ok := op["requestBody"].(map[string]any)
	if !ok {
		return nil
	}
	content, ok := body["content"].(map[string]any)
	if !ok {
		return nil
	}
	// Prefer application/json, fall back to any content type.
	media := pickMedia(content)
	if media == nil {
		return nil
	}
	if s, ok := media["schema"].(map[string]any); ok {
		return s
	}
	return nil
}

func pickMedia(content map[string]any) map[string]any {
	if m, ok := content["application/json"].(map[string]any); ok {
		return m
	}
	for _, m := range content {
		if mm, ok := m.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func paramList(v any) []map[string]any {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, item := range list {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// paramValue extracts an example value for a parameter from
// example/default/enum fields, either on the parameter or its schema.
func paramValue(p map[string]any) string {
	for _, key := range []string{"example", "default", "const"} {
		if v, ok := p[key]; ok && v != nil {
			return scalarString(v)
		}
	}
	if enum, ok := p["enum"].([]any); ok && len(enum) > 0 {
		return scalarString(enum[0])
	}
	if schema, ok := p["schema"].(map[string]any); ok {
		for _, key := range []string{"example", "default", "const"} {
			if v, ok := schema[key]; ok && v != nil {
				return scalarString(v)
			}
		}
		if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
			return scalarString(enum[0])
		}
	}
	return ""
}

// escapeQueryValue escapes characters that would break URL parsing,
// using the same decoding (decodeURIComponent) the GUI applies.
func escapeQueryValue(v string) string {
	replacer := strings.NewReplacer(
		"&", "%26",
		"=", "%3D",
		" ", "%20",
		"+", "%2B",
		"#", "%23",
	)
	return replacer.Replace(v)
}

func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
