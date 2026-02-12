# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based HTTP API testing CLI tool similar to curl/HTTPie. It supports various HTTP methods, custom headers, request bodies, and provides formatted output with sensitive header masking.

## Build and Development Commands

```bash
# Build the binary
go build -o bin/api-tester ./cmd/api-tester

# Run tests
go test ./...
go test -v ./...                    # Verbose output
go test -race ./...                  # With race detection
go test ./internal/config -v        # Test specific package
go test -run TestValidateURL ./...  # Run specific test

# Install to $GOPATH/bin
go install ./cmd/api-tester

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o bin/api-tester-linux ./cmd/api-tester
GOOS=darwin GOARCH=arm64 go build -o bin/api-tester-darwin ./cmd/api-tester
GOOS=windows GOARCH=amd64 go build -o bin/api-tester.exe ./cmd/api-tester
```

## Architecture

The codebase follows Go best practices with a standard project layout:

- **`cmd/api-tester/`**: Minimal entry point (30 lines) that delegates to internal packages
- **`internal/`**: Core business logic, not importable by external packages
  - **`config/`**: Command-line flag parsing and configuration management. Handles multiple `-H` flags using custom `headerList` type that implements `flag.Value` interface
  - **`client/`**: HTTP client orchestration, coordinates request execution using other packages
  - **`request/`**: Request building, URL validation, HTTP method validation, and JSON content detection
  - **`response/`**: Response processing, sensitive header masking (Authorization, X-API-Key, etc.), and JSON formatting

### Key Design Patterns

1. **Flag handling**: Uses `flag.Var()` with custom `headerList` type to support multiple header flags
2. **Sensitive data protection**: Automatic masking of sensitive headers unless `API_TESTER_VERBOSE=true`
3. **Request/Response flow**: Visual indicators using `→` for requests and `←` for responses
4. **Error handling**: All errors wrapped with context using `fmt.Errorf(...%w...)`

### Important Implementation Details

- The `headerList` type in `internal/config/parser.go` is a `[]string` that implements `flag.Value` interface for accumulating multiple `-H` flags
- URL validation includes security checks for localhost/127.0.0.1 connections (SSRF prevention)
- JSON detection and pretty-printing happens automatically based on content
- Test files use table-driven tests extensively with subtests (`t.Run()`)

## Usage Examples

```bash
# Simple GET
./bin/api-tester https://api.example.com/users

# POST with JSON
./bin/api-tester -X POST -d '{"name":"test"}' https://api.example.com/users

# Multiple headers
./bin/api-tester -H 'Authorization: Bearer token' -H 'Accept: application/json' https://api.example.com

# Show sensitive headers (verbose mode)
API_TESTER_VERBOSE=true ./bin/api-tester -H 'X-API-Key: secret' https://api.example.com
```