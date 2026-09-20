# gare

A zero-daemon, rootless deployment CLI and GitOps orchestrator for Podman and Kubernetes workloads on Linux.

## Features

- **Rootless user space**: Runs under unprivileged user accounts with `systemctl --user` and Podman.
- **Direct systemd supervision**: Synthesizes systemd user units directly at `~/.config/systemd/user/<app>.service`.
- **Zero Docker**: Deploys native Kubernetes YAML manifests via `podman kube play`, or multi-container stacks via `podman compose`.
- **Automatic ingress**: Writes Caddy drop-in configuration snippets to `/etc/caddy/conf.d/<app>.caddy`.
- **Static sites**: Serves static assets directly with Caddy without containers or allocated ports.
- **Environment management**: Native `gare env` commands to set, unset, load, and inspect container environment variables.
- **Application tags**: Group and filter workloads with `gare tag` commands, `gare.yml` metadata, and `--tag` list filters.
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
gare app create myapp --repo https://github.com/example/webapp.git --healthcheck /health
gare domain add myapp myapp.example.com
```

Clones the repository, discovers a free TCP port (starting at 8000), synthesizes a Kubernetes pod manifest and a systemd user unit, and records metadata in config.json. Use `gare domain add` to attach a hostname and write the Caddy ingress snippet.

#### Custom Containerfile or monorepos

Specify a custom Containerfile and build context path:

```sh
gare app create api --repo https://github.com/example/monorepo.git \
  --containerfile apps/api/Containerfile --context .
gare domain add api api.example.com
```

#### Static applications

Host a static site directly via Caddy without Podman, containers, or open ports:

```sh
gare app create blog --repo https://github.com/example/blog.git \
  --static dist --build-cmd "bun run build"
gare domain add blog blog.example.com
```

#### Repository configuration (`gare.yml`)

Repositories can optionally define build, workload, and health check configuration in `gare.yml` or `gare.yaml` at the root:

Container workload:

```yaml
type: container
containerfile: apps/web/Containerfile
context: .
build_cmd: make assets
healthcheck: /health
tags:
  - backend
  - production
```

Static workload:

```yaml
type: static
static_dir: dist
build_cmd: bun run build
tags:
  - frontend
```

Compose workload:

```yaml
type: compose
compose_file: docker-compose.yml
port: 8200
healthchecks:
  - name: web
    port: 8200
    path: /health
  - name: admin
    port: 8201
    path: /
```

The `healthchecks` list declares one HTTP readiness probe per workload (or per published port); each entry takes an optional `name`, a `port`, and a `path`. `gare deploy` verifies every probe and names the one that fails.

When a compose file already declares healthchecks, gare derives the probes for you instead: every service whose healthcheck issues an HTTP request to a published container port becomes a probe, named after the service, with the port mapped to its published host port. Non-HTTP healthchecks (such as `pg_isready`) and unpublished ports are skipped, since they cannot be reached from the host. An explicit `healthchecks` list always wins. Without either, the single `port` plus `healthcheck` path is used.

CLI flags take precedence over `gare.yml` settings.

#### Compose workloads

Repositories with a `docker-compose.yml` or `compose.yml` can run it as-is by declaring `type: compose`:

```sh
gare app create stack --repo https://github.com/example/stack.git
gare domain add stack stack.example.com
gare deploy stack
```

`podman compose` requires an external provider (`docker-compose` or `podman-compose`) and the podman API socket. `gare init` reports both, and `gare deploy` fails with a clear error when the provider is missing. Additional behavior:

- `port` is required in `gare.yml` and must match a published host port in the compose file, so Caddy routes to the port the stack actually binds.
- The systemd unit is a `RemainAfterExit` oneshot that requires `podman.socket` (started with the app) and runs the stack detached from the repository checkout, so relative build contexts, bind mounts, and `env_file` paths resolve as written. `stop` and `restart` run `podman compose down`; containers are never left to a watching provider process.
- Each application gets its own compose project name (the app name), so two apps never share containers, networks, or volumes.
- `gare env set` writes to the application env file, which systemd injects into the provider through `EnvironmentFile=` for `${VAR}` interpolation.
- Health probes are derived from the compose file's own HTTP healthchecks when `gare.yml` declares none, so a stack that is already health-checked does not need its ports repeated.
- `gare destroy` runs `podman compose down -v`, removing the stack's containers, networks, **and named volumes**, and reports any container the provider failed to remove instead of claiming a clean teardown.
- Compose images built by `build:` sections are not removed by `gare destroy`; prune them with `podman image prune` when needed.

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

### 4. Manage domains

Attach, inspect, and remove ingress hostnames:

```sh
# Add domain(s) to an app
gare domain add myapp myapp.example.com
gare domain add myapp www.myapp.example.com

# List configured domains
gare domain list
gare domain list myapp
gare domain list --json

# Remove a domain
gare domain rm myapp www.myapp.example.com
```

When an application is active, adding or removing domains automatically updates the Caddy snippet and reloads the proxy.

### 5. Manage tags

Group, categorize, and inspect application tags:

```sh
# Add tag(s) to an app
gare tag add myapp client-acme prod api

# List configured tags across all applications or for a single app
gare tag list
gare tag list myapp
gare tag list --json

# Remove tag(s) from an app
gare tag rm myapp api

# Filter application list by tag
gare list --tag client-acme
gare list -t prod
```

### 6. Deploy the application

```sh
gare deploy myapp
```

Pulls latest Git changes, updates Kubernetes manifests, builds the container image with rootless Podman, reloads systemd and Caddy, and verifies the readiness probe. Compose workloads skip the image build and manifest sync and validate the compose file instead.

### 7. Lifecycle and status inspection

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
gare list --tag client-acme
gare list --json
gare list -q

# Stream real-time logs via journalctl
gare logs myapp -f
```

### 8. Tear down an application

```sh
gare destroy myapp
```

Stops and disables the systemd service, removes the unit file, removes the Caddy snippet, prunes container images, and cleans up app storage. For compose workloads, it additionally runs `podman compose down -v` so no containers, networks, or volumes leak if the unit file is already gone.

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
- `--git-provider`: Override git provider (`github`, `gitlab`)
- `--use-github-cli`: Force use of GitHub CLI
- `--use-gitlab-cli`: Force use of GitLab CLI
- `--credential-helper`: Override credential helper
- `-v, --verbose`: Enable verbose logging
- `--no-color`: Disable colored output (also honoured via the `NO_COLOR` environment variable)

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
