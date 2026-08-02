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
		merged := map[string]any{}
		for _, sub := range allOf {
			if m, ok := sub.(map[string]any); ok {
				if props, ok := m["properties"].(map[string]any); ok {
					merged["properties"] = props
				}
			}
		}
		if len(merged) > 0 {
			return exampleFromSchema(merged, refs, depth)
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

// resolveRef resolves a $ref like "#/components/schemas/Pet" against refs.
func resolveRef(ref string, refs map[string]any) (map[string]any, bool) {
	name := ref
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		name = ref[i+1:]
	}
	schema, ok := refs[name].(map[string]any)
	return schema, ok
}
