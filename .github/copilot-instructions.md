# Gluetarr Context for GitHub Copilot

## Project Overview

Gluetarr is a lightweight Go application that automatically synchronizes port forwarding changes from Gluetun VPN to qBittorrent. It monitors a port file written by Gluetun and updates qBittorrent's listening port accordingly.

## Architecture

- **Language**: Go 1.25+
- **Core Dependencies**:
  - `fsnotify/fsnotify`: File system watching
  - `prometheus/client_golang`: Metrics and observability
  - `rs/zerolog`: Structured logging

## Project Structure

```
cmd/gluetarr/     - Application entrypoint
internal/config/  - Configuration management
internal/qbit/    - qBittorrent API client
internal/sync/    - File watching and sync logic
internal/server/  - HTTP server for health/metrics
pkg/version/      - Build version information
```

## Key Components

### Config (internal/config)

Loads configuration from environment variables with sensible defaults.

### qBittorrent Client (internal/qbit)

- Cookie-based authentication
- Automatic re-authentication on 403 errors
- Type-safe API methods for getting/setting ports

### Sync Watcher (internal/sync)

- Uses fsnotify for efficient file watching
- Fallback ticker for reliability
- Prometheus metrics integration

### HTTP Server (internal/server)

- `/health` - Liveness probe
- `/ready` - Readiness probe (checks qBittorrent connectivity)
- `/metrics` - Prometheus metrics

## Metrics

- `gluetarr_info` - Build information
- `gluetarr_current_port` - Current forwarded port
- `gluetarr_sync_total` - Successful syncs counter
- `gluetarr_sync_errors` - Failed syncs counter
- `gluetarr_last_sync_timestamp` - Last successful sync time

## Coding Style

- Use structured logging with zerolog
- Follow Go standard error handling patterns
- Prefer composition over inheritance
- Keep functions small and focused
- Document exported types and functions

## Testing Considerations

- Mock file system operations for unit tests
- Mock qBittorrent API responses
- Test error handling paths
- Verify metrics are updated correctly

## Common Tasks

- Adding new metrics: Add to `internal/sync/metrics.go`
- Adding new config options: Update `internal/config/config.go`
- Modifying qBittorrent API: Update `internal/qbit/client.go`
- Adding HTTP endpoints: Update `internal/server/handlers.go`

## Docker Deployment

Multi-stage build for minimal image size (~15MB). Runs as non-root user (UID 1000).
