# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **ilo-pana**, a Go-based HTTP API testing CLI tool similar to curl/HTTPie. The name "ilo-pana" means "good tool" in Toki Pona. It supports various HTTP methods, custom headers, request bodies, and provides formatted output with sensitive header masking.

**Module name**: `ilo-pana` (standalone CLI, not using github.com path)

## Build and Development Commands

```bash
# Build the binary (ilo-pana is the new name)
go build -o bin/ilo-pana ./cmd/ilo-pana

# Run tests
go test ./...
go test -v ./...                    # Verbose output
go test -race ./...                  # With race detection
go test ./internal/config -v        # Test specific package
go test -run TestValidateURL ./...  # Run specific test

# Install to $GOPATH/bin
go install ./cmd/ilo-pana

# Release build with GoReleaser (creates multi-platform binaries)
goreleaser build --snapshot --clean

# Cross-compilation manual
GOOS=linux GOARCH=amd64 go build -o bin/ilo-pana-linux ./cmd/ilo-pana
GOOS=darwin GOARCH=arm64 go build -o bin/ilo-pana-darwin ./cmd/ilo-pana
GOOS=windows GOARCH=amd64 go build -o bin/ilo-pana.exe ./cmd/ilo-pana
```

## Architecture

The codebase follows Go best practices with a standard project layout:

- **`cmd/ilo-pana/`**: Minimal entry point (30 lines) that delegates to internal packages
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
./bin/ilo-pana https://api.example.com/users

# POST with JSON
./bin/ilo-pana -X POST -d '{"name":"test"}' https://api.example.com/users

# Multiple headers
./bin/ilo-pana -H 'Authorization: Bearer token' -H 'Accept: application/json' https://api.example.com

# Show sensitive headers (verbose mode)
API_TESTER_VERBOSE=true ./bin/ilo-pana -H 'X-API-Key: secret' https://api.example.com
```

## Release Process

The project uses GitHub Actions and GoReleaser for automated releases:

1. **Tag a release**: `git tag -a v0.1.0 -m "Release v0.1.0"`
2. **Push tag**: `git push origin v0.1.0`
3. **GitHub Actions** automatically builds binaries for multiple platforms using GoReleaser v2
4. **Binary naming**: `ilo-pana_[version]_[OS]_[arch]` (e.g., `ilo-pana_0.1.0_macOS_arm64.tar.gz`)

Configuration files:
- `.goreleaser.yml`: GoReleaser v2 configuration
- `.github/workflows/release.yml`: GitHub Actions workflow (uses GoReleaser v2)