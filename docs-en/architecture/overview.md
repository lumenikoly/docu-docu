# Architecture

Toudocu is a local, dependency-free Go utility. It reads source Markdown
files, validates their relationships, and builds a static HTML portal. The
same model is available through Go code or as CLI JSON. The Go presentation
layer keeps useful semantic content in HTML; interactive browser surfaces do
not own the project model or Portal routing.

`serve` adds a local HTTP server, source-file editor, and internal API
reference page. Only the main portal started with `serve` may make one request
to GitHub for information about the latest stable release. Toudocu needs no
database, CDN, or external service for ordinary operation.

## System boundary

A developer, CI, or program using the Go package provides Toudocu with a
documentation directory and, when needed, a repository root. The utility reads
Markdown, local assets, and recognized OpenAPI contracts. It then either
returns a clear list of errors or creates HTML and JSON files.

After `build`, the portal is read-only. With `serve`, the browser can submit an
allowed file change; the Go process then rereads the documentation and rebuilds
the portal. Only an explicitly selected task verification can run repository
commands. The release-information request can be disabled; it downloads no
executable code and is absent from static and translated portals.

## Map of architectural questions

- [Where is the Toudocu system boundary and who interacts with it?](system-boundary.md)
- [How do runtime components divide responsibilities?](runtime-components.md)
- [What does the Go part do, and what does code in the browser do?](frontend-runtime-boundary.md)
- [Where are the trust boundaries?](trust-boundaries.md)
- [How are documentation and verification failures isolated?](failure-isolation.md)
- [How do Git states become a consistent documentation change set?](documentation-changes.md)
- [How does a message from local documentation reach an external development agent without a direct AI integration?](agent-feedback-delivery.md)
