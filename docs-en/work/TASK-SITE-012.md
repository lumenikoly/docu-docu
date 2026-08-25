<!-- toudocu
id: TASK-SITE-012
status: done
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-011
-->

# TASK-SITE-012: Move Discussions to a Deferred React Island

<!-- toudocu:section result -->
## Result

The Discussions and Agent Feedback interface uses React/Base UI, but a
canonical page does not load React until Discussions is actually opened.

<!-- toudocu:section scope -->
## Scope

- DiscussionPanel, DiscussionComposer, DiscussionState, and deferred
  activation in `web/`.
- The mount point in `internal/site/`, tests, and documents in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Changing server semantics for Discussion, AgentDelivery, anchors, message
  state, leases, or replies; a new browser source of truth; and full Changes migration.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` Discussions loads and mounts only after it is actually opened.
- [x] `AC-02` Components retain server semantics, state, anchors, leases, replies, and errors without a second source of truth.
- [x] `AC-03` Components and API can be reused inside Changes.
- [x] `AC-04` Keyboard behavior, focus, closing, reopening, and soft navigation leave no stale state or handlers.

<!-- toudocu:section plan -->
## Plan

1. Extract reusable discussion view models and components.
2. Connect the deferred island to the existing server contract.
3. Test lifecycle, accessibility, and reusable API.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web run test:browser`
- `AC-02` → `go test ./... && npm --prefix web test`
- `AC-03` → `npm --prefix web run typecheck && npm --prefix web test`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

Agent Feedback documents, the runtime boundary, and MOD-SITE reflect deferred
activation and the reusable Discussions API.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task preserves the existing discussions scenario and changes browser implementation only.
