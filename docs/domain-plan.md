# Domain command: first-class hostnames, off `create`

Checked against: filet (cli limits, no inline comments, public godoc), module-path (`github.com/FacileStudio/gare`). Not applicable: migrations, auth/porte, muse, events (this is a CLI, not an `apps/api` service).

## Goal

Remove `--domain` from `gare app create` and add `gare domain` so hostnames are attached, listed, and detached as their own resource, then written into the existing per-app Caddy drop-in.

## Why (evidence)

Create still takes `--domain` / `-d` (`cmd/gare/app.go:79`) and copies `gare.yml` `domain` into `AppConfig.Domain` (`cmd/gare/app_artifacts.go:35`, `:73`, `:130`). Ingress is a single string, so an app cannot have aliases, a hostname cannot move without editing `config.json`, and create mixes workload setup with public DNS. Caddy already keys sites by hostname and forbids duplicate addresses.

## Approach

Keep the file-based Caddy pattern gare already uses: one drop-in per app at `/etc/caddy/conf.d/<app>.caddy`, glob-imported from `/etc/caddy/Caddyfile`, atomic write, then `caddy reload` / Admin API `POST /load`. Do not switch to On-Demand TLS or the JSON config API.

Store hostnames on the app as `Domains []string`. `gare domain add|rm|list` is the only writer. Create, `gare.yml`, and `--domain` stop touching hostnames. When an app is running (or being deployed/started), regenerate that app's snippet with every attached hostname in one site block, then reload.

Reuse the `gare env` command shape (root management group + nested under `gare app`) and the existing `internal/caddy` write/reload helpers.

## Caddy: what to copy, what to skip

Sources:

- https://caddyserver.com/docs/caddyfile/concepts
- https://caddyserver.com/docs/caddyfile/directives/import
- https://caddyserver.com/docs/automatic-https
- https://caddyserver.com/docs/api
- https://caddyserver.com/docs/caddyfile/options

Use:

1. **Glob import of site files.** `import /etc/caddy/conf.d/*.caddy` is the documented way to compose many sites. Empty glob is not an error. Gare already does this via `EnsureCaddyfilePaths`.
2. **One site block, many addresses.** `app.example.com, www.example.com { reverse_proxy localhost:8000 }` is valid. Addresses in one block share the same definition. At least one space around commas.
3. **Unique addresses.** Caddy rejects the same address twice across the whole config. Gare must enforce uniqueness before write, not after a failed reload.
4. **Named hostnames activate Automatic HTTPS.** A site address with a public hostname gets a managed cert (HTTP-01 on :80, TLS-ALPN on :443) and HTTP→HTTPS redirect. DNS A/AAAA must already point at the host. Local names (`localhost`, `.internal`, `.local`) get local HTTPS. Gare does not obtain certs itself.
5. **Reload the Caddyfile, do not PATCH JSON.** `caddy reload --config <Caddyfile>` and `POST http://127.0.0.1:2019/load` with `Content-Type: text/caddyfile` is what `internal/caddy.Reload` already does. Write the file first, then reload.
6. **Atomic snippet replace.** Write temp, sync, rename (`internal/atomicfile`). One file per app, not one file per hostname, so add/remove rewrites the same path `GetSnippetPath` already uses.

Skip:

- **On-Demand TLS.** Caddy's own docs: do not use it when the hostname is known at config time. It needs a global `ask` HTTP endpoint, which would force a gare daemon. Zero-daemon forbids that. Catch-all `https://` site blocks are the same trap.
- **Caddy JSON API surgery** (`POST /config/apps/http/servers/...`). Two sources of truth with the Caddyfile. Reloading the adapted Caddyfile is enough.
- **Wildcard DNS-01 / `tls { dns ... }`.** `*.example.com` stays valid in our hostname regex, but issuing the cert is the operator's Caddy build (DNS plugin). Gare only writes the address.
- **`www` redirect site blocks, ACME email, staging CA.** Operator Caddyfile global options, not gare.

Snippet shapes after this change:

```caddy
# container, one or more hostnames
app.example.com, www.example.com {
	reverse_proxy localhost:8000
}

# static, hostnames plus the allocated port (current behaviour, now N hostnames)
blog.example.com, www.blog.example.com, :8001 {
	root * "/home/yann/.local/share/gare/apps/blog/repo/dist"
	try_files {path} /index.html
	file_server
}

# static, no hostnames: port only, unchanged
:8001 {
	root * "..."
	try_files {path} /index.html
	file_server
}
```

Container apps with zero hostnames: no snippet (today's `Domain == ""` path). Removing the last hostname of a container app deletes the snippet. Removing the last hostname of a static app rewrites to `:<port>` only.

## CLI contract

Root management command, also nested under `gare app`, same as `env`:

```
gare domain add <app> <hostname>
gare domain remove <app> <hostname>
gare domain list [<app>] [--json]
```

Aliases: `rm` / `delete` for `remove`, `ls` for `list`. `hostname` is a DNS name, optional `*.` left label, optional `:port` (existing `domainRegex` in `cmd/gare/app.go:17-18`).

Rules:

- `add` fails if the app does not exist, the hostname is invalid, or any app already owns it (including this one).
- `remove` fails if that app does not own the hostname.
- `list` with no app prints every hostname and its app. `list <app>` prints that app's hostnames. `--json` is a list of `{hostname, app}` (or `{hostname}` when filtered).
- Persist first (`config.json`), then sync Caddy. If the unit is not `active`, persist only (stop already removes the snippet; start/deploy writes it back from `Domains`).
- If the unit is `active`, rewrite the snippet and reload. Reload errors are warnings, same as start/deploy today.

Examples:

```
gare app create myapp --repo https://github.com/example/webapp.git
gare domain add myapp myapp.example.com
gare domain add myapp www.myapp.example.com
gare domain list
gare domain list myapp --json
gare domain rm myapp www.myapp.example.com
```

## Data model

Replace `AppConfig.Domain string` with `AppConfig.Domains []string` in `internal/storage/storage.go`. Field count stays under filet's 15.

`LoadConfig` migrates legacy `{"domain": "x"}` into `Domains: []string{"x"}` when `domains` is missing or empty. Saves write only `domains`. Do not keep writing `domain`.

Helpers live in a new `internal/storage/domain.go` (storage.go is already at 8 funcs, filet `funcsPerFile: 8`):

- `ValidateDomain(string) error` — move `domainRegex` here from `cmd/gare/app.go`
- `NormalizeDomain(string) string` — trim, lowercase
- `AppByDomain(apps []*AppConfig, hostname string) *AppConfig`
- `(c *AppConfig) AddDomain(hostname string) error`
- `(c *AppConfig) RemoveDomain(hostname string) error`
- `(c *AppConfig) HasDomain(hostname string) bool`

Uniqueness is checked by the command against `storage.ListApps`, not by a separate registry file. Scanning `~/.local/share/gare/apps/*/config.json` is the source of truth.

`GareFile.Domain` / `ResolveDomain` stay parseable so old `gare.yml` files do not fail YAML, but create and deploy **stop applying them**.

## Ingress helper

One function used by domain add/rm, start, restart, and deploy. Stop and destroy keep calling `RemoveSnippet`.

Put it in `cmd/gare/domain_ingress.go` (cmd files are near the filet function cap):

`syncAppIngress(cfg *storage.AppConfig) error`

- container + `len(Domains) > 0` → `WriteSnippet` with all hostnames
- container + no hostnames → `RemoveSnippet`
- static + hostnames and/or port → `WriteStaticSnippet` with all hostnames
- static + nothing to bind → error (already required by `GenerateStaticSnippet`)

`internal/caddy/caddy.go` is also at 8 funcs. Change signatures in place, do not add functions:

- `GenerateSnippet(domains []string, port int)`
- `WriteSnippet(confDir, name string, domains []string, port int)`
- `GenerateStaticSnippet(domains []string, port int, rootDir string)` — join hostnames, then optional `, :port`
- `WriteStaticSnippet` follows

Empty `domains` + port 0 still errors for static. Empty `domains` for container `WriteSnippet` should not be called; `syncAppIngress` removes instead.

While touching `WriteSnippet` / `WriteStaticSnippet` / `Reload`, delete the dead `confErr` branches (`internal/caddy/caddy.go:119-123`, `:138-142`, `:163-167`). They swallow `EnsureCaddyfilePaths` errors today.

## Steps (ordered)

1. `internal/storage/storage.go` + new `internal/storage/domain.go` + `internal/storage/domain_test.go` — `Domains []string`, legacy `domain` migration in `LoadConfig`, validate/normalize/add/remove/lookup helpers. [filet: new file, public godoc, `funcsPerFile` 8, `funcLines` 30]

2. `internal/caddy/caddy.go` + `internal/caddy/caddy_test.go` — multi-hostname site addresses, dead `confErr` removed, tests for 0/1/N hostnames and static port-only. [filet: no new funcs in `caddy.go`]

3. `cmd/gare/app.go`, `cmd/gare/app_artifacts.go`, `cmd/gare/app_test.go`, `cmd/gare/app_options_test.go` — drop `--domain`/`-d`, drop `appCreateOptions.domain`, drop `mergeGareFileDefaults` / `syncGareFileConfig` domain copies, move create-time domain validation tests onto `storage.ValidateDomain`. Create success message stays "on port N" only.

4. New `cmd/gare/domain.go` (cobra tree), `cmd/gare/domain_run.go` (add/rm/list), `cmd/gare/domain_ingress.go` (`syncAppIngress`), `cmd/gare/domain_test.go` — register in `registerManagementCommands` and under `NewAppCmd`. [filet: split files so none exceed 8 funcs / 250 lines]

5. `cmd/gare/deploy_exec.go`, `cmd/gare/lifecycle.go`, `cmd/gare/destroy.go` — start/restart/deploy call `syncAppIngress`; stop/destroy still `RemoveSnippet`. Destroy does not need a domain list; the snippet path is still per-app name.

6. `cmd/gare/list.go`, `cmd/gare/status.go`, `cmd/gare/status_render.go` and their tests — JSON `domains` array; table/status join with comma; `-` when empty. `formatDeployTarget` joins the same way.

7. Docs and changelog: `README.md` (quickstart uses `domain add` after create), `docs/test-app-guide.md`, `docs/dokploy-migration-guide.md`, `CHANGELOG.md` Unreleased. Stop documenting `domain:` in `gare.yml` as a live key.

8. Gate: `mise run test` and `filet check`. Do not raise filet limits.

## Files to Modify / New

Modify:

- `internal/storage/storage.go` — `Domains []string`; migrate in `LoadConfig`
- `internal/storage/storage_test.go` — save/load with `domains`; legacy `domain` key
- `internal/storage/garefile.go` — leave `Domain` field (compat parse only)
- `internal/storage/garefile_network.go` — `ResolveDomain` unused by create/deploy; keep for YAML compat or delete if nothing calls it after step 3
- `internal/caddy/caddy.go` — multi-host templates; drop dead `confErr`
- `internal/caddy/caddy_test.go`
- `cmd/gare/app.go` — remove flag, regex, `domain` field
- `cmd/gare/app_artifacts.go` — no domain on create or gare.yml sync
- `cmd/gare/app_test.go`, `cmd/gare/app_options_test.go`
- `cmd/gare/root.go` — `root.AddCommand(NewDomainCmd())` in management group
- `cmd/gare/deploy_exec.go`, `cmd/gare/lifecycle.go`
- `cmd/gare/list.go`, `cmd/gare/status.go`, `cmd/gare/status_render.go`
- `cmd/gare/list`/`status` tests
- `README.md`, `docs/test-app-guide.md`, `docs/dokploy-migration-guide.md`, `CHANGELOG.md`

New:

- `internal/storage/domain.go`
- `internal/storage/domain_test.go`
- `cmd/gare/domain.go`
- `cmd/gare/domain_run.go`
- `cmd/gare/domain_ingress.go`
- `cmd/gare/domain_test.go`

## Exit criteria

- `gare app create … --domain x` is an unknown flag.
- `gare.yml` `domain:` does not populate `config.json` on create or deploy.
- `gare domain add myapp a.example.com` then `gare domain add myapp b.example.com` writes one snippet whose site address is `a.example.com, b.example.com` (static also keeps `, :port`).
- `gare domain add otherapp a.example.com` fails (duplicate).
- `gare domain list` shows hostname → app; `gare domain rm myapp a.example.com` rewrites or removes the snippet and reloads if the unit is active.
- `gare start` / `deploy` rebuild the snippet from `Domains`; `gare stop` / `destroy` still remove `/etc/caddy/conf.d/<app>.caddy`.
- Existing `config.json` with `"domain": "x"` loads as `["x"]` and the next save writes `"domains"`.
- `mise run test` passes. `filet check` is clean.

## Risks / unknown unknowns

- **Live `config.json` on ruche.** Migration is read-side only. First `domain add`/`rm` or any `SaveConfig` drops the old key. Fine if every caller goes through `LoadConfig`.
- **Caddy reload vs ACME.** Reloading abort in-flight cert tasks (Caddy docs). Batching is already how start/deploy work; do not reload twice in one `domain add`.
- **Duplicate addresses already on disk.** A hand-edited snippet plus a gare snippet can still collide. Uniqueness is only among gare apps.
- **Permissions on `/etc/caddy/conf.d`.** Unchanged; `gare init` already checks write access.
- **filet `funcsPerFile: 8`.** Command files must be split from day one or the gate fails.

## Skip (YAGNI)

- On-Demand TLS, catch-all `https://`, `ask` endpoint, JSON API host matchers.
- Separate `~/.local/share/gare/domains.json` registry.
- `gare.yml` `domain` / `domains` as GitOps ingress (two sources of truth with the new command).
- Moving a hostname in one step (`domain move`). `rm` then `add`.
- `www` → apex redirects, path-based routing, per-domain TLS overrides.
- DNS record checks, cert-expiry hooks, wildcard DNS-01 plugins.
- Root shortcut `gare add-domain`. Domain is not a high-frequency deploy verb.
- Changing snippet path layout (still `<app>.caddy`, not `<hostname>.caddy`).
