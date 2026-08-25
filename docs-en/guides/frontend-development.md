# Frontend development

Frontend source lives in `web/`; Node.js is only a developer build/test
toolchain. Ordinary users and `go build ./...` use the committed assets from
`internal/site/assets/generated/`.

```bash
make web
make web-check
make test
make build
```

The frontend-only loop inside `web/` is:

```bash
npm ci
npm run typecheck
npm test
npm run build
npm run test:browser
```

It does not replace Go vet/tests/race, binary builds, or generated-asset drift
checks performed by the corresponding Make targets.

TypeScript runs in strict mode, and Vite 8 packages multiple JavaScript and CSS
entry points. After Vite, `package-assets.mjs` calculates asset closures for
`static` and `serve` from its manifest, adds pinned Mermaid and Swagger UI, and
creates the Toudocu manifest with SHA-256. The build also retains
`vite-manifest.json` and `licenses.json` for graph and license checks.

Generated asset names and content are deterministic: timestamps and random
values are forbidden. After changing frontend source, commit the rebuilt
`internal/site/assets/generated/`; CI repeats the build and fails on drift.

`appearance.js` and `portal.js` are used by both static and `serve` portals.
`appearance.js` loads before CSS so the saved theme is already active in the
first frame. `serve.js`, `editor.js`, `changes.js`, the roadmap dialog,
CodeMirror, and Swagger UI are available only in `serve`.

`appearance.ts`, `portal.ts`, and `serve.ts` have no static dependency on
React, React DOM, or Base UI. Editor and Changes are separate React application
roots. Roadmap loads as an eager island, while Discussions is an activated
island that loads only after a user action. A regular Portal page does not load
React.

## Browser-code boundaries

Source is organized into internal `design`, `ui`, `docs-ui`, `features`, and
`entries` directories; separate npm packages and workspaces are unnecessary.
Imports point upward only:

- `ui` uses `design`;
- `docs-ui` uses `ui` and `design`;
- `features` use lower UI layers and permitted browser infrastructure from `core`;
- `entries` compose features and infrastructure.

`design`, `ui`, and `docs-ui` do not import `PageBootstrap`, API clients,
product events, `window.ToudocuPage`, or catalogs through `core`. Generic UI
receives strings as props. Documentation-oriented UI receives view models,
callbacks, and a `Translator`, but does not call endpoints itself. Automated
import checks protect this boundary.

Use native HTML for ordinary buttons, links, labels, layout, and static
elements. Add Base UI only when a composable accessibility contract is needed,
such as a dialog, menu, popover, tooltip, select, combobox, or complex tabs.

The shared React layer is in `web/src/ui/`: Button, IconButton, Badge,
Separator, Spinner, EmptyState, and Diagnostic use native HTML, while Dialog,
Tabs, Tooltip, Popover, Menu, and Select provide Base UI compositions.
`web/src/docs-ui/` accepts prepared view models, callbacks, and a Translator;
network access stays in the feature layer.

A lightweight component gallery is available at `/dev/ui.html` during
`npm run dev`. It replaces a separate Storybook and is not included in released
portal assets.

## Visual contract

Shared components use semantic `--td-*` variables from
`web/src/styles/tokens.css`. The contract covers surfaces, text, borders,
accent, states, focus, selection, spacing, radii, control height, and motion.
CSS uses only canonical `--td-*` names; themes set the same semantic variables
and do not create feature-specific palettes.

`--td-font-body`, `--td-font-interface`, `--td-font-heading`, and
`--td-font-mono` define typography roles. `data-site-theme` values `classic`,
`paper`, and `terminal`, `data-theme` for light or dark appearance, and
`data-density` values `comfortable` and `compact` change the same set of
semantic tokens.

The resulting language remains utilitarian: subtle borders separate areas,
shadows mark only raised layers, radii are moderate, and one accent marks a
selection or action. Portal keeps a wide reading column and calm content
hierarchy; Editor and Changes use a denser workspace grid. Task Workspace,
Roadmap, and Discussions use the same control heights, focus outline, forms,
tabs, dialogs, and statuses. Loading, empty, and error states have text or an
accessible name and do not convey meaning through color alone.

Browser components use local Lucide paths from `web/src/design/icons.ts`, while
the generated Go shell uses `internal/app/icons.go`. Add an icon to its relevant
registry and create it through `createIcon`, `Icon`, or `renderIcon`. `Icon` and
`renderIcon` create decorative icons only, with `aria-hidden="true"`; an
accessible name belongs to the containing button or link. `createIcon` also
supports a meaningful standalone icon with `role="img"` and `aria-label`. Line
weight is `2`; no external package or network load is needed.

## React roots and islands

Editor and Changes own the DOM only inside their assigned root. Access to title,
focus, appearance, and navigation uses bounded browser-infrastructure services,
not arbitrary shell modification. CodeMirror and CodeMirror Merge own document,
selection, viewport, transactions, and diff state; do not duplicate this
latency-sensitive state in React.

`web/src/core/react/island-host.ts` provides React-independent `discover`,
`activate`, `mount`, `unmount`, and `unmountAll` operations. Its registry loads
feature modules through dynamic import. A mount point holds its name in
`data-td-island`, a unique instance in `data-td-island-instance`, `activated`
mode for deferred startup, and, when needed, a `data-td-island-model` reference
to `application/json` with an immutable view model. Do not put runtime,
capabilities, permissions, endpoints, locale, theme, or absolute paths there.
One failed island receives local `error` state and does not remove useful
Go-generated fallback. A localized fallback is marked inside the mount point by
`data-td-island-error` and initially hidden. Before first render, a feature
registers cleanup for its root through `onCleanup`, so a rendering error cannot
leave a root behind for repeat activation.

Soft navigation fully validates the target page and `PageBootstrap` before
`toudocu:pagebeforechange`, releases islands, replaces layout and bootstrap,
emits `toudocu:pagechange`, and discovers eager islands on the new page. On a
failure before the commit boundary, current roots remain and the browser makes a
normal full navigation.

Use `createRoot`, not React SSR or hydration. A new island needs no ADR while it
obeys the common lifecycle and capability contract and does not own routing or
the domain model.

Project modeling, document classification, path guards, semantic diff, and
decisions about command execution remain in Go.

All Changes editors use one-based Unicode coordinates. Syntax highlighting is
pinned for Go, Java, JavaScript, JSX, TypeScript, and TSX. Other valid UTF-8
files appear as plain text. Go validates the path, selected text, context, size
limits, and anchor relocation.

React keeps the range, filters, list, and detail shell; comparison content comes
from the Changes API. Do not calculate Git, rendered, or semantic diffs in the
browser. Always return cleanup for CodeMirror, Mermaid, polling, and external
events. If polling finds a new revision while a file is open, keep its current
detail and offer explicit refresh.

In Editor, React manages the workspace while `CodeEditor` remains a thin
CodeMirror adapter: create an instance only for the current `path` and `digest`,
pass changes outward, and always call `destroy()` during cleanup. Do not move
selection, viewport, transactions, or path checking to React; saving, creation,
validation, and preview must use the existing Editor HTTP API and its action
headers.

## Related documents

- [ADR-008: React islands and Vite without turning Portal into an SPA](../decisions/ADR-008.md)
- [Go/frontend boundary](../architecture/frontend-runtime-boundary.md)
- [MOD-SITE](../modules/site.md)
- [Testing changes](testing.md)
