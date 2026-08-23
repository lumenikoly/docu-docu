# Toudocu workflows

Use this reference when invoking the CLI, interpreting diagnostics, building a
portal, or working with `TASK-*`. For any source-language change, also apply
[writing-quality.md](writing-quality.md) and
[semantic-gate.md](semantic-gate.md).

## Discover conventions

Before choosing flags:

1. search repository instructions, CI, scripts, and documentation for existing
   Toudocu commands;
2. reuse the existing documentation directory, repository root, excludes,
   stale policy, output directory, and strict policy;
3. in a monorepo, choose the narrowest repository root that contains both the
   documentation and every referenced code or task-scope path;
4. fall back to `./docs` and its parent only when no convention exists.

Use `toudocu` from `PATH`. In the Toudocu source repository, substitute
`go run ./cmd/toudocu`. The Go CLI has no `init` command. The explicit
`$toudocu init` prompt follows [init.md](init.md); other document creation
uses the bundled templates.

The Go CLI also has no `refresh` command. Explicit `$toudocu refresh` and
`$toudocu refresh diff` prompts follow [refresh.md](refresh.md). The first
reviews all source documentation; the second scopes evidence from the current
worktree relative to `HEAD` and includes affected documentation.

For ordinary CLI, portal, task, and documentation work, use the canonical
documentation root as the only documentation and backlog source. Repository
code, tests, contracts, CI, and other non-translation artifacts remain valid
implementation evidence. Exclude configured translation roots, including
translated work items. A translation root may be read only when the user
explicitly asks to check, find, build, run, or inspect that locale; restrict the
operation to the selected locale and the files required by that operation.
Do not add translation roots to `.gitignore` or global ignore files; explicit
locale operations must remain able to select them.

## Choose the first source

This table selects the first source for the information being requested. It
does not require running every listed command in sequence or make `check`,
`build`, or `task verify --dry-run` mandatory for unrelated operations.

| Required information | First source |
|---|---|
| Context for an existing Ready+ `TASK-*` or `BUG-*` | `toudocu task context ... --format json` |
| Documents or entities by ID, title, or topic | `toudocu search` |
| Documentation changes | `toudocu changes`, `changes file`, or `task changes` |
| Task verification plan | `toudocu task verify ... --dry-run --format json` |
| Structural documentation diagnostics | `toudocu check ... --format json` |
| Details of a document whose path is already known | Read the file directly |
| Implementations, API consumers, symbols, and tests in code | Ordinary repository or code search |

## Documentation gate

For every documentation change, use this sequence:

1. define the audience, useful question or task, and selected document language;
2. identify the source of truth and decide whether to update an existing file;
3. collect authoritative evidence and distinguish current facts, requirements,
   plans, and gaps;
4. draft in natural target-language prose and complete the reader-first review
   from [writing-quality.md](writing-quality.md);
5. complete the author review from [semantic-gate.md](semantic-gate.md);
6. obtain independent semantic review when the change is risk-bearing;
7. run Toudocu as the structural gate.

Use an initial read-only `check` only when establishing a baseline for a large
change, existing diagnostics, a validation or CI failure, or an explicit user
request. Its diagnostics are evidence about the declared structure, not
instructions to invent content. Do not call a document complete until the
reader-first gate, required semantic gates, and final structural check pass.

## Command matrix

| Command | Mutates or executes | Use |
|---|---|---|
| `check` | No | Validate safety, links, IDs, and explicit relationships |
| `build` | Writes output; `--clean` removes safe output | Generate the static portal and report |
| `serve` | Writes output and starts unauthenticated HTTP/editor; explicit save, create, and roadmap-add actions can change canonical sources | Preview with rebuilds |
| `changes` / `changes file` / `task changes` | Read-only unless `-o` writes a report | Inspect Git-backed changes |
| `search` | No | Find source documents |
| `task init` / `scaffold` | Creates one file | Create a neutral ID-based scaffold |
| `task ready` | No | Validate a full Draft or Ready contract |
| `task candidates` | No | List active Draft and Ready candidates with contract and dependency readiness |
| `task context` | No | Read full Ready+ task-local context |
| `task verify --dry-run` | Read-only unless `--report` writes JSON | Inspect the verification plan |
| `task verify --run` | Executes trusted repository commands | Verify an explicitly requested task |
| `task archive` | Moves one file | Archive one valid Done or Cancelled task |
| `task restore` | Moves one file | Restore one task from the yearly archive |
| `skill status` / `version` | No | Inspect the installed skill or CLI version |
| `skill install` / `update` / `uninstall` | Writes the selected skill target | Manage the offline skill package |

Prefer JSON for agent diagnosis and text for human confirmation:

```bash
toudocu check <docs-root> --repository-root <repository-root> --format json
toudocu check <docs-root> --repository-root <repository-root>
```

The ordinary check fails on errors and reports warnings. `--strict` additionally
turns warnings into a failing exit code. Use strict only when repository policy,
CI, or the user requires it.

## Diagnostic loop

1. Confirm that the reader-first and semantic gates for the draft have passed.
2. Run `check --format json`.
3. Group issues by document and stable code.
4. Fix every error at its source without adding unsupported semantics.
5. Evaluate warnings as editorial guidance; fix only those supported by project
   evidence.
6. Re-run an ordinary project-wide check.
7. If strict validation is project policy, run it as an additional final gate.

Do not suppress an integrity error by removing a meaningful relationship. Do
not suppress a warning by inventing status, dates, sections, or
behavior. Report warnings intentionally left unresolved.

## New documentation

Create a neutral `index.md` and required `architecture/overview.md` first. Keep
its question map empty until evidence supports a distinct detailed answer. Add
free-form documents as needed. Opt into typed documents only for semantics the
project will use:

- add a module when stable ownership, boundaries, or business rules matter;
- add a use case when observable behavior should link to a module;
- add a flow when a reusable Mermaid visualization should link through
  `Scenario` to one or more use cases, or directly to an architecture document;
- add `screens/SC-*.md` when product navigation needs stable `SC-*`, a
  searchable catalog, graph validation, actions, states or acceptance context;
- add a roadmap when the project wants global progress aggregation;
- add a work item when an agent or CI needs a checkable task contract.

When adding related types, create targets before references: module and rules,
then use case, then screen documents or flow, then roadmap or work item.
Replace every template placeholder before validation. Write headings, prose,
table cells, and visible diagram labels in the selected document language. A
map or flow never replaces prose requirements or acceptance criteria.

### Free-form drafts

Use `drafts/**/*.md` for provisional text that does not yet belong in a
permanent section. A draft needs only useful Markdown content: do not invent a
stable ID, metadata, required sections, relationships, status, or lifecycle.
Create it with the Editor `draft` template or an ordinary safe file operation;
the CLI `scaffold` command does not create drafts. `drafts/index.md` is
optional. If it exists, its H1 must match the configured `drafts` section
title.

Do not confuse a document in `drafts/` with a `TASK-*` or `BUG-*` whose status
is Draft. Do not create a work item merely to hold provisional text, and do not
move accepted text into a permanent section unless the user explicitly asks.

Do not let the template select the entities or topology. In a flow template,
replace `OPTIONAL_USE_CASES_METADATA` with the complete `Scenario` metadata line
containing one or more `UC-*`, or with an empty value for an architectural
flow. Replace `RELATED_DOCUMENT_LINKS` with one or more links to those use cases
or to the relevant architecture document. Toudocu derives reverse `UC ↔ FLOW`
relationships from the `Scenario` list. Whole-section placeholders such as
`FLOW_DIAGRAM` and `TRANSITION_ROWS` must be replaced with content derived from
product or repository evidence. Keep Mermaid syntax and exact IDs, but express
visible labels as natural actions, questions, conditions, and outcomes rather
than raw variable or event names.

## Task workflow

Create a work item only when the user or repository policy explicitly requires
one, or when substantial work needs durable scope, acceptance criteria,
verification, or handoff. Do not create one for every prompt, ordinary
question, small local edit, or behavior-preserving refactor.

For qualifying new work, start with `search`, `task init`, semantic filling and
read-only `task ready`.

For implementation, continuation, analysis, or debugging of an existing Ready+
`TASK-*` or `BUG-*`, the first operation that gathers documentation context
must be:

```bash
toudocu task context <TASK-ID> <docs-root> \
  --repository-root <repository-root> \
  --format json
```

Before this command, read only what is required to determine the canonical
documentation root, repository root, how to invoke Toudocu, and mandatory
repository-specific CLI flags. Do not search canonical documentation with
`rg`, traverse it with `find`, inventory `docs/` manually, or read related
Markdown files in sequence as a substitute for `task context`.

Use the returned work item, dependencies, related typed entities, `documents`,
`requiredReads`, and diagnostics or issues as the starting map. If the compact
context is insufficient, read those documents in full and inspect the
implementation with ordinary repository or code search. `task context` is
read-only and never implies `task verify --run`.

Never read a translated `TASK-*` or `BUG-*` as task context. The canonical root
is the only task source even when a translation profile contains a complete
read-only mirror of `work/`.

A task with `Flow` includes its `FLOW-*` document. A task with `Screens`
includes selected screen records, incident transitions, and matching `SC-*`
documents.

Task lists are allowed in both `Acceptance criteria` and `Plan`. Acceptance
criteria retain unique `AC-*` identifiers and verification mappings; plan
checkboxes track implementation steps and need neither.

Keep active task files in `work/`. Archive a terminal task only through:

```bash
toudocu task archive TASK-AREA-001 <docs-root> \
  --repository-root <repository-root> --format json
toudocu task restore TASK-AREA-001 <docs-root> \
  --repository-root <repository-root> --format json
```

The archive path is `work/archive/YYYY/`. Archive and restore preserve Markdown
and status, and block when a direct Markdown link would change resolution.
Relationships by `TASK-*` or `BUG-*` ID remain valid. Do not delete archived
work items that still participate in dependencies or traceability.

Run:

```bash
toudocu task verify TASK-AREA-001 <docs-root> --dry-run \
  --repository-root <repository-root> \
  --format json

toudocu task verify TASK-AREA-001 <docs-root> --run \
  --repository-root <repository-root> \
  --format json \
  --report <report-path-outside-docs-root> \
  --timeout 10m
```

`--dry-run` does not execute task commands and may be inspected when task
verification is in scope. It writes a file only when `--report` is supplied;
use that option only when the requested work authorizes the report path.

Run `--run` only when all of these are true:

- the user explicitly asked to execute or verify the task;
- the repository and commands are trusted;
- the task-local validation gate passes;
- the report path is outside source documentation.

The command executes each unique shell command from `AC-*`, `ALL`, `DOCS`, and
the required `QUALITY` target when standards are declared, once from repository
root. A failure does not stop remaining commands.

## Build and serve

Build only when requested or required for verification. Reuse the project's
generated-output convention; otherwise select a dedicated disposable
directory. Add `--clean` only after verifying the resolved output is neither
input, its ancestor, a system root, nor an unsafe symlink target.

When `screens/SC-*.md` exists, the build includes a catalog, Screen Map,
playable flows and traceability by default. Use `--no-screen-map` only to omit
the interactive map page; it does not remove screen documents, flows or JSON
collections.

Use `serve` on `127.0.0.1` by default. It has no TLS or authentication, and its
editor can change canonical source files after explicit save, create, or
roadmap-add browser actions. On the first canonical browser request it may also
query GitHub Releases for latest stable metadata. Add `--no-update-check` when
an outbound request is not part of the requested preview. Use `--host 0.0.0.0`
only for an explicitly requested trusted local-network preview.

## Deliver

Scale the summary to the size and risk of the work. State what changed, which
validation ran, and what remains unverified or needs attention. Add detailed
evidence, review results, diagnostics, or warnings only when they materially
help the user assess the result.
