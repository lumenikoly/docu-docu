<!-- toudocu
id: TASK-SITE-006
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
-->

# TASK-SITE-006: New Toudocu UI System

<!-- toudocu:section result -->
## Result

Toudocu uses one durable UI foundation: Go produces static HTML, a semantic
design system defines the visual contract, React implements application
surfaces, Base UI provides complex controls, and Vite packages browser assets.
Portal does not become an SPA, and `appearance.ts`, `portal.ts`, and `serve.ts`
do not depend on React. `PageBootstrap schema v1`, soft navigation, shared
localization catalogs, and deployment at a nested URL remain intact. Ordinary
pages do not load React, and the released binary does not need Node.js.

<!-- toudocu:section scope -->
## Scope

- Coordinate migration of `web/` and asset embedding through `internal/site/`.
- Final documentation in `docs/` and integration checks in `Makefile`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- An SPA, React Router, or moving Markdown, routing, or security rules out of Go.
- Node.js as a user runtime dependency.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` All direct child tasks are done; the old and new UI foundations do not coexist.
- [x] `AC-02` An ordinary static page does not load React; Portal works at the root and nested URL.
- [x] `AC-03` The released binary runs `check`, `build`, and `serve` without Node.js.
- [x] `AC-04` Documentation describes the final architecture; `make check`, browser tests, and release checks pass.

<!-- toudocu:section plan -->
## Plan

1. Finish the architectural, packaging, and shared UI foundation.
2. Finish island, Editor, and Changes branches.
3. Apply the visual language, remove the previous foundation, and reconcile documentation.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `go run ./cmd/toudocu task tree TASK-SITE-006 ./docs --repository-root . && make web-check && make browser-test`
- `AC-02` → `make browser-test`
- `AC-03` → `go test ./... -run TestReleasedBinaryWithoutNodeRuntime`
- `AC-04` → `make check && make browser-test && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `make check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Documentation impact

After the tree is complete, architectural, module, reference, and user
documents describe only the final UI system. The draft remains an entry point
for decomposition, not a source of current state.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

This task coordinates a technical migration of several existing surfaces and
does not introduce a distinct user scenario.
