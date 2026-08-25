# Configuration reference

The current documentation contract requires `.toudocu/config.yml` at the
repository root. If the file is missing, Toudocu treats the project as using
contract v1 and requires a migration. Toudocu parses the whole configuration
before running the command, so an invalid shared setting or branding asset can
stop an operation that does not use that field directly.

`build`, `check`, and `serve` use project and site settings; `changes` and
`task changes` use `changes`; `task init` and `scaffold` use the project locale.

The root-level `documentationVersion` field selects the source-documentation
contract:

```yaml
documentationVersion: 2
```

The current contract version is `2`. A missing field means version `1` and
requires migration through the Toudocu skill. A value below the current
version stops processing before Markdown is read and returns
`DOCS_MIGRATION_REQUIRED`; a value above it returns
`DOCUMENTATION_VERSION_UNSUPPORTED`. The field must be a positive integer. It
describes the expected format, not the result of the latest check.

## Defaults

| Setting | Default |
|---|---|
| Documentation contract | `2`; a missing field means legacy v1 |
| Documentation root | `./docs` |
| Output | Neighboring `project-docs` directory |
| Repository root | Parent of the documentation root |
| Repository ref | `main` |
| Stale threshold | `90` days |
| CLI format | `text` |
| Per-command task timeout | `10m` |
| `serve` host | `127.0.0.1` |
| `serve` port | `8080` |
| Version check | Enabled; disabled by `--no-update-check` |
| Screen map | Enabled when `screens/SC-*.md` exists |
| Theme | `classic` |
| Color scheme | `system` |
| Accent | `indigo` |
| Density | `comfortable` |
| Content width | `standard` (920 px) |
| Hero | Enabled |

## Portal appearance

```yaml
site:
  title: My Project
  logo: assets/logo.svg
  favicon: assets/favicon.svg
  theme: classic
  colorScheme: system
  accent: indigo
  density: comfortable
  contentWidth: standard
  footer:
    text: My Project documentation
    url: https://example.com
  hero:
    enabled: true
    image: assets/hero.webp
```

Accepted values are:

- `theme`: `classic`, `paper`, `terminal`;
- `colorScheme`: `light`, `dark`, `system`;
- `accent`: `indigo`, `blue`, `teal`, `green`, `amber`, `rose`, `violet`;
- `density`: `compact`, `comfortable`;
- `contentWidth`: `narrow` (760 px), `standard` (920 px), `wide` (1120 px).

The title is selected in this order: `--title`, `site.title`, the H1 from
`index.md`, then the directory name. By default, the Toudocu footer label links
to the project landing page. `footer.text` replaces the label, and
`footer.url` adds an optional HTTPS link.

Visitors can switch theme and color scheme in the browser. Their choice is
stored locally; configuration remains the initial value for a new visitor.

Logo, favicon, and hero image must be regular files under `.toudocu/assets/`.
Absolute paths, `..`, symbolic links, and missing files are rejected. SVG is
served as a separate file and is never inserted into HTML.
When `site.logo` is absent, the header shows a short text mark from the final
project title: the first characters of its first two words, or the first two
characters of one word. The result is uppercased. It uses letters and digits;
when there are none, it uses `T`.

The configuration parser supports mappings, strings, booleans, and comments.
Lists, YAML anchors and aliases, multiline values, unknown keys, and duplicate
keys are forbidden. There is no custom CSS, web-font, or theme-plugin support.

## Repository root

`--repository-root` defines:

- the boundary for links from documentation to code;
- the working directory for `task verify --run`;
- the base for work-item Scope paths.

Point it at the repository's actual top level:

```bash
toudocu check ./docs --repository-root . --strict
```

When both `--repository-url` and an exact `--repository-ref` are set, links to
files outside `docs/` but inside the repository become HTTP(S) `blob` or `tree`
links. Use a commit SHA instead of a moving branch for reproducible output.

## Excludes and stale dates

`--exclude` accepts a comma-separated list and may be repeated. VCS folders,
`node_modules`, `vendor`, `dist`, `build`, coverage output, hidden files, and
symbolic links are excluded by default.

A document date comes from Last updated, then Date, then file mtime.
`--stale-days 0` disables age warnings.

Without `--strict`, only errors make the command fail. With `--strict`, any
warning also produces exit code `1`.

## Screen map

`--screen-map` enables the overall map. `--no-screen-map` removes only that page
and link; the screen catalog, `SC-*` pages, use-case tabs, relationships, and
`report.json` model remain. Both flags are available to `build` and `serve` and
do not change Markdown.

## Git changes

```yaml
changes:
  defaultBaseRef: main
  renameSimilarity: 60
  includeTaskArtifacts: true
  includeAssets: true
  semanticDiff: true
  renderedDiff: true
  maxSourceDiffBytes: 2097152
  maxRenderedFileBytes: 1048576
  exclude:
    - docs/generated/**
    - docs/cache/**
```

This section is optional. Without an explicit range, Toudocu compares
`HEAD → working-tree`. `defaultBaseRef` is used only when no base is supplied
and is never fetched. A size limit disables a heavy view for one file, not the
whole report.

`--include-assets` temporarily includes binary assets while preserving
`changes.exclude`. `--translation-input` selects the complete reader-facing
set: it includes work items and assets regardless of ordinary include settings
and custom excludes, while keeping only `generated/**` and `cache/**` excluded
inside the documentation root.

## Local server

`--host` and `--port` apply only to `serve`. The default is
`127.0.0.1:8080`. `0.0.0.0` exposes the server to the local network without TLS
or authentication, and the CLI prints a warning.

The browser opens only when `--open` is set. There is no `--no-open` or
`--edit` flag.

The main `serve` portal adds Editor, manual rebuild, Changes, discussions, and
`/_toudocu/api-docs/` on one local listener. Static `build` output has no
CodeMirror, Swagger UI, API URLs, or server assets, although it copies detected
OpenAPI files.

On its first request, the main portal may check the latest stable release and
cache the result until shutdown. For a fully self-contained run:

```bash
toudocu serve --no-update-check ./docs
```

The editor workspace includes only regular `.md`, `.yaml`, `.yml`, and `.json`
files inside the documentation root. Hidden paths, configured excludes, output,
and symbolic links are omitted. JSON requests are limited to 3 MiB and file
content to 2 MiB. Same-origin browser checks do not authenticate a direct local
network client.

## Locale and section names

```yaml
project:
  locale: en
  sections:
    architecture: Architecture
    modules: Modules
    use-cases: Use Cases
    flows: Processes
    screens: Screens
    decisions: Architecture Decisions
    contracts: Contracts
    quality: Quality Standards
    runbooks: Runbooks
    reference: Reference
    work: Work Items
    drafts: Drafts
    guides: Guides
```

`project.locale` accepts a normalizable BCP-47-style tag such as `en`,
`en-GB`, `pt-BR`, or `sr-Latn`. `project.sections` defines all 13 built-in
navigation labels. The entry document's H1 must match its explicit label and is
not used as a fallback.

An existing complete 12-label configuration remains valid: Toudocu adds the
localized `drafts` label (`Черновики` for `ru-*`, `Drafts` otherwise). An
explicit `project.sections.drafts` value takes precedence. Without a locale or
complete section list, Toudocu uses English fallback labels
and reports a warning; strict mode treats it as a failure. HTML defaults to
`lang="en"` when no locale is configured.

`project.locale` also selects the portal's built-in interface language. `ru`
and regional `ru-*` variants select the Russian catalog; `en`, any other valid
tag, and a missing setting select English. The same rule applies to server HTML
and browser states. Toudocu does not translate `project.sections`, a custom
footer, or the contents of the selected documentation root.

Without explicit `--lang`, `task init` and `scaffold` use `ru` or `en` from
`project.locale`, including regional variants. Another or missing locale falls
back to `en`; explicit `--lang` takes precedence.

## Translations

```yaml
translations:
  en:
    root: docs-en
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use Cases
      flows: Processes
      screens: Screens
      decisions: Architecture Decisions
      contracts: Contracts
      quality: Quality Standards
      runbooks: Runbooks
      reference: Reference
      work: Work Items
      drafts: Drafts
      guides: Guides
```

Each profile defines a separate root and all 13 labels. An existing complete
12-label profile receives the localized `drafts` label. Its path is relative to
the repository root, stays inside that root, and cannot be absolute, contain
`..`, be a symbolic link, overlap the canonical root, or intersect another
translation root.

When `check`, `build`, or `serve` targets the translation root directly, its
profile temporarily supplies the locale and labels. Allowed operations are
check, build, search, ordinary changes, and read-only `serve`. Work-item
commands, `scaffold`, and Editor writes return
`TRANSLATION_ROOT_READ_ONLY`.

The canonical root remains the only source for ordinary analysis and task
context. An agent reads a translation only on an explicit request for that
locale; `translate diff` processes configured locales one at a time.
Translation roots are not added to ignore files.

`.toudocu/translations/<locale>.json` stores the SHA-256 of every source
Markdown file. The complete process is in the
[AI skill guide](../guides/agent-workflows.md#translation).

## Mermaid

Mermaid has no user configuration. Toudocu fixes:

- `flowchart`, `stateDiagram-v2`, and `sequenceDiagram`;
- at most 50,000 UTF-8 bytes per block;
- `securityLevel: strict`;
- no front matter or directives;
- colors calculated from the active theme.

## Safe cleaning

`--clean` applies only to a separate output directory. It cannot clean a system
root, the source documentation, a parent of the source, or a path that is
itself a symbolic link. Toudocu compares resolved real paths.

## Work-item reports and timeouts

`--report <file.json>` atomically writes `TaskVerifyReport` outside source
documentation. `--timeout` limits each unique command separately, not the whole
verification run.
