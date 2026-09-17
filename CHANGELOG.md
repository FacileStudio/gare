# Changelog

## [Unreleased]

## [0.6.0] - 2026-09-17

### Added

- Environment variable management CLI (`gare env set`, `unset`, `list`, `load`) with atomic Kubernetes manifest updates and automated service reloads.
- Application lifecycle control commands (`gare start`, `stop`, `restart`).
- Detailed application status command (`gare status`) displaying workload, source, systemd properties, memory usage, and health state with `--json` support.
- Healthcheck configuration (`healthcheck` in `gare.yml` / `--healthcheck` flag) and post-deployment HTTP readiness verification in `gare deploy`.

## [0.5.0] - 2026-09-17

### Added

- Automatic user and runtime session detection (`os/user.Current()`, `XDG_RUNTIME_DIR`, `DBUS_SESSION_BUS_ADDRESS`) for rootless systemd user commands.
- Non-interactive passwordless Git credential auto-detection (`GIT_TERMINAL_PROMPT=0`, `GIT_SSH_COMMAND`, GitHub CLI `gh`, GitLab CLI `glab`, token env vars).
- Caddy Admin API reload fallback (`http://127.0.0.1:2019/load`) avoiding Polkit password prompts.
- Systemd user unit cgroup delegation (`Delegate=yes`) and stop timeout (`TimeoutStopSec=70s`).
- Global CLI configuration flags and YAML file loader (`~/.gare.yml`).
- `scripts/check.sh` — suite quality gate for `go vet`, `go test`, and `filet check`.
- `.gitattributes` — Go language detection and line ending specification.

### Fixed

- Caddy static site snippet template syntax (`try_files` placed outside `file_server`).
- Removed duplicate symbol declarations and stubs across `cmd/gare`.
- Resolved all Filet linter errors and in-body comment rule violations.

## [0.4.1] - 2026-09-15

### Fixed

- Caddy static site `try_files` directive nesting: `try_files {path} /index.html` must be inside `file_server` block for valid Caddy v2 configuration.

## [0.4.0] - 2026-09-15

### Added

- Root Caddyfile initialization and verification in `gare init` with disk space and quota error handling.
- `-n` / `--lines` flag for `gare logs` to control the number of streamed journal lines.

### Fixed

- Version injection via GoReleaser ldflags using variable in `main.go`.
- Synchronize Caddy ingress snippets during container application deployments.
- Route warning messages to `stderr` per Facile CLI standard.
- Format status outputs cleanly when domain is not specified.
- Clean up partial storage directory when Git repository cloning fails.
- Fix Caddyfile drop-in import syntax to be top-level.
- Remove trailing periods from single-clause status messages and empty states.

## [0.3.0] - 2026-09-15

### Added

- `gare app create` now accepts creation without `--domain`; the domain can be added later via config.

## [0.2.0] - 2026-09-15

### Added

- Static workload hosting via `static_dir` and `build_cmd` in `gare.yml`.
- `gare.yml` configuration support for container and static app types.
- Port auto-discovery starting at 8000.
- `gare server` webhook receiver for automated deployments.

## [0.1.0] - 2026-09-15

### Added

- Initial release of gare.
