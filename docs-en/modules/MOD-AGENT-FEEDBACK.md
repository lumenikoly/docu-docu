<!-- toudocu
id: MOD-AGENT-FEEDBACK
status: done
updated: 2026-08-25
-->

# MOD-AGENT-FEEDBACK: Local requests to a development agent


This module attaches human messages to specific documentation locations and
delivers them to an external development agent through a local queue. Toudocu
does not invoke a language model and does not depend on Codex, Claude Code,
OpenCode, or another provider.

## Purpose

A reader creates a question or change request on a document page or in Changes.
Saving immediately creates a pending queue entry, while the message remains
editable until the agent retrieves it through the command line. The agent
checks the current document and, when needed, the code, then returns a
structured response to the same discussion.

<!-- toudocu:section code-location -->
## Code location

- `internal/app/review_types.go`, `review_store.go`, and `review_service.go` —
  the model, local store, first-in-first-out queue, and deterministic anchors;
- `internal/app/review_http.go` — the local version 1 API;
- `internal/app/review_cli.go` — `agent next|respond`;
- `web/src/features/discussions/` — deferred React components, composer, and
  Portal-state synchronization for discussions;
- `web/src/features/changes/index.ts` — the view of the same server state in
  Changes;
- `.agents/skills/toudocu/references/agent-feedback.md` — the built-in skill's
  queue-processing rules.

<!-- toudocu:section boundaries -->
## Boundaries

The Portal creates a `document` target only for canonical Markdown. Changes
also creates a `file` target for a regular file in the current working diff. A
range is available only for UTF-8 text up to 2 MiB; a binary, large, or deleted
file receives a whole-file target. A `change_request` may change `target.path`
and additional paths the human explicitly and unambiguously named: for a
`document`, only canonical documentation paths; for a `file`, safe repository
paths. The feedback API does not accept patches or write files.

State lives in the operating system's user-data directory and is keyed by the
canonical repository root. It is not stored in Git and remains available to
the CLI while `serve` is stopped. Static portals and translation roots remain
read-only.

Portal loads the discussions interface only after the panel first opens or the
user acts on selected text. React components keep display state only; after a
revision conflict, they reread the server and do not create a second source of
truth for discussions or deliveries.

<!-- toudocu:section business-rules -->
## Business rules

### BR-AGENT-FEEDBACK-001: Content is independent of transport

`Discussion`, messages, `DocumentAnchor`, and responses store conversation
content. `AgentDelivery` stores delivery information only. A new transport can
therefore be added without changing discussion history.

### BR-AGENT-FEEDBACK-002: A question does not authorize a change

For `intent=question`, the agent responds without changing files. Only
`intent=change_request` permits the agent to validate the request and change
`target.path` plus additional paths the human explicitly and unambiguously
named within the target-kind boundary. The user's claim is not considered
proven in either case.

### BR-AGENT-FEEDBACK-003: A pending message is editable

Saving atomically moves a message to `submitted` and creates one pending
`AgentDelivery`. While the delivery is `pending`, the message can be edited or
deleted. After the agent retrieves it, the message is immutable; a follow-up is
a new message in the same discussion.

### BR-AGENT-FEEDBACK-004: The queue is processed in strict order

`agent next` returns only the oldest unfinished delivery. Until it receives a
response, the next delivery is not returned. After the lease expires, the
agent receives the same oldest delivery again. Repeating an identical response
succeeds, while a different response for a completed delivery is rejected as a
conflict. After a response, the delivery is `responded` and is not returned
again, even when the discussion remains open. A new human message creates a new
delivery.

### BR-AGENT-FEEDBACK-005: A human controls the discussion

An agent response does not close the discussion. The user can close, reopen, or
delete any discussion. Closed threads appear after open threads.

<!-- toudocu:section invariants -->
## Invariants

- A saved message appears in the queue immediately.
- A message can be edited only before the agent atomically retrieves it.
- A question does not grant permission to change files.
- Every retrieved delivery is completed with a response before the next one is
  retrieved.
- Request processing does not run validation commands or expand changes to
  related files the human did not name.
- The feedback API neither writes repository files nor starts an agent.
- Open discussions and unfinished queue entries are never removed
  automatically.

<!-- toudocu:section stable-interfaces -->
## Stable interfaces

- `/_toudocu/api/agent/` — [agent feedback HTTP API](../contracts/agent-feedback.openapi.yaml);
- `toudocu agent next --json`;
- `toudocu agent respond [--input response.json] [--json]`;
- [JSON contract](../reference/agent-feedback-json.md);
- [delivery process](../architecture/agent-feedback-delivery.md);
- [user scenario](../use-cases/UC-AGENT-FEEDBACK-01.md).

<!-- toudocu:section related-use-cases -->
## Related use cases

- [UC-AGENT-FEEDBACK-01: Discuss documentation with a development agent](../use-cases/UC-AGENT-FEEDBACK-01.md)

## Future extension

Interactive human review would add a separate `ReviewGate` capable of passing
an existing `AgentDelivery` to a waiting agent. Blocking waits, reviewer
confirmation, and reviewer decisions are outside the current version.
