// Package curl generates and parses curl command lines.
package curl

import (
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Request is a request representation that maps to/from a curl command.
type Request struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// Generate renders a request as a multi-line curl command.
func Generate(req Request) (string, error) {
	if req.URL == "" {
		return "", errors.New("URL is required")
	}
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = "GET"
	}

	parts := []string{"curl"}
	if method != "GET" {
		parts = append(parts, "-X", quote(method))
	}
	parts = append(parts, quote(req.URL))

	keys := make([]string, 0, len(req.Headers))
	for k := range req.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts = append(parts, "-H", quote(k+": "+req.Headers[k]))
	}

	if req.Body != "" {
		parts = append(parts, "--data-raw", quote(req.Body))
	}

	return strings.Join(parts, " \\\n  "), nil
}

// quote wraps s in single quotes, escaping embedded single quotes the way
// POSIX shells expect ("'\''").
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// parseOptions with a separate value argument.
var flagsWithValue = map[string]bool{
	"-o": true, "--output": true, "-D": true, "--dump-header": true,
	"-F": true, "--form": true, "--form-string": true,
	"-A": true, "--user-agent": true, "-e": true, "--referer": true,
	"--cert": true, "--key": true, "--cacert": true, "--capath": true,
	"--resolve": true, "--connect-timeout": true, "--max-time": true,
	"-w": true, "--write-out": true, "--retry": true, "-x": true, "--proxy": true,
	"--interface": true, "--oauth2-bearer": true, "--trace-ascii": true,
	"-E": true, "--cert-type": true, "--key-type": true, "--pinnedpubkey": true,
	"--proxy-user": true, "--proxy-header": true, "--range": true, "-r": true,
	"-m": true, "--max-filesize": true, "--limit-rate": true,
	"-H": true, "--header": true, "-u": true, "--user": true, "-b": true, "--cookie": true,
}

// Parse parses a curl command line into a Request. Supported flags:
// -X/--request, -H/--header, -d/--data*/--json, -u/--user, -b/--cookie,
// -I/--head, --url. Other flags are ignored.
func Parse(command string) (Request, error) {
	req := Request{Method: "GET", Headers: map[string]string{}}

	tokens, err := tokenize(command)
	if err != nil {
		return req, err
	}

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "curl":
			if i != 0 {
				return req, fmt.Errorf("unexpected argument %q", t)
			}
		case "-X", "--request":
			if i+1 >= len(tokens) {
				return req, fmt.Errorf("%s requires a value", t)
			}
			i++
			req.Method = strings.ToUpper(tokens[i])
		case "-H", "--header":
			if i+1 >= len(tokens) {
				return req, fmt.Errorf("%s requires a value", t)
			}
			i++
			name, value, ok := splitHeader(tokens[i])
			if !ok {
				return req, fmt.Errorf("invalid header %q", tokens[i])
			}
			if value == "" {
				delete(req.Headers, name)
			} else {
				req.Headers[name] = value
			}
		case "-d", "--data", "--data-raw", "--data-ascii", "--data-binary":
			if i+1 >= len(tokens) {
				return req, fmt.Errorf("%s requires a value", t)
			}
			i++
			// curl concatenates repeated -d values with '&'.
			if req.Body != "" {
				req.Body += "&"
			}
			req.Body += tokens[i]
			if req.Method == "GET" {
				req.Method = "POST"
			}
		case "--json":
			if i+1 >= len(tokens) {
				return req, errors.New("--json requires a value")
			}
			i++
			if req.Body != "" {
				req.Body += "&"
			}
			req.Body += tokens[i]
			req.Headers["Content-Type"] = "application/json"
			if req.Method == "GET" {
				req.Method = "POST"
			}
		case "-u", "--user":
			if i+1 >= len(tokens) {
				return req, fmt.Errorf("%s requires a value", t)
			}
			i++
			if strings.Contains(tokens[i], ":") {
				req.Headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(tokens[i]))
			}
		case "-b", "--cookie":
			if i+1 >= len(tokens) {
				return req, fmt.Errorf("%s requires a value", t)
			}
			i++
			req.Headers["Cookie"] = tokens[i]
		case "-I", "--head":
			req.Method = "HEAD"
		case "--url":
			if i+1 >= len(tokens) {
				return req, errors.New("--url requires a value")
			}
			i++
			req.URL = tokens[i]
		default:
			if strings.HasPrefix(t, "-") {
				if flagsWithValue[t] && i+1 < len(tokens) {
					i++
				}
				continue
			}
			if req.URL == "" {
				req.URL = t
			} else {
				return req, fmt.Errorf("unexpected argument %q", t)
			}
		}
	}

	if req.URL == "" {
		return req, errors.New("no URL found in curl command")
	}
	return req, nil
}

// splitHeader splits "Name: value" into name and value.
func splitHeader(h string) (name, value string, ok bool) {
	colon := strings.Index(h, ":")
	if colon < 0 {
		return "", "", false
	}
	name = strings.TrimSpace(h[:colon])
	value = strings.TrimSpace(h[colon+1:])
	return name, value, name != ""
}

// tokenize splits a command line into tokens, honoring single quotes,
// double quotes, and backslash escapes.
func tokenize(s string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	var inSingle, inDouble bool

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				cur.WriteByte(c)
			}
		case inDouble:
			if c == '"' {
				inDouble = false
				continue
			}
			if c == '\\' && i+1 < len(s) {
				next := s[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' {
					cur.WriteByte(next)
					i++
					continue
				}
			}
			cur.WriteByte(c)
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == '\\':
			if i+1 < len(s) && s[i+1] == '\n' {
				// line continuation: backslash-newline is removed
				i++
			} else if i+1 < len(s) && s[i+1] == '\r' && i+2 < len(s) && s[i+2] == '\n' {
				i += 2
			} else if i+1 < len(s) {
				cur.WriteByte(s[i+1])
				i++
			}
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}

	if inSingle || inDouble {
		return nil, errors.New("unterminated quote in curl command")
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens, nil
}
