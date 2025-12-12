# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

```bash
# Build everything (agent + server)
make build

# Build server only (requires CGO for SQLite)
CGO_ENABLED=1 go build -o dist/server ./cmd/server

# Build agent for Linux (no CGO required)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/agent-linux-amd64 ./cmd/agent
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/agent-linux-arm64 ./cmd/agent

# Run server locally
make run-server

# Run agent once (for testing)
make run-agent

# Docker build (local)
make docker-build

# Docker multi-arch build and push
make docker-buildx
```

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Lint check
make lint
```

## Architecture

This is a VPS monitoring system with two components:

### Server (`cmd/server/`)
- HTTP server using standard `net/http` with `http.ServeMux`
- Receives metrics via POST `/api/metrics` with Bearer token auth
- Stores data in SQLite via `internal/storage/sqlite.go`
- Serves embedded HTML dashboard from `internal/dashboard/templates.go`
- Runs daily cleanup job at 3 AM for metrics older than retention period

### Agent (`cmd/agent/`)
- Runs on each VPS, collects metrics every hour (configurable)
- Uses `internal/collector/collector.go` to gather:
  - CPU: reads `/proc/stat` with 1-second delta
  - Memory: reads `/proc/meminfo` (MemTotal - MemAvailable)
  - Disk: uses `syscall.Statfs` on root partition
  - Docker: uses Docker API via socket for container counts and Swarm role
- Sends JSON payload to server with auto-retry

### Key Design Decisions
- **CGO required for server**: SQLite driver (`github.com/mattn/go-sqlite3`) needs CGO
- **No CGO for agent**: Agent is pure Go for easy cross-compilation
- **Linux-only metrics collection**: Uses `/proc` filesystem (Linux specific)
- **Auto-registration**: Machines register automatically on first metric POST
- **Online threshold**: Machine considered online if last seen < 70 minutes ago

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/metrics` | POST | Yes | Receive metrics from agent |
| `/api/machines` | GET | No | List all machines with latest metrics |
| `/api/machines/:id` | GET | No | Get machine details |
| `/api/machines/:id/metrics` | GET | No | Get metrics history |
| `/api/stats` | GET | No | Get summary statistics |
| `/api/health` | GET | No | Health check |
| `/` | GET | No | Dashboard HTML |

## Release Process

Uses Release Please with Conventional Commits:
- `feat:` → minor version bump
- `fix:` → patch version bump
- `feat!:` or `BREAKING CHANGE:` → major version bump

Push to `main` creates a release PR. Merging the PR triggers Docker build and push.
