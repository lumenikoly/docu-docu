<!-- toudocu
id: MOD-CLI
status: done
updated: 2026-08-23
-->

# CLI and work-item operations

This module provides documentation checks, portal builds, local serving,
search, Git comparisons, and work-item operations. Results are reproducible,
exit codes are stable, and machine-readable reports use JSON schema v1.

<!-- toudocu:section code-location -->
## Code locations

- `api.go` and `cmd/toudocu/main.go` — public Go facade and executable entry
  point;
- `internal/app/cli.go` and `internal/app/server.go` — CLI and local server;
- `internal/app/task_context.go`, `task_ready.go`, `task_candidates.go`,
  `task_verify.go`, `task_archive.go`, and `task_tree.go` — work-item operations;
- `internal/app/search.go` and `scaffold.go` — search and document scaffolds;
- `internal/app/command_process_*.go` — process startup and termination;
- `skills/bundle.go`, `internal/skillinstall/`, and
  `internal/app/skill_cli.go` — embedded skill, planning, and filesystem
  operations.

<!-- toudocu:section boundaries -->
## Boundaries

The CLI does not interpret a natural-language request. `task ready`,
`task candidates`, and `task context` are read-only. Only
`task verify --run` starts project commands, after validating the work item and
receiving separate authorization from the user.

`$toudocu init`, `$toudocu refresh`, `$toudocu translate`, and
`$toudocu clarify` are AI-agent workflows, not Go CLI commands. `skill` only
places embedded files and never executes their contents. The skill lifecycle is
not exported through the public Go facade.

<!-- toudocu:section business-rules -->
## Business rules

### BR-CLI-001: Task context does not execute commands

`task context` returns the work item, required documents, relationships, and
diagnostics without invoking the system shell.

### BR-CLI-002: Checks are only run explicitly

`AC-*`, `ALL`, `DOCS`, and optional `QUALITY` commands run only through
`task verify --run`. An identical command runs once, and the report keeps every
target mapped to it.

### BR-CLI-003: Timeout terminates process tree

When `--timeout` expires, Toudocu stops the system shell and every child process
it created. The result is `timed_out`.

### BR-CLI-004: Toudocu does not interpret user request

Toudocu creates a neutral scaffold and validates its format. A person or agent
chooses relationships, writes requirements, changes status, and confirms
acceptance criteria.

### BR-CLI-005: Browser and CLI use the same scaffold registry

`task init`, the `scaffold` variants, and the editor use one registry for field
order, validation, target paths, and templates. A new file is created atomically
with overwrite prohibited through `O_EXCL`.

### BR-CLI-006: A translation root is not a work context

A translation root can be checked, built, searched, compared, and served
read-only. Work-item commands, `scaffold`, and Editor writes return
`TRANSLATION_ROOT_READ_ONLY` before changing data or running commands.
Translated work items are reader-facing mirrors; context and CI use the
canonical root.

### BR-CLI-007: Archiving does not change the task contract

`task archive` and `task restore` move one file without overwriting and do not
change its Markdown or status. The operation is blocked if a direct Markdown
link would resolve differently after the move.

### BR-CLI-008: A managed skill does not overwrite user changes

The lifecycle changes only an absent target or an exact managed copy with a
valid manifest. Extra, changed, or removed files, incorrect permissions,
symbolic links, unmanaged directories, corrupt manifests, and newer installed
versions block the write. There is no `--force`.

### BR-CLI-009: The skill lifecycle works offline

The canonical skill package is embedded in the binary. `skill install`,
`status`, `update`, and `uninstall` do not use the network, a system shell, or a
marketplace and do not run scripts from the package.

### BR-CLI-010: Multi-target operations are planned before writing

With `--agent all`, the CLI resolves and classifies every unique target before
writing. It then processes targets independently; one failure does not stop the
others, but a partial result returns exit code `1`.

### BR-CLI-011: The task tree does not widen the execution boundary

`task tree` only shows `TASK-*` decomposition. `task changes --tree` aggregates
documentation impact for that tree and does not accept `BUG-*`, while
`task verify --run` executes commands only for the selected task and never for
its children.

### BR-CLI-012: The built-in CLI interface uses English

Help, text-report labels, success messages, and warnings produced by the Go CLI
are in English regardless of `project.locale`. Reader-authored values remain
unchanged, `--lang ru` affects only scaffold text, and hidden machine
annotations remain identical in both languages.

### BR-CLI-013: Task candidates do not choose the next task

`task candidates` builds the model once and reports contract completeness and
dependency state for every active `Draft` and `Ready` item. It neither ranks
the candidates nor declares one of them next; the implementer makes that
decision.

<!-- toudocu:section invariants -->
## Invariants

- JSON output is not mixed with streaming human-readable text.
- Work-item commands run sequentially and continue after one fails.
- Every command starts from the repository root.
- At most the last 1 MiB of stdout and 1 MiB of stderr is retained per command.
- A build requires explicit `toudocu build`; a bare path is rejected.
- `init`, `refresh`, `translate`, and `clarify` are rejected as top-level CLI
  commands.
- Work-item and document creation never uses a translation root.
- Built-in CLI text is English; source Markdown language is selected
  separately.
- `serve` binds to loopback by default; another address must be explicit.
- The browser opens only with `--open`; there is no `--no-open` or `--edit`.
- `skill status` is read-only. Mutating skill commands recheck the target before
  publication and restore the backup if publication fails.

<!-- toudocu:section stable-interfaces -->
## Stable interfaces

- commands and options in the [CLI contract](../contracts/cli.md);
- schema-v1 `ProjectReport`, `TaskContextReport`, `TaskTreeReport`,
  `SearchReport`, `TaskInitReport`, `ScaffoldReport`, `TaskReadyReport`,
  `TaskCandidatesReport`, `TaskMoveReport`, and `TaskVerifyReport`;
- exit code `0` only for success or an allowed no-op.

<!-- toudocu:section related-use-cases -->
## Related use cases

- [Work-item context](../use-cases/task-workflow.md)
- [Work-item verification](../use-cases/task-verify.md)
- [Prepare a work item](../use-cases/UC-TASK-03.md)
- [Archive and restore](../use-cases/UC-TASK-04.md)
- [Check documentation](../use-cases/check-documentation.md)
- [Local server](../use-cases/serve-portal.md)
- [Install the AI skill](../use-cases/UC-AGENT-01.md)

## Related flows

- [FLOW-DOCS-CHECK](../flows/FLOW-DOCS-CHECK.md)
- [FLOW-DOCS-SERVE](../flows/FLOW-DOCS-SERVE.md)
- [FLOW-TASK-WORKFLOW](../flows/FLOW-TASK-WORKFLOW.md)
