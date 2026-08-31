<!-- toudocu
status: in-progress
stage: Maintaining version 0.0.7
updated: 2026-08-31
-->

# Current status

The current stable version is `0.0.7`. The GitHub Release installers for POSIX
and PowerShell verify the checksum before replacing the program.

<!-- toudocu:section summary -->
## Summary

The CLI checks documentation, builds a portal, and opens it for local work. It
can return JSON reports and collect focused context from a work item's explicit
relationships, but it runs that item's verification commands only after an
explicit request. The same core operations are available from Go code. Mermaid
diagrams work without a CDN or Node.js. Tests cover unsafe paths, HTML,
diagrams, HTTP serving, and command timeouts.

Use cases (`UC-*`) and flows (`FLOW-*`) have separate catalogs and stable URLs.
A use-case page shows its description, map, steps, and relationships. Screens
have a shared catalog, transition map, hierarchy, interactive hotspots, and
relationship table. Static portals work over HTTP(S), support keyboard and
touch input, and show clear error states; `serve` is used for local viewing.

The portal derives active work, blockers, and the next result from work items
and the roadmap. The local Changes workspace can inspect files across the
repository, filter by kind, open the complete file, and use the same
documentation discussions as the Portal. Saving a message immediately creates
a queue entry that can be edited or deleted until an agent receives it.

A discussion does not start an agent by itself. Separately, the main local
`serve` instance provides Agent Console: a ready work item can start an
installed Codex or OpenCode, show conversation and commands, accept follow-up,
and explicitly stop a response or the whole session. This capability is
disabled in published output and when `serve` is exposed beyond loopback.

## Release readiness

Source code and embedded resources ship as one binary with no external runtime.
The pinned Goldmark and OpenAPI YAML Go dependencies are linked into that
binary. `make check` checks Go formatting, runs ordinary and race tests, checks
the browser assets and Go modules, then strictly validates the canonical
Russian documentation. `make docs` builds the Russian and English portals.
`make release` runs `make check`, builds six operating-system and architecture
targets, and prepares the release bundle with two installers. The installers
select the appropriate binary and verify its SHA-256 before replacing the file
in the user's program directory.

## Next focus

Maintain version `0.0.7`: fix reported defects, keep the documentation current,
and verify the installers on supported platforms.
