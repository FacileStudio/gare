# Gare Lifecycle, Environment & Healthcheck Implementation Plan

## 1. Overview

This document outlines the architecture and execution plan for adding environment variable management (`gare env`), service lifecycle controls (`gare start`, `gare stop`, `gare restart`, `gare status`), and post-deployment health check verification in `gare`.

---

## 2. Architecture & Design

### A. Environment Management (`internal/storage/manifest.go`, `cmd/gare/env.go`)

- **Manifest Storage**: Container environment variables reside directly in the Kubernetes Pod manifest (`manifest.yaml`) under `spec.containers[0].env`.
- **Atomic Modification**: Modifying environment variables parses `manifest.yaml` via `gopkg.in/yaml.v3`, updates or removes matching `- name: KEY, value: VAL` items while preserving comments and structure, and writes back atomically via `internal/atomicfile`.
- **Subcommands**:
  - `gare env set <app> KEY=VALUE [KEY2=VALUE2...]`: Adds or updates environment variables.
  - `gare env unset <app> KEY [KEY2...]`: Removes environment variables.
  - `gare env list <app> [--json]`: Displays active environment variables.
  - `gare env load <app> -f <env-file>`: Parses `.env` key-value pairs (ignoring empty lines and `#` comments) and sets them.
- **Service Reload**: If the application is a container and its systemd unit is active, `gare env set/unset/load` triggers `systemd.Restart(ctx, appName)`.

### B. App Lifecycle Commands (`internal/systemd/`, `cmd/gare/lifecycle.go`, `cmd/gare/status.go`)

- **Service Controls**:
  - `gare start <app>`: Calls `systemd.Start(ctx, appName)` then `systemd.WaitForState(ctx, appName, "active")`. The drafted `systemd.EnableAndStart` was never used and has been removed; enabling happens on deploy through `systemd.Enable`.
  - `gare stop <app>`: Calls `systemd.Stop(ctx, appName)`.
  - `gare restart <app>`: Calls `systemd.Restart(ctx, appName)`.
  - For static applications, `start`/`stop`/`restart` output appropriate notices since static sites are served continuously by Caddy without systemd units.
- **Detailed Status (`gare status <app>`)**:
  - Queries `systemctl --user show <app>.service -p ActiveState -p SubState -p MainPID -p ActiveEnterTimestamp -p MemoryCurrent`.
  - Retrieves current Git commit hash from `storage.GetRepoDir(appDir)`.
  - Inspects app config (`config.json`), port allocation, domain routing, and healthcheck path.
  - Formats output into a styled terminal summary or computer-parsable JSON (`--json`).

### C. Healthcheck Configuration & Deployment Verification (`internal/storage/garefile.go`, `internal/health/`, `cmd/gare/deploy.go`)

- **`gare.yml` Schema**:
  ```yaml
  type: container
  healthcheck: /health
  # or nested
  health_check: /ready
  ```
- **CLI Flags**: `--healthcheck` / `-H` on `gare app create` and `gare deploy`.
- **Verification Loop**:
  - During `gare deploy`, after starting or restarting the service, `gare` polls `http://127.0.0.1:<port><healthcheck-path>` with exponential backoff up to a timeout (default: 30 seconds).
  - Returns clean success when HTTP 200-399 is received.
  - Returns clear error if service fails to become healthy.

---

## 3. Implementation Steps & File Allocations

1. **Storage & Manifest Package (`internal/storage/`)**:
   - `manifest_env.go`: Implement `GetManifestEnv`, `SetManifestEnv`, `UnsetManifestEnv`, `LoadDotEnv`.
   - `garefile.go`: Add `Healthcheck` and `HealthCheck` fields to `GareFile` with `ResolveHealthcheck()`.
   - `storage.go`: Add `Healthcheck` field to `AppConfig`.
   - Tests: `manifest_env_test.go`, `garefile_test.go`.

2. **Systemd & Health Packages (`internal/systemd/`, `internal/health/`)**:
   - `internal/systemd/systemctl.go`: Add `Start` wrapper and `GetServiceProperties` querying systemctl properties.
   - `internal/health/probe.go`: Implement lightweight HTTP/TCP health probe with retry loop.
   - Tests for systemd properties parsing and health probes.

3. **CLI Commands (`cmd/gare/`)**:
   - `cmd/gare/env.go`: CLI implementation for `gare env` (`set`, `unset`, `list`, `load`).
   - `cmd/gare/lifecycle.go`: CLI implementation for `gare start`, `gare stop`, `gare restart`.
   - `cmd/gare/status.go`: CLI implementation for `gare status`.
   - `cmd/gare/app.go` & `cmd/gare/deploy.go`: Wire healthcheck flags and deploy probe execution.
   - `cmd/gare/root.go`: Register all new commands.

4. **Verification & Quality Gate**:
   - `mise run test`: All unit and package tests pass.
   - `filet check`: 100% compliance with Filet rules (no body comments, line limits, etc.).
