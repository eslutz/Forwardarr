# Gluetarr

[![CI](https://github.com/eslutz/gluetarr/actions/workflows/ci.yml/badge.svg)](https://github.com/eslutz/gluetarr/actions/workflows/ci.yml)
[![Security](https://github.com/eslutz/gluetarr/actions/workflows/security.yml/badge.svg)](https://github.com/eslutz/gluetarr/actions/workflows/security.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/eslutz/gluetarr)](https://goreportcard.com/report/github.com/eslutz/gluetarr)
[![License](https://img.shields.io/github/license/eslutz/gluetarr)](LICENSE)
[![Release](https://img.shields.io/github/v/release/eslutz/gluetarr)](https://github.com/eslutz/gluetarr/releases/latest)

A lightweight, production-ready Go application that automatically synchronizes port forwarding changes from Gluetun VPN to qBittorrent. Built with observability and reliability in mind.

## Features

- 🔄 **Automatic Port Synchronization**: Monitors Gluetun's forwarded port file and updates qBittorrent instantly
- 📊 **Full Observability**: Prometheus metrics for monitoring and alerting
- 🏥 **Health & Readiness**: Kubernetes-compatible health check endpoints
- ⚡ **Efficient File Watching**: Uses fsnotify for real-time file system events
- 🔒 **Secure by Default**: Runs as non-root user in Docker, minimal attack surface
- 🪶 **Lightweight**: ~15MB Docker image, minimal resource footprint
- 🛡️ **Production Ready**: Automatic re-authentication, graceful error handling, fallback polling

## Quick Start

### Docker Compose

An example `docker-compose.yml` file is available at [docker-compose.example.yml](docker-compose.example.yml).

### Docker CLI

```bash
docker run -d \
  --name gluetarr \
  -e GLUETUN_PORT_FILE=/tmp/gluetun/forwarded_port \
  -e QBIT_ADDR=http://qbittorrent:8080 \
  -e QBIT_USER=admin \
  -e QBIT_PASS=adminadmin \
  -v gluetun-data:/tmp/gluetun:ro \
  -p 9090:9090 \
  ghcr.io/eslutz/gluetarr:latest
```

## Configuration

All configuration is done via environment variables. An example configuration file is available at [docs/.env.example](docs/.env.example).

| Variable | Default | Description |
|----------|---------|-------------|
| `GLUETUN_PORT_FILE` | `/tmp/gluetun/forwarded_port` | Path to Gluetun's port file |
| `QBIT_ADDR` | `http://localhost:8080` | qBittorrent WebUI address |
| `QBIT_USER` | `admin` | qBittorrent username |
| `QBIT_PASS` | `adminadmin` | qBittorrent password |
| `SYNC_INTERVAL` | `60` | Fallback polling interval (seconds) |
| `METRICS_PORT` | `9090` | HTTP server port for metrics/health |
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |

## Architecture

```txt
┌─────────┐         ┌──────────┐         ┌────────────┐
│ Gluetun │ writes  │  Port    │ watched │            │
│   VPN   ├────────►│   File   ├────────►│  Gluetarr  │
└─────────┘         └──────────┘         └─────┬──────┘
                                                │
                                                │ updates
                                                ▼
                                         ┌─────────────┐
                                         │ qBittorrent │
                                         └─────────────┘
```

**How it works:**

1. Gluetun establishes a VPN connection with port forwarding
2. Gluetun writes the forwarded port to a file
3. Gluetarr watches this file for changes using fsnotify
4. When the port changes, Gluetarr updates qBittorrent's listening port via API
5. A fallback ticker ensures sync even if file events are missed

## HTTP Endpoints

| Endpoint | Purpose | Response |
|----------|---------|----------|
| `GET /health` | Liveness probe | `200 OK` if running |
| `GET /ready` | Readiness probe | `200 OK` if qBittorrent is reachable |
| `GET /metrics` | Prometheus metrics | Metrics in OpenMetrics format |

## Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `gluetarr_info` | Gauge | Build information (version, commit, date) |
| `gluetarr_current_port` | Gauge | Current forwarded port from Gluetun |
| `gluetarr_sync_total` | Counter | Total number of successful port syncs |
| `gluetarr_sync_errors` | Counter | Total number of failed sync attempts |
| `gluetarr_last_sync_timestamp` | Gauge | Unix timestamp of last successful sync |

### Example Prometheus Queries

```promql
# Current forwarded port
gluetarr_current_port

# Sync success rate (last 5m)
rate(gluetarr_sync_total[5m]) / (rate(gluetarr_sync_total[5m]) + rate(gluetarr_sync_errors[5m]))

# Time since last successful sync
time() - gluetarr_last_sync_timestamp
```

## Grafana Dashboard

A pre-built Grafana dashboard is available at [docs/dashboard.json](docs/dashboard.json). Import it into your Grafana instance to visualize:

- Application version and build info
- Current forwarded port with change history
- Sync operation success rates and error counts
- Go runtime metrics (memory, goroutines, CPU)
- Time since last successful sync

## Building from Source

```bash
# Clone the repository
git clone https://github.com/eslutz/gluetarr.git
cd gluetarr

# Build binary
go build -o gluetarr ./cmd/gluetarr

# Build Docker image
docker build -t gluetarr .
```

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test -v ./...

# Run linter
golangci-lint run

# Run locally
export GLUETUN_PORT_FILE=/path/to/port/file
export QBIT_ADDR=http://localhost:8080
export QBIT_USER=admin
export QBIT_PASS=adminadmin
go run ./cmd/gluetarr
```

## Troubleshooting

### Gluetarr can't connect to qBittorrent

- Verify qBittorrent is accessible at the configured address
- Check credentials are correct
- Ensure network connectivity between containers
- Check logs: `docker logs gluetarr`

### Port not updating

- Verify Gluetun is writing to the port file: `cat /tmp/gluetun/forwarded_port`
- Check the port file path is correct in Gluetarr config
- Ensure the volume mount is working: `docker exec gluetarr cat /tmp/gluetun/forwarded_port`
- Increase log level to debug: `LOG_LEVEL=debug`

### High resource usage

- Increase `SYNC_INTERVAL` to reduce polling frequency
- Check for excessive file system events in the watched directory

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](.github/PULL_REQUEST_TEMPLATE.md) before submitting PRs.

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run linters and tests
6. Submit a pull request

## Security

Please see our [Security Policy](.github/SECURITY.md) for reporting vulnerabilities.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Gluetun](https://github.com/qdm12/gluetun) - VPN client with port forwarding
- [qBittorrent](https://www.qbittorrent.org/) - BitTorrent client
- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications

## Related Projects

- [Torarr](https://github.com/eslutz/torarr) - Similar project for Transmission
