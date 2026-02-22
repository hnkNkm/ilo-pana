# ilo-pana 🔧

A simple and powerful HTTP API testing tool written in Go. The name "ilo-pana" means "good tool" in Toki Pona.

## Features

- 🚀 Fast and lightweight
- 🎨 Colorized JSON output
- 🔒 Automatic sensitive header masking
- 📝 Support for all HTTP methods
- ⚡ Multiple headers support
- 🌐 Clean request/response formatting
- 🔄 **NEW: Variable expansion with {{syntax}}**
- 📁 **NEW: Environment file support (.env)**
- 🍪 **NEW: Session management with cookies**

## Installation

### Download Binary (Recommended)

Download the latest release from [GitHub Releases](https://github.com/hnkNkm/ilo-pana/releases/latest).

#### macOS
```bash
# Intel Mac
curl -L https://github.com/hnkNkm/ilo-pana/releases/latest/download/ilo-pana_0.2.0_macOS_x86_64.tar.gz | tar xz
sudo mv ilo-pana /usr/local/bin/

# Apple Silicon (M1/M2)
curl -L https://github.com/hnkNkm/ilo-pana/releases/latest/download/ilo-pana_0.2.0_macOS_arm64.tar.gz | tar xz
sudo mv ilo-pana /usr/local/bin/
```

#### Linux
```bash
# x86_64
curl -L https://github.com/hnkNkm/ilo-pana/releases/latest/download/ilo-pana_0.2.0_linux_x86_64.tar.gz | tar xz
sudo mv ilo-pana /usr/local/bin/

# ARM64
curl -L https://github.com/hnkNkm/ilo-pana/releases/latest/download/ilo-pana_0.2.0_linux_arm64.tar.gz | tar xz
sudo mv ilo-pana /usr/local/bin/
```

#### Windows
Download the Windows binary from [Releases page](https://github.com/hnkNkm/ilo-pana/releases/latest) and add it to your PATH.

### Build from Source

Requires Go 1.24 or later.

```bash
# Clone the repository
git clone https://github.com/hnkNkm/ilo-pana.git
cd ilo-pana

# Build
go build -o ilo-pana ./cmd/ilo-pana

# Or install to $GOPATH/bin
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
ilo-pana -v -H 'X-API-Key: secret' https://api.example.com
```

### Using variables
```bash
# Command-line variables
ilo-pana -var BASE_URL=https://api.example.com -var TOKEN=abc123 \
  -H "Authorization: Bearer {{TOKEN}}" "{{BASE_URL}}/users"

# Environment file
ilo-pana --env-file .env.dev "{{BASE_URL}}/api/{{VERSION}}/users"
```

### Session management
```bash
# Create session with login
ilo-pana --session dev --session-new \
  -X POST -d '{"user":"test","pass":"password"}' \
  https://api.example.com/login

# Use session for authenticated requests
ilo-pana --session dev https://api.example.com/profile

# Manage sessions
ilo-pana session list
ilo-pana session show dev
ilo-pana session clear dev
```

## Options

```
  -X string              HTTP method (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) (default "GET")
  -H value               HTTP headers (format: 'Key: Value'), can be specified multiple times
  -d string              Request body data
  -timeout duration      Request timeout (default 30s)
  -v                     Verbose output (show all headers without masking)
  -var value             Variables (format: 'key=value'), can be specified multiple times
  --env-file string      Path to environment file (default: ./.env if exists)
  --session string       Session name to use for cookies and headers
  --session-new          Create new session (overwrites existing)
  --session-save-headers Save custom headers to session
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

### Variable Expansion (v0.2.0+)
Use `{{variable}}` syntax to reference variables in URLs, headers, and request bodies:
- Variables can be defined via `-var key=value` flags
- Load from `.env` files with `--env-file`
- Automatic fallback to system environment variables
- Precedence: CLI args > env file > system environment

### Session Management (v0.3.0+)
Sessions persist cookies and custom headers between requests:
- Cookies are automatically saved and sent
- Custom headers can be saved with `--session-save-headers`
- Sessions are stored in `~/.ilo-pana/sessions/`
- Secure file permissions (0600) for session files

## Development

### Project Structure
```
ilo-pana/
├── cmd/ilo-pana/        # Main entry point
│   └── commands/       # Subcommands (session, etc.)
├── internal/            # Internal packages
│   ├── client/         # HTTP client logic
│   ├── config/         # Configuration and flag parsing
│   ├── env/            # Environment file loader
│   ├── request/        # Request building and validation
│   ├── response/       # Response processing and formatting
│   ├── session/        # Session management
│   └── variables/      # Variable expansion
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