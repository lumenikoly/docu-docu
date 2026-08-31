# Toudocu

Toudocu turns Markdown in your repository into documentation you can trust. It
checks links and relationships, presents the project in a usable portal, helps
you review changes, and prepares focused context for work items and development
agents.

It is a local Go CLI. Ordinary files in Git remain the source of truth, and you
do not need a database, npm, or a separate documentation management system to
run it.

## Problems Toudocu solves

- **Documentation quietly drifts away from the project.** Links break, roadmap
  entries disagree with the status of linked use cases, and important
  relationships must be checked by hand.
- **New contributors cannot see the whole project.** Architecture, use cases,
  decisions, and current work live in separate files without a coherent view.
- **A development agent receives either too little context or the entire
  repository.** It has to rediscover requirements, constraints, and
  verification commands before doing useful work.
- **Publishing Markdown grows into a separate toolchain.** Local preview,
  search, editing, and the published site often require different tools.

Toudocu does not replace Markdown with its own database. It adds a verifiable
structure to the files you already have and uses the same data in the CLI,
local portal, static publication, and JSON reports.

## What you can do

- **Catch problems before publication.** `toudocu check ./docs` reports broken
  links, duplicate identifiers, invalid relationships, and violations of the
  declared structure without changing files.
- **Read and edit documentation in one place.** `toudocu serve ./docs` opens a
  local portal with search, source editing, preview, Git changes, and
  discussions.
- **Understand what changed.** The Changes workspace shows the patch, the full
  file, rendered Markdown before and after the edit, and differences between
  entities known to Toudocu.
- **Connect work to its purpose and proof.** A work item keeps its outcome,
  scope, dependencies, acceptance criteria, and verification commands
  together. The portal presents work as a board, list, and tree.
- **Work with an agent without copying context by hand.** From the main local
  portal, a ready work item can start an installed Codex or OpenCode. Agent
  Console separates conversation from command output, accepts follow-up or stop
  actions, and keeps the session alive while you navigate the portal.
- **Maintain documentation with an agent.** On explicit request, the bundled
  skill can create project documentation, compare it with the repository,
  update only affected files, clarify a decision, or translate a selected
  locale.
- **Publish without a backend.** `toudocu build ./docs` creates a self-contained
  portal for ordinary HTTP(S) static hosting. Toudocu does not need to run on
  the server.
- **Use the same data in automation.** The CLI returns versioned JSON reports,
  and core operations are also available directly from Go code.

The [feature catalog](reference/features.md) lists implemented behavior and
links to detailed instructions.

## Where to start

Choose the path that matches your task:

| Your task | First step | Result |
|---|---|---|
| Add Toudocu to an existing project with an agent | Install the [bundled skill](guides/skill-installation.md) and send the agent `$toudocu init` | Minimal documentation based on repository evidence |
| Check an existing Markdown tree | `toudocu check ./docs` | Errors and warnings without source changes |
| Work with documentation locally | `toudocu serve ./docs` | Portal, editor, changes, work items, and discussions |
| Publish documentation | `toudocu build ./docs` | A static site and `report.json` |
| Update docs after a code change | Send the agent `$toudocu refresh diff` | Review of affected documentation and its dependencies |

Entries that start with `$toudocu` are messages for an agent with the bundled
skill, not terminal commands. See the [agent workflow guide](guides/agent-workflows.md)
for the complete process.

## Who Toudocu is for

Toudocu is useful for teams that keep documentation next to code, want to
validate it in CI, publish without a separate server platform, or give agents
bounded and reproducible work context. You can also use the CLI without an AI
agent.

## Important boundaries

- `check` validates declared structure and relationships; it does not replace a
  semantic review of the writing;
- `task verify --run` executes trusted repository commands only after an
  explicit request;
- `build` output is read-only;
- the editor is available only in `serve`, while Agent Console additionally
  requires the main `serve` instance on a loopback address; network-facing and
  published portals do not start agents;
- global project progress comes only from `roadmap.md`.
