package openapi

import "strings"

// maxRefDepth guards against infinite recursion on recursive $refs.
const maxRefDepth = 6

// exampleFromSchema generates a representative example value from a JSON
// schema. It prefers explicit example/default/enum/const values and falls
// back to structural generation for objects and arrays.
func exampleFromSchema(schema map[string]any, refs map[string]any, depth int) any {
	if schema == nil {
		return nil
	}

	if ref := str(schema["$ref"]); ref != "" {
		if depth >= maxRefDepth {
			return nil
		}
		resolved, ok := resolveRef(ref, refs)
		if !ok {
			return nil
		}
		return exampleFromSchema(resolved, refs, depth+1)
	}

	for _, key := range []string{"example", "const", "default"} {
		if v, ok := schema[key]; ok && v != nil {
			return v
		}
	}
	if enum, ok := schema["enum"].([]any); ok && len(enum) > 0 {
		return enum[0]
	}

	if allOf, ok := schema["allOf"].([]any); ok && len(allOf) > 0 {
		if props := mergeAllOfProperties(allOf, refs, depth); len(props) > 0 {
			return exampleFromSchema(map[string]any{"properties": props}, refs, depth)
		}
	}
	for _, key := range []string{"oneOf", "anyOf"} {
		if subs, ok := schema[key].([]any); ok && len(subs) > 0 {
			if m, ok := subs[0].(map[string]any); ok {
				return exampleFromSchema(m, refs, depth)
			}
		}
	}

	if props, ok := schema["properties"].(map[string]any); ok && len(props) > 0 {
		obj := map[string]any{}
		for name, propSchema := range props {
			if m, ok := propSchema.(map[string]any); ok {
				if v := exampleFromSchema(m, refs, depth); v != nil {
					obj[name] = v
				}
			}
		}
		return obj
	}

	if items, ok := schema["items"].(map[string]any); ok {
		if v := exampleFromSchema(items, refs, depth); v != nil {
			return []any{v}
		}
		return []any{}
	}

	switch str(schema["type"]) {
	case "object":
		return map[string]any{}
	case "array":
		return []any{}
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "null":
		return nil
	case "string":
		switch str(schema["format"]) {
		case "date-time":
			return "2024-01-01T00:00:00Z"
		case "date":
			return "2024-01-01"
		case "email":
			return "user@example.com"
		case "uuid":
			return "00000000-0000-0000-0000-000000000000"
		case "uri", "url":
			return "https://example.com"
		default:
			return "string"
		}
	default:
		return nil
	}
}

// mergeAllOfProperties merges the properties of every allOf member, resolving
// $refs and nested allOf recursively, so the generated example contains the
// fields of all members instead of only the last one.
func mergeAllOfProperties(allOf []any, refs map[string]any, depth int) map[string]any {
	props := map[string]any{}
	for _, sub := range allOf {
		m, ok := sub.(map[string]any)
		if !ok {
			continue
		}
		if ref := str(m["$ref"]); ref != "" {
			if depth >= maxRefDepth {
				continue
			}
			if resolved, ok := resolveRef(ref, refs); ok {
				m = resolved
			}
		}
		if nested, ok := m["allOf"].([]any); ok && len(nested) > 0 {
			for name, propSchema := range mergeAllOfProperties(nested, refs, depth+1) {
				props[name] = propSchema
			}
		}
		if mProps, ok := m["properties"].(map[string]any); ok {
			for name, propSchema := range mProps {
				props[name] = propSchema
			}
		}
	}
	return props
}

// resolveRef resolves a $ref like "#/components/schemas/Pet" against refs.
// Keys are indexed under their full path (e.g. "components/schemas/Pet") so
// parameters and schemas sharing a name cannot collide; the last path segment
// is used as a fallback for backwards compatibility.
func resolveRef(ref string, refs map[string]any) (map[string]any, bool) {
	if full, ok := refs[strings.TrimPrefix(ref, "#/")].(map[string]any); ok {
		return full, true
	}
	name := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		name = ref[i+1:]
	}
	schema, ok := refs[name].(map[string]any)
	return schema, ok
}
