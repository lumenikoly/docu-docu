<!-- toudocu
id: TASK-SITE-007
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
-->

# TASK-SITE-007: Establish the Architecture of the New UI Foundation

<!-- toudocu:section result -->
## Result

An architectural decision accepts React 19, Base UI 1, and Vite 8 and records
their boundaries relative to the Go core, presentation layer, static Portal,
and `serve` mode.

<!-- toudocu:section scope -->
## Scope

- The ADR for React islands and Vite, the architecture map, MOD-SITE, and
  browser-development guidance in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Runtime code, browser packaging, visual redesign, and a new runtime contract.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` The ADR assigns the documentation and application model, routing, `PageBootstrap`, and security decisions to the Go core, and safe semantic HTML to Go presentation code, without requiring package refactoring.
- [x] `AC-02` `appearance.ts`, `portal.ts`, and `serve.ts` remain framework-independent; React is limited to islands, Editor, and Changes, and Base UI to complex interaction controls.
- [x] `AC-03` The decision prohibits turning Portal into an SPA or adding React Router; it defines the IslandHost contract and keeps `PageBootstrap v1`, capability-based UI, shared localization, nested URLs, island failure isolation, and Vite only for packaging.
- [x] `AC-04` `docs/architecture/overview.md`, the browser runtime boundary, MOD-SITE, and browser-development guide agree with ADR-008 and the `design/ui/docs-ui` boundaries.

<!-- toudocu:section plan -->
## Plan

1. Create ADR-008 with context, decision, and consequences.
2. Update the browser runtime boundary and architecture overview.
3. Align MOD-SITE and browser-development guidance.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-02` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-03` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-04` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Documentation impact

ADR-008 is created; the browser runtime boundary, MOD-SITE, architecture
overview, and frontend-development guide are updated.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task records a technical architectural decision and does not change an
independent user journey.
