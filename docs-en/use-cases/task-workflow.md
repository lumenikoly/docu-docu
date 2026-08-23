<!-- toudocu
id: UC-TASK-01
status: done
priority: high
module: MOD-CLI
updated: 2026-08-23
-->

# UC-TASK-01: Collect work-item context

Actor: implementer — a developer or software agent.

Before changing code or documentation, the implementer retrieves a bounded context
for one ready work item.

## Inputs

- stable `TASK-*` identifier or a need to select from the current task
  candidates;
- documentation root;
- repository root;
- `text` or `json` output.

<!-- toudocu:section prerequisites -->
## Preconditions

- Toudocu is available;
- the implementer can read the documentation and repository.

<!-- toudocu:section main-scenario -->
## Main flow

1. If no task has been selected, the implementer lists the current task
   candidates once:

   ```bash
   toudocu task candidates ./docs --format json
   ```

   To select within one decomposition, the implementer adds `--parent TASK-ID`.
   Toudocu reports all active Draft and Ready candidates, including contract
   completeness, dependency waits, and whether each candidate is ready for
   work. The implementer chooses one using priority and request context.
2. The implementer runs:

   ```bash
   toudocu task context TASK-ID ./docs --format json
   ```

3. Toudocu finds exactly one work item with that identifier.
4. Its status must be Ready, In progress, Blocked, or Done.
5. The report contains the complete task contract, compact references to its
   parent, ancestors, and direct children, required sections from explicitly
   related documents, declared documentation impact, business rules, and
   diagnostics.
6. `TaskContextReport` schema v1 lists `requiredReads`: the files the implementer
   must actually read before working.
7. The implementer plans changes within the task's goal, scope, constraints, and
   exclusions.

The report does not embed complete descendant documents. For a tree overview,
the implementer runs `toudocu task tree TASK-ID ./docs`; a specific child receives
its own `task context` call.

## Error flows

- A missing identifier or several work items with the same identifier returns
  code `1`.
- Draft and Cancelled are not valid implementation context.
- Problems in related documents remain in `issues` so the implementer sees them
  before work starts.
- A project read failure ends the command before context is created.

<!-- toudocu:section postconditions -->
## Postconditions

The implementer has the selected task context. Files are unchanged and no command
from Verification has run.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] The implementer receives context for exactly one selected work item.
- [x] Before selecting one, the implementer can retrieve ready tasks and subtasks,
  plus the reasons why other candidates are not ready, in one call.
- [x] Collecting context leaves files unchanged and runs no command from
  Verification.

<!-- toudocu:section business-rules -->
## Business rules

- [BR-CLI-001](../modules/cli.md#br-cli-001-task-context-does-not-execute-commands)

<!-- toudocu:section implementation -->
## Implementation

- [FLOW-TASK-WORKFLOW](../flows/FLOW-TASK-WORKFLOW.md)
- [CLI and work-item operations](../modules/cli.md)
- [Project model](../modules/model.md)
- [CLI contract](../contracts/cli.md)
- [Work-item guide](../guides/work-items.md)

## Scenario verification

Coverage includes JSON composition, missing and duplicate identifiers, related
rules and documents, and the guarantee that no commands run.
