# gare

A zero-daemon, rootless deployment CLI and GitOps orchestrator for Podman and Kubernetes workloads on Linux.

## Features

- **Rootless user space**: Runs under unprivileged user accounts with `systemctl --user` and Podman.
- **Direct systemd supervision**: Synthesizes systemd user units directly at `~/.config/systemd/user/<app>.service`.
- **Zero Docker**: Deploys native Kubernetes YAML manifests via `podman kube play`.
- **Automatic ingress**: Writes Caddy drop-in configuration snippets to `/etc/caddy/conf.d/<app>.caddy`.
- **Static sites**: Serves static assets directly with Caddy without containers or allocated ports.
- **Environment management**: Native `gare env` commands to set, unset, load, and inspect container environment variables.
- **Healthcheck verification**: Built-in HTTP readiness polling during deployment with configurable probes.
- **Service lifecycle**: First-class `start`, `stop`, `restart`, and detailed `status` inspection.
- **GitOps webhooks**: Built-in HTTP daemon verifying GitHub HMAC-SHA256 and GitLab tokens.
- **Passwordless operations**: Auto-detects user session buses (`XDG_RUNTIME_DIR`, `DBUS_SESSION_BUS_ADDRESS`) and Git credentials.

## Quickstart

### 1. Initialize host prerequisites

```sh
gare init
```

Verifies that `podman`, `caddy`, and `git` are available, checks user lingering (`loginctl enable-linger`), validates pause container setup, and initializes storage directories.

### 2. Create an application

```sh
gare app create myapp --repo https://github.com/example/webapp.git --domain myapp.example.com --healthcheck /health
```

Clones the repository, discovers a free TCP port (starting at 8000), synthesizes a Kubernetes pod manifest and a systemd user unit, and writes the Caddy ingress configuration.

#### Custom Containerfile or monorepos

Specify a custom Containerfile and build context path:

```sh
gare app create api --repo https://github.com/example/monorepo.git --domain api.example.com \
  --containerfile apps/api/Containerfile --context .
```

#### Static applications

Host a static site directly via Caddy without Podman, containers, or open ports:

```sh
gare app create blog --repo https://github.com/example/blog.git --domain blog.example.com \
  --static dist --build-cmd "bun run build"
```

#### Repository configuration (`gare.yml`)

Repositories can optionally define build, workload, and health check configuration in `gare.yml` or `gare.yaml` at the root:

```yaml
# For container workloads:
type: container
containerfile: apps/web/Containerfile
context: .
build_cmd: make assets
healthcheck: /health

# For static workloads:
type: static
static_dir: dist
build_cmd: bun run build
```

CLI flags take precedence over `gare.yml` settings.

### 3. Manage environment variables

Manage container environment variables directly in the Kubernetes manifest:

```sh
# Set key-value pairs
gare env set myapp DATABASE_URL=postgres://localhost/db LOG_LEVEL=info

# Load from .env file
gare env load myapp -f .env

# List configured variables
gare env list myapp
gare env list myapp --json

# Remove variables
gare env unset myapp LOG_LEVEL
```

When updated, active application services are automatically restarted to apply new environment values.

### 4. Deploy the application

```sh
gare deploy myapp
```

Pulls latest Git changes, updates Kubernetes manifests, builds the container image with rootless Podman, reloads systemd and Caddy, and verifies the readiness probe.

### 5. Lifecycle and status inspection

```sh
# Start, stop, or restart an application
gare start myapp
gare stop myapp
gare restart myapp

# Inspect detailed status and systemd properties
gare status myapp
gare status myapp --json

# View all applications in a styled terminal table
gare list
gare list --json
gare list -q

# Stream real-time logs via journalctl
gare logs myapp -f
```

### 6. Tear down an application

```sh
gare destroy myapp
```

Stops and disables the systemd service, removes the unit file, removes the Caddy snippet, prunes container images, and cleans up app storage.

## Global Configuration (`~/.gare.yml`)

Gare optionally loads settings from `~/.gare.yml`:

```yaml
verbose: false
config_path: ~/.gare.yml
git_provider: github
use_github_cli: true
use_gitlab_cli: false
credential_helper: ""
```

All configuration values can be overridden via CLI flags:
- `-c, --config`: Path to config file (default: `~/.gare.yml`)
- `-p, --git-provider`: Override git provider (`github`, `gitlab`)
- `--use-github-cli`: Force use of GitHub CLI
- `--use-gitlab-cli`: Force use of GitLab CLI
- `-H, --credential-helper`: Override credential helper
- `-v, --verbose`: Enable verbose logging

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
