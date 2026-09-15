# gare

Zero-daemon, rootless deployment CLI and GitOps orchestrator for Podman and Kubernetes workloads on Linux.

## Overview

`gare` provides lightweight, self-contained application management by embracing native Linux primitives:

- **Zero Docker**: Runs natively with rootless Podman. Workloads are standard Kubernetes Pod manifests executed via `podman kube play`.
- **Direct systemd supervision**: Process supervision, auto-restarts, boot behavior, and cgroup resource limits are managed natively by systemd user units synthesized directly without Quadlet.
- **Automatic Caddy ingress**: Caddy provides automatic Let's Encrypt TLS and reverse proxy routing by importing drop-in snippets from `/etc/caddy/conf.d/*.caddy`.
- **Stateless & file-driven**: All state is stored predictably on the filesystem (no central daemon, database, or Redis).
- **Built-in GitOps webhook**: A lightweight HTTP daemon triggers deployments upon receiving authenticated Git webhooks (GitHub HMAC-SHA256 and GitLab tokens).

## Architecture

- **Base App Store**: `~/.local/share/gare/apps/<app-name>/`
  - `repo/`: Git working copy of the project.
  - `manifest.yaml`: Kubernetes YAML file defining the pod.
  - `config.json`: Metadata tracking the assigned port, domain, repository URL, and branch.
- **Systemd User Units**: `~/.config/systemd/user/<app-name>.service`
- **Caddy Ingress**: `/etc/caddy/conf.d/<app-name>.caddy`

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/FacileStudio/gare/main/install.sh | bash
```

Installs to `~/.local/bin` via [facile](https://github.com/FacileStudio/facile), the suite installer. Pass `--bin-dir <dir>` to change that, `--source` to build from source, `--no-skill` to skip AI agent skill registration.

Already have `facile`:

```sh
facile install gare
```

## Quickstart

### 1. Initialize host prerequisites

```sh
gare init
```

Verifies that `podman`, `caddy`, and `git` are available, checks systemd user lingering (`loginctl enable-linger`), validates pause container setup, and ensures base directories exist.

### 2. Create an application

```sh
gare app create myapp --repo https://github.com/example/webapp.git --domain myapp.example.com
```

Clones the repository, discovers a free TCP port (starting at 8000), synthesizes a Kubernetes pod manifest and a systemd user unit, and writes the Caddy ingress configuration.

#### Custom Containerfile or monorepos

Specify a custom Containerfile and build context path:

```sh
gare app create api --repo https://github.com/example/monorepo.git --domain api.example.com \
  --containerfile apps/api/Dockerfile --context .
```

#### Static applications

Host a static site directly via Caddy without Podman, containers, or open ports:

```sh
gare app create blog --repo https://github.com/example/blog.git --domain blog.example.com \
  --static dist --build-cmd "bun run build"
```

#### Repository configuration (`gare.yml`)

Repositories can optionally define build and workload configuration in `gare.yml` or `gare.yaml` at the root:

```yaml
# For container workloads:
type: container
containerfile: apps/web/Dockerfile
context: .
build_cmd: make assets

# For static workloads:
type: static
static_dir: dist
build_cmd: bun run build
```

CLI flags always take precedence over `gare.yml` settings.

### 3. Deploy the application

```sh
gare deploy myapp
```

Pulls latest Git changes, re-synchronizes Kubernetes manifests, builds the container image with rootless Podman, and reloads systemd and Caddy.

### 4. Inspect status and logs

```sh
# View all applications in a styled terminal table
gare list

# Machine-readable output for scripts
gare list --json
gare list -q

# Stream real-time logs via journalctl
gare logs myapp -f
```

### 5. Tear down an application

```sh
gare destroy myapp
```

Stops and disables the systemd service, removes the unit file, removes the Caddy snippet, prunes container images, and cleans up app storage.

## GitOps Webhook Daemon

Start the webhook receiver to automate deploys on push:

```sh
gare server --port 8080 --secret your-webhook-secret
```

Endpoints:
- `POST /webhook/<app-name>`: Authenticated webhook endpoint.
- `GET /health`: Healthcheck probe.

Supported authentication:
- GitHub: `X-Hub-Signature-256` header (HMAC-SHA256).
- GitLab: `X-Gitlab-Token` header.

## Development

`gare` uses [mise](https://mise.sh) for task management.

```sh
mise run build        # build bin/gare
mise run test         # run test suite
mise run all          # build and test
mise run lint         # run go vet ./...
```

Code quality and architectural constraints are enforced by [filet](https://github.com/FacileStudio/filet).