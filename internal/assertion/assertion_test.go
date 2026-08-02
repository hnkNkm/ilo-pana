package assertion

import (
	"testing"

	"ilo-pana/internal/response"
)

func testData() *response.ResponseData {
	return &response.ResponseData{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       `{"data": {"items": [{"name": "apple"}, {"name": "banana"}], "count": 2}, "ok": true, "err": null}`,
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name   string
		rule   Rule
		passed bool
	}{
		{"status equals pass", Rule{Kind: KindStatusEquals, Target: "200"}, true},
		{"status equals fail", Rule{Kind: KindStatusEquals, Target: "404"}, false},
		{"status range pass", Rule{Kind: KindStatusRange, Target: "200-299"}, true},
		{"status range fail", Rule{Kind: KindStatusRange, Target: "300-399"}, false},
		{"status 2xx pass", Rule{Kind: KindStatusRange, Target: "2xx"}, true},
		{"status 4xx fail", Rule{Kind: KindStatusRange, Target: "4xx"}, false},
		{"body contains pass", Rule{Kind: KindBodyContains, Target: "banana"}, true},
		{"body contains fail", Rule{Kind: KindBodyContains, Target: "cherry"}, false},
		{"body not contains pass", Rule{Kind: KindBodyNotContains, Target: "cherry"}, true},
		{"body not contains fail", Rule{Kind: KindBodyNotContains, Target: "banana"}, false},
		{"json path exists pass", Rule{Kind: KindJSONPathExists, Target: "data.items[1].name"}, true},
		{"json path exists fail", Rule{Kind: KindJSONPathExists, Target: "data.missing"}, false},
		{"json path equals pass", Rule{Kind: KindJSONPathEquals, Target: "data.count", Expected: "2"}, true},
		{"json path equals fail", Rule{Kind: KindJSONPathEquals, Target: "data.count", Expected: "3"}, false},
		{"json path equals string pass", Rule{Kind: KindJSONPathEquals, Target: "data.items[0].name", Expected: "apple"}, true},
		{"json path contains pass", Rule{Kind: KindJSONPathContains, Target: "data.items[0].name", Expected: "app"}, true},
		{"json path contains fail", Rule{Kind: KindJSONPathContains, Target: "data.items[0].name", Expected: "zzz"}, false},
		{"json path root", Rule{Kind: KindJSONPathExists, Target: "ok"}, true},
		{"json path array element exists", Rule{Kind: KindJSONPathExists, Target: "data.items.1.name"}, true},
		{"unknown kind", Rule{Kind: "bogus"}, false},
		{"invalid status expr", Rule{Kind: KindStatusEquals, Target: "abc"}, false},
		{"invalid status range", Rule{Kind: KindStatusRange, Target: "a-b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Evaluate(testData(), []Rule{tt.rule})
			if len(results) != 1 {
				t.Fatalf("got %d results, want 1", len(results))
			}
			if results[0].Passed != tt.passed {
				t.Errorf("passed = %v, want %v (message: %s)", results[0].Passed, tt.passed, results[0].Message)
			}
		})
	}
}

func TestEvaluateNilResponse(t *testing.T) {
	results := Evaluate(nil, []Rule{{Kind: KindStatusEquals, Target: "200"}})
	if results[0].Passed {
		t.Error("nil response should fail the rule")
	}
}

func TestEvaluateEmptyRules(t *testing.T) {
	if results := Evaluate(testData(), nil); len(results) != 0 {
		t.Errorf("empty rules should return no results, got %d", len(results))
	}
}

func TestEvaluateInvalidJSON(t *testing.T) {
	data := &response.ResponseData{StatusCode: 200, Body: "not json"}
	results := Evaluate(data, []Rule{{Kind: KindJSONPathExists, Target: "a"}})
	if results[0].Passed {
		t.Error("json path on invalid body should fail")
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"a.b.c", []string{"a", "b", "c"}},
		{"a[0].b", []string{"a", "0", "b"}},
		{"a.b[1][2]", []string{"a", "b", "1", "2"}},
		{"$.a.b", []string{"a", "b"}},
		{"$", nil},
		{"", nil},
		{"a..b", []string{"a", "b"}},
		{"data.items.1.name", []string{"data", "items", "1", "name"}},
	}
	for _, tt := range tests {
		got := parsePath(tt.path)
		if len(got) != len(tt.want) {
			t.Errorf("parsePath(%q) = %v, want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parsePath(%q) = %v, want %v", tt.path, got, tt.want)
				break
			}
		}
	}
}
