<!-- toudocu
id: UC-TASK-04
status: done
priority: medium
module: MOD-CLI
updated: 2026-08-12
-->

# UC-TASK-04: Archive or Restore a Work Item


The assignee moves a completed or cancelled task out of the active list into a
yearly archive, or returns an archived work item to the active directory without
changing its contract.

## Inputs

- a stable `TASK-*` or `BUG-*` ID;
- the canonical documentation directory;
- repository root;
- result format: `text` or `json`.

<!-- toudocu:section prerequisites -->
## Preconditions

- the task being archived has status Done or Cancelled;
- the destination path does not exist;
- direct Markdown links will still resolve after the move.

<!-- toudocu:section main-scenario -->
## Main scenario

1. The assignee runs `toudocu task archive TASK-ID ./docs`.
2. Toudocu finds exactly one work item and checks its contract, status,
   source path, and destination path.
3. Toudocu verifies that the move will not break relative Markdown links.
4. The file is moved without overwriting to `work/archive/YYYY/`.
5. The command returns a `TaskMoveReport` with status `archived`.
6. When needed, the assignee runs `task restore` for the same ID.
7. The file returns to `work/` without content changes, and the report receives
   status `restored`.

## Error scenarios

- a missing or ambiguous ID blocks the operation;
- an active task cannot be archived;
- a destination-path conflict is not overwritten;
- an unsafe archive path, symlink, or escape from the repository root is rejected;
- a direct Markdown link that could break leaves the file in place;
- a configured translation root is rejected without moving the work item.

<!-- toudocu:section postconditions -->
## Postconditions

On success, only the location of one work item has changed. Its Markdown,
status, stable ID, dependencies, and history are preserved; when blocked, the
filesystem is unchanged.

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

- [x] A successful operation changes only one work item's location and
  preserves its Markdown, status, stable ID, dependencies, and history.
- [x] A blocked operation leaves the filesystem unchanged.

<!-- toudocu:section business-rules -->
## Business rules

- [BR-CLI-006](../modules/cli.md#br-cli-006-the-selected-locale-is-a-work-context) — task workflow operates only on the selected locale.
- [BR-CLI-007](../modules/cli.md#br-cli-007-archiving-does-not-change-the-task-contract) — moving a task preserves its contract.

<!-- toudocu:section implementation -->
## Implementation

- [CLI and task workflow](../modules/cli.md)
- [CLI contract](../contracts/cli.md)
- [Work-item guide](../guides/work-items.md#archive)

## Verification

- positive archive and restore tests;
- rejection of an active status, a destination conflict, and an unsafe path;
- preservation of content and the archive year in `TaskMoveReport`;
- rejection when direct Markdown links would resolve differently.
