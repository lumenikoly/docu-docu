<!-- toudocu
id: FLOW-DOCS-SERVE
module: MOD-SITE
useCase: UC-DOCS-03
updated: 2026-08-10
-->

# FLOW-DOCS-SERVE: Work with the portal locally


The diagram shows the main `serve` lifecycle. Detailed errors and alternative
paths are described in [UC-DOCS-03](../use-cases/serve-portal.md).

## Flow

```mermaid
flowchart TD
    Start["Run toudocu serve ./docs"] --> Build["Build the portal"]
    Build --> Built{"Did the build succeed?"}
    Built -->|No| Stop["Return code 1 without starting the server"]
    Built -->|Yes| Listen["Print the local address and accept requests"]
    Listen --> Action{"What does the user do?"}
    Action -->|Read| Read["Serve the latest successful page"]
    Action -->|Edit| Guard["Validate the path, size, and file version"]
    Guard --> Accepted{"Is the write safe?"}
    Accepted -->|No| Conflict["Show the error without changing the source"]
    Accepted -->|Yes| Save["Safely write the file and rebuild the portal"]
    Action -->|Review a diff| Changes["Open /changes/"]
    Action -->|Add a deliverable| Roadmap["Validate the stage and append one DLV-*"]
    Action -->|Open API docs| API["Show the local OpenAPI reference"]
    Action -->|Press Ctrl+C| Finish["Stop the server and release the port"]
    Read --> Action
    Conflict --> Action
    Save --> Action
    Changes --> Action
    Roadmap --> Action
    API --> Action
```

## What happens without an explicit browser action

- When an external editor changes a file, Toudocu waits for the contents to
  settle and rebuilds. If rebuilding fails, the server keeps its last working
  portal snapshot.
- Main HTML pages may navigate without a full reload. Editor, Changes, API,
  translations, and external links always use ordinary browser navigation.
- Search loads when first used; Mermaid loads when a diagram approaches the
  viewport.
- Configured locales rebuild independently and receive the same workspace.
- The main portal checks for a newer version at most once per process unless
  `--no-update-check` is set. Failure does not interrupt local work.

## Boundaries

- The server binds to `127.0.0.1` by default. `0.0.0.0` must be selected
  explicitly; there is no built-in TLS or authentication.
- The editor sees only allowed files inside the documentation root.
- Static `build` output contains no editor, local API, Changes workspace,
  polling, or manual rebuild controls.
- API docs use embedded assets and allow Try it out only for `GET` and `HEAD`.
- A rebuild failure does not stop an already running server.

## Related documents

- [UC-DOCS-03: View documentation on a local server](../use-cases/serve-portal.md)
- [FLOW-DOCS-BUILD: Build a static HTTP portal](FLOW-DOCS-BUILD.md)
- [MOD-SITE: Static portal](../modules/site.md)
- [CLI contract](../contracts/cli.md)
- [Editor HTTP API](../contracts/editor-http.md)
