# ilo-pana 🔧

A simple and powerful HTTP API testing tool written in Go. The name "ilo-pana" means "good tool" in Toki Pona.

## Features

- 🚀 Fast and lightweight
- 🎨 Colorized JSON output
- 🔒 Automatic sensitive header masking
- 📝 Support for all HTTP methods
- ⚡ Multiple headers support
- 🌐 Clean request/response formatting

## Installation

```bash
# Clone the repository
git clone https://github.com/hnkNkm/ilo-pana.git
cd ilo-pana

# Build from source
go build -o ilo-pana ./cmd/ilo-pana

# Or install directly
go install ./cmd/ilo-pana
```

## Usage

### Simple GET request
```bash
ilo-pana https://api.example.com/users
```

### POST with JSON data
```bash
ilo-pana -X POST -d '{"name":"John","email":"john@example.com"}' https://api.example.com/users
```

### Multiple headers
```bash
ilo-pana -H 'Authorization: Bearer token' -H 'Accept: application/json' https://api.example.com/protected
```

### With timeout
```bash
ilo-pana -timeout 10s https://slow-api.example.com
```

### Verbose mode (show sensitive headers)
```bash
API_TESTER_VERBOSE=true ilo-pana -H 'X-API-Key: secret' https://api.example.com
```

## Options

```
  -X string      HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) (default "GET")
  -H value       HTTP headers (format: 'Key: Value'), can be specified multiple times
  -d string      Request body data
  -timeout       Request timeout (default 30s)
  -v            Verbose output (show all headers without masking)
```

## Examples

### Testing REST API
```bash
# GET request
ilo-pana https://jsonplaceholder.typicode.com/posts/1

# POST request
ilo-pana -X POST -d '{"title":"Test","body":"Content","userId":1}' \
  https://jsonplaceholder.typicode.com/posts

# PUT request
ilo-pana -X PUT -d '{"id":1,"title":"Updated"}' \
  https://jsonplaceholder.typicode.com/posts/1

# DELETE request
ilo-pana -X DELETE https://jsonplaceholder.typicode.com/posts/1
```

### Testing with authentication
```bash
# Bearer token
ilo-pana -H 'Authorization: Bearer your-token-here' \
  https://api.example.com/protected

# API key
ilo-pana -H 'X-API-Key: your-api-key' \
  https://api.example.com/data
```

## Features in Detail

### Automatic JSON Detection
The tool automatically detects JSON content and formats it with proper indentation.

### Sensitive Header Masking
Headers like `Authorization`, `X-API-Key`, `Cookie`, etc., are automatically masked in the output for security. Use verbose mode to see full values.

### Visual Request/Response Flow
- `→` indicates outgoing request details
- `←` indicates incoming response details

## Development

### Project Structure
```
ilo-pana/
├── cmd/ilo-pana/        # Main entry point
├── internal/            # Internal packages
│   ├── client/         # HTTP client logic
│   ├── config/         # Configuration and flag parsing
│   ├── request/        # Request building and validation
│   └── response/       # Response processing and formatting
└── README.md
```

### Building
```bash
# Build
go build -o ilo-pana ./cmd/ilo-pana

# Run tests
go test ./...

# Run with race detection
go test -race ./...
```

## License

MIT

## Author

hnkNkm

## Name Origin

"ilo-pana" comes from Toki Pona:
- **ilo** = tool, device
- **pana** = give, send, emit

Perfect for a tool that sends HTTP requests!