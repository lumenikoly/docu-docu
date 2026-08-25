<!-- toudocu
id: TASK-SITE-014
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-010
-->

# TASK-SITE-014: Move the Editor Workspace to React

<!-- toudocu:section result -->
## Result

The Editor workspace shell is implemented with React/Base UI, while the secure
server part and CodeMirror 6 semantics are fully retained.

<!-- toudocu:section scope -->
## Scope

- Workspace shell and state in `web/`.
- Editor integration in `internal/site/`, tests, and documents in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Rewriting CodeMirror 6, the Editor HTTP API, SHA-256 CAS, atomic saving,
  preview semantics, file restrictions, or dependency on IslandHost.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` React manages the shell, file tree, toolbar, tabs, conflicts, creation dialog, diagnostics, notifications, and responsive state.
- [x] `AC-02` CodeMirror remains an imperative surface inside a React component and releases resources correctly.
- [x] `AC-03` The HTTP API, CAS conflicts, atomic saving, preview, and file restrictions retain their behavior and security checks.
- [x] `AC-04` Keyboard behavior, focus, errors, creation, and mobile mode are covered by tests; the prior shell is removed.

<!-- toudocu:section plan -->
## Plan

1. Wrap CodeMirror in a stable React lifecycle.
2. Move the shell and state without changing transport or the server part.
3. Test saving, conflict, preview, creation, and accessibility; remove the prior shell.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web test && npm --prefix web run test:browser`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `go test ./... && npm --prefix web run test:browser`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

Editor documents, runtime components, MOD-SITE, and the development guide
separate the React shell, imperative CodeMirror, and server contract.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task retains the existing Editor scenario and changes its shell only.
