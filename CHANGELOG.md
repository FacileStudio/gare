# Changelog

## [Unreleased]

### Added

- Compose workload type: `type: compose` in `gare.yml` runs a repository-owned compose file with `podman compose`, with `compose_file` selecting the file explicitly or falling back to discovery.
- Compose workloads require an explicit `port`, validated against the host ports the compose file publishes, so ingress never routes to a port the stack does not bind.
- Compose systemd units run detached (`up -d`) as `Type=oneshot` with `RemainAfterExit=yes` and a per-app project name via `-p`, so `start`, `stop`, `restart`, and `status` behave like every other workload type without stalling the stop path.
- Compose units require `podman.socket`, which systemd starts with the app, because the external compose provider talks to the podman API socket.
- `healthchecks:` in `gare.yml` declares one HTTP probe per published port, and `gare deploy` verifies each probe and names the one that fails.
- Compose workloads derive probes from the compose file when `gare.yml` declares none: services whose healthcheck issues an HTTP request to a published container port become probes, mapped to their published host port.
- `gare destroy` tears compose workloads down with `podman compose down -v`, independent of whether the unit file still exists, and reports containers the provider failed to remove.
- `gare deploy` preflights the external compose provider and fails with an actionable error when neither `docker-compose` nor `podman-compose` is available; `gare init` reports the provider and the podman socket.

### Changed

- `gare app env` now stores environment variables for static and compose workloads in the same application env file, which compose units load through systemd `EnvironmentFile=`.
- Renamed the static env storage to app env storage (`storage.GetAppEnv`, `SetAppEnv`, `UnsetAppEnv`) now that it serves every non-Pod workload.
- Health configuration moved to a `health` section in `config.json` holding the primary path and the probe list; the legacy `healthcheck` key is migrated on load.
- `gare stop` accepts any non-running final state (including a provider that exited non-zero) instead of waiting out a stop timeout, and reports the result honestly.
- Unknown `type` values in `gare.yml` are rejected instead of silently falling back to a single-container Pod.
- Deployment warns when a repository contains a compose file but no `type: compose`, instead of silently ignoring it.

## [0.10.0] - 2026-09-18

### Added

- `gare tag add|rm|list` commands to manage application tags.
- Support for `tags` in `gare.yml` and `config.json` with GitOps synchronization on deployment.
- `--tag` / `-t` flag in `gare list` to filter applications by tag.
- `TAGS` column in `gare list` table and `tags` field in JSON output.
- `--tag` flag in `gare app create` to attach initial tags during creation.

## [0.9.0] - 2026-09-18

### Added

- `gare domain add|rm|list` commands for managing hostnames as first-class resources attached to applications.

### Changed

- Removed `--domain` flag from `gare app create`; hostnames are now attached after creation with `gare domain add`.
- Hostnames are stored as `Domains []string` in `config.json`.

## [0.8.6] - 2026-09-18

### Added

- Nested application lifecycle and management commands under `gare app` (`create`, `deploy`, `list`, `status`, `start`, `stop`, `restart`, `logs`, `destroy`, `env`).
- Added command aliases: `apps` for `app`, `delete` and `rm` for `destroy`, and `ls` and `ps` for `list`.
- Grouped root commands into "MANAGEMENT COMMANDS" and "APPLICATION SHORTCUTS" in help output while preserving root command shortcuts.

## [0.8.5] - 2026-09-18

### Fixed

- Resolved container port not wired to host port in Kubernetes Pod manifests (`manifest.yaml`) executed via `podman kube play`.
- Added automatic detection of `EXPOSE` directives from `Containerfile` and `Dockerfile` in the workload repository.
- Synchronized port configuration on deployment (`gare deploy`) and updated manifest ports in-place while preserving environment variables.
- Added `--container-port` flag to `gare app create` and `container_port` support in `gare.yml`.
- Displayed container port wiring in `gare status` output.

## [0.8.4] - 2026-09-17

### Fixed

- Prevented `gare app create` from marking static applications active upon creation. Applications now remain inactive until explicitly started or deployed.
- Resolved static application port connection failure when Caddy daemon is not running on the host by supervising static workloads directly via systemd user units (`caddy file-server --listen :<port> --root <dir>`).
- Re-enabled `gare logs` journal streaming for static workloads.
- Restored uniform lifecycle state inspection (`status`, `list`, `start`, `stop`, `restart`) across both container and static workloads via live systemd property polling.


### Fixed

- Suppressed connection refused errors during Caddy configuration reload when the Caddy daemon or Admin API is stopped.


### Added

- Initialization verification in `gare init` checking snippet directory existence, Caddyfile presence, and import directive persistence with success confirmation.

## [0.8.1] - 2026-09-17

### Added

- Automatic `sudo` fallback in `gare init` to configure `/etc/caddy` permissions seamlessly.
- Plain-text formatted manual instructions when permission setup requires user action.

### Changed

- Transitioned static site hosting to direct Caddy drop-in `file_server` snippets, removing systemd user unit synthesis and port allocation overhead for static workloads.
- Updated `gare start`, `stop`, `restart`, and `status` to control static sites directly via Caddy drop-in snippets and reloads.
- Removed dead `systemd.static_unit` code.

## [0.8.0] - 2026-09-17

### Added

- Automatic Caddyfile and drop-in snippet directory initialization in `caddy.Reload`, `WriteSnippet`, and `WriteStaticSnippet`.
- Automatic appending of `import /etc/caddy/conf.d/*.caddy` to existing Caddyfiles when snippet imports are absent.
- Automatic application name inference from Git repository URLs in `gare app create` when the name argument is omitted.

### Fixed

- Resolved Caddy reload failure on missing `/etc/caddy/Caddyfile` during `gare destroy`, `deploy`, and lifecycle operations.
- Resolved Cobra flag collision where root persistent flag shorthand `-p` shadowed `--port` (`-p`) on child commands.

## [0.7.0] - 2026-09-17

### Added

- Static workload systemd user unit synthesis running `caddy file-server` with automated port binding and health check verification.
- Static application environment variable support (`~/.local/share/gare/apps/<name>/env`) injected into systemd user units via `EnvironmentFile`.
- Automatic free port discovery starting from port 8000 and collision detection for declared ports in `gare.yml`.
- Systemd lifecycle state verification (`WaitForState`) in `gare start`, `stop`, and `restart` with status output and 90s shutdown timeout.
- Default limit of 100 lines for `gare logs` and clean signal handling for `SIGINT` / `Ctrl+C`.
- Robust `.env` parser supporting inline comments, single/double quotes, escape sequences, and multiline values.
- Preservation of configured environment variables across `gare deploy` repository manifest updates.

### Fixed

- Prevented `gare restart` crash on static sites attempting to reload missing `/etc/caddy/Caddyfile`.
- Resolved no-op behavior in `gare start` and `gare stop` for static applications.
- Suppressed `signal: interrupt` exit code 1 error when canceling `gare logs -f`.


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
