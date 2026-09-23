# Gare Architecture, Deployment & Code Health Fix Plan

## 1. Root Cause Analysis: User/Password Prompts & Failures

### A. Caddy Reload Polkit/Sudo Prompts
- **Cause**: `internal/caddy/caddy.go` falls back to `systemctl reload caddy` when `caddy reload` fails. In a rootless deployment, calling system-level `systemctl` triggers Polkit authorization (`org.freedesktop.systemd1.manage-units`), prompting for user/root password on the terminal.
- **Fix**:
  - Rely on Caddy's Admin API (`localhost:2019`) and `caddy reload --config /etc/caddy/Caddyfile`.
  - Provide clear setup instructions and checks in `gare init` for Caddy Admin API and snippet directory permissions (`/etc/caddy/conf.d`).
  - Eliminate the blind interactive system-level `systemctl reload caddy` fallback that triggers Polkit prompts.

### B. Interactive Git Credentials & SSH Prompts
- **Cause**: `internal/builder/builder.go` executes `git clone` and `git pull` without `GIT_TERMINAL_PROMPT=0` or SSH batch mode flags. When accessing repositories requiring authentication or unknown SSH host keys, Git hangs or prompts for username/password on `/dev/tty`.
- **Fix**:
  - Set `GIT_TERMINAL_PROMPT=0` and `GIT_SSH_COMMAND="ssh -o BatchMode=yes"` in all Git executions.
  - Auto-detect credentials from GitHub CLI (`gh auth token`), GitLab CLI, environment variables (`GH_TOKEN`, `GITHUB_TOKEN`, `GITLAB_TOKEN`), and system Git credential helpers.
  - Wire configuration options from global config into the builder.

### C. User & Runtime Session Auto-Detection
- **Cause**: `CheckLinger` and paths relied on `os.Getenv("USER")` which can be empty in non-interactive environments, webhooks, or background services. Furthermore, `systemctl --user` and `podman` fail if `XDG_RUNTIME_DIR` (`/run/user/<UID>`) and `DBUS_SESSION_BUS_ADDRESS` (`unix:path=/run/user/<UID>/bus`) are missing from the environment.
- **Fix**:
  - Implement robust user resolution via standard library `os/user.Current()` with fallback to UID and environment variables.
  - Auto-populate `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS` in child process environments for `systemctl` and `podman` when not already set.

---

## 2. Compilation Errors & Symbol Redeclarations

### A. `cmd/gare/config.go`
- Add missing `package main` clause.
- Add required imports (`time`, `github.com/FacileStudio/gare/cmd/gare/config`).
- Remove duplicate `UseGitLabCLI: false` field initialization.

### B. `cmd/gare/deploy.go`
- Remove duplicate UI printing functions (`printInfo`, `printSuccess`, `printWarning`, `printError`) that conflict with `cmd/gare/ui.go`.
- Remove duplicate `syncRepoManifest` stub that conflicts with `cmd/gare/app_artifacts.go`.
- Remove unused `getCommitHash` helper. Resolved the other way: the helper survived as `builder.GetCommitHash` (`internal/builder/builder.go`) and is now called during deploy, so nothing was removed.

### C. `cmd/gare/config/loader.go`
- Remove duplicate declaration of `DefaultConfigPath()`.
- Refactor `Load()` and methods to meet Filet complexity, statement, line count, and function count limits.

---

## 3. Caddyfile & Systemd User Unit Synthesis

### A. Caddy Static File Server Template
- Fix `staticSnippetTemplate` in `internal/caddy/caddy.go`: move `try_files {path} /index.html` outside `file_server` block into valid Caddyfile format. Superseded: a static site now runs in its own container, and the template is `staticServerConfigTemplate` in `internal/caddy/static_server.go`. `internal/caddy/caddy.go` keeps only `snippetTemplate`, for ingress.

### B. Systemd User Unit Generation
- Update `unitTemplate` in `internal/systemd/unit.go`. There is no `unitTemplate`: units are one template per workload type, `kubeUnitTemplate` / `containerUnitTemplate` / `composeUnitTemplate`, in `internal/systemd/kube_unit.go`, `internal/systemd/container_unit.go` and `internal/systemd/compose_unit.go`. The three directives landed as follows:
  - Pod and static units replaced the network target with `After=` / `Wants=podman-user-wait-network-online.service`, the user-session equivalent. The item's premise holds: `systemctl --user status network-online.target` reports `LoadState=not-found`, and systemd drops ordering on an unknown unit without a warning. `composeUnitTemplate` was the last unit still naming `network-online.target` and now uses the same bridge service.
  - `Delegate=yes` is on the static and compose units.
  - `TimeoutStopSec=70s` is on the pod and compose units; the static unit matches what Podman's own generator produced for the same container.

---

## 4. Filet & Code Quality Compliance

- Remove all comments inside function bodies and inline comments in accordance with project rules in `AGENTS.md`.
- Ensure all Go files pass `filet check` and `mise run test`.
