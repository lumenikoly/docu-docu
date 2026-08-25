<!-- toudocu
id: TASK-SITE-008
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-007
-->

# TASK-SITE-008: Move Browser and Asset Packaging to Vite 8

<!-- toudocu:section result -->
## Result

Vite 8 packages browser assets instead of the esbuild application build, while
preserving the chain “sources → working assets → committed generated assets →
`go:embed` → one binary” and Portal behavior.

<!-- toudocu:section scope -->
## Scope

- Vite dependencies, configuration, and packaging scripts in `web/`.
- Assets in `internal/site/`, commands in `Makefile`, CI, and release packaging
  in `.github/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Moving the interface to React, visual redesign, Editor, Changes, Discussions,
  and Roadmap.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` Vite 8 packages multiple entry points through `build.rolldownOptions` and creates a reproducible Vite manifest.
- [x] `AC-02` A separate `package-assets.mjs` calculates the transitive asset closure for `static` and `serve`, creates the Toudocu manifest, SHA-256 values, and JSON licenses, and retains Mermaid and Swagger UI notices.
- [x] `AC-03` `static` and `serve` assets are isolated; a nested URL does not depend on absolute `/assets/...`, and `appearance.js` loads before CSS.
- [x] `AC-04` Generated assets are reproducible, the prior esbuild application build is removed, and release and program operation need no Node.js.

<!-- toudocu:section plan -->
## Plan

1. Introduce Vite packaging for multiple entry points.
2. Extract packaging, asset closure, licenses, and hashes.
3. Move embedding, Makefile, CI, and release packaging; remove the prior build script.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web run build`
- `AC-02` → `make web-check`
- `AC-03` → `make browser-test`
- `AC-04` → `make web && git diff --exit-code -- internal/site/assets/generated && make build`
- `ALL` → `make web-check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Documentation impact

Asset architecture, development guidance, release packaging, and license notices
describe Vite and the new manifest.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The packaging infrastructure changes while observable user behavior remains the same.
