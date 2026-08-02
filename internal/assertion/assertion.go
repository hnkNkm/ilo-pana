// Package assertion evaluates validation rules against HTTP responses.
package assertion

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ilo-pana/internal/response"
)

// Kind identifies the type of an assertion rule.
type Kind string

const (
	KindStatusEquals     Kind = "status_equals"
	KindStatusRange      Kind = "status_range"     // e.g. "200-299" or "2xx"
	KindBodyContains     Kind = "body_contains"
	KindBodyNotContains  Kind = "body_not_contains"
	KindJSONPathExists   Kind = "json_path_exists"
	KindJSONPathEquals   Kind = "json_path_equals"
	KindJSONPathContains Kind = "json_path_contains"
)

// Rule is a single assertion configured by the user.
type Rule struct {
	Name     string `json:"name"`
	Kind     Kind   `json:"kind"`
	Target   string `json:"target,omitempty"`   // JSON path or status expression
	Expected string `json:"expected,omitempty"` // expected value for equals/contains rules
}

// Result is the outcome of evaluating one rule.
type Result struct {
	Rule    Rule   `json:"rule"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// Evaluate runs every rule against the response and returns the results
// in rule order. A nil response fails all rules.
func Evaluate(data *response.ResponseData, rules []Rule) []Result {
	results := make([]Result, 0, len(rules))
	for _, rule := range rules {
		results = append(results, evaluateOne(data, rule))
	}
	return results
}

func evaluateOne(data *response.ResponseData, rule Rule) Result {
	result := Result{Rule: rule}
	if data == nil {
		result.Message = "no response to evaluate"
		return result
	}

	switch rule.Kind {
	case KindStatusEquals:
		want, err := strconv.Atoi(strings.TrimSpace(rule.Target))
		if err != nil {
			result.Message = fmt.Sprintf("invalid status expression %q", rule.Target)
			return result
		}
		result.Passed = data.StatusCode == want
		result.Message = fmt.Sprintf("status %d %s %d", data.StatusCode, passWord(result.Passed), want)
	case KindStatusRange:
		ok, msg := statusInRange(data.StatusCode, strings.TrimSpace(rule.Target))
		result.Passed, result.Message = ok, msg
	case KindBodyContains:
		result.Passed = strings.Contains(data.Body, rule.Target)
		result.Message = fmt.Sprintf("body %s contain %q", passWord(result.Passed), rule.Target)
	case KindBodyNotContains:
		result.Passed = !strings.Contains(data.Body, rule.Target)
		result.Message = fmt.Sprintf("body %s not contain %q", passWord(result.Passed), rule.Target)
	case KindJSONPathExists, KindJSONPathEquals, KindJSONPathContains:
		value, found := resolveJSONPath(data.Body, rule.Target)
		if !found {
			result.Message = fmt.Sprintf("JSON path %q not found", rule.Target)
			return result
		}
		switch rule.Kind {
		case KindJSONPathExists:
			result.Passed = true
			result.Message = fmt.Sprintf("JSON path %q exists", rule.Target)
		case KindJSONPathEquals:
			result.Passed = stringify(value) == rule.Expected
			result.Message = fmt.Sprintf("JSON path %q = %s (expected %q)", rule.Target, stringify(value), rule.Expected)
		case KindJSONPathContains:
			result.Passed = strings.Contains(stringify(value), rule.Expected)
			result.Message = fmt.Sprintf("JSON path %q contains %q", rule.Target, rule.Expected)
		}
	default:
		result.Message = fmt.Sprintf("unknown assertion kind %q", rule.Kind)
	}
	return result
}

func passWord(passed bool) string {
	if passed {
		return "does"
	}
	return "does not"
}

// statusInRange checks a status code against "200-299", "2xx", or "200" style
// expressions. The first form is a closed range, "xx" matches any code
// starting with the given digit, and a bare number matches exactly.
func statusInRange(status int, expr string) (bool, string) {
	if i := strings.Index(expr, "-"); i > 0 {
		lo, err1 := strconv.Atoi(strings.TrimSpace(expr[:i]))
		hi, err2 := strconv.Atoi(strings.TrimSpace(expr[i+1:]))
		if err1 != nil || err2 != nil {
			return false, fmt.Sprintf("invalid status range %q", expr)
		}
		ok := status >= lo && status <= hi
		return ok, fmt.Sprintf("status %d %s in %d-%d", status, passWord(ok), lo, hi)
	}
	if strings.HasSuffix(expr, "xx") {
		digit := strings.TrimSuffix(expr, "xx")
		if len(digit) != 1 || digit < "0" || digit > "9" {
			return false, fmt.Sprintf("invalid status pattern %q", expr)
		}
		ok := strings.HasPrefix(strconv.Itoa(status), digit)
		return ok, fmt.Sprintf("status %d %s match %s", status, passWord(ok), expr)
	}
	want, err := strconv.Atoi(expr)
	if err != nil {
		return false, fmt.Sprintf("invalid status expression %q", expr)
	}
	ok := status == want
	return ok, fmt.Sprintf("status %d %s %d", status, passWord(ok), want)
}

// resolveJSONPath extracts a value from a JSON body using a dotted path that
// supports array indexes, e.g. "data.items[0].name" or "data.0.name".
func resolveJSONPath(body, path string) (any, bool) {
	var doc any
	if err := json.Unmarshal([]byte(body), &doc); err != nil {
		return nil, false
	}
	segments := parsePath(path)
	var cur any = doc
	for _, seg := range segments {
		switch t := cur.(type) {
		case map[string]any:
			next, ok := t[seg]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(t) {
				return nil, false
			}
			cur = t[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// parsePath splits "a.b[0].c" into ["a", "b", "0", "c"].
func parsePath(path string) []string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$.")
	path = strings.TrimPrefix(path, "$")
	if path == "" {
		return nil
	}

	var segments []string
	var cur strings.Builder
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '.':
			if cur.Len() > 0 {
				segments = append(segments, cur.String())
				cur.Reset()
			}
		case '[':
			if cur.Len() > 0 {
				segments = append(segments, cur.String())
				cur.Reset()
			}
			if j := strings.IndexByte(path[i:], ']'); j >= 0 {
				if inner := strings.TrimSpace(path[i+1 : i+j]); inner != "" {
					segments = append(segments, inner)
				}
				i += j
			}
		case ']':
			// skip stray closing brackets
		default:
			cur.WriteByte(path[i])
		}
	}
	if cur.Len() > 0 {
		segments = append(segments, cur.String())
	}
	return segments
}

// stringify renders a JSON value for comparison: strings verbatim,
// everything else as compact JSON.
func stringify(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return fmt.Sprintf("%v", v)
}
