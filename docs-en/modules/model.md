<!-- toudocu
id: MOD-MODEL
status: done
updated: 2026-08-21
-->

# Project model and validation

This module reads the documentation root, connects known entities, and reports
problems with structure, identifiers, paths, and dependencies. `check`, the
portal, and JSON reports all consume the result. `BuildDocumentationModel`
always builds one monolingual Model from the selected `locales.<locale>.root`;
documents, KnowledgeModel, roadmap, search index, and work items from another
locale are not mixed in, and no cross-locale ID or task-state diagnostic exists.

<!-- toudocu:section code-location -->
## Code locations

- `internal/app/docs_core.go` and `types.go` — documents and project model;
- `internal/app/knowledge.go` — known entities and work items;
- `internal/app/documentation_links.go` — links;
- `internal/app/utils.go` — safe path handling.

<!-- toudocu:section boundaries -->
## Boundaries

The model does not generate HTML or run work-item commands. It returns data and
diagnostics without changing sources. For preview, the editor can supply one
in-memory file version; an ordinary model build reads a filesystem snapshot.

<!-- toudocu:section business-rules -->
## Business rules

### BR-MODEL-001: Roadmap is the only source of global coverage

Global progress counts only items from `roadmap.md`. A related `UC-*` is
complete only when its status belongs to the `done` group, its Acceptance
criteria section contains at least one checkbox, and every checkbox in that
section is checked. `CON-*`, `CONTRACT-*`, `DLV-*`, and `DELIVERABLE-*` keep
their roadmap checkbox state.

### BR-MODEL-002: Links do not go beyond repository root

A local link may leave the documentation root only for a file inside the
explicit repository root. Any path beyond that boundary is blocked.

### BR-MODEL-003: A ready-to-run task has a full verifiable contract

From Ready onward, a work item contains scope, exclusions, `AC-*` criteria, a
plan, verification commands, and documentation impact.

### BR-MODEL-004: Screen documents are the source of the graph

Metadata, states, and tables in `screens/SC-*.md` create the `SC-*` and `TR-*`
model. Arbitrary Mermaid text does not become requirements or relationships.

### BR-MODEL-005: Overview is a direct map of architectural issues

`architecture/overview.md` is required and has type `Architecture Overview`.
Every other architecture file asks one non-empty question and is linked
directly from the overview; a transitive link is not enough.

### BR-MODEL-006: OpenAPI contracts are validated as a separate source type

Only `contracts/**/*.openapi.{yaml,yml,json}` receives OpenAPI 3.0/3.1
validation. Toudocu checks root fields, operations and responses, unique
`operationId` values, path parameters, and internal `$ref` values. It never
loads external references or invents a schema for arbitrary YAML.

### BR-MODEL-007: Decomposition and completion order are independent

The parent relationship models decomposition. Declared dependencies determine
completion order; neither relationship is inferred from the other.

<!-- toudocu:section invariants -->
## Invariants

- Every stable identifier is unique within its model.
- `STD-*` and `RB-*` are unique, and work items refer only to existing entities
  of the required kind.
- A minimal project needs `index.md` and `architecture/overview.md`.
- A populated `quality/`, `runbooks/`, or custom section without a complete
  `index.md` produces a warning.
- An unsafe or broken architecture link is an error; a person reviews whether
  the question itself is meaningful.
- Transitions point only to existing `SC-*`; screen routes are unique.
- Previews and hotspots cannot escape the repository root.
- A playable flow reaches a terminal screen or receives a graph diagnostic.
- One `work/*.md` file contains one `TASK-*` or `BUG-*` work item.
- A completed `UC-*` has at least one acceptance checkbox, and every checkbox
  in that section is checked.
- Public report lines start at one; an OpenAPI column is included when known.
- Model construction runs no commands and never writes calculated state back
  into Markdown.

<!-- toudocu:section stable-interfaces -->
## Stable interfaces

- `BuildDocumentationModel(Options)` and other exports from `api.go`;
- `ProjectReport` schema v1;
- `RoadmapItem.effectiveCompleted` includes the `UC-*` status and criteria,
  while `completionSource` remains `use-case-status`;
- `documents[].type = architecture` and optional
  `documents[].sectionType`;
- `Issue` codes;
- [identifier and relationship rules](../reference/document-model.md);
- [work items](../guides/work-items.md);
- [quality standards](../quality/).

<!-- toudocu:section related-use-cases -->
## Related use cases

- [Check documentation](../use-cases/check-documentation.md)
- [Work-item context](../use-cases/task-workflow.md)
- [Work-item verification](../use-cases/task-verify.md)

## Related flows

- [FLOW-DOCS-CHECK](../flows/FLOW-DOCS-CHECK.md)
- [FLOW-TASK-WORKFLOW](../flows/FLOW-TASK-WORKFLOW.md)
