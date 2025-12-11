# Forwardarr - GitHub Copilot Instructions

## Project Overview

**Forwardarr** is a lightweight, production-ready Go application that automatically synchronizes port forwarding changes from Gluetun VPN to qBittorrent. Built with observability and reliability in mind.

### Core Purpose
- Monitors Gluetun's forwarded port file
- Automatically updates qBittorrent's listening port via API
- Provides Prometheus metrics and health check endpoints

## Technical Stack

- **Language**: Go 1.25+
- **Core Dependencies**:
  - `fsnotify/fsnotify`: File system watching for real-time port file monitoring
  - `prometheus/client_golang`: Metrics and observability
  - `log/slog`: Structured JSON logging (Go standard library)
- **Build**: Multi-stage Docker builds for minimal image size (~15MB)
- **Deployment**: Runs as non-root user (UID 1000) for security

## Project Structure

```
forwardarr/
├── cmd/forwardarr/          # Application entrypoint
│   └── main.go              # Main application logic, signal handling, logging setup
├── internal/
│   ├── config/              # Configuration management
│   │   ├── config.go        # Environment variable loading
│   │   └── config_test.go   # Config tests (100% coverage)
│   ├── qbit/                # qBittorrent API client
│   │   ├── client.go        # HTTP client with cookie-based auth
│   │   └── client_test.go   # API client tests (80% coverage)
│   ├── sync/                # File watching and sync logic
│   │   ├── watcher.go       # fsnotify-based port file watcher
│   │   ├── metrics.go       # Prometheus metrics definitions
│   │   └── watcher_test.go  # Watcher tests (16% coverage)
│   └── server/              # HTTP server for health/metrics
│       ├── server.go        # HTTP server setup
│       ├── handlers.go      # Endpoint handlers
│       └── handlers_test.go # Handler tests (59% coverage)
├── pkg/version/             # Build version information
│   ├── version.go           # Version constants and metrics
│   └── version_test.go      # Version tests (100% coverage)
├── docs/                    # Documentation and examples
│   ├── .env.example         # Example environment configuration
│   ├── docker-compose.example.yml  # Docker Compose example
│   └── forwardarr-grafana-dashboard.json  # Grafana dashboard
├── .github/workflows/       # CI/CD workflows
│   ├── ci.yml              # Tests, linting, CodeQL, Docker build on PRs
│   ├── release.yml         # Tests, CodeQL, Docker publish on tags
│   └── security.yml        # Trivy container scanning
└── Dockerfile              # Multi-stage production build
```

## Key Components

### 1. Configuration (internal/config)

**Purpose**: Load and validate environment variables with sensible defaults.

**Environment Variables**:
| Variable | Default | Description |
|----------|---------|-------------|
| `GLUETUN_PORT_FILE` | `/tmp/gluetun/forwarded_port` | Path to Gluetun's port file |
| `QBIT_ADDR` | `http://localhost:8080` | qBittorrent WebUI address |
| `TORRENT_CLIENT_USER` | `admin` | Torrent client username |
| `TORRENT_CLIENT_PASSWORD` | `adminadmin` | Torrent client password |
| `SYNC_INTERVAL` | `60` | Fallback polling interval (seconds) |
| `METRICS_PORT` | `9090` | HTTP server port for metrics/health |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

**Key Functions**:
- `Load()`: Loads all configuration from environment variables
- `getEnv()`: Helper for string environment variables with defaults
- `getDurationEnv()`: Helper for duration environment variables (parses seconds)

### 2. qBittorrent Client (internal/qbit)

**Purpose**: HTTP client for qBittorrent Web API with authentication.

**Features**:
- Cookie-based authentication with automatic session management
- Automatic re-authentication on 403 (Forbidden) responses
- Type-safe API methods for getting/setting ports

**Key Methods**:
- `NewClient(baseURL, user, pass)`: Creates client and performs initial login
- `Login()`: Authenticates with qBittorrent (returns error if credentials invalid)
- `GetPort()`: Retrieves current listening port from preferences
- `SetPort(port)`: Updates listening port in preferences
- `Ping()`: Health check endpoint for readiness probes

**API Endpoints Used**:
- `POST /api/v2/auth/login`: Authentication
- `GET /api/v2/app/preferences`: Get current settings (including port)
- `POST /api/v2/app/setPreferences`: Update settings (including port)
- `GET /api/v2/app/version`: Health check

### 3. Sync Watcher (internal/sync)

**Purpose**: Monitor port file and synchronize changes to qBittorrent.

**Implementation**:
- Uses `fsnotify` for efficient file system event monitoring
- Watches parent directory (file may not exist at startup)
- Fallback ticker for periodic sync (reliability if events are missed)
- Prometheus metrics for observability

**Key Methods**:
- `NewWatcher()`: Creates watcher and adds directory to watch list
- `Start()`: Main loop handling file events, errors, and periodic syncs
- `syncPort()`: Core sync logic - read file, compare, update if different
- `readPortFromFile()`: Reads and validates port from file (1-65535)

**Metrics Functions**:
- `SetCurrentPort(port)`: Updates current port gauge
- `IncrementSyncTotal()`: Increments successful sync counter
- `IncrementSyncErrors()`: Increments error counter
- `UpdateLastSyncTimestamp()`: Updates last sync timestamp

### 4. HTTP Server (internal/server)

**Purpose**: Expose health checks and Prometheus metrics.

**Endpoints**:
- `GET /health`: Liveness probe (200 OK if running)
- `GET /ready`: Readiness probe (200 OK if qBittorrent reachable)
- `GET /status`: JSON diagnostics (version, status, qBittorrent connectivity)
- `GET /metrics`: Prometheus metrics in OpenMetrics format

**Usage in Kubernetes/Docker**:
- Configure `/health` as liveness probe (restart on failure)
- Configure `/ready` as readiness probe (don't route traffic if unhealthy)
- Scrape `/metrics` with Prometheus for monitoring

### 5. Version Package (pkg/version)

**Purpose**: Build-time version information and info metric.

**Variables** (injected via `-ldflags` during build):
- `Version`: Version string (e.g., "v1.0.0", "dev")
- `Commit`: Git commit SHA
- `Date`: Build timestamp

**Metrics**:
- `forwardarr_info`: Gauge with labels for version, commit, date, go_version

## Prometheus Metrics

All metrics use the `forwardarr_` prefix:

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `forwardarr_info` | Gauge | version, commit, date, go_version | Build information (always 1) |
| `forwardarr_current_port` | Gauge | - | Current forwarded port from Gluetun |
| `forwardarr_sync_total` | Counter | - | Total successful port syncs |
| `forwardarr_sync_errors` | Counter | - | Total failed sync attempts |
| `forwardarr_last_sync_timestamp` | Gauge | - | Unix timestamp of last successful sync |

**Example Queries**:
```promql
# Current forwarded port
forwardarr_current_port

# Sync success rate (last 5m)
rate(forwardarr_sync_total[5m]) / (rate(forwardarr_sync_total[5m]) + rate(forwardarr_sync_errors[5m]))

# Time since last successful sync
time() - forwardarr_last_sync_timestamp
```

## Coding Standards

### Style Guidelines
- **Logging**: Use `log/slog` with structured JSON output
  - Use appropriate levels: Debug, Info, Warn, Error
  - Include context in log fields (e.g., "port", "error", "file")
- **Error Handling**: 
  - Return errors with context using `fmt.Errorf("context: %w", err)`
  - Log errors at appropriate levels before returning
  - Never ignore errors silently
- **Testing**:
  - Use table-driven tests for multiple scenarios
  - Mock external dependencies (HTTP, file system)
  - Aim for high coverage on critical paths
- **Code Organization**:
  - Keep functions small and focused (single responsibility)
  - Document exported types and functions with godoc comments
  - Prefer composition over inheritance
  - Use interfaces for testability

### Package Conventions
- `internal/`: Private application code (not importable by other projects)
- `pkg/`: Public, reusable packages
- `cmd/`: Application entrypoints (one per binary)

## Testing Strategy

### Unit Tests
All packages have comprehensive unit tests:

1. **Config Tests** (100% coverage)
   - Default values, custom values, partial overrides
   - Invalid input handling (e.g., invalid duration)

2. **qBit Client Tests** (80% coverage)
   - Mock HTTP server for API responses
   - Authentication success/failure scenarios
   - Re-authentication on 403 errors
   - Port get/set operations

3. **Sync Watcher Tests** (16% coverage)
   - Port file parsing with various formats
   - Validation of port ranges (1-65535)
   - Edge cases (whitespace, invalid values)

4. **Server Handler Tests** (59% coverage)
   - Health endpoint (running/stopped states)
   - Ready endpoint (qBittorrent reachable/unreachable)
   - Status endpoint JSON response

5. **Version Tests** (100% coverage)
   - String formatting with different values
   - Variable initialization

### Running Tests
```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run specific package tests
go test -v ./internal/config
```

### Building
```bash
# Local build
go build -o forwardarr ./cmd/forwardarr

# Build with version info
go build -ldflags="-X github.com/eslutz/forwardarr/pkg/version.Version=v1.0.0" -o forwardarr ./cmd/forwardarr

# Docker build
docker build -t forwardarr .
```

### Linting
```bash
# Run golangci-lint (used in CI)
golangci-lint run
```

## CI/CD Workflows

### CI Workflow (`.github/workflows/ci.yml`)
Runs on pull requests to `main`:
1. **Tests**: Run all unit tests with race detection
2. **Lint**: golangci-lint for code quality
3. **CodeQL**: Security analysis for Go and GitHub Actions
4. **Build**: Multi-platform Docker build (amd64, arm64) and push to GHCR with `pr-<number>` tag

### Release Workflow (`.github/workflows/release.yml`)
Runs on version tags (`v*`):
1. **Tests**: Run all unit tests and linting
2. **CodeQL**: Security analysis
3. **Build & Publish**: Multi-platform Docker images with semantic versioning
4. **GitHub Release**: Create release with auto-generated notes

### Security Workflow (`.github/workflows/security.yml`)
Runs on PRs, main branch, and weekly schedule:
- **Trivy**: Container image vulnerability scanning (OS packages and libraries)

## Common Development Tasks

### Adding a New Configuration Option
1. Add field to `Config` struct in `internal/config/config.go`
2. Add `getEnv()` call in `Load()` function with default value
3. Add test cases in `internal/config/config_test.go`
4. Update `README.md` configuration table
5. Update `docs/.env.example`

### Adding a New Prometheus Metric
1. Add metric definition to `internal/sync/metrics.go`
2. Add helper function to update metric (e.g., `IncrementXxx()`)
3. Call helper function where metric should be updated
4. Update `README.md` metrics table
5. Update Grafana dashboard if applicable

### Adding a New HTTP Endpoint
1. Add handler function in `internal/server/handlers.go`
2. Register route in `Start()` method in `internal/server/server.go`
3. Add tests in `internal/server/handlers_test.go`
4. Update `README.md` endpoints table

### Modifying qBittorrent API Integration
1. Update methods in `internal/qbit/client.go`
2. Add/update tests in `internal/qbit/client_test.go` with mock server
3. Test against real qBittorrent instance if changing API calls

## Architecture Flow

```
1. Application Start
   ├─> Load configuration from environment
   ├─> Setup structured logging (JSON)
   ├─> Create qBittorrent client & authenticate
   ├─> Create file watcher (fsnotify)
   └─> Start HTTP server (health/metrics)

2. Main Loop (sync watcher)
   ├─> Watch for file system events (fsnotify)
   │   └─> On file write/create: syncPort()
   ├─> Periodic ticker (fallback)
   │   └─> Every SYNC_INTERVAL: syncPort()
   └─> Handle errors and log appropriately

3. syncPort() Flow
   ├─> Read port from file
   ├─> Get current port from qBittorrent
   ├─> Compare ports
   ├─> If different:
   │   ├─> Update qBittorrent port
   │   ├─> Update Prometheus metrics
   │   └─> Log success
   └─> If same: log debug message

4. HTTP Server (concurrent)
   ├─> /health  → Return 200 if running
   ├─> /ready   → Ping qBittorrent, return 200 if reachable
   ├─> /status  → Return JSON with version & connectivity
   └─> /metrics → Return Prometheus metrics

5. Graceful Shutdown
   ├─> Receive SIGINT/SIGTERM
   ├─> Stop accepting new requests
   ├─> Complete in-flight operations (10s timeout)
   └─> Log shutdown and exit
```

## Troubleshooting Guide

### Common Issues

**Authentication Failures**
- Error: `"login failed: status 200, body: Fails."`
- Cause: Incorrect username or password
- Solution: Verify `TORRENT_CLIENT_USER` and `TORRENT_CLIENT_PASSWORD` match qBittorrent credentials

**Port File Not Found**
- Error: `"failed to read port file"`
- Cause: Gluetun hasn't created the port file yet, or path is incorrect
- Solution: Verify `GLUETUN_PORT_FILE` path and ensure Gluetun is running with port forwarding enabled

**Connection Refused**
- Error: `"failed to get preferences: ... connection refused"`
- Cause: qBittorrent is not accessible at the configured address
- Solution: Verify `QBIT_ADDR` and network connectivity (e.g., same Docker network)

### Debugging
- Set `LOG_LEVEL=debug` for verbose logging
- Check `/status` endpoint for connectivity status
- Monitor `/metrics` endpoint for sync errors counter
- Verify qBittorrent logs for API request errors

## Docker Deployment

### Image Details
- **Base Image**: `alpine:3.23` (minimal, security-focused)
- **Size**: ~15MB (multi-stage build)
- **User**: Non-root (UID 1000, username: `forwardarr`)
- **Exposed Port**: 9090 (metrics/health)
- **Healthcheck**: `wget http://localhost:9090/health` every 30s

### Build Arguments
- `VERSION`: Version string (default: "dev")
- `COMMIT`: Git commit SHA (default: "unknown")
- `DATE`: Build timestamp (default: "unknown")

### Volume Mounts
- **Required**: Gluetun port file directory (read-only)
  - `-v gluetun-data:/tmp/gluetun:ro`

### Example docker-compose.yml
See `docs/docker-compose.example.yml` for a complete setup with Gluetun and qBittorrent.

## Related Projects

- [Gluetun](https://github.com/qdm12/gluetun): VPN client with port forwarding support
- [qBittorrent](https://www.qbittorrent.org/): BitTorrent client with Web API
- [Torarr](https://github.com/eslutz/torarr): Similar project for Transmission BitTorrent client

## References

- [qBittorrent Web API Documentation](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))
- [Prometheus Naming Best Practices](https://prometheus.io/docs/practices/naming/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [fsnotify Documentation](https://github.com/fsnotify/fsnotify)
