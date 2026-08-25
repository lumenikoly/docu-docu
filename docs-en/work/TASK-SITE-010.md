<!-- toudocu
id: TASK-SITE-010
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-009
-->

# TASK-SITE-010: Introduce the Shared React/Base UI Foundation

<!-- toudocu:section result -->
## Result

Toudocu has a testable, reusable React 19 and Base UI layer that is not tied to
one large application surface.

<!-- toudocu:section scope -->
## Scope

- React/Base UI dependencies, `ui`, `docs-ui`, tests, and a lightweight
  component-development page in `web/`.
- Shared-layer boundaries in `docs/`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- Migrating application surfaces, Storybook, an npm workspace, and premature
  package extraction.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] `AC-01` React 19, React DOM, Base UI, Vitest, React Testing Library, and user-event are included with pinned versions and licenses.
- [x] `AC-02` `ui` provides Button, IconButton, Badge, Separator, Spinner, EmptyState, Diagnostic, Dialog, Tabs, Tooltip, Popover, Menu, and Select; complex controls use Base UI.
- [x] `AC-03` `ui` knows neither `PageBootstrap`, HTTP, nor localization keys; `docs-ui` receives translator and view model through parameters, makes no backend requests, and does not read `window.ToudocuPage`.
- [x] `AC-04` `web/dev/ui.html` replaces Storybook; tests cover keyboard behavior, focus, Escape, tabs, dialogs, menus, labels, and reduced motion.

<!-- toudocu:section plan -->
## Plan

1. Add the minimal runtime and test dependencies.
2. Create `ui`/`docs-ui` boundaries and shared controls.
3. Add the development page and component tests.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `npm --prefix web run build`
- `AC-02` → `npm --prefix web test`
- `AC-03` → `npm --prefix web run typecheck && npm --prefix web test`
- `AC-04` → `npm --prefix web test && npm --prefix web run test:browser`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Documentation impact

The development guide and module boundaries explain dependencies, `ui`/`docs-ui`,
localization, and the development page.

<!-- toudocu:section use-case-omission-reason -->
## Use-case omission reason

The task creates shared component infrastructure without a new user journey.
