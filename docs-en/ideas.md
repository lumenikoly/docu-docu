# Ideas

The capabilities below have not been accepted into the roadmap and are not
promised to users. Each idea needs a defined data source and observable result
before implementation.

## A complete agent-workflow experience

An embedded terminal and quick task actions, such as starting or discussing a
task. Start tasks with prompt templates.

## Add ideas from the local portal

The `ideas.md` file can currently be changed through the `serve` editor, but
there is no separate “Add idea” button or “implemented” state. Such a button
only makes sense after the project defines how an idea differs from a roadmap
deliverable and how a completed idea leaves this list.

## Show project dependencies

The portal does not currently build a separate catalog of libraries and tools.
Before implementation, the project must select authoritative sources for Go
modules, browser packages, and embedded assets and decide which version and
license facts readers need. Until then, `go.mod`, `web/package.json`, and
`THIRD_PARTY_NOTICES.md` remain the sources.

## Improve prompts for the agent

Refine the prompts that help the agent make documentation clearer, more
precise, and more consistent.

## Improve the portal design

Make the portal interface more cohesive, easier to understand, and more
comfortable for everyday work.

## Interactive plan review

Add interactive collaboration with an AI agent for reviewing documentation and
plans.
