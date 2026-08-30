---
name: toudocu
description: >-
  Create, update, review, validate, translate, or operate Toudocu-managed
  documentation, the CLI and portal, discussions, the Agent Feedback queue,
  explicit $toudocu workflows, and supported typed entities. Also use when code
  or product changes affect documented behavior, public interfaces,
  configuration, architecture, workflows, operational procedures, or
  user-visible behavior. Produce evidence-backed, reader-first text in the
  selected document language. Do not use for general code or text questions
  unless they require Toudocu-managed documentation or affect its accuracy,
  structure, or content. Do not use for code-only changes that explicitly
  preserve public and documented behavior. Clarification is available only
  through explicit `$toudocu clarify` or an explicit request for a Toudocu
  clarification interview; ordinary questions and implementation requests do
  not activate it. Never infer initialization or run `task verify --run`
  without explicit authorization.
---

# Toudocu

Toudocu maintains source Markdown, explicit relationships, and safe repository
paths. Use repository evidence to keep that documentation accurate and useful.

## Route the request

Use the first matching route. Explicit `$toudocu` operations take precedence;
do not add `workflows.md` when their operation reference already defines the
workflow.

| Request | Read | Rule |
|---|---|---|
| `$toudocu init` | [references/init.md](references/init.md) | Only when explicitly invoked |
| `$toudocu refresh` or `$toudocu refresh diff` | [references/refresh.md](references/refresh.md) | Only the requested refresh mode |
| `$toudocu translate <target-locale> --from <source-locale>` | [references/translate.md](references/translate.md) | Only the explicitly selected locale pair and selection mode |
| `$toudocu clarify <subject>` or an explicit request for a Toudocu clarification interview | [references/clarify.md](references/clarify.md) | Investigate facts, exhaust the decision frontier, and stop before implementation |
| `$toudocu feedback`, Toudocu discussions, or the local Agent Feedback queue | [references/agent-feedback.md](references/agent-feedback.md) | Use its isolated transport and lifecycle |
| Ordinary source-documentation mutation, CLI, portal, or task operation | [references/workflows.md](references/workflows.md) | Follow the requested operation |
| Read-only review, analysis, or explanation | Only the applicable references below | Skip `workflows.md` unless CLI or diagnostics are required |

Load additional references only when the request needs them:

- [references/writing-quality.md](references/writing-quality.md) when drafting,
  revising, or reviewing reader-facing prose, headings, tables, diagram labels,
  messages, or translations;
- [references/semantic-gate.md](references/semantic-gate.md) before changing
  source documentation;
- [references/document-model.md](references/document-model.md) when creating,
  selecting, or reviewing typed documents, stable IDs, or relationships;
- [references/architecture-gate.md](references/architecture-gate.md) for
  architecture documents;
- [references/screen-model.md](references/screen-model.md) for `FLOW-*`, `SC-*`,
  `TR-*`, screen states, or hotspots;
- [references/work-item-model.md](references/work-item-model.md) for `TASK-*`,
  `BUG-*`, or work-item lifecycle operations.

## Gather context with Toudocu first

For operations that activate this skill, use a purpose-built read-only Toudocu
command as the first source when one provides the required information. Do not
replace it with broad search across canonical documentation. Treat its result
as the map of documents, relationships, and entities, then read the identified
Markdown files and inspect source code with ordinary repository tools.

Skip a Toudocu discovery command when the exact file is already known, no
relationships or entity search are needed, and the task is to read that file's
details. Ordinary code search remains valid for implementations, API consumers,
symbols, tests, and other source-code details.

If Toudocu returns `DOCS_MIGRATION_REQUIRED`, read `Migration` from the
diagnostic and open `references/migrations/<Migration>.md`. Apply that guide
only to canonical documentation, set the guide's target
`documentationVersion` before validation, run `toudocu check`, and fix the
remaining migration-related errors. Preserve unrelated information and never
invent a value that project evidence cannot establish. After the check passes,
continue with the ordinary current-format workflow.

## Preserve global invariants

1. Give repository evidence priority over assumptions. Never invent behavior,
   status, relationships, procedures, terminology, or other facts to fill gaps
   or silence diagnostics.
2. Treat generated portals, builds, reports, and example output as derived
   artifacts, never as documentation sources to edit.
3. Never infer `$toudocu init` from missing files, first use, or an ordinary
   documentation request.
4. Treat `$toudocu init`, `$toudocu refresh`, `$toudocu refresh diff`,
   translation workflows, and `$toudocu clarify` as agent workflows, not
   Toudocu Go CLI commands.
5. Enter clarification only when the user explicitly requests
   `$toudocu clarify` or asks Toudocu for a clarification interview. It
   authorizes the investigation and documentation actions in `clarify.md`,
   never implementation.
6. Run `task verify --run` only when the user explicitly requests execution of
   repository verification commands and the repository is trusted.
7. Each configured locale root is an independent documentation and backlog
   source when that root is selected. Read only the active root except for an
   explicit translation workflow, which reads one source/target pair.
8. Process Agent Feedback only through `toudocu agent next|respond`. Its
   operation reference owns validation and delivery; do not run ordinary
   checks, tests, or builds for feedback.
9. Create a durable work item only when the user or repository explicitly
   requires one, or substantial work needs durable scope, acceptance,
   verification, or handoff. Do not create one for an ordinary request or small
   local edit.

## Gather minimum evidence

Resolve or read paths, CLI, CI, glossary, standards, runbooks, and diagnostics
only when required by the current operation. Follow repository instructions,
start with the directly relevant sources, and expand evidence only when needed
for a reliable result.

When Toudocu provides structured documentation or task context, use it before
generic filesystem search over canonical documentation. Search the filesystem
after Toudocu narrows the context or for evidence outside Toudocu's model.

An initial read-only check is optional when establishing a baseline for a large
change, existing diagnostics, a validation or CI failure, or an explicit user
request. Read-only review needs no check unless the answer depends on structural
diagnostics.

## Validate

Ordinary source-documentation mutations require the ordinary project check
unless the selected operation reference defines its own validation or delivery
policy. Run strict validation only when repository policy or the user requires
it, and build the portal only when requested or needed for verification.
