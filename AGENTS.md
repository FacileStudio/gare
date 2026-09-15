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

`gare` synthesizes systemd user units directly. It uses no Quadlet and no Compose wrappers.

- Rootless user space: runs under unprivileged accounts with `systemctl --user`.
- Zero Docker: relies entirely on `podman kube play` and `podman kube down`.
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

## Rules and constraints

- Do not use Cgo. Produce a single static binary.
- Never add inline comments in Go code or comments in function bodies.
- Package `cmd/gare` must remain `package main`.
- Write systemd service units and Caddy configurations atomically: write temporary file, sync, rename.
- Stream stdout and stderr directly to the terminal during long-running tasks (`podman build`, `git clone`, `git pull`).
- Clean up resources completely on destroy: stop and disable unit, remove unit file, remove Caddy snippet, remove storage directory, remove local container image, reload systemd and Caddy.
- Port discovery begins at port 8000 and increments upwards, avoiding both recorded app ports and active system listeners.
