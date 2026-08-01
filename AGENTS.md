# AGENTS.md

**ilo-pana** — HTTP API testing tool (curl/HTTPie-like). One Go module `ilo-pana` (NOT a github.com path — internal imports look like `import "ilo-pana/internal/config"`).

Two apps share `internal/`:
- CLI: `cmd/ilo-pana` (main entrypoint, `session` subcommand in `cmd/ilo-pana/commands/`)
- GUI: `gui/` (Wails v2 + Svelte 5/TS frontend in `gui/frontend`)

`CLAUDE.md` also exists but is partially stale — trust code over it (see Gotchas).

## Commands

```bash
go build -o bin/ilo-pana ./cmd/ilo-pana   # CLI build
go test ./...                             # all tests
go test -v ./internal/config              # one package
go test -run TestValidateURL ./internal/request  # one test
go test -race ./...

# Frontend (run inside gui/frontend/, pnpm only):
pnpm install && pnpm check                # svelte-check typecheck
pnpm build                                # outputs to gui/frontend/dist

# GUI (run inside gui/):
wails dev                                 # live dev
wails build                               # production app
```

Release: push a `v*` tag → GitHub Actions runs GoReleaser v2 (`.goreleaser.yml`). No lint config exists; `golangci-lint` (from the Nix flake) runs with defaults.

## Gotchas (verified — don't relearn these)

- **Fresh clone breaks `go build ./...`**: `gui/main.go` has `//go:embed all:frontend/dist`, but `gui/frontend/dist` is gitignored. Build the frontend first (`cd gui/frontend && pnpm install && pnpm build`) or build only `./cmd/ilo-pana`.
- **`internal/config/parser.go` is dead code**: `HeaderParser`/`URLValidator` there have zero callers. The live paths are `parseHeaders()` in `config.go` and `request.ValidateURL()` in `internal/request/request.go`. Edit those, not parser.go.
- **CLAUDE.md stale claims**: verbose mode is the `-v` flag (there is NO `API_TESTER_VERBOSE` env var); localhost/127.0.0.1 is NOT blocked — `request.ValidateURL` only prints a warning to stderr.
- **Wails CLI comes from the Nix flake** (`wails` package in `flake.nix`, pinned via `flake.lock`). Do NOT `go install` it — a stray `~/go/bin/wails` shadows the nixpkgs one and is removed by design.
- **`gui/frontend/wailsjs/` is generated but committed**. After changing exported methods on `App` in `gui/app.go`, regenerate bindings with `wails generate module` (or via `wails dev/build`), and commit the result.
- Dev shell: Nix flake + direnv (`.envrc` = `use flake`) provides Go, nodejs_22, pnpm, wails CLI, Go tooling. After editing `flake.nix`, run `nix flake update nixpkgs` + `direnv reload`. Without Nix, install Go 1.24+, pnpm, and the wails CLI manually.

## Conventions

- **Commit in work units**: split commits by logical unit of work (one issue / one feature / one bugfix per commit, or per-file-group when that maps to a unit). Do not batch unrelated changes into one commit, and do not leave multiple units uncommitted together. One branch can contain multiple unit commits.
- CLI `client.Execute()` prints formatted output to stdout; `client.ExecuteForGUI()` returns structured `*response.ResponseData` — GUI bindings must use the latter.
- Variable expansion `{{VAR}}`: CLI auto-loads `./.env` if present; precedence is `-var` flags > env file > system env.
- Sessions persist to `~/.ilo-pana/sessions/` (files are 0600); session tests/fixtures must not touch the real home dir — `NewFileStorage(dir)` accepts an override.
- Tests are table-driven with subtests; HTTP tests use `net/http/httptest` (no real network). `internal/session` and `cmd/` currently have no tests.
- Sensitive headers (Authorization, X-API-Key, X-Auth-Token, Cookie, Set-Cookie) are masked in CLI output unless `-v`.
