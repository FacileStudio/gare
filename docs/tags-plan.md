# Tags command: metadata grouping for applications

Checked against: filet (cli limits, no inline comments, public godoc), module-path (`github.com/FacileStudio/gare`).

## Goal

Add tags to applications as first-class metadata so operators can group, filter, and inspect 50+ workloads without introducing filesystem or systemd hierarchy.

## Why (evidence)

Dokploy organized services into nested project folders inside its UI and SQLite database. 

`gare` runs on Linux primitives:
- Native systemd user units (`~/.config/systemd/user/<app>.service`)
- Caddy drop-in snippets (`/etc/caddy/conf.d/<app>.caddy`)
- Flat storage (`~/.local/share/gare/apps/<app>/`)

Introducing folder hierarchies (`apps/<project>/<app>/`) creates artificial friction:
- Systemd and Caddy namespaces remain flat, forcing prefixed unit names (`<project>-<app>.service`).
- Path helpers, destroy logic, and webhook routes (`/webhook/<app>`) become complex and brittle.
- Moving an app between projects requires directory renames, unit file regeneration, and Caddy snippet rewrites.

Tags provide full grouping power while preserving the flat, predictable OS integration.

## Approach

Keep the storage and systemd layer flat. Store tags in `AppConfig.Tags []string` inside each app's `config.json`.

Support both declarative GitOps tags (via `gare.yaml`) and imperative CLI tag management:
1. **GitOps sync**: `gare.yaml` supports `tags: [...]`. When an app is created or deployed (`gare deploy`), tags defined in Git are loaded and synced to `config.json`.
2. **Conflict resolution**: If `gare.yaml` defines `tags`, Git is the declarative source of truth on deployment. If `gare.yaml` omits tags, tags set via CLI are preserved.
3. **CLI management**: Provide a `gare tag` command group (and nested `gare app tag`) to add, remove, and list tags.
4. **Filtering**: Add `--tag` / `-t` to `gare list` (and `gare app list`) and display a `TAGS` column.

## CLI contract

### Management commands

Root command, also nested under `gare app`, following the `domain` and `env` convention:

```sh
gare tag add <app> <tag...>
gare tag remove <app> <tag...>
gare tag list [<app>] [--json]
```

Aliases:
- `rm`, `delete` for `remove`
- `ls` for `list`

Rules:
- `tag` identifiers must be alphanumeric with hyphens or underscores (e.g. `client-acme`, `prod`, `api-gateway`).
- `add` normalizes tags (lowercase, trimmed) and ignores duplicates.
- `remove` removes specified tags from the app.
- `list` with no arguments lists all unique tags across all apps and their app counts.
- `list <app>` prints tags attached to that specific application.

### App creation and listing

```sh
# Create with initial tags
gare app create myapp --repo https://github.com/org/repo.git --tag client-a --tag backend

# Filter list by tag
gare list --tag client-a
gare list -t prod
gare list --json
```

Output format for `gare list`:

```text
NAME      TYPE       STATUS   PORT   DOMAINS          TAGS              REVISION
myapi     container  active   8001   api.example.com  client-a,backend  a1b2c3d
myweb     static     active   8002   web.example.com  client-a,frontend e4f5g6h
```

## Data model

### Configuration updates

1. `storage.AppConfig` in `internal/storage/storage.go`:
   ```go
   type AppConfig struct {
       // ... existing fields ...
       Tags []string `json:"tags,omitempty"`
   }
   ```

2. `storage.GareFile` in `internal/storage/garefile.go`:
   ```go
   type GareFile struct {
       // ... existing fields ...
       Tags []string `yaml:"tags,omitempty"`
   }
   ```

### Storage helpers

Create `internal/storage/tag.go`:
- `ValidateTag(tag string) error`: Validates alphanumeric format and length (1-63 chars).
- `NormalizeTag(tag string) string`: Trims whitespace and lowercases.
- `(c *AppConfig) AddTags(tags ...string) bool`: Adds unique normalized tags.
- `(c *AppConfig) RemoveTags(tags ...string) bool`: Removes matching tags.
- `(c *AppConfig) HasTag(tag string) bool`: Reports whether the tag is present.
- `FilterAppsByTag(apps []*AppConfig, tag string) []*AppConfig`: Filters a slice of apps.
- `CollectAllTags(apps []*AppConfig) map[string]int`: Returns unique tags with app counts.

## Steps (ordered)

1. **Storage model and helpers**:
   - Add `Tags []string` to `storage.AppConfig`.
   - Implement `internal/storage/tag.go` with validation, normalization, and list helpers.
   - Add comprehensive unit tests in `internal/storage/tag_test.go`.

2. **GareFile parsing and deployment sync**:
   - Add `Tags []string` to `storage.GareFile` in `internal/storage/garefile.go`.
   - Update `syncGareFileConfig` in `cmd/gare/app_artifacts.go` to sync `Tags` when defined in `gare.yaml`.
   - Add unit tests in `internal/storage/garefile_test.go` and `cmd/gare/config/loader_test.go`.

3. **App creation flags**:
   - Add `--tag` / `-t` flag (string slice) to `gare app create` in `cmd/gare/app.go`.
   - Populate `AppConfig.Tags` in `saveAppMetadata` in `cmd/gare/app_artifacts.go`.
   - Update `cmd/gare/app_test.go` and `cmd/gare/app_options_test.go`.

4. **Tag command implementation**:
   - Create `cmd/gare/tag.go` (Cobra command setup).
   - Create `cmd/gare/tag_run.go` (execution logic for add, remove, list).
   - Create `cmd/gare/tag_test.go` for CLI testing.
   - Register in `registerManagementCommands` in `cmd/gare/root.go` and under `NewAppCmd` in `cmd/gare/app.go`.

5. **List command filtering and rendering**:
   - Add `--tag` / `-t` filter flag to `gare list` in `cmd/gare/list.go`.
   - Add `TAGS` column in `cmd/gare/status_render.go`.
   - Include `tags` in `--json` output.
   - Update `cmd/gare/status_test.go`.

6. **Documentation**:
   - Update `README.md` with tag usage and filtering examples.
   - Update `docs/dokploy-migration-guide.md` explaining how to map Dokploy projects to Gare tags.
   - Update `CHANGELOG.md`.

7. **Verification Gate**:
   - Run `mise run test`.
   - Run `filet check` (ensure functions per file <= 8, file lines <= 250, no inline comments).

## Exit criteria

- `gare app create myapp --repo ... --tag client-a` records `tags: ["client-a"]` in `config.json`.
- `gare tag add myapp prod api` adds `prod` and `api` tags.
- `gare tag rm myapp api` removes `api`.
- `gare tag list` outputs all tags across all applications with counts.
- `gare tag list myapp` outputs tags for `myapp`.
- `gare list --tag client-a` returns only applications carrying the `client-a` tag.
- If `gare.yaml` contains `tags: [team-core, prod]`, `gare deploy` syncs these tags to `config.json`.
- `mise run test` and `filet check` pass with zero warnings.

## Skip (YAGNI)

- Hierarchical folder nesting (`apps/<project>/<app>`).
- Batch destructive operations (`gare destroy --tag <tag>`).
- Complex boolean tag expressions (`--tag "prod AND (api OR web)"`). Single tag matching is sufficient.
