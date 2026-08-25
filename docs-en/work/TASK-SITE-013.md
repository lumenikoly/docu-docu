<!-- toudocu
id: TASK-SITE-013
status: done
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-011
-->

# TASK-SITE-013: Move Roadmap Controls to a React Island

<!-- toudocu:section result -->
## Result

The “Add outcome” Roadmap operation uses an eagerly mounted React island and
Base UI while retaining its constrained write contract.

<!-- toudocu:section scope -->
## Scope

- The Roadmap island, form, and dialog in `web/`.
- Mount point in `internal/site/`, tests, and documents in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- New write capabilities or changing `suggestedId`, `digest`, `expectedDigest`,
  `stale_digest`, `roadmap-add`, or `X-Toudocu-Action`.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` The eager island loads state and retains the constrained write contract without new operations.
- [x] `AC-02` Input validation, successful addition, and stale `digest` are clear; the form state remains on a recoverable error.
- [x] `AC-03` The dialog is keyboard accessible, manages focus correctly, and works at a mobile size.
- [x] `AC-04` Remounting on soft navigation leaves no duplicate root, handlers, or stale form; the DOM implementation is removed.

<!-- toudocu:section plan -->
## Plan

1. Move the current form and state to React/Base UI.
2. Connect the eager island to the unchanged endpoint contract.
3. Test errors, accessibility, mobile behavior, and remounting; remove the DOM implementation.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `go test ./... && npm --prefix web test`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The Roadmap flow and screen, MOD-SITE, and runtime boundary describe the eager
island while the write contract remains unchanged.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task retains the existing Roadmap operation and changes its UI implementation only.
