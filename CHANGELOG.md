# Changelog

## 0.0.6

- Added recursive task trees to work-item pages and documented dependency-aware
  task decomposition.
- Reduced Changes workspace requests by returning rendered Markdown with file
  details, caching Git status during report construction, and pausing polling
  while the page is hidden or busy.
- Derived the default portal brand mark from the project title when no logo is
  configured.
- Expanded the embedded skill guidance for focused Toudocu discovery and safer
  work-item planning.

## 0.0.5

- Removed redundant per-annotation `version: 1` metadata from the contract,
  scaffolds, templates, and documentation.
- Added an explicit documentation contract version. Projects on an older
  contract now receive one migration diagnostic, while projects created by a
  newer Toudocu version are rejected without partial parsing.
- Added `task tree`, parent/child task relationships, hierarchy-aware task
  context, and change reports that include affected descendants.
- Standardized CLI diagnostics and generated report labels in English while
  preserving reader-authored document values.
- Added semantic change analysis and improved annotations, Changes rendering,
  and CodeMirror behavior.

## 0.0.4

- Portal and Changes now share one accessible discussion panel implementation,
  keeping their controls and labels consistent while reducing duplicated browser
  code.
- Installation, change-reporting, quality, runbook, configuration, and document
  model guidance now describes the supported workflows more explicitly.
- The embedded Toudocu skill is now a shorter routing and safety layer: explicit
  operations take precedence, read-only reviews load only applicable guidance,
  Context7-specific instructions are removed, and activation cases are more
  precise.

## 0.0.3

- Document selections are now hints: when rendered text, such as a Mermaid
  label, cannot be matched to the Markdown source, Toudocu keeps the question
  as a document-level discussion and passes the original selection to the
  agent. Exact matches still retain their range, and file selections in
  Changes remain strict.

## 0.0.2

- Changes discussions now support every regular file in the working diff,
  including whole-file targets for binary, large, and deleted files. Index and
  revision comparisons remain read-only.

- Added the built-in `drafts/` section for free-form Markdown drafts.
- Drafts appear in navigation, search, the static portal, and `ProjectReport`
  with `sectionType: "drafts"` and `type: "draft"`.
- Editor creates a draft from its required title, derives a safe file name,
  adds sequential suffixes on collisions, and never overwrites an existing
  file.
- Existing complete 12-section configurations receive a localized `drafts`
  label automatically; an explicit setting still takes precedence.
- Updated Russian and English documentation, the OpenAPI contract, and the
  embedded portal assets.

## 0.0.1

### Documentation and portal

- Added Markdown validation, search, static portal generation, and local
  `serve` with an editor and automatic rebuilds.
- The document model covers architecture questions, modules, rules, use cases,
  Mermaid processes, screens, transitions, standards, runbooks, and work items.
- The portal provides dedicated catalogs for known entities, search,
  relationships, a screen map, themes, and localization.
- A `UC-*` is complete only when its status group is `done`, it has at least
  one acceptance criterion, and every criterion is checked. The roadmap,
  statistics, and diagnostics use this single calculation, while
  `ProjectReport` retains schema version 1 and
  `completionSource: "use-case-status"`.
- `build` creates a read-only portal that needs no running Toudocu process
  after generation. It can be published over HTTP(S), including below a nested
  URL path.
- Opening `index.html` directly through `file://` is not supported. Use
  `toudocu serve` for local viewing.
- Browser code lives in `web/`. Built TypeScript and CSS assets are committed
  and embedded in the Go binary, so users do not need Node.js.
- Static and local modes are separate: Editor, Changes, CodeMirror, Swagger UI,
  and local APIs are available only under `serve`.
- Two OpenAPI 3.1.0 contracts describe the Editor and Changes APIs. Canonical
  `serve` displays them with the bundled Swagger UI 5.32.12 without a CDN.
- Toudocu uses Goldmark 1.8.5 and one CommonMark/GFM parser with tables, task
  lists, strikethrough, and literal autolinks enabled.
- Raw HTML and leading front matter enclosed by matching `---` or `+++` lines
  are errors. The safe preview still renders that source as escaped text.
- The landing page and documentation portal use the same favicon.

### Changes and local discussions

- `changes`, `changes file`, and `task changes` report documentation changes
  in text, JSON, and Markdown without modifying the working tree, index, refs,
  or history.
- The `/changes/` workspace covers the local repository, opens the first
  matching file automatically, and shows the exact Git diff.
- The workspace can display a complete UTF-8 file and filter all files,
  documentation, or other files. Its primary tab is named “Diff”.
- Git range details and diagnostics live in disclosure panels. On narrow
  screens, the file list and discussions open as overlays without making the
  page scroll horizontally.
- The Changes panel opens over the workspace from the right and shares the
  Portal discussion panel's dimensions, scrim, focus handling, and mobile
  behavior.
- In Portal and Changes, users can select Markdown content, including a
  heading, copy its text or context, and create a question or change request.
- The unified diff preserves the browser selection and lets users copy the
  text or context or attach a question. New lines receive an exact range;
  removed lines from a modified document are quoted without introducing a new
  anchor type. The “+” control appears when an eligible line is hovered or
  focused and remains visible on touch devices.
- Discussions and anchors are stored locally outside the repository. Toudocu
  reanchors an unambiguous passage after an edit or marks it as stale or
  deleted.
- Saving a question immediately creates a pending queue record. Until the agent
  receives it, the message can be edited or deleted; cancelling an empty form
  does not trigger required-field validation.
- Discussion anchors account for browser whitespace normalization and the
  selected occurrence of repeated text.
- A shared discussion button appears in the header of every page served from
  the canonical root. Its panel lists all project threads but permits a new
  thread only for the current canonical document.
- “Copy prompt” copies an explicit request to process the queue and never
  pretends that an agent has started.
- The installed skill receives the oldest record through
  `toudocu agent next --json` and returns a structured response through
  `toudocu agent respond`. A question does not authorize documentation edits;
  a change request first requires fact verification.
- Complete-file views highlight Go, Java, JavaScript, and TypeScript. Other
  valid UTF-8 files remain available as plain text.

### Commands, work items, and embedded skill

- The CLI provides `check`, `build`, `serve`, `search`, `changes`, `scaffold`,
  and the complete work-item lifecycle.
- `task verify` does not execute commands unless `--run` is explicitly present.
- Public JSON reports use `schemaVersion: 1` and stable exit codes.
- The embedded skill can be installed, checked, updated, and removed for Codex,
  Claude Code, and Copilot. Toudocu does not overwrite local changes in its
  managed copy.
- `$toudocu init`, `$toudocu refresh`, and `$toudocu translate` are embedded
  skill workflows. Documentation requests use the separate provider-neutral
  CLI transport `toudocu agent next|respond`.
- `$toudocu refresh diff` starts from staged, unstaged, and untracked changes
  relative to `HEAD`, then includes dependent documents.
- `$toudocu translate diff` updates every configured language mirror in order.
  Translation roots remain isolated and read-only.
- `--include-assets` and `--translation-input` explicitly include binary assets
  and complete translation input.
- `make update-local` builds the current source and updates the local binary in
  `$(INSTALL_DIR)`.

### Distribution

- The release bundle contains AMD64 and ARM64 binaries for Linux, macOS, and
  Windows, including Windows ARM64.
- The release workflow distinguishes stable releases from `X.Y.Z-rc.N`
  candidates. An RC is published as a prerelease only after an explicit
  workflow dispatch.
- POSIX and PowerShell installers select the correct binary, verify its SHA-256
  and reported version, and only then replace the user's executable.
- `releases/latest` installs the newest stable release.
- The root Go package provides a typed facade for embedding Toudocu from the
  source tree. The CLI remains the primary public interface.
