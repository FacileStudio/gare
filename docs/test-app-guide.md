# Creating a test application with gare

This guide walks through building, configuring, deploying, and destroying a test application using `gare`.

`gare` runs containerized workloads using rootless Podman, supervises them with systemd user units, and configures TLS ingress through Caddy drop-in snippets. It requires no central daemon, no Docker socket, and no root privileges during routine operations.

## Host prerequisites

Verify four prerequisites before creating an application:

1. `podman`, `caddy`, and `git` are installed and present in `PATH`.
2. Systemd user lingering is enabled for the target user. Lingering ensures systemd user units start on boot and remain running after logout:

```sh
loginctl enable-linger $USER
```

3. The Podman pause container or `catatonit` init binary is available. If using an infra pause image, configure it in `~/.config/containers/containers.conf`:

```ini
[engine]
infra_image = "registry.k8s.io/pause:3.9"
```

4. The Caddy snippet directory exists and is writable by your user account:

```sh
sudo mkdir -p /etc/caddy/conf.d
sudo chown -R $USER: /etc/caddy/conf.d
```

Ensure your main `/etc/caddy/Caddyfile` contains the import directive:

```caddyfile
import /etc/caddy/conf.d/*.caddy
```

Enable and start Caddy:

```sh
sudo systemctl enable --now caddy
```

Run `gare init` to confirm your host passes all checks:

```sh
gare init
```

The command verifies dependencies, creates `~/.local/share/gare/apps/` and `~/.config/systemd/user/`, and confirms Caddy directory permissions.

## Build a minimal test application

`gare` requires a Git repository containing either a `Dockerfile` or a `Containerfile`.

Create a local Git repository for testing:

```sh
mkdir -p /tmp/gare-test-app
cd /tmp/gare-test-app
git init -b main
```

Create a simple HTTP server in Go. Write `main.go`:

```go
package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from gare test app!\nHost: %s\nPath: %s\n", r.Host, r.URL.Path)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	fmt.Printf("Server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
```

Write a `Containerfile` or `Dockerfile`:

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY main.go .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/app main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/app /bin/app
EXPOSE 8080
ENTRYPOINT ["/bin/app"]
```

Commit the test app to git:

```sh
git add main.go Containerfile
git commit -m "feat: initial test app commit"
```

Push this repository to a remote host (GitHub, GitLab, or a local bare repository). For a local bare repo test:

```sh
git clone --bare /tmp/gare-test-app /tmp/gare-test-app.git
```

## Create the application in gare

Register the application with `gare app create`:

```sh
gare app create test-app \
  --repo /tmp/gare-test-app.git \
  --domain test.example.com \
  --branch main
```

You can optionally specify `--port <port>`. If omitted or set to 0, `gare` scans ports starting from 8000 to find an unused port.

`gare app create` performs six operations:

1. Clones the repository into `~/.local/share/gare/apps/test-app/repo`.
2. Resolves or generates the Kubernetes Pod manifest at `~/.local/share/gare/apps/test-app/manifest.yaml`.
3. Synthesizes a systemd user unit at `~/.config/systemd/user/test-app.service`.
4. Generates an ingress snippet at `/etc/caddy/conf.d/test-app.caddy`.
5. Records metadata in `~/.local/share/gare/apps/test-app/config.json`.
6. Prints the assigned port and domain confirmation.

### Inspect the generated artifacts

View the synthesized Kubernetes Pod manifest:

```sh
cat ~/.local/share/gare/apps/test-app/manifest.yaml
```

The manifest exposes the allocated port and maps it to `containerPort`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-app
  labels:
    app: test-app
spec:
  containers:
  - name: test-app
    image: localhost/test-app:latest
    ports:
    - containerPort: 8000
      hostPort: 8000
```

View the synthesized systemd unit:

```sh
cat ~/.config/systemd/user/test-app.service
```

Notice the systemd unit calls `podman kube play --replace -w`:

```ini
[Unit]
Description=Gare Managed App: test-app
After=network-online.target
Wants=network-online.target

[Service]
Environment=PODMAN_SYSTEMD_UNIT=%n
Type=exec
KillMode=mixed
Restart=on-failure
RestartSec=5s
ExecStart=/usr/bin/podman kube play --replace -w /home/yann/.local/share/gare/apps/test-app/manifest.yaml
ExecStopPost=-/usr/bin/podman kube down /home/yann/.local/share/gare/apps/test-app/manifest.yaml
SyslogIdentifier=%N

[Install]
WantedBy=default.target
```

View the Caddy configuration snippet:

```sh
cat /etc/caddy/conf.d/test-app.caddy
```

```caddyfile
test.example.com {
	reverse_proxy localhost:8000
}
```

## Deploy the application

Run the deployment pipeline:

```sh
gare deploy test-app
```

`gare deploy` executes the deployment steps in order:

1. Pulls the latest commits from the Git repository.
2. Checks for `Containerfile` or `Dockerfile`.
3. Builds the container image tagged `localhost/test-app:latest` using rootless Podman.
4. Synchronizes any changes from `repo/manifest.yaml` if defined in the repository.
5. Runs `systemctl --user daemon-reload`.
6. Enables and restarts the systemd user unit `test-app.service`.
7. Reloads Caddy to activate the ingress snippet.
8. Prunes dangling Podman container images.

## Verify the running application

Check status with the `gare list` command:

```sh
gare list
```

The command prints a formatted table displaying the application name, domain, assigned port, systemd active status, and git commit hash.

For scripted output, use JSON or quiet mode:

```sh
gare list --json
gare list -q
```

Inspect the systemd user service directly:

```sh
systemctl --user status test-app.service
```

Stream live application logs through `gare logs`:

```sh
gare logs test-app -f
```

Inspect the Podman pod directly:

```sh
podman pod ps
podman ps
```

Send a test request directly to the local port:

```sh
curl http://localhost:8000/health
```

Send a request through Caddy using the domain:

```sh
curl -H "Host: test.example.com" http://localhost/health
```

## Customizing the Kubernetes manifest

If your application requires custom environment variables, volume mounts, or health probes, add a `manifest.yaml` file to your Git repository root.

When present, `gare deploy` syncs your repository's `manifest.yaml` directly over the generated default:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-app
  labels:
    app: test-app
spec:
  containers:
  - name: test-app
    image: localhost/test-app:latest
    env:
    - name: LOG_LEVEL
      value: "debug"
    - name: PORT
      value: "8000"
    ports:
    - containerPort: 8000
      hostPort: 8000
```

Commit and push changes to the repository, then run `gare deploy test-app`.

## Configuring with `gare.yml`

Instead of passing CLI flags on every setup, repositories can declare build and deployment settings using a `gare.yml` or `gare.yaml` file at the repository root:

```yaml
type: container
containerfile: apps/web/Dockerfile
context: .
build_cmd: make assets
```

For static applications served directly by Caddy:

```yaml
type: static
static_dir: dist
build_cmd: bun run build
```

When creating the application:

```sh
gare app create blog --repo https://github.com/example/blog.git --domain blog.example.com
```

Gare reads `gare.yml` automatically, routes traffic through Caddy's `file_server`, runs the optional build command, and skips Podman container synthesis entirely. CLI flags override `gare.yml` values when specified.

## Automated webhook deployments

To trigger deployments automatically on `git push`, start the built-in webhook receiver:

```sh
gare server --port 9000 --secret super-secret-token
```

The server exposes two endpoints:

- `POST /webhook/<app-name>`: Authenticated webhook endpoint. Accepts GitHub `X-Hub-Signature-256` HMAC signatures or GitLab `X-Gitlab-Token` headers.
- `GET /health`: Healthcheck probe.

To run `gare server` continuously under systemd user supervision, create `~/.config/systemd/user/gare-server.service`:

```ini
[Unit]
Description=Gare Webhook Server
After=network-online.target

[Service]
Type=exec
ExecStart=%h/.local/bin/gare server --port 9000 --secret super-secret-token
Restart=always
RestartSec=5s

[Install]
WantedBy=default.target
```

Enable and start the service:

```sh
systemctl --user daemon-reload
systemctl --user enable --now gare-server.service
```

Add an ingress snippet in `/etc/caddy/conf.d/gare-server.caddy`:

```caddyfile
gare-hook.example.com {
	reverse_proxy localhost:9000
}
```

Reload Caddy:

```sh
caddy reload --config /etc/caddy/Caddyfile
```

Now configure GitHub or GitLab to point to `https://gare-hook.example.com/webhook/test-app`.

## Tear down the application

To remove the test application completely, run:

```sh
gare destroy test-app
```

`gare destroy` cleans up every resource:

1. Stops and disables `test-app.service`.
2. Removes `~/.config/systemd/user/test-app.service`.
3. Removes `/etc/caddy/conf.d/test-app.caddy`.
4. Deletes the container image `localhost/test-app:latest`.
5. Removes the application directory `~/.local/share/gare/apps/test-app/`.
6. Reloads systemd and Caddy.

Verify all traces are removed:

```sh
gare list
systemctl --user status test-app.service
ls /etc/caddy/conf.d/test-app.caddy
```
