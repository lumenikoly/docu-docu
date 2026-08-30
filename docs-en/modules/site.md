<!-- toudocu
id: MOD-SITE
status: done
updated: 2026-08-25
-->

# Static portal

The module produces backend-independent HTML pages, navigation, static JSON
resources, and a typed `report.json` from the completed project model.

## Purpose

Make project documentation convenient for people on ordinary HTTP(S) static
hosting or through local `serve`, while also providing the complete model to CI
and agents.

<!-- toudocu:section code-location -->
## Code location

- application services and report: `internal/app/site.go`, `internal/app/report_types.go`;
- typed bootstrap, templates, asset manifest, and embed: `internal/site/`;
- browser sources and packaging: `web/src/`, `web/vite.config.mjs`, and
  `web/package-assets.mjs`;
- derived embedded assets: `internal/site/assets/generated/`;
- process and use-case catalogs: `internal/app/process_site.go`;
- Screen Map, catalog, and screen pages: `internal/app/screen_site.go`;
- local HTTP serving and offline API docs: `internal/app/server.go`,
  `internal/app/api_docs.go`;
- editor workspace, API, and platform-specific atomic replacement: `internal/app/editor_*.go`;
- shared Editor and Changes shell: `internal/site/workspace.go`;
- theme and safe-branding configuration: `internal/app/site_config.go`.

<!-- toudocu:section boundaries -->
## Module boundaries

Static generation does not revalidate business entities or edit Markdown. Only
the explicit `serve` mode provides workspace operations, after which it rebuilds
the model through `MOD-MODEL`.

<!-- toudocu:section business-rules -->
## Business rules

### BR-SITE-001: Cleaning output does not affect protected directories

`--clean` rejects the system root, source documentation, its parent directories,
and direct output symlinks. The decision is based on resolved paths.

### BR-SITE-002: The portal works on static HTTP hosting

The `build` result requires no Toudocu backend, database, Node.js, CDN, or
external runtime. HTML, CSS, JavaScript, and JSON reside in output, use relative
URLs, and work both at the root of an HTTP(S) host and under a nested URL path.
Direct opening through `file://` is not supported. Use HTTP(S) hosting or
`toudocu serve`.

### BR-SITE-003: The local server exposes files only through explicit interfaces

Ordinary `serve` routes expose the built portal, not arbitrary filesystem
paths. The Editor API accepts only regular `.md`, `.yaml`, `.yml`, and `.json`
files inside the documentation root and rejects hidden, excluded, output, and
symlink paths.

Changes is a separate explicit interface in canonical `serve`. It can read
tracked and new non-ignored files inside the current repository to display the
diff, full UTF-8 text, and discussions. It validates path, kind, and size,
never returns absolute paths, and does not turn the server into a general file
browser. By default, the listener uses loopback; `--host 0.0.0.0` explicitly
includes reachable local-network clients in the trust boundary.

### BR-SITE-004: Mermaid works autonomously and in strict mode

The pinned classic Mermaid Tiny bundle is copied from `go:embed`, loaded only
when a diagram approaches the viewport, and started with `securityLevel: strict`.
A syntax error does not break the page: the portal shows a message and the
original diagram source.

### BR-SITE-005: The Screen Map works autonomously

The map, filters, SVG links, zoom, pan, sidebar, and step-by-step viewer work on
local JavaScript and CSS without a CDN or backend requests.

“Use Cases” is an independent top-level section for `UC-*`, while “Processes”
is the only catalog of `FLOW-*` documents at `processes/index.html`. The
canonical use-case page combines the description, map, playback, and
relationships. The “Screens” section opens the `SC-*` catalog; the overall map
is available as a separate item when its generation is enabled. Cards show the
number of incoming and outgoing transitions. Transition types differ by line
shape; hotspots appear on hover and focus, and the terminal screen links to the
map and the use-case description.

The main navigation follows the stable registry order of built-in sections; it
does not depend on Go map iteration order. Built-in section names come from
`project.sections`. The `flows` section is emitted under the `processes` route,
while `drafts` appears after work items and before guides. The generated
`drafts/index.html` catalog lists free-form Markdown drafts and does not require
a source `index.md`.

### BR-SITE-006: Themes do not expand the trusted surface

`classic`, `paper`, and `terminal`, their tokens, the color-scheme switch,
fallback favicon, and browser resources are embedded through `go:embed`.
Configuration selects only fixed options; custom CSS, fonts, and theme plugins
are not loaded.

Custom logo, favicon, and hero files are read only as regular files from
`.toudocu/assets/`, validated when the model is built, and copied to
`assets/branding/`. `build`, `check`, and `serve` use the same diagnostics and
remain offline-first.

### BR-SITE-007: Build and serve have different capabilities

`GenerateSite` always creates a backend-independent read-only result for static
HTTP hosting without editor markup, API UI, Swagger UI, CodeMirror,
discussions, or server-only rebuild code. It copies discovered OpenAPI specs as
ordinary portal assets. `serve` separately adds the live workspace,
editor/source actions, Changes and review capabilities, polling, API, watcher,
and vendored Swagger UI for canonical contracts.

### BR-SITE-008: Writes are protected by optimistic concurrency

Content is identified by a SHA-256 digest. Save checks the digest before and
after writing a same-directory temporary file, preserves mode, synchronizes the
data, and atomically replaces the source. A conflict does not lose local text
and requires a separate overwrite with the current digest and explicit
`confirmOverwrite`; if the source was deleted, the dirty buffer can be
downloaded. Diagnostics do not block saving.

### BR-SITE-009: Locale portals are peer workspaces

Configured `locales.<locale>` roots create independent workspaces at
`/_toudocu/locales/<locale>/`.
The switch receives URLs only from server-computed targets: Markdown is matched
by relative source path, a generated page by an existing output path, and
otherwise the locale homepage is used. Each mount receives Editor, Changes,
Task Workspace and Agent Console under the same network security policy and
writes only its own root. Each model remains single-language; Toudocu does not
synchronize another locale.

### BR-SITE-010: Soft navigation limited to canonical serve portal

Only the canonical portal in `serve` mode intercepts ordinary same-origin
transitions between HTML documents. It preloads up to the last eight pages
after pointer hover or keyboard focus, checks the workspace revision, and
replaces the document shell without a rebuild. Back/Forward, anchors, scroll
restoration, and main focus preserve browser semantics. Editor, changes, API,
locale, external, and special transitions always use full navigation; a network
error, unsuitable HTML, or a new revision also causes a full load.

The search index loads only on first use of search and remains in memory across
soft transitions. The Mermaid bundle loads when the first diagram approaches
and is reused until a full page load.

### BR-SITE-011: API docs remain offline and read-mostly

`/_toudocu/api-docs/` exists only in canonical `serve`, uses same-origin specs
and pinned Swagger UI 5.32.12 without a CDN. CSP prohibits external network
access, and Try it out is available only for `GET`/`HEAD`. Locale mounts, direct
translation serve, and static build contain no UI, assets, or navigation for it.

### BR-SITE-012: Work surfaces use consistent appearance

The canonical portal in `serve`, Editor, and Changes use the same `localStorage`
keys for `classic`/`paper`/`terminal` and `system`/`light`/`dark`. A shared
blocking `appearance.js` applies the saved theme, scheme, accent, density, and
content width before CSS loads and publishes `toudocu:themechange` on later
changes. Deferred surface bundles do not repeat this initialization.

Editor and Changes receive a shared header with project branding, “Portal /
Editor / Changes” navigation, an active `aria-current`, and theme selectors.
Their work actions remain in a separate contextual panel. CodeMirror switches
the theme compartment without recreating editor state, while an active Mermaid
diff rerenders without resetting the report, filters, or URL state.

Shared Portal, Editor, and Changes components use semantic `--td-*` variables.
`classic`, `paper`, and `terminal` themes; `system`, `light`, and `dark` color
schemes; and `comfortable` and `compact` density change those variables rather
than feature-specific palettes. Typography roles body, interface, heading, and
mono remain shared by all work surfaces. New shared controls use local Lucide
paths from `web/src/design/icons.ts` and `internal/app/icons.go` with line
weight `2`. Documents, workspace sections, and actions keep distinct icons by
meaning; a decorative SVG is hidden from the accessibility tree, while a
meaningful standalone icon has an accessible label.

### BR-SITE-013: Go explicitly defines frontend capabilities

Every page contains a safely serialized `application/json` bootstrap with
`schemaVersion`, runtime, page reference, relative asset/data bases, and
capabilities. Static runtime always disables `editor`, `changes`, `rebuild`, and
`taskWorkspace`. Serve-only endpoints occur only in the serve bootstrap and
remain same-origin. The frontend ignores unknown fields, but explicitly shows
an error when bootstrap is missing or its schema version is unsupported.

### BR-SITE-014: Roadmap changes use only a constrained operation

Canonical `serve` adds an action only on the `roadmap.md` page and only for a
new unfinished `DLV-*` in an existing H2 stage. Go returns the stages, validates
the ID and text with the same one-roadmap-ID rule, performs digest CAS, and
applies a targeted atomic insertion while preserving line endings. The frontend
does not parse Markdown or decide whether a write is allowed. `build`, locale
portals, and direct translation serve remain read-only.

In canonical `serve`, the button connects to an eagerly mounted React island.
Its Base UI dialog retains entered fields on `stale_digest`, but sends the same
single `roadmap-add` with `expectedDigest`; it creates neither a new browser API
nor a new source of truth.

### BR-SITE-015: Version check does not affect portal availability

Only canonical `serve` enables the version-check capability. On the first
same-origin request, Go fetches latest stable release metadata once from a
fixed GitHub endpoint, applies a timeout and response-size limit, and caches
the result until the process stops. A newer version is shown as a non-blocking
suggestion to open the official release; dismissal applies only to that
version. Every failure remains silent. `--no-update-check`, static builds,
locale mounts, and direct translation serves keep the capability disabled, the
endpoint unavailable, and perform no check.

### BR-SITE-016: A parent task shows its current subtree

A `TASK-*` page with children shows the current task and every descendant as a
nested tree. Each node has a link and text status from the shared project model;
static portal and `serve` produce the same view. A nested parent's page is
limited to its subtree while ancestors remain in breadcrumbs. Source Markdown
does not store the computed child list.

### BR-SITE-017: Side navigation shows the work-item hierarchy

Side navigation shows root `TASK-*` and `BUG-*` work items. Child tasks nest
under their parent and collapse independently; the open page and its ancestors
remain expanded. The selected expansion state is local and works consistently
in static portal and `serve`.

### BR-SITE-018: Work items have a specialized workspace

`work/index.html` uses Task Workspace rather than the generic document catalog.
Go prepares derived state, readiness, dependencies, `parentTask` relationships,
and acceptance-criteria progress; the browser only filters, groups, and changes
Board, List, and Tree. Static HTML already contains the Board and task links.
The workspace has no API to change status, run commands, or drag cards.

### BR-SITE-019: React is limited to application surfaces

Editor and Changes use separate React application roots, while canonical
`serve` uses `IslandHost`-managed islands. Regular Portal pages,
`appearance.ts`, `portal.ts`, and `serve.ts` do not depend on React. A new
island is allowed without a separate ADR when it obtains capabilities from
`PageBootstrap v1`, mounts dynamically, follows the shared lifecycle, and does
not own routing or the project model.

Editor and Changes receive a Go-generated shell and own DOM only inside their
root. CodeMirror and CodeMirror Merge keep their document, selection, viewport,
transactions, and diff state; React does not mirror it. Editor owns the tree,
actions, tabs, conflicts, diagnostics, notices, and creation dialog while
retaining server path, action-header, and CAS checks. Changes owns comparison
range, filters, file tree, detail shell, and notices, and renders prepared Go
projections. It retains an open detail until the user explicitly refreshes it.

Discussions is an activated island: a regular canonical page does not load
React until its panel opens or the user acts on a selection. Within a browser
session it restores the island on following pages to keep count and replies
current. The panel, composer, and confirmation use one React API; server state
remains the sole source of truth.

### BR-SITE-020: Islands isolate lifecycle and failure

`IslandHost.discover()` is idempotent and an island instance has at most one
React root. Before replacing `.site-layout`, soft navigation calls `unmountAll()`
only after complete target-page validation. It emits `toudocu:pagechange` after
replacing `PageBootstrap` and synchronizing assets.

A loading, feature-JSON, or first-render failure changes only that instance to
`data-td-island-state="error"`. Go-generated content, other islands, and
navigation continue. An activated island can restart after a user action; an
eager island retries after the next page transition.

### BR-SITE-021: Feature JSON does not decide bootstrap security

Go safely serializes a feature-specific immutable view model into
`application/json`. A mount point may reference it but does not hold
permissions, capabilities, endpoints, runtime, locale, theme, or absolute
paths. Those values come only from `PageBootstrap v1`. A feature validates its
minimal data shape and handles failure locally.

### BR-SITE-022: Base UI complements native HTML

Native HTML remains the base primitive. Base UI is used only for composable
behavior that a browser element does not provide itself: dialog, menu, context
menu, popover, tooltip, select, combobox, or complex tabs. Ordinary buttons,
links, labels, panels, headings, badges, and static tables have no mandatory
Base UI wrapper.

<!-- toudocu:section invariants -->
## Invariants

- the source `index.md` is displayed by the dashboard rather than a duplicate
  page: the home page presents project information, a compact current focus, no
  more than five recommended entry points, and the substantive content of
  `index.md` in order, without repeating its H1 or structural metadata inside
  the always-visible detailed overview;
- the current-status line exists only when status, roadmap, work items, or risks
  exist, names the status and nearest deliverable, and selects one destination
  in this order: `status.md` → deliverable document → work catalog → risks;
  counters and detailed items remain on the status page and in catalogs;
- pages for source Markdown documents, including the dashboard and canonical
  use case, allow the title and safe source path to be copied; dashboard actions
  sit inside the always-visible overview, where `serve` also exposes editor,
  source, and changes;
- side navigation colors its type icon according to recognized document status
  and, for `TASK-*` and `BUG-*`, additionally distinguishes incomplete `☐` from
  completed `☑`; the textual status label remains available regardless of color;
- the active navigation group is expanded, other groups are collapsed by
  default, and the user's explicit selection is stored locally;
- the dashboard does not duplicate the full catalog, detailed roadmap and risk
  cards, or active-task lists; global search, side navigation, and section
  catalogs provide the complete document overview;
- the detailed overview always shows the substantive part of `index.md`, and
  the print version preserves it in full;
- catalogs, Screen Map, traceability, and the health page do not emit synthetic
  document context;
- `work/index.html` is Task Workspace rather than a generic document catalog:
  Go prepares readiness, derived state, dependencies, `parentTask`, and
  acceptance-criteria progress, while the browser only filters, groups, and
  switches Board, List, and Tree; the initial HTML contains the Board and task
  links, and the workspace has no status mutation, command, or drag-and-drop API;
- in canonical `serve`, shared surface navigation opens portal, Editor, and
  Changes using full navigation, while rebuild remains a separate portal
  action; static output contains none of the special routes, actions, or
  serve-only assets;
- compact navigation preserves accessible names on 40×40 control surfaces;
  contextual panels collapse without horizontal page overflow, while trees,
  metrics, and diffs retain local scrolling;
- save/create and a stable external change synchronously update the model, HTML,
  search, diagnostics, and workspace revision;
- an ordinary HTTP request does not trigger rebuild; the watcher publishes a
  snapshot only after a successful build, and a locale rebuild does not change
  canonical editor or changes state;
- a soft transition in canonical `serve` does not trigger rebuild and accepts
  HTML only with the current workspace revision; watcher and manual rebuild end
  in a full reload that synchronizes runtime and snapshot;
- an ordinary page does not load the search index or Mermaid until search is
  used or a diagram approaches the viewport;
- Screen Map and playable flow are reinitialized for the new layout; when the
  DOM is replaced, the previous page lifecycle cancels listeners and observers;
- the serve-only roadmap dialog is reinitialized after soft navigation,
  preserves fields on a CAS conflict, and does not block page reading when the
  API fails;
- the update notice persists across soft navigation, is not shown again for a
  dismissed version, and never blocks the main content;
- a service-output conflict receives a separate safe path;
- `ProjectReport` and HTML are built from the same model;
- generated files never become editable documentation sources.

<!-- toudocu:section stable-interfaces -->
## Stable interfaces

- `GenerateSite`;
- `BuildReport`;
- CLI command `serve`;
- [Editor OpenAPI](../contracts/editor.openapi.yaml) and [Changes OpenAPI](../contracts/changes.openapi.yaml);
- [Editor API behavior](../contracts/editor-http.md) and [Changes API behavior](../contracts/changes-http.md);
- `ProjectReport` schema v1;
- HTML entrypoint `index.html` and machine-readable `report.json`.

<!-- toudocu:section related-use-cases -->
## Related use cases

- [UC-DOCS-01: Build the portal](../use-cases/build-portal.md)
- [UC-DOCS-03: Local server](../use-cases/serve-portal.md)
- [UC-DOCS-04: Screen Map](../use-cases/screen-map.md)

## Related processes

- [FLOW-DOCS-BUILD: Build a static HTTP portal](../flows/FLOW-DOCS-BUILD.md)
- [FLOW-DOCS-SERVE: Browse the portal locally](../flows/FLOW-DOCS-SERVE.md)
