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

## Architecture

`gare` synthesizes systemd user units directly. It uses no Quadlet and no generated Compose files.

- Rootless user space: runs under unprivileged accounts with `systemctl --user`.
- Zero Docker: relies entirely on `podman kube play`/`podman kube down` for `container` workloads and `podman compose up`/`down` for `compose` workloads declared in `gare.yml`.
- Supervision: systemd user units at `~/.config/systemd/user/<app-name>.service`.
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
```

## Workload types

`gare.yml` selects the workload type and gare never guesses: `container` (Kubernetes Pod manifest via `kube play`), `static` (Caddy file server), and `compose` (a repo-owned compose file executed by `podman compose`).

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
- Clean up resources completely on destroy: stop and disable unit, remove unit file, remove Caddy snippet, remove storage directory, remove local container image, reload systemd and Caddy.
- Port discovery begins at port 8000 and increments upwards, avoiding both recorded app ports and active system listeners.
