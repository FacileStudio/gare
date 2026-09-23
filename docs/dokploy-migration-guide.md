# Dokploy and Traefik to gare migration guide for ruche

This guide details the complete migration of `ruche` from Dokploy, Traefik, and Docker to a 100% `gare` deployment architecture.

By completing this migration, you eliminate the centralized Docker daemon, discard Dokploy's SQLite and tRPC control plane, retire Traefik, and run every workload under rootless Podman supervised directly by systemd user units and routed through Caddy.

## Architectural comparison

| Concern | Dokploy + Traefik (Current) | gare + Caddy (Target) |
|---|---|---|
| Runtime | Docker daemon (root) | Podman (rootless, user-space) |
| Workload format | Docker Compose (`docker-compose.yml`) | Kubernetes Pod manifests (`manifest.yaml`) |
| Supervision | Docker engine restart policies | Native systemd user units (`systemctl --user`) |
| Control plane | NodeJS server, tRPC API, Postgres, Redis | Zero daemon; state on filesystem (`~/.local/share/gare`) |
| Ingress & TLS | Traefik with dynamic file provider | Caddy with drop-in snippets (`/etc/caddy/conf.d/*.caddy`) |
| Network model | Docker bridge networks per stack | Localhost pod network namespaces |
| Logging | Docker json-file logs via daemon | Systemd journal (`journalctl --user -u <app>`) |
| Grouping | Database-backed UI projects | Application tags (`gare tag`, `--tag` filter) |

## Phase 0: Pre-migration inventory and extraction

Before stopping any service on `ruche`, extract all live configuration, secrets, and database dumps from the running Docker containers.

### 1. Extract environment variables and secrets

Dokploy stores compose environment blocks internally. Running containers hold the ground truth.

Dump the exact runtime environment for every running container:

```sh
mkdir -p ~/migration-backup/envs
for container in $(docker ps --format '{{.Names}}'); do
  docker inspect "$container" --format '{{range .Config.Env}}{{println .}}{{end}}' > ~/migration-backup/envs/"$container".env
done
```

Review critical secrets such as `DATABASE_URL`, `JWT_SECRET`, `ENCRYPTION_SECRET`, and OIDC client credentials.

### 2. Identify persistent volumes

Map out all Docker volumes and host bind mounts:

```sh
docker inspect $(docker ps -q) --format '{{.Name}}: {{range .Mounts}}{{.Source}} -> {{.Destination}} ({{.Type}}){{println}}{{end}}' > ~/migration-backup/volumes.txt
```

Note specifically:
- Postgres data directories (for `registre`, client databases).
- SQLite database files.
- Static assets and file uploads.

### 3. Record domain mappings and routing

Traefik configuration in Dokploy reads from `/etc/dokploy/traefik/dynamic/*.yml` and container labels. Extract all active hostnames:

```sh
docker ps --format '{{.Names}}' | while read -r name; do
  labels=$(docker inspect "$name" --format '{{json .Config.Labels}}')
  echo "=== $name ==="
  echo "$labels" | grep -o 'Host(`[^`]*`)' || true
done > ~/migration-backup/domains.txt
```

Verify the canonical suite endpoints:
- `sso.facile.studio` (`registre`)
- `mycelium.facile.studio` (`mycelium`)
- `sonde.facile.studio` (`sonde`)
- `sablier.facile.studio` (`sablier`)
- `furet.facile.studio` (`furet`)
- `courrier.facile.studio` (`courrier`)
- `journal.facile.studio` (`journal`)
- `boite.facile.studio` (`boite`)
- External client sites (such as `gfconseiletformation.fr`)

## Phase 1: Host preparation on ruche

Prepare the unprivileged user account `yann` and system packages. `gare` needs Podman 5.0 or newer: the systemd units it writes pass `--service-container` to `podman kube play`, which earlier releases reject.

### 1. Configure systemd user lingering

Systemd must run user services without an active SSH session:

```sh
loginctl enable-linger yann
```

Confirm linger status:

```sh
ls /var/lib/systemd/linger/yann
```

### 2. Verify subuid and subgid allocations

Rootless Podman requires subordinate user and group IDs:

```sh
grep yann /etc/subuid
grep yann /etc/subgid
```

Expected output shows a range of 65,536 IDs assigned to `yann`:

```text
yann:100000:65536
```

### 3. Configure Podman pause container

Podman uses an infra container to hold the network namespace for Kubernetes pods. Create `~/.config/containers/containers.conf`:

```ini
[engine]
infra_image = "registry.k8s.io/pause:3.9"
```

Pull the pause container image to verify network access:

```sh
podman pull registry.k8s.io/pause:3.9
```

### 4. Install and prepare Caddy

Install Caddy from the official Debian/Ubuntu repository if not already installed:

```sh
sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLF 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt-get update
sudo apt-get install -y caddy
```

Create the drop-in snippet directory and grant ownership to `yann`:

```sh
sudo mkdir -p /etc/caddy/conf.d
sudo chown -R yann: /etc/caddy/conf.d
```

Update `/etc/caddy/Caddyfile` so Caddy imports every snippet:

```caddyfile
{
	email admin@facile.studio
}

import /etc/caddy/conf.d/*.caddy
```

Keep Caddy stopped for now because Traefik currently occupies ports 80 and 443:

```sh
sudo systemctl stop caddy
sudo systemctl disable caddy
```

### 5. Install and initialize gare

Build or install `gare` into `~/.local/bin/gare`:

```sh
mise run build
cp bin/gare ~/.local/bin/gare
```

Run `gare init`:

```sh
gare init
```

Confirm all checks pass.

## Phase 2: Workload translation

In Dokploy, applications ran as Docker Compose files. In `gare`, applications run as Kubernetes Pod manifests (`manifest.yaml`) executed via `podman kube play`.

### Single-container applications

For standard web APIs (such as `journal` or `furet`), `gare app create` synthesizes the manifest automatically.

The generated `manifest.yaml` looks like:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: furet
  labels:
    app: furet
spec:
  containers:
  - name: furet
    image: localhost/furet:latest
    imagePullPolicy: Never
    ports:
    - containerPort: 8001
      hostPort: 8001
```

`imagePullPolicy: Never` is required on every `localhost/` image gare builds: that image lives only
in Podman's storage, so an unset policy has `podman kube play` look it up as
`docker://localhost/furet:latest` and fail wherever nothing answers as a registry on `localhost`.

If the application requires environment variables, add them to `repo/manifest.yaml` under `spec.containers[0].env`:

```yaml
    env:
    - name: PORT
      value: "8001"
    - name: LOG_LEVEL
      value: "info"
```

### Multi-container stacks (API + Database)

In Docker Compose, services communicate across custom Docker bridge networks using service names as DNS hosts.

In a Kubernetes Pod, all containers share the exact same network namespace. Containers communicate directly over `localhost`.

Translate a compose stack with an API and a Postgres database into a multi-container Pod manifest:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: registre
  labels:
    app: registre
spec:
  volumes:
  - name: pgdata
    hostPath:
      path: /home/yann/.local/share/gare/apps/registre/volumes/postgres
      type: DirectoryOrCreate
  containers:
  - name: postgres
    image: docker.io/library/postgres:16-alpine
    env:
    - name: POSTGRES_DB
      value: registre
    - name: POSTGRES_USER
      value: postgres
    - name: POSTGRES_PASSWORD
      value: secretpassword
    - name: PGDATA
      value: /var/lib/postgresql/data/pgdata
    volumeMounts:
    - name: pgdata
      mountPath: /var/lib/postgresql/data
  - name: api
    image: localhost/registre:latest
    imagePullPolicy: Never
    env:
    - name: DATABASE_URL
      value: postgres://postgres:secretpassword@localhost:5432/registre?sslmode=disable
    - name: PORT
      value: "8002"
    ports:
    - containerPort: 8002
      hostPort: 8002
```

Key advantages of this pattern:
- No bridge networks or overlay drivers.
- Database ports do not need to be published to the host; only the app port is exposed via `hostPort`.
- The database is only reachable by containers inside that specific pod.

### Managing volume permissions under rootless Podman

Rootless Podman maps container UID 0 (root) to the host user UID (`1000`). Internal UIDs (like Postgres UID `70` or `999`) map to subordinate IDs in `/etc/subuid`.

When mounting a host directory into a container running as a non-root UID, set ownership using `podman unshare`:

```sh
mkdir -p ~/.local/share/gare/apps/registre/volumes/postgres
podman unshare chown -R 70:70 ~/.local/share/gare/apps/registre/volumes/postgres
```

`podman unshare` enters the user namespace of `yann`, allowing correct assignment of container UIDs on host files.

## Phase 3: Service-by-service migration sequence

Migrate workloads in order of blast radius:

1. Pilot application (e.g. `boite` or standalone test app).
2. Standalone APIs with minimal state (`furet`, `journal`).
3. Internal suite applications (`sonde`, `sablier`, `courrier`).
4. Daemon and client sites (`mycelium`, `gfconseil`).
5. Central authentication root: `registre` (`sso.facile.studio`).

### Procedure for stateful services (example: registre)

#### 1. Dump database from running Docker container

```sh
docker exec -t registre-postgres-1 pg_dump -U postgres registre > ~/migration-backup/registre.sql
```

#### 2. Stop Dokploy compose stack for the application

Using Dokploy CLI or direct Docker command:

```sh
docker stop $(docker ps -q -f name=registre)
```

#### 3. Register application with gare

```sh
gare app create registre \
  --repo https://github.com/FacileStudio/registre.git \
  --port 8002 \
  --branch main

gare domain add registre sso.facile.studio
```

#### 4. Configure manifest and restore database

Copy the multi-container manifest containing the Postgres container and API container into `~/.local/share/gare/apps/registre/manifest.yaml`.

Start the Postgres container pod briefly or initialize the volume:

```sh
mkdir -p ~/.local/share/gare/apps/registre/volumes/postgres
podman unshare chown -R 70:70 ~/.local/share/gare/apps/registre/volumes/postgres
```

#### 5. Deploy application under gare

```sh
gare deploy registre
```

Verify the pod is running:

```sh
podman pod ps
systemctl --user status registre.service
```

#### 6. Restore database dump

Restore the SQL dump into the Podman Postgres container:

```sh
cat ~/migration-backup/registre.sql | podman exec -i registre-postgres psql -U postgres registre
```

Restart the service:

```sh
systemctl --user restart registre.service
```

#### 7. Verify local endpoint

Test the local application port directly:

```sh
curl -I http://localhost:8002/ready
```

Confirm HTTP 200 response before proceeding.

## Phase 4: Ingress cutover (Traefik to Caddy)

Once all applications are deployed under `gare` and responding on their assigned local ports, execute the ingress switch.

### 1. Pre-check Caddy snippets

Verify every migrated application has an ingress snippet in `/etc/caddy/conf.d/`:

```sh
ls -la /etc/caddy/conf.d/
```

Validate snippet syntax with Caddy:

```sh
caddy validate --config /etc/caddy/Caddyfile
```

### 2. Stop Traefik and Dokploy

Stop Traefik to release ports 80 and 443:

```sh
docker stop dokploy-traefik || true
```

Stop the Dokploy server container to prevent automated restarts:

```sh
docker stop dokploy-server || true
```

Confirm ports 80 and 443 are free:

```sh
sudo ss -tulpn | grep -E ':(80|443)\b'
```

### 3. Start Caddy

Enable and start Caddy:

```sh
sudo systemctl enable --now caddy
```

Watch Caddy logs as it requests certificates from Let's Encrypt and ZeroSSL:

```sh
sudo journalctl -u caddy -f
```

### 4. Verify external connectivity

Test endpoints externally:

```sh
curl -Iv https://sso.facile.studio/ready
curl -Iv https://mycelium.facile.studio/health
curl -Iv https://sonde.facile.studio/health
```

Confirm valid TLS certificates, HTTP/2 or HTTP/3 negotiation, and correct reverse proxy responses.

## Phase 5: Decommissioning Dokploy and Docker

Once Caddy is handling all traffic and `gare list` reports all services healthy, purge Dokploy and Docker.

### 1. Stop and remove all remaining Docker containers

```sh
docker stop $(docker ps -aq)
docker rm $(docker ps -aq)
docker volume prune -f
docker network prune -f
docker system prune -a --volumes -f
```

### 2. Disable and stop Docker daemon

Stop the Docker service and its socket listener:

```sh
sudo systemctl stop docker.service docker.socket
sudo systemctl disable docker.service docker.socket
```

Verify Docker is inactive:

```sh
sudo systemctl status docker.service
```

### 3. Remove Dokploy configuration and data directories

Archive the Dokploy directory just in case, then remove it:

```sh
sudo tar -czf ~/dokploy-archive-$(date +%F).tar.gz /etc/dokploy
sudo rm -rf /etc/dokploy
```

Remove the global `@dokploy/cli` package:

```sh
bun remove -g @dokploy/cli || true
npm uninstall -g @dokploy/cli || true
```

Remove any Dokploy cron entries in `/etc/cron.*` or user crontabs:

```sh
crontab -l | grep -v dokploy | crontab -
```

### 4. Optional: Purge Docker packages

If you no longer need the Docker engine installed on `ruche`:

```sh
sudo apt-get purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo apt-get autoremove -y
sudo rm -rf /var/lib/docker
sudo rm -rf /var/lib/containerd
```

## Phase 6: Establish GitOps automation

To re-enable automatic deployments on Git push without Dokploy, deploy the `gare server` webhook daemon.

### 1. Create the systemd service for gare server

Write `~/.config/systemd/user/gare-server.service`:

```ini
[Unit]
Description=Gare GitOps Webhook Server
After=podman-user-wait-network-online.service
Wants=podman-user-wait-network-online.service

[Service]
Type=exec
ExecStart=%h/.local/bin/gare server --port 9000 --secret YOUR_WEBHOOK_SECRET
Restart=always
RestartSec=5s

[Install]
WantedBy=default.target
```

Reload and start the server:

```sh
systemctl --user daemon-reload
systemctl --user enable --now gare-server.service
```

### 2. Configure Caddy ingress for webhooks

Create `/etc/caddy/conf.d/gare-webhook.caddy`:

```caddyfile
gare.facile.studio {
	reverse_proxy localhost:9000
}
```

Reload Caddy:

```sh
caddy reload --config /etc/caddy/Caddyfile
```

### 3. Update repository webhooks

In each GitHub or GitLab repository settings:
- Payload URL: `https://gare.facile.studio/webhook/<app-name>`
- Content type: `application/json`
- Secret: `YOUR_WEBHOOK_SECRET`
- Events: `push`

When commits hit the target branch, `gare server` verifies the HMAC signature and triggers `gare deploy <app-name>`.

## Phase 7: Verification and post-migration checklist

Execute the following checklist to ensure complete system health:

- [ ] Run `gare list` to confirm all workloads are reported as `active`.
- [ ] Run `systemctl --user list-units --type=service` to confirm user units are active and running.
- [ ] Verify reboot persistence:
  ```sh
  loginctl show-user yann | grep Linger
  ```
- [ ] Test journald logs:
  ```sh
  gare logs sso -n 20
  ```
- [ ] Verify memory and CPU reduction:
  ```sh
  free -h
  top -b -n 1 | head -n 20
  ```
  Note the memory savings from dropping the Dokploy control plane, Traefik, Docker daemon, and Redis.
- [ ] Verify Let's Encrypt certificate renewal: Caddy automatically renews certificates starting 30 days before expiration. Check status with `sudo caddy storage`.

## Rollback procedure

If a critical issue occurs during the Traefik to Caddy cutover before Docker has been uninstalled:

1. Stop Caddy:
   ```sh
   sudo systemctl stop caddy
   ```
2. Restart Traefik and Dokploy:
   ```sh
   docker start dokploy-traefik dokploy-server
   ```
3. Restart original containers:
   ```sh
   docker start $(docker ps -a -q -f status=exited)
   ```
4. Verify original routing is restored.
