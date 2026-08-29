# Feature catalog

This page lists implemented behavior and links to its detailed contract.
Markdown remains the source: JSON and portal output are derived. The canonical
root changes only after an explicit `serve` action or a separate mutating CLI
command.

For a quick interface choice, see the [API map](api.md).

## CLI

Toudocu is delivered as one Go binary. Running it requires no Node.js,
database, or package installation.

| Capability | Command | Result |
|---|---|---|
| Check the project | `toudocu check ./docs` | Text or `ProjectReport` |
| Build the portal | `toudocu build ./docs` | Static HTML and `report.json` |
| Work locally | `toudocu serve ./docs` | Portal, editor, changes, and local APIs |
| View the Git diff | `toudocu changes ./docs` | Text, Markdown, or `ChangeSetReport` |
| Inspect one file | `toudocu changes file PATH ./docs` | Details for the selected change |
| Find a document | `toudocu search "query" ./docs` | `SearchReport` |
| Create a work item | `toudocu task init ./docs --area AREA --title TITLE --type TYPE` | New draft without overwrite |
| Create a document | `toudocu scaffold TYPE ID ./docs --title TITLE` | Typed scaffold without overwrite |
| Check readiness | `toudocu task ready TASK-ID ./docs` | Read-only `TaskReadyReport` |
| List task candidates | `toudocu task candidates ./docs` | `TaskCandidatesReport` with ready and waiting work items |
| Collect context | `toudocu task context TASK-ID ./docs` | Read-only `TaskContextReport`; no commands run |
| Show decomposition | `toudocu task tree TASK-ID ./docs` | Read-only `TaskTreeReport` |
| Plan or run verification | `toudocu task verify TASK-ID ./docs --dry-run|--run` | Plan or `TaskVerifyReport` |
| Compare a task with the diff | `toudocu task changes TASK-ID ./docs` | Report and declared-document warnings |
| Archive or restore | `toudocu task archive|restore TASK-ID ./docs` | Move one file |
| Manage the AI skill | `toudocu skill install|status|update|uninstall` | State of the embedded offline package |
| Process a local documentation request | `toudocu agent next|respond` | One queue entry or a structured response |
| Show the version | `toudocu version` | Generator version |

A bare path is rejected. There are no top-level `init`, `refresh`, `translate`,
or `clarify` commands; those are prompts to the installed AI skill.

All options and exit codes are in the [CLI contract](../contracts/cli.md).

For a first project, the installed binary and the two required source files,
`docs/index.md` and `docs/architecture/overview.md`, are enough:

```bash
toudocu check ./docs
toudocu build ./docs
toudocu serve ./docs
```

## Changes and discussions

`changes` compares the working tree, index, local commits, or a branch merge
base. The browser provides the exact patch, full file, rendered Markdown before
and after, semantics, relationships, and applicable OpenAPI, Mermaid, asset,
and screen-map views.

The main `serve` also shows local discussions on document pages and in Changes.
The Portal targets canonical Markdown, while Changes can target any regular
file in the working diff or an available text range within that file.
Saving immediately creates a queue entry; the message can be edited or deleted
until the agent retrieves it. Copy prompt only copies the request for the
agent. A response does not close the discussion automatically.

Details:

- [changes guide](../guides/documentation-changes.md);
- [Changes workspace](../screens/SC-CHANGES-WORKSPACE.md);
- [agent request flow](../use-cases/UC-AGENT-FEEDBACK-01.md).

## Public Go API

The root package exports typed operations for the model, portal, search, work
items, and changes. A direct call returns Go values and requires neither a
separate process nor JSON.

The `toudocu` import path is intended for the source tree or an explicit local
`replace`. Exported declarations and comments in `api.go` define the actual Go
API surface; the CLI remains the public distribution interface.

## AI skill

The installed skill adds workflows that do not exist in the Go CLI:

- `$toudocu init` safely creates the minimal structure and managed guidance,
  only on explicit request;
- `$toudocu refresh` compares every canonical document with code, contracts,
  CI, and decisions;
- `$toudocu refresh diff` starts from current changes against `HEAD` and adds
  affected documents;
- `$toudocu translate <target-locale> --from <source-locale>` translates one
  explicit locale pair with one selection mode;
- `$toudocu clarify <subject>` investigates facts, interviews the user across
  the complete decision frontier, preserves confirmed durable decisions in
  canonical documents, and stops before implementation;
- `$toudocu feedback` runs `agent next` first and performs only retrieved
  deliveries through `agent respond`;
  `pending=false` ends the workflow without reading or changing files.

Send these entries to the development agent in a message; do not run them in a
terminal:

```text
$toudocu refresh diff
$toudocu translate en --from ru --base main
$toudocu clarify the configuration format migration
$toudocu prepare context for TASK-AREA-001
```

Each locale root is an independent work source. An ordinary operation reads
only the active locale; translation reads one explicit pair. The model is in the
[AI skill guide](../guides/agent-workflows.md).

## Document model

A minimal project contains `index.md` and `architecture/overview.md`. Toudocu
can also recognize status, roadmap, risks, ideas, modules, use cases, flows,
screens, decisions, contracts, standards, runbooks, guides, references, and
work items.

Stable identifiers and explicit fields create validated relationships.
Mermaid remains an illustration; the model does not extract relationships from
diagram text. Global progress comes only from `roadmap.md`.

- [Choose a document type](document-types.md)
- [Review exact relationships](document-model.md)
- [Define a work item](../guides/work-items.md)

## Markdown

Goldmark `v1.8.5` parses CommonMark with its Table, TaskList, Strikethrough, and
Linkify extensions. Toudocu supports safe links, images, tables, lists, code,
and Mermaid `flowchart`, `stateDiagram-v2`, and `sequenceDiagram`.

Raw HTML, completed front matter, dangerous URLs, and active HTML, XML, SVG,
and JavaScript assets are blocked. Mermaid runs from an embedded package with
`securityLevel: strict` and requires no CDN.

## Static portal

`build` creates the home page, document and section pages, search, screen map,
relationships, documentation health, and `report.json`. It runs on ordinary
HTTP hosting at the root or a nested path. No Go server is needed after the
build.

When present, repository-root `CHANGELOG.md` appears as a separate project
changelog but does not enter task context, the editor, or the project model.
`docs/changelog.md` has no special meaning.

Navigation, search, themes, Mermaid, and the map use local assets. See
[MOD-SITE](../modules/site.md) and the
[deployment guide](../guides/deployment.md).

Task Workspace at `work/index.html` is also part of the static portal: Board,
List, Tree, search, filters, and completed-item disclosures work without an API
and remain read-only.

## Local `serve`

`serve` adds source editing for `.md`, `.yaml`, `.yml`, and `.json`, automatic
and manual rebuild, Git changes, discussions, and embedded OpenAPI docs. Saving
uses a SHA-256 version check so an external edit is not lost.

The main portal may check the latest stable release once; `--no-update-check`
disables that request. Configured translations use separate read-only routes.

See the [complete local workflow](../guides/local-workflow.md).

## Screens and use cases

`SC-*` documents and `TR-*` transitions produce the screen map and step-by-step
playback for a `UC-*`. Every transition belongs to exactly one use case; global
transitions without `UC-*` are unsupported. Playback simulates documentation
and never calls the real product.

See [screens and transitions](../guides/screens.md).

## Product boundaries

Toudocu is not a network CMS, collaborative editor, or runtime for the product
being documented. `serve` stores only local discussion state, does not import
an interface from Figma or source code, and makes no real API calls during
playback.
