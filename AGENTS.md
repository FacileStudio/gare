# gare

A zero-daemon, rootless deployment CLI and GitOps orchestrator for Podman and Kubernetes workloads on Linux.

## Commands

```sh
mise run build        # build bin/gare
mise run test         # run unit and package tests
mise run all          # build and test
mise run dev          # run local application
mise run lint         # run go vet ./...
mise run help         # display available tasks
```

Tests that need a working podman are skipped on a machine that cannot run the generated units, and fail instead under `GARE_REQUIRE_PODMAN=1`, which CI sets. CI runs on the `ubuntu-26.04` image for its podman 5.7: the 24.04 image was rolled back to podman 4.9, which cannot run these units at all.

## Architecture

`gare` supervises every workload through a systemd user unit it writes itself at `~/.config/systemd/user/<app-name>.service`: `container` workloads get `podman kube play`/`podman kube down` around their pod manifest, `static` workloads get `podman run`/`podman rm` around the bundled Caddy image serving `static_dir`, and `compose` workloads get `podman compose up -d`/`down`.

- Rootless user space: runs under unprivileged accounts with `systemctl --user`.
- Zero Docker: relies entirely on `podman kube play`/`podman kube down` for `container` workloads, `podman run` for `static`, and `podman compose up`/`down` for `compose` workloads declared in `gare.yml`.
- Supervision: gare-synthesized systemd user units for every workload type. gare owns the pod and container lifecycle, so each unit spells out the `podman` command it runs, including `ExecStop`/`ExecStopPost` teardown.
- Static serving: the unit runs `docker.io/library/caddy:2-alpine` with the static directory bind-mounted read-only at `/srv` and a host port published to container port 80. No host Caddy binary serves site content; the host Caddy is ingress only.
- Static containers are addressed through a cidfile (`%t/%N.cid`) so `ExecStop` can remove the exact container the unit started.
- Static content is served from a gare-written Caddyfile (`~/.local/share/gare/apps/<app-name>/Caddyfile`) mounted at `/etc/caddy/Caddyfile`, not from `caddy file-server` flags: `file-server` cannot express the SPA fallback (`try_files {path} /index.html`) that static sites depend on for deep links.
- Ingress: Caddy drop-in snippets at `/etc/caddy/conf.d/<app-name>.caddy`.
- Storage: predictable filesystem paths under `~/.local/share/gare/apps/<app-name>/`.
- Server: lightweight webhook receiver verifying HMAC-SHA256 signatures.

## Directory layout

```
cmd/gare/             CLI entrypoint and Cobra commands (package main)
internal/
  atomicfile/         crash-safe atomic file writes
  builder/            git operations and container image builds
  caddy/              caddy configuration snippets and reloads
  server/             webhook HTTP daemon and HMAC verification
  storage/            app state, manifests, and port discovery
  systemd/            unit file synthesis and systemctl operations
  xdg/                XDG base directory resolution for the current user
```

## Workload types

`gare.yml` selects the workload type and gare never guesses: `container` (Kubernetes Pod manifest via `kube play`), `static` (bundled Caddy container serving a directory), and `compose` (a repo-owned compose file executed by `podman compose`).

- Compose workloads require an explicit `port` that the compose file actually publishes, because the host port belongs to the compose file and gare only routes to it.
- Compose units are `Type=oneshot` with `RemainAfterExit=yes`: `up -d` runs detached, `Requires=podman.socket` pulls in the API socket the external provider talks to, and stop/restart run `compose down`. Never attach the unit to a watching provider process: a `Type=exec` unit whose `ExecStop`/`ExecStopPost` tears the stack down marks itself failed and stalls for the stop timeout.
- Each compose unit passes `-p <app-name>` so every app owns its own project; without it, every app's checkout is named `repo` and `down` targets the wrong stack.
- Compose file names come from `compose_file` or discovery (`compose.yml`, `compose.yaml`, `docker-compose.yml`, `docker-compose.yaml`); a name with whitespace or path traversal is rejected before it reaches a unit file.
- `healthchecks:` in `gare.yml` declares one HTTP probe per published port; deploy verifies each and names the failing probe. Probes live in `health` in `config.json`, with the legacy `healthcheck` key migrated on load.
- When `healthchecks:` is absent for a compose workload, probes are derived from the compose file: only services whose healthcheck issues an HTTP request to a *published* container port qualify, so container-internal checks (and unpublished ports) are skipped rather than probed and failed.

## Rules and constraints

- Do not use Cgo. Produce a single static binary.
- Never add inline comments in Go code or comments in function bodies.
- Package `cmd/gare` must remain `package main`.
- Write systemd service units and Caddy configurations atomically: write temporary file, sync, rename.
- Stream stdout and stderr directly to the terminal during long-running tasks (`podman build`, `git clone`, `git pull`).
- Clean up resources completely on destroy: stop unit, disable it, remove the unit file, its enable link and any Quadlet source an older gare left behind, remove Caddy snippet, remove storage directory, remove local container image, reload systemd and Caddy.
- Fail fast when podman cannot run the generated units: probe `podman kube play` for `--service-container` before touching anything else, so a podman older than 5.0 cannot leave a workload stopped or half-retired.
- Tenable restarts: the kube unit must pass `--service-exit-code-propagation=any` to `podman kube play`, because podman otherwise exits the service zero even when a container failed, leaving `Restart=on-failure` inert.
- Order user units after `podman-user-wait-network-online.service`, never `network-online.target`: the user manager has no `network-online.target` (`LoadState=not-found`), and systemd discards `After=`/`Wants=` on an unknown unit without warning, so the workload silently ends up with no network ordering at all.
- Never overwrite a unit file gare did not write: `~/.config/systemd/user/<app-name>.service` belongs to the operator until gare writes it. gare's own file is rewritten in place, so changing workload type costs nothing, while any other occupant fails loudly instead of deploying silently behind it.
- One unit source per app: every workload type writes the same `<app-name>.service` path rather than coexisting, so a workload type change cannot leave two definitions racing for one name.
- A workload type change retires the workload it replaces: write the new unit first, then stop the old one before any `daemon-reload`. systemd keeps serving the loaded definition until it reloads, so the stop still runs the outgoing unit's own `ExecStop` — a compose stack goes down through `podman compose down` rather than being replaced by a `kube down` that names no pod, which would leave the stack running in front of the new unit.
- Each template's `Description=` comes from `systemd.KubeUnitDescription`, `StaticUnitDescription` or `ComposeUnitDescription`, and `GareUnitDescriptions` proves the set. Those strings are the ownership marker that tells a deploy whether it may replace a unit file, so never spell one out in a template or a guard: a description that drifts from its template makes gare refuse to redeploy its own unit.
- Retire legacy Quadlet sources on write: an older gare left `~/.config/containers/systemd/<app-name>.kube` or `.container` for the Podman generator. Remove them, stopping the workload first, because after `daemon-reload` the generated definition is gone and its own teardown never runs, leaving an orphaned pod or container in front of the new unit.
- Resolve user paths through the XDG base directories: `$XDG_CONFIG_HOME` (defaulting to `~/.config`) locates the systemd user units, so gare agrees with systemd on a host that sets it.
- Paths written into a unit must be absolute and free of whitespace: systemd splits `ExecStart` on whitespace without honouring quotes, so a space silently becomes an extra argument and the unit starts the wrong command.
- Port discovery begins at port 8000 and increments upwards, avoiding both recorded app ports and active system listeners.
