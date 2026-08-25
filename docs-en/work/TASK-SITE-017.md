<!-- toudocu
id: TASK-SITE-017
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-016
-->

# TASK-SITE-017: Complete the Migration and Remove the Previous UI Foundation

<!-- toudocu:section result -->
## Result

The working version contains only the new browser architecture; temporary
bridges and the previous UI foundation are removed, and `static`, `serve`, and
release pass the full integration cycle.

<!-- toudocu:section scope -->
## Scope

- Removing the previous implementation in `web/` and `internal/site/`.
- Full verification through `Makefile` and final documentation in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- New capabilities, retaining compatibility bridges, and documentation of a
  transitional state.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` Helper DOM components, manual Dialog/Tabs/Tooltip/Menu, unused selectors, prior variables, Unicode interface icons, stale CSS, the esbuild application build, and compatibility bridges are removed.
- [x] `AC-02` `static` at the root and nested path does not load React on a regular page; search, themes, and Mermaid work.
- [x] `AC-03` `serve` passes soft navigation, island lifecycle, Roadmap, Discussions, Editor, Changes, update checking, and translation and API Docs isolation.
- [x] `AC-04` The binary without Node.js performs `check`, `build`, and `serve`; documentation describes only the final architecture.

<!-- toudocu:section plan -->
## Plan

1. Remove remnants of the previous foundation and compatibility bridges.
2. Test critical `static` and `serve` paths.
3. Run release verification without Node.js.
4. Update the browser boundary, modules, guide, reference, and affected typed documents.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `make web-check && git diff --exit-code -- internal/site/assets/generated`
- `AC-02` → `make browser-test`
- `AC-03` → `make browser-test && go test ./...`
- `AC-04` → `go test ./... -run TestReleasedBinaryWithoutNodeRuntime && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `make check && make web-check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The browser runtime boundary, runtime components, MOD-SITE, development guide,
feature reference, and every affected use case, flow, screen, and contract are
updated; transitional wording is removed.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task completes a technical migration of existing scenarios and adds no new user journey.
