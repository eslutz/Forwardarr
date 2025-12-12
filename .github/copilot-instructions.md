# Forwardarr AI Coding Instructions

## Project Overview

Forwardarr (formerly Gluetarr) is a Go application that synchronizes port forwarding changes from Gluetun VPN to qBittorrent. It is designed for reliability and observability in containerized environments.

## Architecture

- **Entry Point**: `cmd/forwardarr/main.go` initializes configuration, clients, and starts the server and watcher.
- **Core Logic**:
  - `internal/sync`: Watches the Gluetun port file using `fsnotify`. Updates qBittorrent when the file changes or on a ticker interval.
  - `internal/qbit`: Client for interacting with qBittorrent API (auth, get/set preferences).
  - `internal/server`: HTTP server providing health, readiness, and metrics endpoints.
- **Configuration**: Handled in `internal/config` via environment variables.

## Development Workflows

### Build & Run

- **Local**: `go run ./cmd/forwardarr`
- **Docker**: `docker build -t forwardarr .`
- **Version Injection**: The `Dockerfile` injects version info using `-ldflags`.
  ```bash
  -ldflags="-w -s -X github.com/eslutz/forwardarr/pkg/version.Version=${VERSION} ..."
  ```

### Testing

- **Framework**: Standard Go `testing` package.
- **Pattern**: Prefer table-driven tests for logic (e.g., `internal/config/config_test.go`).
- **File System**: Use `t.TempDir()` for tests involving file operations (e.g., `internal/sync/watcher_test.go`).

### Release Process

The release process is fully automated and driven by the `VERSION` file.

1.  **Update Version**: Modify the `VERSION` file in the root of the repository (e.g., `1.1.0`).
2.  **Commit & Push**: Commit the change and push to `main` (or merge a PR).
3.  **CI Workflow**: The `CI` workflow (`.github/workflows/ci.yml`) runs linting, tests, and security checks.
4.  **Release Workflow**:
    - Triggered automatically after `CI` completes successfully on `main`.
    - Reads the `VERSION` file.
    - Checks if a git tag `v{VERSION}` already exists.
    - **If the tag is new**:
      - Creates and pushes the git tag `v{VERSION}`.
      - Builds and pushes the Docker image to GHCR.
      - Creates a GitHub Release.
    - **If the tag exists**: The workflow skips the release steps.

**Crucial**: Do not manually create git tags. The `Release` workflow handles tagging based on the `VERSION` file content.

## Conventions & Patterns

### Error Handling

- Use `fmt.Errorf` with `%w` to wrap errors.
- Return errors up the stack; log only at the top level or in background goroutines.

### Logging

- Use `log/slog` for structured logging.
- Levels: `Info` for startup/shutdown, `Debug` for operational details (e.g., "port file changed"), `Error` for failures.
- Include context fields: `slog.Info("msg", "key", value)`.

### Configuration

- All config is driven by environment variables (see `internal/config/config.go`).
- Default values are provided for all settings.

### Concurrency

- Use `context.Context` for cancellation and graceful shutdown.
- `main.go` manages the lifecycle of background goroutines (server, watcher) using `signal.NotifyContext`.

## Key Files

- `VERSION`: Source of truth for the project version.
- `cmd/forwardarr/main.go`: Application wiring and lifecycle.
- `internal/sync/watcher.go`: File watching and sync logic.
- `internal/qbit/client.go`: qBittorrent API interaction.
- `.github/workflows/ci.yml`: CI pipeline (lint, test, security).
- `.github/workflows/release.yml`: Release pipeline (tag, build, publish).
