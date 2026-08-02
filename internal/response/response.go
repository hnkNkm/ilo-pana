// Package response handles HTTP response processing and formatting
package response

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SensitiveHeaders contains header names that should be masked in output
var SensitiveHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"x-auth-token":        true,
	"cookie":              true,
	"x-access-token":      true,
	"x-secret-key":        true,
	"proxy-authorization": true,
}

// ResponseData is a structured response for GUI consumption
type ResponseData struct {
	StatusCode int               `json:"statusCode"`
	Status     string            `json:"status"`
	Headers    map[string]string  `json:"headers"`
	Body       string            `json:"body"`
	ElapsedMs  int64             `json:"elapsedMs"`
	Warning    string            `json:"warning,omitempty"`
}

// Handler processes and displays HTTP responses
type Handler struct {
	Verbose bool
}

// New creates a new response handler
func New(verbose bool) *Handler {
	return &Handler{Verbose: verbose}
}

// MaxResponseSize is the maximum response body size (10MB)
const MaxResponseSize = 10 * 1024 * 1024

// ProcessToStruct processes the HTTP response and returns structured data for GUI
func (h *Handler) ProcessToStruct(resp *http.Response, elapsed time.Duration) (*ResponseData, error) {
	// Collect headers with masking
	headers := make(map[string]string)
	for key, values := range resp.Header {
		masked := make([]string, len(values))
		for i, v := range values {
			masked[i] = h.maskSensitiveValue(key, v)
		}
		headers[key] = strings.Join(masked, ", ")
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize+1)
	respBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := ""
	if len(respBody) > MaxResponseSize {
		// Response exceeds size limit
		actualSize := len(respBody)
		contentType := resp.Header.Get("Content-Type")
		
		if strings.Contains(contentType, "application/json") {
			// For JSON, don't show truncated content as it would be invalid
			bodyStr = fmt.Sprintf(
				"{\n"+
				"  \"error\": \"RESPONSE_TOO_LARGE\",\n"+
				"  \"message\": \"Response size exceeds %d MB limit\",\n"+
				"  \"actual_size_bytes\": %d,\n"+
				"  \"limit_bytes\": %d,\n"+
				"  \"suggestion\": \"Consider using pagination or filtering on the API side\"\n"+
				"}",
				MaxResponseSize/(1024*1024), actualSize, MaxResponseSize)
		} else {
			// For non-JSON, show truncated content with clear markers
			bodyStr = fmt.Sprintf(
				"[TRUNCATED RESPONSE - showing first %d MB of %d bytes total]\n\n%s\n\n[... %d bytes truncated ...]",
				MaxResponseSize/(1024*1024), 
				actualSize,
				string(respBody[:MaxResponseSize]),
				actualSize-MaxResponseSize)
		}
	} else if len(respBody) > 0 {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			bodyStr = formatJSONAsString(string(respBody))
		} else {
			bodyStr = string(respBody)
		}
	}

	return &ResponseData{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       bodyStr,
		ElapsedMs:  elapsed.Milliseconds(),
	}, nil
}

// formatJSONAsString pretty-prints JSON and returns as string
func formatJSONAsString(jsonStr string) string {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(prettyJSON)
}

// Process handles the HTTP response and prints it
func (h *Handler) Process(resp *http.Response, elapsed time.Duration) error {
	// Print status line with timing
	fmt.Printf("← HTTP/%d.%d %s (%.2fs)\n",
		resp.ProtoMajor, resp.ProtoMinor, resp.Status, elapsed.Seconds())

	// Print response headers
	for key, values := range resp.Header {
		for _, value := range values {
			displayValue := h.maskSensitiveValue(key, value)
			fmt.Printf("← %s: %s\n", key, displayValue)
		}
	}
	fmt.Println()

	// Read and print response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if len(respBody) == 0 {
		fmt.Println("← (empty response body)")
		return nil
	}

	// Pretty print based on content type
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := PrettyPrintJSON("", string(respBody)); err != nil {
			// Fall back to raw output if JSON parsing fails
			fmt.Println(string(respBody))
		}
	} else {
		fmt.Println(string(respBody))
	}

	return nil
}

// maskSensitiveValue masks sensitive header values unless verbose mode is enabled
func (h *Handler) maskSensitiveValue(key, value string) string {
	if h.Verbose {
		return value
	}

	lowerKey := strings.ToLower(key)
	if SensitiveHeaders[lowerKey] {
		if len(value) > 10 {
			return value[:6] + "***MASKED***"
		}
		return "***MASKED***"
	}

	return value
}

// PrettyPrintJSON attempts to pretty-print JSON data
func PrettyPrintJSON(prefix, jsonStr string) error {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return err
	}

	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if prefix != "" {
		lines := strings.Split(string(prettyJSON), "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("%s%s\n", prefix, line)
			}
		}
	} else {
		fmt.Println(string(prettyJSON))
	}

	return nil
}

// PrintRequest prints the request details to stdout
func PrintRequest(method, url string, headers http.Header, data string, verbose bool) {
	fmt.Printf("\n→ %s %s\n", method, url)

	handler := &Handler{Verbose: verbose}

	// Print headers with sensitive value masking
	for key, values := range headers {
		for _, value := range values {
			displayValue := handler.maskSensitiveValue(key, value)
			fmt.Printf("→ %s: %s\n", key, displayValue)
		}
	}

	// Print request body if present
	if data != "" {
		fmt.Printf("→\n")
		// Pretty print JSON if possible
		if isJSON(data) {
			PrettyPrintJSON("→ ", data)
		} else {
			fmt.Printf("→ %s\n", data)
		}
	}
	fmt.Println()
}

// isJSON is a helper to detect if a string is JSON
func isJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}