# Compose workload type: first-class support in gare

Checked against: filet (cli limits, no inline comments, public godoc), module-path (`github.com/FacileStudio/gare`). Not applicable: migrations, auth/porte, muse, events.

> **Implemented 2026-09-20, with the shapes below corrected by a live smoke test.** `Type=oneshot` with `RemainAfterExit=yes` and `up -d` is the shipped shape. A foreground `up` was tried first and rejected on evidence: with an attached provider, `ExecStop`'s `compose down` races the provider's own attach loop (docker-compose logs `Error while Stopping` for every container and exits 1), the unit ends `failed` after a normal stop, and stop takes the full `TimeoutStopSec`. Detached, the same `down` is clean. The unit also `Requires=podman.socket`, because the external provider reaches podman through the API socket and nothing else starts it. Other deviations: `gare.yml` `port:` is authoritative and validated against the compose file's published host ports (gare cannot rewrite a compose port mapping); each app gets its own compose project name via `-p` (without it every app's checkout is named `repo` and `down` targets the wrong stack); compose env vars are injected through the application env file plus systemd `EnvironmentFile=` rather than `--env-file`; `gare destroy` runs `compose down -v` explicitly instead of relying on the unit, so a missing unit file cannot leak containers or volumes. The "Health probes" section below has been rewritten to the three-tier resolution gare now ships (explicit `healthchecks:`, then probes derived from the compose file, then the legacy single-probe key).

## Goal

Add `type: compose` as an opt-in workload type in `gare.yml`, alongside the existing `container` (Pod manifest) and `static` types. When set, gare runs `podman compose up -d` instead of `podman kube play`, and `podman compose down` instead of `kube down`. This gives repos a known workflow path for multi-container stacks without inventing a new orchestration layer.

## Why (evidence)

### What compose brings

A repository that already has a `docker-compose.yml` or `compose.yml` can author multi-container workloads in the format it already knows. Networks, volume mounts, `depends_on`, named volumes with drivers, `secrets`, and per-service image targets all work without translation. No Pod YAML to hand-author, no manual mapping of services to containers, no network alias confusion.

### What kube play brings today

Single-container apps with a generated Pod manifest, full declarative control, no second binary dependency. This stays the default and unchanged.

### The friction today

A repo with a `docker-compose.yml` and no `manifest.yaml` has a silent failure: gare generates a default single-container Pod, ignores the compose file entirely, and deploys a degenerate workload nobody asked for. The migration guide walks through translating compose to Pod by hand, but nothing in the code enforces this or tells the user it is required.

### The binary cost

`podman compose` is not part of podman; it shells out to an external provider (`docker-compose` or `podman-compose`), chosen via `containers.conf`. gare already requires podman — this adds a second install surface. If the provider is missing at deploy time, the failure is a confusing "executing external compose provider" message from the wrapper, not a clean Gare error.

### The supervision model divergence

This is the real structural cost. `podman kube play` is a oneshot: start, wait, done. `podman compose up -d` is a long-lived process managed by podman's container lifecycle. The systemd unit, lifecycle commands (`start`/`stop`/`restart`), and destroy path all need a per-type branch. That is a divergence budget, not a bug, but it is real.

## Approach

### Explicit opt-in, never auto-detect

`type: compose` is set in `gare.yml`. A repo with a stray `docker-compose.yml` and no `gare.yml` still fails loudly — pointing at the need to write a `manifest.yaml`. Compose detection is not added. Auto-translation is not added.

The yaml shape:

```yaml
type: compose
compose_file: docker-compose.yml   # optional; defaults to compose.yml then docker-compose.yml
```

### Where the supervision diverges

Today all apps share one codepath: systemd unit template in `unit.go` hardcodes `kube play`/`kube down`. Each divergence point:

| Concern | `container` (Pod) | `compose` | `static` |
|---|---|---|---|
| Systemd unit | `ExecStart=kube play`, `ExecStopPost=kube down` | `ExecStart=compose up -d`, `ExecStop=compose down`, `Requires=podman.socket` | `ExecStart=caddy file-server`, `ExecStopPost=systemctl stop` |
| Unit name | `<name>.service` | `<name>.service` (same) | `<name>.service` (same) |
| Health probe | HTTP GET on port + healthcheck path | HTTP GET on port + healthcheck path | HTTP GET on port + healthcheck path |
| Destroy | `kube down` then remove image | `compose down` (removes compose containers, not necessarily images) | remove Caddy snippet, stop unit |
| Env injection | manifest YAML env section | compose env blocks + host env vars | host env vars only |

`compose down` stops containers but does not remove images by default. `kube down` destroys the entire Pod including its image references. `gare destroy` therefore calls `compose down -v` (containers, networks, and named volumes) and then reports any project container the provider failed to remove, rather than claiming a clean teardown. Images built by `build:` sections are left in place; prune them with `podman image prune`.

### Health probes

Shipped: gare resolves probes in three tiers, first match wins.

1. An explicit `healthchecks:` list in `gare.yml` (one entry per `name`/`port`/`path`).
2. Probes derived from the compose file itself: any service whose `healthcheck` issues an HTTP request (`curl`/`wget`) to a **published** container port becomes a probe named after the service, with the container port mapped to its published host port. Non-HTTP checks and unpublished ports are skipped, since they are not reachable from the host.
3. The legacy single `port` plus `healthcheck` path.

A per-container probe model is no longer required: probes target host ports through Caddy's upstream, so derived probes cover a multi-service stack without any compose-file changes. Compose `depends_on: condition: service_healthy` still does the in-stack ordering; gare's probes are the host-side gate.

### Env injection

Compose `environment:` blocks and `env_file:` declarations become the compose `env` and `env_file` directives in the `podman compose up -d` command. Host-level env vars (set via `gare env set` or in the compose file) are passed through the same way. No secrets management yet (see Security below).

### Port mapping

Compose port mappings (`ports: - "8000:8000"`) become the systemd unit's `Port=8000` directive for Caddy routing, same as today. Port discovery on deploy still allocates a free host port and syncs it into `config.json`. The compose host port and gare's host port stay in sync via `syncGareFilePorts` in `app_artifacts.go`, which already does this for `container` type.

### Systemd unit template

The unit template in `unit.go` needs a small injection point. Rather than three separate unit templates, introduce a workload type abstraction:

```go
type Workload struct {
    UpCommand   string
    DownCommand string
    IsCompose   bool
}
```

The existing unit template parameter `ManifestPath` generalises to `ExecArg` — for Pod it is the manifest path, for compose it is the compose file path. The systemd unit no longer hardcodes `kube play`/`kube down`.

This is a breaking change to `unit.go`'s template interface, but scoped to one function (`GenerateUnit`). The `systemd_test.go` coverage updates accordingly.

## CLI contract

### `gare.yml` schema

Existing keys (`containerfile`, `context`, `build_cmd`, `port`, `container_port`, `healthcheck`, `static_dir`) are unchanged. New keys:

```yaml
type: compose                      # required; container | static | compose
compose_file: docker-compose.yml   # optional; defaults to compose.yml, compose.yaml, docker-compose.yml, docker-compose.yaml
```

### Commands affected

No new commands. `gare deploy`, `gare destroy`, `gare start`, `gare stop`, `gare restart`, `gare status` all work. The difference is in the systemd unit and lifecycle hooks, transparent to the user.

## Data model

### `GareFile` updates (`internal/storage/garefile.go`)

```go
type WorkloadType string

const (
    WorkloadTypeContainer WorkloadType = "container"
    WorkloadTypeStatic    WorkloadType = "static"
    WorkloadTypeCompose   WorkloadType = "compose"
)

type GareFile struct {
    // existing fields unchanged ...

    WorkloadType WorkloadType `yaml:"type"`
    ComposeFile  string       `yaml:"compose_file,omitempty"`
}
```

### `AppConfig` updates (`internal/storage/storage.go`)

Add `WorkloadType string` field (or replace with typed enum if later languages add more). Keep as string for now — YAGNI on enum overloading.

## Implementation steps

### Phase 1: Fail-fast and schema

1. **Compose detection warning** (`cmd/gare/deploy.go`, `cmd/gare/app.go`):
   - Add `detectComposeFile` helper: checks `docker-compose.yml`, `docker-compose.yaml`, `compose.yml`, `compose.yaml` in repo root.
   - If a compose file exists and `type` is not `compose`, log a warning: "compose file found in repo; set type: compose in gare.yml or write manifest.yaml".
   - Not an error — allows repos that have both a compose file and a manifest to coexist without noise.

2. **`GareFile` schema** (`internal/storage/garefile.go`):
   - Add `WorkloadType` field with `container | static | compose` values.
   - Add `ResolveWorkloadType()` returning the parsed type.
   - `ResolveType()` deprecated in favor of `ResolveWorkloadType()` for workload-specific paths.
   - Load both `gare.yaml` and `gare.yml`.

3. **Merge into AppConfig** (`cmd/gare/app_artifacts.go`, `cmd/gare/deploy.go`):
   - `syncGareFileConfig` copies `WorkloadType` and `ComposeFile` into `AppConfig`.
   - `syncGareFilePorts` and `applyGareWorkloadConfig` gain a workload-type branch.

4. **Unit template rewrite** (`internal/systemd/unit.go`):
   - Change `GenerateUnit` to accept a `WorkloadType` and an `ExecArg` instead of hardcoded `kube play`/`kube down`.
   - Add a compose template string alongside the Pod template.
   - Keep one `UnitData` struct with conditional fields.

### Phase 2: Compose lifecycle

5. **Build command divergence** (`cmd/gare/deploy.go`, `cmd/gare/deploy_exec.go`):
   - Container type: `buildAppImage` (existing).
   - Compose type: no image build — compose up pulls or builds as configured. `buildAppImage` is skipped.

6. **Deploy command** (`cmd/gare/deploy.go`):
   - Container type: `syncManifest`, then unit with `kube play`.
   - Compose type: `compose up -d` via systemd unit with `ExecArg` = compose file path.

7. **Lifecycle commands** (`cmd/gare/lifecycle.go`):
   - `runStartApp`, `runStopApp`, `runRestartApp` call `systemd.Start`/`Stop`/`Restart` with the same unit path regardless of type (the unit file already reflects the correct command). No new code needed in lifecycle — it is unit-file-driven.

8. **Destroy** (`cmd/gare/destroy.go`):
   - Container type: `kube down` then image prune (existing).
   - Compose type: `compose down -v` (removes containers and named volumes). Add compose image pruning after.

### Phase 3: Env and health

9. **Env injection for compose** (`cmd/gare/env.go`, `cmd/gare/deploy_exec.go`):
   - Container: env vars written into `manifest.yaml`.
   - Compose: env vars passed as environment variables to the `compose up -d` command, or injected via `compose.env` file.
   - Verify: podman compose supports `--env-file` and inline env vars in the compose command.

10. **Health probe** (`internal/health/probe.go`, `cmd/gare/deploy_exec.go`):
    - No change needed. Probes work the same regardless of supervision model.

### Phase 4: Cleanup

11. **Systemd unit tests** (`internal/systemd/systemd_test.go`):
    - Update tests to cover compose unit template path.

12. **Destroy tests** (`cmd/gare/destroy.go`):
    - Verify `compose down -v` is called for compose apps.

13. **GareFile tests** (`internal/storage/garefile_test.go`):
    - Test parse of `type: compose`, `compose_file: custom.yml`.

## Files to modify / new

Modify:
- `internal/storage/garefile.go` — add `WorkloadType`, `ComposeFile`
- `internal/storage/garefile_test.go` — parse coverage
- `internal/systemd/unit.go` — workload-type-aware unit generation
- `internal/systemd/systemd_test.go` — compose unit template path
- `cmd/gare/app.go` — warn on compose file without `type: compose`
- `cmd/gare/app_artifacts.go` — sync `WorkloadType`/`ComposeFile` to `AppConfig`
- `cmd/gare/app_options_test.go` — new options
- `cmd/gare/deploy.go` — container vs compose deploy branch
- `cmd/gare/deploy_exec.go` — skip `buildAppImage` for compose
- `cmd/gare/destroy.go` — `compose down -v` + image prune for compose
- `cmd/gare/config/loader.go` — resolve config path before `LoadGareFile` call

Do not add:
- Auto-detection of compose files (fail-fast via warning only, see Phase 1 step 1)
- Auto-translation compose → Pod manifest
- New commands (`gare compose up`, etc.)
- Per-container health probes
- Secret management

## Exit criteria

- `gare.yml` with `type: compose` generates a systemd unit calling `podman compose up -d <file>` on start and `podman compose down -v <file>` on stop.
- `gare.yml` with `type: compose` on `gare deploy` does not call `buildAppImage` or `syncManifest`.
- `gare.yml` with `type: compose` and `compose_file: custom.yml` uses `custom.yml` instead of the default.
- `gare destroy` for a compose app calls `podman compose down -v`.
- A repo with a `docker-compose.yml` and no `type: compose` in `gare.yml` logs a warning but does not fail.
- `gare start`, `stop`, `restart`, `status` work identically for compose apps (unit-driven, no new code needed).
- `mise run test` and `filet check` pass.

## Risks and open questions

- **Provider availability**: `podman compose` requires an external provider (`docker-compose` or `podman-compose`). If neither is installed, the unit will fail at runtime when systemd tries to start it. This is a user responsibility, not a gare error. We could add a preflight check in `gare deploy` that validates the compose provider exists, but that adds complexity to the deploy path for an edge case.
- **Image naming conflict**: The existing `localhost/<name>:latest` image naming is container-type-specific. For compose, images come from the compose file and registry references. No conflict today, but `gare destroy` image pruning for compose apps needs to target compose-managed image names, not the hardcoded `<name>:latest` pattern.
- **Volume lifecycle**: `compose down -v` removes named volumes. `kube down` destroys the Pod (volumes are ephemeral, hostPath is untouched). This is a semantic difference in behaviour. Document it.
- **`secrets` in compose**: Maps to `--secret` flags in podman compose, which read from files. This is out of scope for now — env vars are flat strings, files are separate. No `secretKeyRef` or file-based secret injection.
