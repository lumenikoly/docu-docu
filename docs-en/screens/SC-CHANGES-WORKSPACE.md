<!-- toudocu
id: SC-CHANGES-WORKSPACE
status: done
screenKind: screen
module: MOD-CHANGES
route: /changes/
updated: 2026-08-23
-->

# SC-CHANGES-WORKSPACE: Changes


This screen lets a developer review the current Git diff and continue a
discussion about any changed file without leaving the local portal. New
messages to the agent are allowed for regular files in a comparison against the
working tree. The public
`ChangeSetReport` remains limited to documentation.

No current screenshot is available yet.
`assets/screens/changes-workspace.png` shows an older layout and is therefore
not declared as this document's Preview.

## What the user sees

The top bar links to Portal, Editor, and Changes and contains a
“Discussions · N” button together with theme and color-scheme controls. The
count covers every open project thread. Below it, the page shows the Git range
and the changed file and line counts.

Range settings are inside a disclosure panel. The user can choose start and end
states and an optional base branch, then see the resolved Git refs, current
branch, and dirty state. The panel closes after apply, on `Esc`, or after an
outside click, and restores focus to the range control.

## File list

Files are sorted by path and split into two groups:

- Changed — files in the current Git diff;
- Related — files the user added for discussion context.

The panel provides search, status and scope filters, plus a Kind filter for all
files, documentation together with repository-root `CHANGELOG.md`, or other
repository files. A related file enters local discussion state only after its
first saved comment.

The first matching file opens automatically. After a refresh, Toudocu tries to
preserve the selected file, tab, and filters. When nothing matches, the screen
states that the list is empty.

After a file is selected, the list and URL update immediately. While Toudocu
loads details for that file only, the main area shows its heading and a
“Loading file…” message. For Markdown, the same response includes safely
rendered Before and after versions. Switching back to an already loaded tab
reuses its data instead of repeating the request. A late response cannot
replace a newer file selection. After an error, the user can select the file
again.

## File views

Diff is the main tab and shows the exact Git patch. It supports unified and
side-by-side layouts and patch copying. There is no Summary tab; old links with
`tab=summary` or `tab=source` open Diff.

Full file loads the current UTF-8 file only when opened. For a deleted file, it
shows the last version before deletion. Markdown, JSON, YAML, Go, Java,
JavaScript, and TypeScript receive syntax highlighting; other text files appear
as plain text.

Documentation gets only applicable extra tabs: Before and after, Semantics,
Relationships, OpenAPI, Mermaid, assets, or screen map. Diagnostics remain
collapsed until needed; an error opens them automatically.

## Comments and agent responses

A message can target an entire file, a new patch line, or selected content
in Full file or unified diff. In both views, the selection menu can copy clean
text, copy context, or add a question, and releasing the pointer does not clear
the selection.

The new side of a unified diff stores an exact file range. For the old side,
Toudocu creates a file anchor without a range and
adds a visible quote containing the path and old line number to the message.
Mixed selections remain available only for copying. A fully deleted, binary,
or large file supports a whole-file discussion even when its content cannot be
displayed. A separate “+” appears on hover and focus for lines that support an
exact or file anchor and remains visible on touch screens.

The form lets the user choose a question or change request. `Ctrl`+`Enter` or
`Cmd`+`Enter` saves the message and immediately adds it to the queue; `Esc`
cancels.

The discussion panel opens over the workspace on the right and shows every
project thread, open threads first. A pending message can be edited or deleted
until the agent retrieves it through `agent next`. Any open or closed thread
can be deleted in full after confirmation, permanently removing its messages
and unprocessed queue entries. A message retrieved by the agent is immutable;
the thread can continue with a new message after the agent responds.

Copy prompt immediately copies the request to process the queue and does not
change state. An agent result appears as a response, change, no change needed,
clarification request, or error and never closes a discussion automatically.

## Live updates and responsive layout

Toudocu watches repository and discussion changes separately. When the open
file changes, the screen offers to refresh it. Draft comment text is preserved,
but the anchor must be selected again.

On a wide screen, the file list and diff sit side by side while discussions
slide over the workspace from the right with a backdrop. On phones, the
discussion panel fills the viewport and the file list remains a separate
drawer. `Esc` closes both panels; the discussion panel also closes when the
backdrop is pressed and restores focus to the header button. Motion is disabled
under `prefers-reduced-motion`, and state is always expressed with text as well
as color.

## Availability

The screen and discussion writes are available only in the main portal started
through `serve` and only with `target=working-tree`. Index and revision
comparisons hide discussions, and the server rejects writes. Static builds, translation portals, and
direct serving of a translation root do not include this capability.

## Related documents

- [UC-DOCS-05](../use-cases/UC-DOCS-05.md)
- [UC-AGENT-FEEDBACK-01](../use-cases/UC-AGENT-FEEDBACK-01.md)
- [MOD-AGENT-FEEDBACK](../modules/MOD-AGENT-FEEDBACK.md)
- [FLOW-AGENT-FEEDBACK](../flows/FLOW-AGENT-FEEDBACK.md)
- [Changes HTTP API](../contracts/changes-http.md)
- [TASK-CHANGES-002](../work/TASK-CHANGES-002.md)
