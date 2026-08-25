<!-- toudocu
status: in-progress
stage: Сопровождение версии 0.0.6
updated: 2026-08-23
-->

# Current status

The current stable version is `0.0.6`. The GitHub Release installers for POSIX
and PowerShell select the appropriate binary and verify its checksum before
replacing the program.

<!-- toudocu:section summary -->
## Summary

The CLI checks, builds, and locally serves documentation, produces typed JSON
reports, returns context for a selected task, and runs its verification
commands only after an explicit request. The root Go package provides a stable
model, generation, and reporting facade for local embedding from the source
tree; the CLI remains the public distribution interface. The embedded Mermaid
Tiny renders related processes without a CDN or Node.js. Tests for prohibited
scenarios cover path, HTML, diagram, HTTP serving, and execution-timeout safety.

The process model gives `UC-*` and `FLOW-*` separate catalogs, stable ID-based
URLs, and a use-case page with its description, map, step-by-step playback, and
relationships. The screen model provides a catalog, an overall map with SVG
links, hierarchy, hotspots, and a relationship table. Static portals work over
HTTP(S), support keyboards and touch input, and expose clear error states;
`serve` is used for local viewing.

Active tasks, blockers, and the next deliverable are derived from work items
and the roadmap and displayed in HTML automatically. The local Changes
workspace can inspect files across the repository, filter by file kind, open
the complete file, store anchored discussions, and send comments to an
installed AI skill. The interface never starts the agent itself.

## Release readiness

Source code and embedded resources ship as one binary with no external runtime.
The pinned Goldmark and OpenAPI YAML Go dependencies are linked into that
binary. Formatting, ordinary tests, race detection, `go vet`, module
verification, strict checks for both documentation roots, and builds for six
target platforms are part of `make check` and `make release`. The release
bundle includes two installers; they select the appropriate binary and verify
its SHA-256 before replacing the file in the user's program directory.

## Next focus

Maintain version `0.0.6`: fix reported defects, keep the documentation current,
and verify the installers on supported platforms.
