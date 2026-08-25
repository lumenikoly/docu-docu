<!-- toudocu
id: TASK-SITE-011
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-010
-->

# TASK-SITE-011: Integrate React Islands with Soft Navigation

<!-- toudocu:section result -->
## Result

Canonical `serve` safely mounts eager and deferred React islands and changes
pages through existing soft navigation without leaking React roots or effects.

<!-- toudocu:section scope -->
## Scope

- Framework-independent IslandHost and navigation lifecycle in `web/`.
- Mount points in `internal/site/`, tests, and runtime documentation in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- React Router, a new `PageBootstrap` version, and migration of one application surface.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` A framework-independent IslandHost performs `discover`, `activate`, `mount`, `unmount`, and `unmountAll`, with at most one React root per island instance.
- [x] `AC-02` Once the target page validates, `toudocu:pagebeforechange` is sent, `unmountAll` runs, layout and bootstrap data change, then `toudocu:pagechange` is sent and eager islands mount.
- [x] `AC-03` Cache bounds, history, scroll restoration, full navigation on failure, `PageBootstrap v1`, `window.ToudocuPage`, capabilities, endpoints, and relative paths are retained.
- [x] `AC-04` A→B, A→B→A, and A→B→A→B leave no duplicate root or handler, stale dialog or bootstrap data, or unfinished `AbortController`.

<!-- toudocu:section plan -->
## Plan

1. Implement IslandHost without a static React import.
2. Add the navigation commit boundary and lifecycle events.
3. Test discovery, remounting, cleanup, and full navigation.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web test`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `go test ./internal/site/... && npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The browser runtime boundary and MOD-SITE describe IslandHost, eager and
deferred activation, and soft-navigation commit order.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task changes the shared lifecycle of existing navigation without a new user scenario.
