# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**TakeSort** — Go daemon that watches `/temp` for new media files (from DJI drones, GoPro cameras, Zoom H4e recorder) and automatically moves them into an organized folder structure at `/media/{YYYY}/{YYYY-MM-DD}/`. Runs as a Docker container on TrueNAS.

## Build & Dev Commands

```bash
make build          # Compile binary to bin/takesort
make test           # Run all tests (go test ./... -v)
make lint           # Run golangci-lint
make run            # Build and run locally
make clean          # Remove bin/
make docker-build   # Build Docker image
make docker-up      # Start container (docker compose up -d)
make docker-down    # Stop container
make docker-logs    # Tail container logs
make release VERSION=v1.0.0  # Tag and push a release

# Run a single package's tests
go test ./internal/mover/ -v

# Run a specific test
go test ./internal/orchestrator/ -v -run TestIntegration_FullMP4Flow
```

## Architecture

Clean architecture with dependency injection. Module: `github.com/pinn/takesort`.

**Entry point**: `cmd/takesort/main.go` is the composition root — it wires all dependencies and starts the service. No business logic lives here.

**Orchestrator** (`internal/orchestrator/`) coordinates the pipeline through 8 interfaces (DIP). Each internal package implements one concern:

| Package | Responsibility |
|---------|---------------|
| `config` | Env var parsing, configurable paths (watch/media dirs, derived conflicts/errors) |
| `logger` | slog JSON handler setup |
| `classifier` | Extension-based file type detection |
| `mover` | File movement + date-based dest path building + cross-device fallback |
| `proxy` | Rename LRF/LRV to .mp4 before moving |
| `trash` | Delete THM/SRT files |
| `conflict` | Detect duplicate filenames, redirect to `/temp/conflicts/` |
| `errsink` | Move problem files to `/temp/errors/`, pre-validation (zero-size, no extension) |
| `debounce` | Poll file size until stable (prevents moving incomplete uploads) |
| `watcher` | fsnotify-based directory monitoring, ignores conflicts/errors subdirs |
| `safepath` | Symlink detection and path-traversal validation |

**Adapter pattern**: `main.go` defines adapter structs that bridge package-level functions to the orchestrator's interfaces. This keeps packages decoupled — they export plain functions, not interface implementations.

## File Processing Pipeline

`orchestrator.ProcessFile()` runs this sequence for each detected file:

1. **Reject symlinks** — symlinks are moved to `/temp/errors/`
2. **Validate** — reject zero-size or no-extension files to `/temp/errors/`
3. **Classify** — map extension to FileType
4. **Route** — Unknown → errors, Trash → delete, all others continue
5. **Build dest path** — `/media/{YYYY}/{YYYY-MM-DD}/[subfolder]` from file's ModTime
6. **Check conflict** — if dest file exists, redirect to `/temp/conflicts/`
7. **Process** — Proxy files: rename extension to .mp4 then move; others: move directly
8. **On error** — move source file to `/temp/errors/`

On startup, orphan files already in `/temp` are scanned and processed through the same pipeline.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TAKESORT_WATCH_DIR` | `/media/temp` | Directory to watch for new files |
| `TAKESORT_MEDIA_DIR` | `/media` | Root directory for organized output |
| `TAKESORT_DEBOUNCE_INTERVAL` | `2s` | Interval between file size checks |
| `TAKESORT_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

`ConflictsDir` and `ErrorsDir` are derived automatically: `{WatchDir}/conflicts` and `{WatchDir}/errors`. For best performance, mount watch and media dirs on the same filesystem so `os.Rename` works without copy.

## File Classification & Routing

| Extension | Type | Destination |
|-----------|------|-------------|
| `.mp4` | Video | `{date}/` (root) |
| `.lrf`, `.lrv` | Proxy | `{date}/proxy/` (renamed to .mp4) |
| `.jpg` | PhotoJPEG | `{date}/photos/jpeg/` |
| `.dng` | PhotoRAW | `{date}/photos/raw/` |
| `.wav` | Audio | `{date}/audio/` |
| `.thm`, `.srt` | Trash | Deleted |
| other | Unknown | `/temp/errors/` |

Classification is case-insensitive.

## Key Patterns

- **Debounce**: 4 consecutive size checks at configurable interval before processing (critical for large 4K video files still being copied)
- **Cross-device move**: `os.Rename` with fallback to copy+delete when source and dest are on different filesystems (common with Docker volumes)
- **Graceful shutdown**: SIGINT/SIGTERM cancel the context, stopping watcher and orchestrator cleanly
- **Version injection**: `main.version` is set via `-ldflags` at build time (Dockerfile `ARG VERSION=dev`)

## Testing

Tests use `testify` (assert/require). Orchestrator tests wire real package adapters (not mocks) against `t.TempDir()` — they exercise the full pipeline on the real filesystem. Integration tests in `internal/orchestrator/integration_test.go` cover end-to-end flows.

## Branching & CI/CD

- `main` — production: CI auto-increments semver tag, builds multi-arch Docker image (amd64/arm64), pushes to Docker Hub with `latest`, `vX.Y.Z`, and `vX.Y` tags
- `develop` — dev builds: pushes `dev-latest` and `dev-{sha}` tags to Docker Hub
- CI runs tests and `golangci-lint` on both branches and PRs
