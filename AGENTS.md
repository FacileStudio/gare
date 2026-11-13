# gare

A zero-daemon, rootless deployment CLI and GitOps orchestrator for Podman and Kubernetes workloads on Linux.

## Commands

```sh
mise run build        # build bin/gare from apps/api
mise run test         # run unit and package tests
mise run all          # build and test
mise run dev          # run local application
mise run lint         # run go vet ./...
mise run check        # the full gate: sh scripts/check.sh
mise run help         # display available tasks
```

The gate is `sh scripts/check.sh`: gofmt, `go vet` and `go test` in `apps/api`, the client check once `apps/client/package.json` exists, then `filet check .`. `--go-only` skips the client and filet; `--format` rewrites the Go sources. It is POSIX sh, because `/bin/sh` on the deployment host is dash.

Tests that need a working podman are skipped on a machine that cannot run the generated units, and fail instead under `GARE_REQUIRE_PODMAN=1`, which CI sets. CI runs on the `ubuntu-26.04` image for its podman 5.7: the 24.04 image was rolled back to podman 4.9, which cannot run these units at all.

## Releasing

The version lives in two places and both move in the same `chore: release vX` commit: the git tag, which GoReleaser injects with `-X main.version={{.Version}}`, and the fallback literal in `apps/api/version.go`, which is what a plain `go build` reports. Bump only the tag and the repository declares the previous version to everyone who builds it from source.

## Architecture

`gare` supervises every workload through a systemd user unit it writes itself at `~/.config/systemd/user/<app-name>.service`: `container` workloads get `podman kube play`/`podman kube down` around their pod manifest, `static` workloads get `podman run`/`podman rm` around the bundled Caddy image serving `static_dir`, and `compose` workloads get `podman compose up -d`/`down`.

- Rootless user space: runs under unprivileged accounts with `systemctl --user`.
- Zero Docker: relies entirely on `podman kube play`/`podman kube down` for `container` workloads, `podman run` for `static`, and `podman compose up`/`down` for `compose` workloads declared in `gare.yml`.
- Supervision: gare-synthesized systemd user units for every workload type. gare owns the pod and container lifecycle, so each unit spells out the `podman` command it runs, including `ExecStop`/`ExecStopPost` teardown.
- Static serving: the unit runs `docker.io/library/caddy:2-alpine` with the static directory bind-mounted read-only at `/srv` and a host port published to container port 80. No host Caddy binary serves site content; the host Caddy is ingress only.
- Static containers are addressed through a cidfile (`%t/%N.cid`) so `ExecStop` can remove the exact container the unit started.
- Static content is served from a gare-written Caddyfile (`~/.local/share/gare/apps/<app-name>/Caddyfile`) mounted at `/etc/caddy/Caddyfile`, not from `caddy file-server` flags: `file-server` cannot express the SPA fallback (`try_files {path} /index.html`) that static sites depend on for deep links.
- Ingress: Caddy drop-in snippets at `/etc/caddy/conf.d/<app-name>.caddy`.
- Storage: predictable filesystem paths under `$XDG_DATA_HOME/gare/apps/<app-name>/`, defaulting to `~/.local/share/gare/apps/<app-name>/`.
- Server: lightweight webhook receiver verifying HMAC-SHA256 signatures.

## Why gare ships as a host binary

Other suite apps ship as a distroless image deployed through Dokploy. gare does not, and has no root `Dockerfile`. It writes systemd user units, runs `systemctl --user`, drives the host's rootless podman and reloads the host Caddy, so it must run as the unprivileged user that owns those units, on the host itself. A container would need the user's systemd bus, podman socket and home directory mounted in, which is the host with extra steps. The release archive carries the `gare` binary, and `facile install gare` installs it.

## Directory layout

```
apps/api/               the Go module github.com/FacileStudio/gare/apps/api
  main.go               the only package main: run() calls cli.Execute(version)
  version.go            the version literal GoReleaser overrides with -X main.version
  cli/                  Cobra commands (package cli, entry point Execute): flags, prompts, rendering and the terminal sink
    config/             `~/.gare.yml` loading and the flag overrides that outrank it
  internal/
    appconfig/          Resolve: the one precedence rule for an app's settings (flag, gare.yml, stored, default)
    appname/            application name validation shared by the CLI and the webhook server
    atomicfile/         crash-safe atomic file writes
    caddy/              caddy configuration snippets and reloads
    compose/            compose file discovery, published ports, derived probes
    dotenv/             dotenv parsing and writing
    engine/             the app lifecycle: create, deploy, start, stop, restart, destroy, status, env, domain and tag edits
    git/                git clone, fetch-and-reset sync, commit lookup and credential plumbing
    health/             the readiness probe type and HTTP polling
    manifest/           Pod manifest generation and env/port rewriting
    podman/             podman build, image, Containerfile and compose CLI invocation
    repopath/           keeps configured paths (static_dir, context, containerfile, compose_file) inside the repo
    server/             webhook HTTP daemon and HMAC verification
    storage/            app state, workload types, tags, domains, health, ports
    systemd/            unit file synthesis and systemctl operations
    testkit/            test-only helpers: recording podman stub, unit file fixtures
    workload/           one adapter per workload type (container, static, compose), selected by workload.For
    xdg/                XDG base directory resolution for the current user
scripts/check.sh        the quality gate (POSIX sh)
mise.toml               root tasks; apps/api/mise.toml holds the module's own
```

## Workload types

`gare.yml` selects the workload type and gare never guesses: `container` (Kubernetes Pod manifest via `kube play`), `static` (bundled Caddy container serving a directory), and `compose` (a repo-owned compose file executed by `podman compose`).

- `workload.For` turns the type into the adapter (`internal/workload/container.go`, `static.go`, `compose.go`) that settles, prepares, tears down and stores the env of that type. The engine never asks which type it holds; type-specific behaviour goes into the adapters, so a fourth type is one new file.

- Every setting follows one precedence, at create and at every deploy alike, decided by `appconfig.Resolve` only: a flag the operator passed, then `gare.yml`, then the value stored in `config.json`, then the default. Never merge `gare.yml` into a configuration anywhere else, or create and deploy drift apart again.
- Compose workloads require an explicit `port` that the compose file actually publishes, because the host port belongs to the compose file and gare only routes to it.
- Compose units are `Type=oneshot` with `RemainAfterExit=yes`: `up -d` runs detached, `Requires=podman.socket` pulls in the API socket the external provider talks to, and stop/restart run `compose down`. Never attach the unit to a watching provider process: a `Type=exec` unit whose `ExecStop`/`ExecStopPost` tears the stack down marks itself failed and stalls for the stop timeout.
- Each compose unit passes `-p <app-name>` so every app owns its own project; without it, every app's checkout is named `repo` and `down` targets the wrong stack.
- Compose file names come from `compose_file` or discovery (`compose.yml`, `compose.yaml`, `docker-compose.yml`, `docker-compose.yaml`); a name with whitespace or path traversal is rejected before it reaches a unit file.
- `healthchecks:` in `gare.yml` declares one HTTP probe per published port; deploy verifies each and names the failing probe. Probes live in `health` in `config.json`, with the legacy `healthcheck` key migrated on load.
- A declared probe (a `healthcheck` path, a `healthchecks` entry or a compose-derived probe) passes only on 2xx or 3xx. The implicit probe gare adds when none is declared sends `GET /` and passes on any HTTP answer, for the same reason the ingress probe never inspects the status: an API with no root route answers 404 at `/` and is up. `health.Probe.AnyAnswer` carries that distinction and is never serialised, so a stored probe is always a declared one.
- When `healthchecks:` is absent for a compose workload, probes are derived from the compose file: only services whose healthcheck issues an HTTP request to a *published* container port qualify, so container-internal checks (and unpublished ports) are skipped rather than probed and failed.

## Rules and constraints

- Do not use Cgo. Produce a single static binary.
- Never add inline comments in Go code or comments in function bodies.
- Only `apps/api/main.go` (with `apps/api/version.go`) is `package main`. The Cobra commands live in `apps/api/cli` as `package cli`, so a future daemon or test can import them, and `main` only maps `cli.Execute` to an exit status.
- Write systemd service units and Caddy configurations atomically: write temporary file, sync, rename.
- Stream stdout and stderr directly to the terminal during long-running tasks (`podman build`, `git clone`, `git fetch`). The engine never prints: it hands subprocesses the writers its `Sink` returns from `Output`, and reports every progress line through `Sink.Event`, so the CLI's terminal sink, the webhook daemon and the API all run the same code.
- Clean up resources completely on destroy: stop unit, disable it, remove the unit file, its enable link and any Quadlet source an older gare left behind, remove Caddy snippet, remove storage directory, remove local container image, reload systemd and Caddy. Compose named volumes go only with `--volumes`, and a destroy runs only with `--yes` or the app name typed back on a terminal, because it cannot be undone.
- Fail fast when podman cannot run the generated units: probe `podman kube play` for `--service-container` before touching anything else, so a podman older than 5.0 cannot leave a workload stopped or half-retired.
- Every `localhost/<name>` image a generated or adopted Pod manifest runs carries `imagePullPolicy: Never`. The image exists only in Podman's local storage, since gare builds it, so an unset policy leaves `podman kube play` to resolve it as `docker://localhost/<name>` and fail wherever nothing answers as a registry on `localhost` — a restart loop whose journal looks like a deployment bug.
- A deploy of an app with domains is not complete until its ingress is: the running Caddy must accept the snippet and answer an HTTP request for every hostname the app is configured with, because a snippet on disk is not the configuration a running server serves and a bound socket says nothing about which hostnames it answers. The probe reports only whether the ingress answered, never what it answered, since an API served at the root of its own hostname may legitimately answer 404 and Caddy answers 502 for a workload that is not up yet — the probe runs before the restart, so inspecting the status would fail the deploy it exists to guard. It does not verify the certificate either, since the Caddy on this host may serve an internal one. An app reachable on its assigned port alone only warns, since nothing then depends on ingress.
- Image pruning after a deploy is best-effort and reported only in verbose mode: `podman image prune -f` exits non-zero when a leftover buildah working container holds a dangling image, which says nothing about the deployment.
- Tenable restarts: the kube unit must pass `--service-exit-code-propagation=any` to `podman kube play`, because podman otherwise exits the service zero even when a container failed, leaving `Restart=on-failure` inert.
- Order user units after `podman-user-wait-network-online.service`, never `network-online.target`: the user manager has no `network-online.target` (`LoadState=not-found`), and systemd discards `After=`/`Wants=` on an unknown unit without warning, so the workload silently ends up with no network ordering at all.
- Never overwrite a unit file gare did not write: `~/.config/systemd/user/<app-name>.service` belongs to the operator until gare writes it. gare's own file is rewritten in place, so changing workload type costs nothing, while any other occupant fails loudly instead of deploying silently behind it.
- One unit source per app: every workload type writes the same `<app-name>.service` path rather than coexisting, so a workload type change cannot leave two definitions racing for one name.
- A workload type change retires the workload it replaces: write the new unit first, then stop the old one before any `daemon-reload`. systemd keeps serving the loaded definition until it reloads, so the stop still runs the outgoing unit's own `ExecStop` — a compose stack goes down through `podman compose down` rather than being replaced by a `kube down` that names no pod, which would leave the stack running in front of the new unit.
- Each template's `Description=` comes from `systemd.KubeUnitDescription`, `StaticUnitDescription` or `ComposeUnitDescription`, and `GareUnitDescriptions` proves the set. Those strings are the ownership marker that tells a deploy whether it may replace a unit file, so never spell one out in a template or a guard: a description that drifts from its template makes gare refuse to redeploy its own unit.
- Retire legacy Quadlet sources on write: an older gare left `~/.config/containers/systemd/<app-name>.kube` or `.container` for the Podman generator. Remove them, stopping the workload first, because after `daemon-reload` the generated definition is gone and its own teardown never runs, leaving an orphaned pod or container in front of the new unit.
- Resolve user paths through the XDG base directories: `$XDG_CONFIG_HOME` (defaulting to `~/.config`) locates the systemd user units and `$XDG_DATA_HOME` (defaulting to `~/.local/share`) locates application storage, so gare agrees with systemd and the rest of the desktop on a host that sets them.
- Every documented global flag must change behaviour. The `~/.gare.yml` loader feeds verbosity and git authentication, and only a flag the operator actually passed may override the file, so a setting in the file survives unless the command line contradicts it.
- Paths written into a unit must be absolute and free of whitespace: systemd splits `ExecStart` on whitespace without honouring quotes, so a space silently becomes an extra argument and the unit starts the wrong command.
- Every engine operation that changes an app (create, deploy, destroy, and env, domain and tag edits) holds the app's lock for its whole run: `storage.Store.Lock`, an `flock` on `<base>/.locks/<app-name>.lock`. The lock file lives outside the app directory so destroy keeps holding it while it deletes that directory, and it is never removed, because unlinking a lock file someone waits on lets a third process lock a fresh file of the same name. Load `config.json` only after taking the lock, or the command saves over what the previous holder wrote.
- Port discovery begins at port 8000 and increments upwards, avoiding both recorded app ports and active system listeners.
