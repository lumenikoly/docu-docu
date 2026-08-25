<!-- toudocu
id: TASK-SITE-016
status: done
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-013, TASK-SITE-014, TASK-SITE-015
-->

# TASK-SITE-016: Deliver the Target Visual Redesign

<!-- toudocu:section result -->
## Result

The new design system becomes the visible, consistent interface of a
professional developer tool: dense but readable, with subtle borders, limited
shadows, moderate radii, one accent, and keyboard-first operation.

<!-- toudocu:section scope -->
## Scope

- Shared presentation, controls, and states for Portal, Editor, and Changes in `web/`.
- Description of the visual language in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Changing architecture, API, or capabilities, and a generic SaaS dashboard.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` Header, navigation, sidebar, global search, buttons, selection lists, tabs, dialogs, forms, and statuses use one visual language.
- [x] `AC-02` Task Workspace, Roadmap, Discussions, Editor, and Changes align while Portal remains content-oriented.
- [x] `AC-03` Diagnostic, loading, empty, and error states are understandable, keyboard accessible, and do not depend on color alone.
- [x] `AC-04` Mobile states, themes, density, and reduced motion pass browser tests without behavior changes.

<!-- toudocu:section plan -->
## Plan

1. Apply typography, layout, borders, and accent to shared presentation.
2. Align controls and states of migrated surfaces.
3. Test themes, density, keyboard behavior, reduced motion, and mobile mode.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web run test:browser -- --grep 'visual language'`
- `AC-02` → `npm --prefix web run test:browser -- --grep 'Portal and workspaces'`
- `AC-03` → `npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The development guide and screen descriptions reflect the final visual language,
accessibility, and distinction between Portal and workspaces.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task aligns presentation of existing scenarios and adds no new journey.
