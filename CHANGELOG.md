# Changelog

## [0.13.4] - 2026-09-23

### Added

- `gare deploy`, `start` and `restart` verify the ingress instead of trusting the snippet on disk. An application with domains is only reported active once Caddy has accepted the configuration and a listener answers on the ports those domains resolve to, and the failure names the app, its domains and the silent ports. An application reachable on its assigned port alone is left alone, because nothing then depends on ingress.

### Fixed

- A generated or adopted Pod manifest left `imagePullPolicy` unset, so `podman kube play` resolved the locally built `localhost/<name>:latest` as a registry reference and restart-looped on a failing lookup instead of running the image gare had just built. Every `localhost/` image is now pinned to `imagePullPolicy: Never`, and `gare start`/`restart` name `gare deploy` when the image is genuinely missing rather than letting podman emit a registry error.
- A compose workload that publishes on loopback only and has no domain is reachable from the host alone; deploy and start now say so and point at `gare domain add <name> <hostname>`.
- Image pruning after a deploy is best-effort and no longer reads as a failed deployment. `podman image prune -f` exits non-zero when a leftover buildah working container holds a dangling image, which says nothing about the workload, so its output is shown only with `--verbose`.

### Changed

- Compose workloads name `podman-compose` as the provider to install, in `gare init` and the README, instead of offering `docker-compose-v2`.

## [0.13.3] - 2026-09-23

### Added

- The unit handover is pinned by a test of its own: a workload type change stops the outgoing unit, and the stop is recorded against a unit file that already holds the replacement, so the order the handover depends on cannot be reordered without failing. A redeploy of the same workload type is asserted to stop nothing, and the recording stub makes the container and static cases run wherever the tests run rather than skipping on a host without podman.
- The atomic write, XDG path resolution and per-workload-type destroy teardown the release before this one changed are covered: a rewrite replaces the file and its mode, the temp file is staged beside its target, is removed when the rename fails, and is never visible as a partial read, and a destroy tears down only the workload type it resolved, leaving the image of an unknown type alone.

## [0.13.2] - 2026-09-23

### Changed

- The unit handover every workload writer runs after it writes its unit is one function instead of three copies: guard the name, write, report, stop the workload being replaced while systemd still holds its own definition, then retire any Quadlet source an older gare left behind. That sequence decides whether an outgoing compose stack goes down through `podman compose down` or is orphaned in front of its replacement, so it can no longer drift between workload types; a test drives all three writers through a workload type change.
- The static workload's unit is named for the workload it renders: `static_unit.go`, `StaticUnitData`, `GenerateStaticUnit` and `WriteStaticUnit`, where `ContainerUnit` previously meant both the kube-play workload and the Caddy container serving a static site.
- A managed application's lookup has a home of its own rather than `app_errors.go`, and `gare deploy` uses it instead of repeating the same validation and config load.
- A rule both sides needed is now a leaf package of its own: the readiness probe type moved from `storage` to `health`, and application name validation from `storage` to `appname`. The compose file parser no longer imports the config schema, and the webhook server no longer imports storage.
- `gare env` resolves the store its workload reads once per command, through one type whose operations are named fields rather than two identically typed closures.
- `gare init` checks for `podman.socket` through the same user-session systemctl invocation as every other systemd query, instead of a bare `systemctl --user` that never received `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`, so the check reaches the user manager when gare runs outside a login shell instead of reporting the socket missing.
- `systemctlCmd` sits in `systemctl.go` beside the verbs it serves, and the delay `WaitForState` gives an already-active service is declared where the wait owns it instead of hiding inside a state comparison.

### Fixed

- A failed `gare env` write names the store it could not write — the app env file or the pod manifest — instead of reporting a generic failure, and `gare env list` names it when a read fails too.
- `gare destroy` resolves the workload type it is tearing down explicitly instead of reading a `config.json` it could not load as a container workload, which removed a compose stack's unit and storage while leaving the stack running. An unreadable configuration or an unknown type now tears down only the type-independent artifacts — the unit, the ingress snippet and the storage — and says which workload and image it left behind for `podman ps`/`podman images`.

## [0.13.1] - 2026-09-23

### Fixed

- Static workloads start again. The generated unit appended `run --config ...` to the Caddy image, which declares no `Entrypoint` — its `Cmd` carries the whole invocation — so the arguments replaced it and the runtime tried to exec a binary named `run`. Every static workload failed with status 127; the unit now names `/usr/bin/caddy` in full.
- `~/.gare.yml` takes effect. The file was parsed and discarded, so `--verbose`, `--git-provider`, `--use-github-cli`, `--use-gitlab-cli` and `--credential-helper` were documented no-ops. A flag now overrides the file only when it is actually passed, and a malformed config file fails the command instead of being ignored.
- A failed reload of a running Caddy is reported instead of being swallowed as "connection refused", which had a deploy report success while the server kept serving the previous configuration.
- `gare status` nests each section under its own heading instead of appending every line to the root as a sibling.
- `gare list` renders `CREATED` in the operator's timezone rather than the stored UTC value.
- The reverse proxy snippet is written without a stray extra tab on both lines.
- A unit that refuses to start reports its own journal first, instead of only systemd's "control process exited with error code".
- `gare app create` reports a failed rollback of a partial clone rather than returning the clone error alone.

### Changed

- Duplication removed across the CLI and unit synthesis: one systemctl runner replaces six identical wrappers, one template render-and-write pair replaces three copies across the workload types, one application lookup replaces nine, and `gare destroy`'s six single-purpose steps collapse into two.
- Storage's manifest, dotenv and compose concerns and the builder's git and podman concerns are their own packages, and one shared deploy tail replaces three near-identical pipelines.
- `gare.yml` dropped its undocumented aliases and their wrapper resolvers.

### Added

- The compose unit is verified by `systemd-analyze --user verify`, as the container and static units already were.

### Removed

- Dead symbols, a duplicated JSON encoder, and five plan documents for already-shipped features.

## [0.13.0] - 2026-09-23

### Changed

- Every workload is now supervised by a systemd unit gare writes itself at `~/.config/systemd/user/<app>.service`. `container` workloads run `podman kube play`/`podman kube down`, `static` workloads run `podman run`/`podman rm` against the bundled Caddy image, and `compose` workloads are unchanged. The units carry the same `[Unit]` and `[Service]` directives the Podman Quadlet generator produced for them, verified line for line against its output.
- `gare init` no longer requires the Quadlet generator and no longer creates `~/.config/containers/systemd/`. It probes the installed podman for the service-container flags the units pass to `podman kube play`, which makes **Podman 5.0 or newer** the supported floor where a Quadlet generator (4.4) was enough before.
- `gare destroy` removes the unit, disables it, and additionally deletes any Quadlet source an older gare left behind.

### Added

- A unit file gare did not write is no longer overwritten: `~/.config/systemd/user/<app>.service` belonging to the operator fails the deploy by name instead of being replaced, for every workload type.
- Redeploying an application already supervised by gare rewrites its unit in place and retires any Quadlet source an older gare left behind, stopping that workload first so its own teardown still runs.
- Changing an application's workload type retires the workload it replaces instead of leaving it running. The unit is rewritten and the outgoing workload stopped while its own definition is still loaded, so a compose stack goes down through `podman compose down` and a pod through `podman kube down`, rather than being orphaned in front of the new unit by a teardown that names the workload it never started.

### Removed

- Quadlet sources. gare no longer writes `~/.config/containers/systemd/<app>.kube` or `.container`, and `gare init` no longer requires the Podman generator or creates that directory. An app deployed by an older gare keeps running; its next deploy retires the sources. Compose workloads are unaffected.

### Fixed

- Compose units order after `podman-user-wait-network-online.service` instead of `network-online.target`. The user manager has no `network-online.target`, so systemd discarded both the `After=` and the `Wants=` without a warning and `podman compose up -d` could start before the network was online — which is exactly when a stack's `pull` or `build:` fails.

## [0.12.1] - 2026-09-22

### Fixed

- Static workloads deploy again on podman 4.x: the `.container` source no longer sets `Entrypoint=`, a key Quadlet only accepts from podman 5.0. On 4.4 to 4.9 the generator rejected the key and dropped the whole source, so the unit was never generated; the caddy image's own entrypoint already runs `caddy run --config /etc/caddy/Caddyfile --adapter caddyfile`.
- `TestWriteAndRemoveSnippet` no longer creates the default Caddyfile, which made the `ci` workflow fail wherever the test ran without root to write `/etc/caddy`.

## [0.12.0] - 2026-09-22

### Added

- Quadlet supervision for container and static workloads: gare writes a `.kube` source (Podman pod) or a `.container` source (bundled Caddy image) under `~/.config/containers/systemd/`, and the Podman generator produces the `<app>.service` unit and owns the pod and container lifecycle.
- `gare init` reports whether the Podman Quadlet generator is installed and creates the Quadlet source directory.
- A workload moving to Quadlet retires gare's own unit file in place — stopped while its own definition is still loaded, then removed with its enable link — so upgrading or changing workload type keeps the application's configuration, domains, tags and environment.

### Changed

- Container and static workloads are supervised by Quadlet-generated units; gare synthesizes a unit file itself only for compose, which Quadlet cannot express.
- Static sites are served by the bundled `docker.io/library/caddy:2-alpine` image with the static directory mounted read-only at `/srv` and the assigned port published, instead of a host Caddy `file-server` process. A static app now needs the Quadlet generator and pulls the Caddy image on its first deploy.
- Static content is served from a gare-written Caddyfile mounted at `/etc/caddy/Caddyfile`, because `caddy file-server` cannot express the SPA fallback (`try_files {path} /index.html`) that deep links depend on.
- Static ingress is a reverse proxy to the container port like every other workload, so the host Caddy no longer serves site content.
- Pod workloads set `ExitCodePropagation=any`, so a container that fails exits the unit non-zero and `Restart=on-failure` restarts it instead of leaving the unit `inactive (dead)`.
- Unit and Quadlet source directories both resolve through `$XDG_CONFIG_HOME` (defaulting to `~/.config`), so gare, systemd and the Podman generator agree on a host that sets it.
- Deploying an application gare created before Quadlet stops its old unit, removes it with its enable link, and writes the Quadlet source in its place; run `gare init` to create the Quadlet directory first.

### Fixed

- Changing a workload type retires the other supervision's source in both directions and stops the previous workload while its own teardown definition is still loaded, so no stale source keeps generating a unit of the same name and no orphaned pod or container is left in front of the new one.
- `gare destroy` removes the `default.target.wants` enable link it created, instead of leaving a dangling symlink pointing at a deleted unit.
- A unit file gare does not own still fails the deploy loudly, naming the path, rather than starting a workload behind a shadowing unit.

## [0.11.0] - 2026-09-20

### Added

- Compose workload type: `type: compose` in `gare.yml` runs a repository-owned compose file with `podman compose`, with `compose_file` selecting the file explicitly or falling back to discovery.
- Compose workloads require an explicit `port`, validated against the host ports the compose file publishes, so ingress never routes to a port the stack does not bind.
- Compose systemd units run detached (`up -d`) as `Type=oneshot` with `RemainAfterExit=yes` and a per-app project name via `-p`, so `start`, `stop`, `restart`, and `status` behave like every other workload type without stalling the stop path.
- Compose units require `podman.socket`, which systemd starts with the app, because the external compose provider talks to the podman API socket.
- `healthchecks:` in `gare.yml` declares one HTTP probe per published port, and `gare deploy` verifies each probe and names the one that fails.
- Compose workloads derive probes from the compose file when `gare.yml` declares none: services whose healthcheck issues an HTTP request to a published container port become probes, mapped to their published host port.
- `gare destroy` tears compose workloads down with `podman compose down -v`, independent of whether the unit file still exists, and reports containers the provider failed to remove.
- `gare deploy` preflights the external compose provider and fails with an actionable error when neither `docker-compose` nor `podman-compose` is available; `gare init` reports the provider and the podman socket.
- `--no-color` disables colored output; it is a persistent flag, so it applies to every subcommand, and the standard `NO_COLOR` environment variable is honoured too.

### Changed

- `gare app env` now stores environment variables for static and compose workloads in the same application env file, which compose units load through systemd `EnvironmentFile=`.
- Renamed the static env storage to app env storage (`storage.GetAppEnv`, `SetAppEnv`, `UnsetAppEnv`) now that it serves every non-Pod workload.
- Health configuration moved to a `health` section in `config.json` holding the primary path and the probe list; the legacy `healthcheck` key is migrated on load.
- `gare stop` accepts any non-running final state (including a provider that exited non-zero) instead of waiting out a stop timeout, and reports the result honestly.
- Unknown `type` values in `gare.yml` are rejected instead of silently falling back to a single-container Pod.
- Deployment warns when a repository contains a compose file but no `type: compose`, instead of silently ignoring it.
- Failure messages name the failure and the remedy: an unknown app points at `gare list`, a failed probe at `gare logs <app>`, and an unavailable port asks for another one.
- `gare start` and `gare restart` wait long enough for a compose cold start (320s and 400s) instead of giving up at 30s and 90s while the stack is still starting.
- The probe list reported after a deploy is capped at four names so a large stack does not print an unreadable line.

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
