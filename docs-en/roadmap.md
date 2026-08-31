# Roadmap

The roadmap shows what Toudocu already includes and which areas continue to
evolve. A linked `UC-*` is complete only when its document has a `done`-group
status, at least one acceptance checkbox, and every checkbox in that section
checked. The roadmap checkbox must match that result. Versioned sections
describe releases.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Implemented


- [x] `UC-DOCS-01` Build a self-contained documentation portal.
- [x] `UC-DOCS-02` Check documentation structure and relationships.
- [x] `UC-DOCS-03` Open documentation on a local server.
- [x] `UC-DOCS-04` Explore the screen map and transitions.
- [x] `UC-DOCS-05` Review documentation changes.
- [x] `UC-TASK-01` Get context for a selected work item.
- [x] `UC-TASK-02` Explicitly run that work item's verification commands.
- [x] `UC-TASK-03` Create a new work item.
- [x] `UC-TASK-04` Archive or restore a work item.
- [x] `CON-CLI-V1` Stabilize the public CLI contract and versioned JSON schemas.
- [x] `DLV-INSTALL-01` Document installation and updates from GitHub Release
  and building from source.
- [x] `DLV-SKILL-WORKFLOWS-01` Document the skill's `init`, `refresh`, and
  `translate` processes.
- [x] `DLV-SELF-DOCS-01` Maintain this repository's own Toudocu documentation.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.6

- [x] `DLV-RELEASE-07` Add `task candidates`, the hierarchical work-item tree,
  Changes optimizations, portal branding from the project title when no logo is
  set, and clearer bundled-skill guidance.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.7

- [x] `DLV-TASK-WORKSPACE-01` Add a work-item workspace to the static portal
  with Board, List, Tree, search, and filters.
- [x] `DLV-AGENT-ACTIONS-01` Add consistent work-item actions and a prepared
  handoff for an external development agent to `serve`.
- [x] `DLV-AGENT-OPENCODE-01` Add OpenCode as a second structured agent
  provider with the same shared events and explicitly reported capabilities.

This version also added [Agent Console](reference/features.md#agent-console),
which can run a ready work item through an installed agent in the main loopback
`serve` instance.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.5

- [x] `DLV-RELEASE-06` Record the documentation contract version and add its
  migration, `task tree`, hierarchical `task context`, aggregated `task
  changes`, and semantic change analysis.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.4

- [x] `DLV-RELEASE-05` Give Portal and Changes one accessible discussion panel
  and document the supported installation, change-review, quality,
  configuration, and document-model workflows.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.3

- [x] `DLV-RELEASE-04` Treat a text selection as a hint: when it cannot be
  mapped exactly to Markdown, store the question at document level and still
  pass the original selection to the agent.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.2


- [x] `DLV-RELEASE-03` Add the built-in drafts section to the portal, search, Editor, and public report; synchronize release documentation in Russian and English.

<!-- toudocu:section roadmap-stage -->
<!-- toudocu
status: done
-->

## Version 0.0.1


- [x] `UC-AGENT-01` Install and maintain the embedded AI skill.
- [x] `DLV-ROADMAP-001` Add an outcome to an existing roadmap stage from
  canonical `serve`.
- [x] `UC-AGENT-FEEDBACK-01` Discuss documentation and deliver separate
  requests to a local development agent in arrival order.
- [x] `DLV-RELEASE-02` Build and document the stable `0.0.1` release bundle.
