# Changelog

## [Unreleased]

### Added

- `gare app create` now accepts creation without `--domain`; the domain can be added later via config.

## [0.2.0] - 2026-09-15

### Added

- Static workload hosting via `static_dir` and `build_cmd` in `gare.yml`.
- `gare.yml` configuration support for container and static app types.
- Port auto-discovery starting at 8000.
- `gare server` webhook receiver for automated deployments.
