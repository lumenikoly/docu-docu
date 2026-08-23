# Toudocu document model

Use this reference when deciding whether a document should remain free-form or
participate in Toudocu's typed knowledge model. Apply the
[reader-first writing gate](writing-quality.md) to the content of either form.

## Validation boundary

Validate only promises the documentation makes:

| Signal | Meaning | Result |
|---|---|---|
| Ordinary Markdown | Human-readable documentation | Render safely; report basic editorial warnings |
| Typed path such as `modules/` or `use-cases/` | Opt-in machine-readable entity | Validate its ID and explicit relationships |
| `toudocu` annotation | Canonical metadata, section, or typed-table contract | Parse exact identifiers without reading display labels |
| Stable ID or local link | Declared identity or relationship | Require uniqueness and a valid target |
| `roadmap.md` checklist item | Declared global scope | Require one supported stable ID and derive `UC-*` readiness |
| `task ready` | Read-only readiness request | Require the complete task schema even for Draft |
| `task candidates` | Read-only work-front request | Reuse readiness and require Done dependencies for execution |
| `task verify --run` | Permission to execute repository commands | Apply the task-local validation gate before execution |

Errors protect safety, identity, relationships, and executable contracts.
Warnings describe editorial completeness, recognized statuses, and staleness.
Do not turn a warning into invented content merely to make a strict check pass.
Structural validation may define the serialization of an opted-in typed
contract. It does not decide which entities the product contains, which
relationships are meaningful, or whether the document should exist. Apply
[semantic-gate.md](semantic-gate.md) before the final structural check.

## Paths and document types

Every project requires `index.md` and `architecture/overview.md`. A missing
`index.md` is a warning; a missing or structurally invalid architecture overview
is an error. All other document types are optional.

| Path | Use when the project needs | Machine-readable behavior |
|---|---|---|
| `index.md` | An entry point and navigation | Project title and description |
| `status.md` | A current snapshot | Project status metadata; checklists are rejected |
| `roadmap.md` | Global scope and progress | Validates checklist IDs and derives linked `UC-*` state |
| `risks.md` | Risk tracking | Extracts `RISK-*` sections and mitigation progress |
| `ideas.md` | Feature ideas and future plans | Free-form; no required metadata or sections |
| `notes.md` | Notes, observations, and temporary context | Free-form; no required metadata or sections |
| `drafts/**/*.md` | Provisional text not yet accepted into a permanent section | Type `draft`; no required ID, metadata, sections, relationships, status, or lifecycle |
| `modules/*.md` | Stable component boundaries and rules | Requires a unique `MOD-*` and validates relationships |
| `use-cases/*.md` | Observable actor behavior | Requires a unique `UC-*` and an existing module |
| `flows/*.md` | A reusable visual process | Requires a unique `FLOW-*`, Mermaid, and a use-case or architecture relationship |
| `screens/SC-*.md` | A significant screen and its outgoing transitions | Parses metadata, states and `TR-*`; validates modules, routes and topology |
| `screens/hotspots.json` | Optional percentage-based interactive areas | Validates screen and transition references and geometry |
| `decisions/*.md` | Durable decisions | Requires a unique `ADR-*` |
| `quality/STD-*.md` | Enforceable project standards | Requires a unique `STD-*`; validates status and replacement links |
| `runbooks/RB-*.md` | Operational procedures | Requires a unique `RB-*`; derives review freshness |
| `work/TASK-*.md`, `work/BUG-*.md`, and yearly archive paths | Agent- or CI-readable work and terminal history | Requires exactly one work item, status-dependent fields, and terminal archive status |
| `architecture/overview.md` | Required architecture boundary and question map | The path defines the type; requires direct links to every detailed architecture document |
| Other `architecture/**/*.md` | One evidence-backed architectural question per document | Requires canonical `architectureQuestion` metadata and a direct overview listing |
| `contracts/`, `guides/`, `reference/` | Specialized human documentation | Classifies and renders the document |
| Any other path | Free-form documentation | Renders safely without a typed entity contract |

A repository-root `CHANGELOG.md`, when present, is the only special release
journal. Do not create a portal-specific duplicate inside the documentation
root, and do not create a root changelog only to populate the portal.

When an unknown top-level directory contains Markdown, its `index.md` is its
custom-section manifest by structure. It needs a description; its H1 supplies
the navigation title. No localized `Type: Custom` field is used.

The built-in `drafts/` directory is not a custom section. Its `index.md` is
optional; when present, its H1 matches the configured section title. A draft
document is distinct from a work item whose workflow status is Draft.

Use one H1 and a concise introduction when useful. These improve navigation but
are not reasons to invent project metadata. Reader-facing content may use any
project-selected language and must be idiomatic in that language. Exact code
tokens remain unchanged and receive a plain-language explanation when needed.
Reader-facing content may use any headings and language. Put all machine
semantics in exact, non-localized annotations:

```md
<!-- toudocu
id: UC-AUTH-01
status: planned
module: MOD-AUTH
-->

# Any reader-facing title

<!-- toudocu:section acceptance-criteria -->
## Any reader-facing heading
```

Never derive an ID from H1. Canonical field names, enum values, section kinds,
table kinds, and table columns are not translated.

## Stable IDs and relationships

When opting into a typed entity:

- give modules unique `MOD-*` IDs;
- give use cases unique `UC-*` IDs and reference an existing module;
- declare business rules as module headings such as
  `### BR-AREA-001: Title`;
- give decisions unique `ADR-*` IDs;
- give reusable flows unique `FLOW-*` IDs and link product flows to one or more
  requirements;
- give screens unique `SC-<AREA>-<NAME>` IDs and transitions unique
  `TR-<AREA>-<NUMBER>` IDs;
- give ordinary work items unique `TASK-AREA-NNN` IDs and bugs unique `BUG-AREA-NNN` IDs;
- keep IDs stable when titles or filenames change;
- update all references together when identity genuinely changes;
- use relative Markdown links that remain inside repository root.

For a `UC-*` roadmap item, Toudocu sets `effectiveCompleted` only when the
linked use case has canonical status `done`, at least one checkbox in its
`acceptance-criteria` section, and every checkbox in that section checked. Nested
subsections count; the next heading of the same or higher level ends the section. The public
`ProjectReport` remains schema v1 and `completionSource` remains
`use-case-status`. `CON-*`, `CONTRACT-*`, `DLV-*`, and `DELIVERABLE-*` retain
their roadmap checkbox state. Other checklists do not contribute to global
progress.

`check` reports these readiness errors without changing Markdown:

- `done-use-case-missing-acceptance-criteria`;
- `done-use-case-has-open-acceptance-criteria`;
- `roadmap-item-completion-mismatch`;
- `roadmap-section-status-mismatch`.

## Typed-document guidance

The module template includes purpose, code location, boundaries, business
rules, invariants, stable interfaces, and related use cases. The use-case
template includes its main scenario, postconditions, an unchecked acceptance
criterion placeholder, rules, and implementation.
The flow and screen templates participate in the separate
[screen and flow model](screen-model.md). The ADR template includes context,
decision, and consequences. Work-item schemas are defined in
[work-item-model.md](work-item-model.md).

Toudocu reports missing recommended sections as warnings. Use these sections
when they add information; do not create empty, speculative, or terminology-led
prose. If the project does not need typed semantics, use the generic
`document.md` template outside the reserved typed paths.

Common schema-driven anti-patterns include:

| Document | Reject when |
|---|---|
| Module | It mirrors a directory without a stable product or ownership boundary |
| Use case | It restates functions or routes without observable actor behavior |
| Flow | Its nodes are copied from code structure rather than one reusable scenario |
| Screen model | Its screens or transitions are derived from router entries instead of product navigation |
| ADR | No durable decision or considered tradeoff can be evidenced |
| Roadmap or status | Desired work is presented as current fact or global scope is inferred from local checklists |
| Task | Scope, criteria, or commands are invented to complete the task schema |

## Supported Markdown

Use CommonMark headings, paragraphs, emphasis, links, safe raster images,
blockquotes, lists, task lists, tables, strikethrough, literal autolinks, inline
code, and fenced code. Do not use raw HTML, front matter, attributes, footnotes,
definition lists, active SVG/XML/HTML assets, or JavaScript URLs. Raw HTML is a
policy error except for exact `toudocu` annotations; annotations are omitted
from HTML, plain text, and search. Fenced code is not parsed as headings,
annotations, links, or tasks.
