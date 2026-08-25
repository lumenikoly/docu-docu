<!-- toudocu
id: TASK-SITE-015
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-012
-->

# TASK-SITE-015: Move the Changes Workspace to React

<!-- toudocu:section result -->
## Result

The Changes workspace shell runs on React/Base UI and reuses the new
Discussions interface while keeping Git and diff calculation in Go.

<!-- toudocu:section scope -->
## Scope

- Controls, file list, detail shell, and review interface in `web/`.
- Changes integration in `internal/site/`, tests, and documents in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Moving Git, semantic and rendered diffs to React; rewriting CodeMirror Merge;
  or changing the Changes API, review anchors, or Git semantics.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` React manages range, filters, file list, detail shell, notices, mobile file panel, review editor, related-file selection, and state messages.
- [x] `AC-02` Changes reuses Discussions components and API from TASK-SITE-012 without a second implementation.
- [x] `AC-03` `ChangeSetReport`, the Changes API, semantic and rendered diffs, CodeMirror Merge, review anchors, and Git semantics are retained.
- [x] `AC-04` Range, filters, file selection, review, errors, mobile behavior, and cleanup are covered by tests; the prior shell is removed.

<!-- toudocu:section plan -->
## Plan

1. Move the shell and local state to React.
2. Embed the shared Discussions interface and retain API and view model.
3. Test diffing, change review, mobile behavior, and cleanup; remove the DOM shell.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web test && npm --prefix web run test:browser`
- `AC-02` → `npm --prefix web test`
- `AC-03` → `go test ./... && npm --prefix web run test:browser`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

Changes and review documents, runtime components, and MOD-SITE separate the
React shell from unchanged Go, Git, and diff contracts.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task retains the existing Changes scenario and changes its browser implementation.
