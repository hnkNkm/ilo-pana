package curl

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    Request
		wantErr bool
	}{
		{
			name:    "simple get",
			command: "curl https://api.example.com/users",
			want:    Request{Method: "GET", URL: "https://api.example.com/users", Headers: map[string]string{}},
		},
		{
			name:    "post with headers and body",
			command: `curl -X POST 'https://api.example.com/users' -H 'Content-Type: application/json' -d '{"name":"alice"}'`,
			want: Request{
				Method:  "POST",
				URL:     "https://api.example.com/users",
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    `{"name":"alice"}`,
			},
		},
		{
			name:    "quoted values with spaces and escapes",
			command: `curl -X POST 'https://example.com' -d 'it'\''s here' -H 'X-Auth: a b'`,
			want: Request{
				Method:  "POST",
				URL:     "https://example.com",
				Headers: map[string]string{"X-Auth": "a b"},
				Body:    "it's here",
			},
		},
		{
			name:    "double quotes",
			command: `curl "https://example.com" -d "hello \"world\""`,
			want:    Request{Method: "POST", URL: "https://example.com", Headers: map[string]string{}, Body: `hello "world"`},
		},
		{
			name:    "json flag sets content type",
			command: `curl --json '{"a":1}' https://example.com`,
			want: Request{
				Method:  "POST",
				URL:     "https://example.com",
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    `{"a":1}`,
			},
		},
		{
			name:    "basic auth",
			command: `curl -u 'alice:secret' https://example.com`,
			want: Request{
				Method:  "GET",
				URL:     "https://example.com",
				Headers: map[string]string{"Authorization": "Basic YWxpY2U6c2VjcmV0"},
			},
		},
		{
			name:    "cookie header",
			command: `curl -b 'session=abc; theme=dark' https://example.com`,
			want: Request{
				Method:  "GET",
				URL:     "https://example.com",
				Headers: map[string]string{"Cookie": "session=abc; theme=dark"},
			},
		},
		{
			name:    "head flag",
			command: `curl -I https://example.com`,
			want:    Request{Method: "HEAD", URL: "https://example.com", Headers: map[string]string{}},
		},
		{
			name:    "ignores unrelated flags",
			command: `curl -sS -L --compressed -o out.json https://example.com -H 'A: 1'`,
			want:    Request{Method: "GET", URL: "https://example.com", Headers: map[string]string{"A": "1"}},
		},
		{
			name:    "url flag",
			command: `curl --url 'https://example.com/x'`,
			want:    Request{Method: "GET", URL: "https://example.com/x", Headers: map[string]string{}},
		},
		{
			name:    "empty header value removes header",
			command: `curl -H 'Accept:' https://example.com`,
			want:    Request{Method: "GET", URL: "https://example.com", Headers: map[string]string{}},
		},
		{
			name:    "multiple data flags concatenate with &",
			command: `curl -d 'a=1' -d 'b=2' https://example.com`,
			want:    Request{Method: "POST", URL: "https://example.com", Headers: map[string]string{}, Body: "a=1&b=2"},
		},
		{
			name:    "multiple json flags concatenate with &",
			command: `curl --json '{"a":1}' --json '{"b":2}' https://example.com`,
			want:    Request{Method: "POST", URL: "https://example.com", Headers: map[string]string{"Content-Type": "application/json"}, Body: `{"a":1}&{"b":2}`},
		},
		{
			name:    "no url",
			command: `curl -X GET`,
			wantErr: true,
		},
		{
			name:    "unterminated quote",
			command: `curl 'https://example.com`,
			wantErr: true,
		},
		{
			name:    "invalid header",
			command: `curl -H 'not-a-header' https://example.com`,
			wantErr: true,
		},
		{
			name:    "multiple urls",
			command: `curl https://a.com https://b.com`,
			wantErr: true,
		},
		{
			name:    "flag missing value",
			command: `curl -X`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	req := Request{
		Method:  "POST",
		URL:     "https://api.example.com/users",
		Headers: map[string]string{"Content-Type": "application/json", "X-Auth": "a b"},
		Body:    `{"name":"it's alice"}`,
	}
	cmd, err := Generate(req)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Round trip: generating then parsing yields the same request.
	got, err := Parse(cmd)
	if err != nil {
		t.Fatalf("Parse(generated) error = %v\ncommand:\n%s", err, cmd)
	}
	if !reflect.DeepEqual(got, req) {
		t.Errorf("round trip mismatch:\n%s\ngot  %+v\nwant %+v", cmd, got, req)
	}

	// Deterministic header order regardless of map order.
	req2 := req
	cmd2, _ := Generate(req2)
	if cmd != cmd2 {
		t.Errorf("Generate() not deterministic:\n%s\n%s", cmd, cmd2)
	}
}

func TestGenerateErrors(t *testing.T) {
	if _, err := Generate(Request{Method: "GET"}); err == nil {
		t.Error("Generate() with empty URL should error")
	}
}

func TestQuote(t *testing.T) {
	if got := quote("a'b"); got != "'a'\\''b'" {
		t.Errorf("quote(a'b) = %q", got)
	}
}

func TestTokenizeComplex(t *testing.T) {
	tokens, err := tokenize(`curl "a b" 'c d' e\ f "g\"h"`)
	if err != nil {
		t.Fatalf("tokenize() error = %v", err)
	}
	want := []string{"curl", "a b", "c d", "e f", `g"h`}
	if !reflect.DeepEqual(tokens, want) {
		t.Errorf("tokenize() = %v, want %v", tokens, want)
	}
	if !strings.Contains(strings.Join(tokens, " "), "e f") {
		t.Error("joined tokens should contain escaped space")
	}
}
