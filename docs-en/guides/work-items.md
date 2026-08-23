# Work items

Each work item lives in its own `work/*.md` file. Features, maintenance,
documentation, and research use `TASK-*`; defects use `BUG-*`. Reports treat
both as `WorkItem` entities with the same status model, dependencies, context
and verification commands, and annual archive.

## Minimal draft

One file contains exactly one work item:

```md
<!-- toudocu
id: TASK-AUTH-014
status: draft
taskType: feature
-->

# TASK-AUTH-014: Add password recovery

<!-- toudocu:section result -->
## Outcome

A user can request a password-recovery link.
```

A draft needs canonical `id`, `status`, and `taskType` fields and a non-empty
section marked `result`. Add the other fields as the contract becomes clearer.

## When a work item is useful

Do not create a `TASK-*` for every prompt, question, code-reading session,
formatting change, small local edit, or behavior-preserving refactor.

A separate work item is justified when the user or project policy requires it,
or when substantial work needs a durable contract: clear boundaries,
acceptance criteria, verification commands, and handoff between people or
sessions. Contract and architecture changes, migrations, and multi-step
features commonly meet that threshold. Reuse an existing Ready work item when
one already covers the change.

`$toudocu init`, `$toudocu refresh`, `$toudocu refresh diff`, and
`$toudocu clarify` do not create a work item automatically. They use the same
threshold.

## Statuses

| Status | Use it when | Additional requirement |
|---|---|---|
| `draft` | The task is still being defined | Type and observable outcome |
| `ready` | The contract is agreed | Every work section and every `AC-*` verification |
| `in-progress` | Work is underway | Same contract as ready |
| `blocked` | Work cannot continue | A non-empty `blocker` section |
| `done` | The result is accepted | Every `AC-*` checked; `ALL` and `DOCS` present; dependencies complete |
| `cancelled` | The work is no longer needed | A non-empty `cancellation-reason` section |

Other spellings and translations are not recognized as machine values.

## Types

The type describes the nature of the work, not its state.

| Type | Meaning | Use case field |
|---|---|---|
| `feature` | New or changed user-facing behavior | Required |
| `bug` | Fix for observable incorrect behavior | Required, except for the technical case below |
| `maintenance` | Refactoring, infrastructure, or upkeep | Optional |
| `documentation` | Documentation-only work | Optional |
| `research` | Investigation or hypothesis testing | Optional |

A technical work item without `useCase` needs a section marked
`use-case-omission-reason`. When a `UC-*` is named, it must exist.

`task init --type Bug` creates `work/BUG-AREA-NNN.md`; other types receive a
`TASK-AREA-NNN` identifier. Machine value `bug` requires the `BUG-` prefix, and `BUG-*`
cannot be used for another type.

## Fields

| Field | Rule |
|---|---|
| `status` | Always required; one of the canonical statuses above |
| `taskType` | Always required |
| `module` | Required from `ready`; the `MOD-*` must exist |
| `useCase` | Required for `feature` and user-facing `bug`; the `UC-*` must exist |
| `flow` | Optional existing `FLOW-*`; included in task context |
| `standards` | Optional existing `STD-*` identifiers; requires a `QUALITY` target |
| `runbooks` | Optional existing `RB-*` identifiers; included in context |
| `parentTask` | One optional `TASK-*`; declares decomposition |
| `dependsOn` | Comma-separated `TASK-*` and `BUG-*` identifiers |
| `priority` | Optional canonical priority |
| `updated` | Optional ISO date used by stale-document policy |

Identifiers are unique across all of `work/**`, including the archive. A
dependency must exist and must not form a cycle. A work item cannot be marked
`done` before all dependencies are complete.

## Decomposing a large task

When substantial work no longer fits one compact, independently verifiable
work item, split it into `TASK-*` items by observable outcomes. `parentTask`
answers which larger body of work a task belongs to. `dependsOn` independently
defines completion order; the parent relationship alone does not order sibling
tasks.

Design the complete tree and dependency graph before creating files. Create the
root parent first because `task init --parent` accepts only an existing parent.
Then create descendants so each parent exists before its children and each
dependency receives a lower ID than the task that depends on it. Preserve the
agreed plan order for independent tasks, and create a prerequisite branch first
when another branch depends on it. This keeps natural ID ordering in
`task tree` and the portal close to execution order. Still declare every real
dependency through `dependsOn`; an ID number never creates a relationship.

Children are derived from the parent field; Markdown has no separate
`Children` field. The parent must exist anywhere under `work/**`, including the
archive, must be a `TASK-*`, and must not create a cycle. The combined graph
also rejects deadlocks in which a parent waits for a child while that child
depends on the parent.

Every work item from `ready` onward keeps its own outcome, boundaries, criteria,
plan, verification, and documentation impact. A `done` parent requires every
direct child to be `done`; a cancelled child does not count as complete. A
cancelled parent cannot retain active children.

Use a parent task as the coordination contract for the shared outcome and
integration criteria. Do not copy every child's detailed criteria into it. Do
not split work mechanically into backend, frontend, tests, and documentation
unless each part produces an independently verifiable outcome.

The portal recursively shows a parent task's current subtree, links, and
statuses. This view is derived from `parentTask`; do not copy it into Markdown
or add a `Children` field.

```bash
toudocu task init ./docs --area AUTH --title "New API" --type Feature \
  --parent TASK-AUTH-100
toudocu task tree TASK-AUTH-100 ./docs
toudocu task candidates ./docs --parent TASK-AUTH-100 --format json
```

`task candidates` builds one report for choosing the next work item. Without
`--parent`, it includes every active `Draft` and `Ready` item. With `--parent`,
it includes matching descendants at any depth. For each candidate, the report
shows the same contract completeness checked by `task ready`, unfinished
`dependsOn` relationships, and the resulting `readyForWork`. A task is ready
for work only when its status is `Ready`, its contract is complete, and every
dependency is `Done`.

The report intentionally retains incomplete and waiting candidates with their
reasons, but it does not choose the next task. `task tree` still describes only
decomposition; the implementer evaluates priority and request context.

The generated child draft keeps the relationship in one source field:

```md
<!-- toudocu
id: TASK-AUTH-101
status: draft
taskType: feature
parentTask: TASK-AUTH-100
-->

# TASK-AUTH-101: New API
```

## Required sections for a Ready work item

From `ready` onward, a task needs sections with these kinds:

1. `result` — what a user or system can observe.
2. `behavior-change` with `before` and `after` subsections — for `feature`.
   Bugs use the separate contract below.
3. `scope` — files and directories that may change.
4. `out-of-scope` — work explicitly excluded.
5. `acceptance-criteria` — testable `AC-*` conditions.
6. `plan` — the expected work sequence.
7. `verification` — one command mapping per criterion plus `ALL` and `DOCS`.
8. `documentation-impact` — exact files to update, or why no documentation
   change is needed.

Backtick paths in Scope are relative to `--repository-root`. Existing paths
must stay inside that root. A new file is allowed when its safe parent exists.
A glob must match at least one path.

## Acceptance criteria and commands

Each criterion has a unique local identifier:

```md
<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [ ] `AC-01` An expired token returns `INVALID_TOKEN`.
- [ ] `AC-02` A valid token changes the password once.
```

Verification contains exactly one entry for each criterion, with at least one
command:

```md
<!-- toudocu:section verification -->
## Verification

- `AC-01` → `go test ./internal/auth -run TestInvalidToken`
- `AC-02` → `go test ./internal/auth -run TestResetPassword`
- `ALL` → `go test ./...`
- `DOCS` → `toudocu check ./docs --strict`
```

| Target | What it confirms |
|---|---|
| `AC-*` | One acceptance criterion |
| `ALL` | The project's complete accepted verification cycle |
| `DOCS` | The complete documentation check |
| `QUALITY` | Explicitly related standards, through commands in the work item |

Readiness requires every `AC-*`, `ALL`, and `DOCS`. If `standards` is non-empty,
one `QUALITY` entry is also required. `task context` includes named standards
and runbooks in `documents` and `requiredReads`; Toudocu does not infer their
applicability from scope. A `done` work item has every criterion marked `[x]`.

## Actual documentation impact

```bash
toudocu task changes TASK-AUTH-014 ./docs --base HEAD --target working-tree
```

The command and Changes workspace compare paths from Documentation impact with
the current Git diff. The `TASK-*` file itself appears separately as a work
contract and does not increase the count of permanent documents. Explicit
paths take precedence over relationships inferred from a module, use case,
flow, screen, or contract.

`declared-document-not-changed` means a promised file did not change;
`undeclared-document-change` means an unlisted document did. These are warnings
for human review by default, not proof of a defect. The comparison range is
always shown; Toudocu does not guess a hidden task-start commit.

## Additional bug contract

A bug first establishes the defect, then proves a correction of that same
behavior. Severity, Priority, Reproducibility, Regression, Module, Use case,
and Last updated are always required.

| Field | Accepted values |
|---|---|
| `Severity` | `Critical`, `High`, `Medium`, `Low` |
| `Priority` | `Urgent`, `High`, `Normal`, `Low` |
| `Reproducibility` | `Always`, `Often`, `Sometimes`, `Rarely`, `Not reproducible`, `Unknown` |
| `Regression` | `Yes`, `No` |

Severity describes harm; priority describes ordering. Toudocu does not derive
one from the other. When Regression is Yes, name the version or period when it
appeared.

Even a draft bug needs Symptom, Expected behavior, Actual behavior, and either
Steps to reproduce or Evidence. From Ready onward it also needs Cause, Scope,
Out of scope, Plan, Acceptance criteria, Verification, and Documentation
impact.

If the cause is unknown, write `Not established` and put tentative explanations
in a Hypotheses section. A completed bug must have an established cause. A bug
plan is a numbered list without checkboxes; checkboxes remain reserved for
acceptance criteria.

A Ready bug includes a criterion for a regression test. When automation is not
technically possible, a Regression test section explains why and gives an exact
manual path.

A purely technical defect may use `Use case: Not applicable`, but then it needs
a User-behavior relationship section. Every named `MOD-*`, `UC-*`, `SC-*`,
`TR-*`, and dependency must exist.

Never put passwords, tokens, keys, personal data, or raw production dumps with
sensitive data in evidence.

## Archive

Active work items live directly in `work/`; completed and cancelled items can
move to `work/archive/YYYY/`:

```text
work/TASK-AUTH-014.md
work/BUG-AUTH-021.md
work/archive/2026/TASK-AUTH-009.md
```

Use the CLI instead of moving files by hand:

```bash
toudocu task archive TASK-AUTH-009 ./docs --format json
toudocu task restore TASK-AUTH-009 ./docs --format json
```

Only Done and Cancelled items can be archived. The command checks status,
paths, destination conflicts, incoming links, and the work item's own relative
links. If a link would resolve differently after the move, the file stays in
place. The command changes neither contents nor status.

`task restore` returns the file to `work/` and can also recover an active item
archived by mistake. Identifiers and dependencies span all of `work/**`, so
`task init` never reuses an archived number.

## Complete example

```md
<!-- toudocu
id: TASK-AUTH-014
status: ready
taskType: feature
priority: high
module: MOD-AUTH
useCase: UC-AUTH-03
flow: FLOW-AUTH-RECOVERY
dependsOn: TASK-MAIL-004
updated: 2026-07-27
-->

# TASK-AUTH-014: Add password recovery

<!-- toudocu:section result -->
## Outcome

A user can recover a password securely through a one-time link.

<!-- toudocu:section behavior-change -->
## Behavior change

<!-- toudocu:section before -->
### Before

A user cannot recover a forgotten password without support.

<!-- toudocu:section after -->
### After

A user follows a one-time link and sets a new password.

<!-- toudocu:section scope -->
## Scope

- `internal/auth/`;
- `internal/mail/`;
- `docs/modules/auth.md`.

<!-- toudocu:section out-of-scope -->
## Out of scope

- changing registration rules;
- adding a new mail provider.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [ ] `AC-01` An expired token is rejected.
- [ ] `AC-02` A valid token changes the password once.

<!-- toudocu:section plan -->
## Plan

- [ ] Add token creation and storage.
- [ ] Enforce expiry and one-time use.
- [ ] Update the authentication documentation.

<!-- toudocu:section verification -->
## Verification

- `AC-01` → `go test ./internal/auth -run TestExpiredResetToken`
- `AC-02` → `go test ./internal/auth -run TestResetTokenSingleUse`
- `ALL` → `go test ./...`
- `DOCS` → `toudocu check ./docs --strict`

<!-- toudocu:section documentation-impact -->
## Documentation impact

Update the password-recovery use case and authentication module rules.
```

## From draft to verification

1. `task ready` validates a complete draft without changing its status.
2. After a manual move to Ready, `task context` collects the limited set of
   required documents.
3. `task verify --dry-run` displays the exact command plan without executing it.
4. Only after separate authorization, `task verify --run` executes commands in
   sequence from the repository root.

An identical command runs once even when mapped to several targets. One failed
command does not cancel the others. Commands are trusted repository code and
run with the current user's permissions.

Report formats and exit codes are defined in the
[CLI contract](../contracts/cli.md).
