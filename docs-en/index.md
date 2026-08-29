# Toudocu

Toudocu is a Go CLI that checks project documentation written in Markdown and
turns it into an HTML portal. The finished portal can be published on ordinary
static HTTP(S) hosting and does not require a running Toudocu server. The
repository also exposes a public Go facade for local embedding.

## The problem

Project documentation quickly drifts away from the code when statuses,
relationships, acceptance criteria, and verification commands live in
unrelated files or must be synchronized by hand.

## The approach

Toudocu builds one project model from Markdown and checks stable identifiers,
links, business rules, the roadmap, and work items. People read the result in
the HTML portal; automation receives the same information in `ProjectReport`
schema v1.

## What Toudocu provides

- **One project portal.** Search, navigation, the architecture map, use cases,
  and current work status live in one coherent interface.
- **A verifiable document model.** Toudocu checks links, stable identifiers,
  document structure, and explicitly declared relationships.
- **End-to-end traceability.** A use case and screen can be traced to a work
  item, an acceptance criterion, and the command that verifies the result.
- **Meaningful change review.** The local Changes workspace shows the exact Git
  patch, rendered pages before and after a change, and differences between
  documented entities.
- **Local documentation discussions.** In the main portal running under
  `serve`, a user can attach a question or change request to an entire canonical
  Markdown document or a selected range. Toudocu stores the thread and queue
  outside the repository but does not start an AI agent; messages are delivered
  only after an explicit user request.
- **Focused context for an AI agent.** A compact read-only package contains the
  requirements, rules, screens, dependencies, and checks that belong to one
  selected task.
- **An embedded documentation skill.** The offline package connects Toudocu to
  Codex, Claude Code, and Copilot and adds explicit workflows for project
  initialization, change clarification, full or scoped documentation refresh,
  and translation. The CLI installs, checks, updates, and removes its managed
  copy offline without overwriting user changes. Start with the
  [skill installation guide](guides/skill-installation.md), then read the
  [agent workflow guide](guides/agent-workflows.md).
- **Verifiable work items.** A task combines its allowed scope, acceptance
  criteria, and verification commands. Toudocu checks that contract and runs
  commands only after explicit authorization.
- **Self-contained publication.** One Go binary builds a static portal without
  a CDN, database, npm, or an external runtime. Use `serve` for local work and
  static HTTP hosting for publication.

## Intended readers

- developers and technical leads;
- teams that maintain documentation next to the code;
- AI agents that need bounded, machine-readable context;
- CI systems that enforce a documentation contract;
- Go developers embedding Toudocu from the source module.

## Current limits

- the Markdown dialect is CommonMark plus the explicitly enabled tables, task
  lists, strikethrough, and automatic links;
- commands under `task verify --run` are trusted repository code;
- `build` creates a read-only portal and never edits source documents;
- `serve` writes only explicitly selected files in the canonical documentation
  root through the Editor API; every configured locale root is a peer workspace;
- global progress comes only from `roadmap.md`.
