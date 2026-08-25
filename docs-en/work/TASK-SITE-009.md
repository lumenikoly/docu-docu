<!-- toudocu
id: TASK-SITE-009
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-008
-->

# TASK-SITE-009: Introduce a Semantic Design System and Unified Icon Contract

<!-- toudocu:section result -->
## Result

Portal, Editor, and Changes use one semantic visual contract; their appearance
may initially remain close to the existing design.

<!-- toudocu:section scope -->
## Scope

- Tokens, themes, typography, motion, layout, and icons in `web/`.
- The design-system contract in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- A complete visual redesign, removing every prior icon, and publishing the
  design system as npm packages.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` `--td-*` variables cover surfaces, text, borders, accent, statuses, focus, selection, spacing, radii, control height, and motion.
- [x] `AC-02` The body/interface/heading/mono typography roles, classic/paper/terminal themes, system/light/dark color schemes, and compact/comfortable density are retained.
- [x] `AC-03` Theme and density change semantic tokens without feature-specific palettes.
- [x] `AC-04` A unified local Lucide-based SVG icon contract defines line weight, accessible labels, and `aria-hidden` for decorative icons.

<!-- toudocu:section plan -->
## Plan

1. Introduce minimal semantic scales and existing appearance options.
2. Move shared styles to `--td-*` without redesigning them.
3. Add an icon registry and accessibility contract.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `make web-check`
- `AC-02` → `make browser-test`
- `AC-03` → `make web-check && make browser-test`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The development guide and MOD-SITE describe tokens, themes, density, typography
roles, and icons.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task establishes a shared visual contract without a separate user scenario.
