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

`gare` supervises container and static workloads through Quadlet: it writes a `.kube` source (Podman pod) or a `.container` source (bundled Caddy image serving `static_dir`) at `~/.config/containers/systemd/<app-name>.<type>` and lets the Podman generator produce the `<app-name>.service` unit. Compose workloads keep a gare-synthesized unit at `~/.config/systemd/user/<app-name>.service`, because Quadlet has no compose support.

- Rootless user space: runs under unprivileged accounts with `systemctl --user`.
- Zero Docker: relies entirely on `podman kube play`/`podman kube down` for `container` workloads and `podman compose up`/`down` for `compose` workloads declared in `gare.yml`.
- Supervision: Quadlet-generated systemd user units for `container` and `static` workloads, synthesized units for `compose`. Quadlet sources never carry `ExecStart`/`ExecStop`: the generator owns the pod and container lifecycle, so gare must not reintroduce `podman kube play` or `podman run` commands of its own.
- Static serving: a `.container` source runs `docker.io/library/caddy:2-alpine` with the static directory bind-mounted read-only at `/srv` and a host port published to container port 80. No host Caddy binary serves site content; the host Caddy is ingress only.
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
  quadlet/            Quadlet source files supervising Podman workloads
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
- Clean up resources completely on destroy: stop unit, disable it when gare synthesized it, remove the unit file and the Quadlet source, remove Caddy snippet, remove storage directory, remove local container image, reload systemd and Caddy.
- Fail fast when the Quadlet generator is missing: never fall back to hand-written container units.
- Tenable restarts: a `.kube` source must carry `ExitCodePropagation=` in `[Kube]`, because Quadlet's default (`none`) exits the service zero even when a container failed, leaving `Restart=on-failure` inert.
- Never write a Quadlet source that a unit file would shadow: `~/.config/systemd/user/<app-name>.service` outranks the generated unit, so a Quadlet-backed workload fails loudly when that path is occupied instead of deploying silently behind it.
- One Quadlet source per app: writing a `.kube` or `.container` source retires the other, because two sources in one directory would generate the same `<app-name>.service` and a workload type change must not leave both behind.
- Paths written into a Quadlet source must be absolute and free of whitespace: Quadlet re-quotes `Yaml=` but not `Volume=`/`EnvironmentFile=`, so a space silently breaks the generated mount.
- Port discovery begins at port 8000 and increments upwards, avoiding both recorded app ports and active system listeners.
