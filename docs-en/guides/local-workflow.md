# Working with the portal locally

Use `serve` for everyday work. It builds the portal, starts a local HTTP server,
and adds source editing, change review, discussions, and internal API docs.

## Complete workflow

1. From the project root, run:

   ```bash
   toudocu serve ./docs
   ```

2. Open the printed address, normally `http://127.0.0.1:8080`.
3. Find a document through navigation or search. Document pages offer Edit,
   Show changes, and Open source actions.
4. Edit and save the file. The server checks that its version has not changed,
   writes the update safely, and rebuilds the model, HTML, and search index.
5. If an external editor or agent changes files, wait for the automatic
   refresh. Unsaved text in the built-in editor is preserved; on conflict, the
   UI asks what to do.
6. Open Changes to review the current Git diff. You can also leave local
   comments there and prepare them for an agent.
7. On `roadmap.html`, if needed, select Add deliverable, choose an existing
   stage, review the suggested `DLV-ROADMAP-NNN`, and enter one line of text.
8. Press `Ctrl+C` in the terminal to stop the server.

Manual rebuild is useful when you want to reread documentation immediately. It
shows what is being rebuilt and does not reload the page until the operation
finishes.

## Available routes

- `/` — the main portal;
- `/_toudocu/editor/` — editor for allowed source files;
- `/changes/` — Git changes and local discussions;
- `/_toudocu/api-docs/` — Editor, Changes, and agent feedback HTTP API
  reference;
- `/_toudocu/locales/<locale>/` — a peer locale portal.

Each locale portal exposes the same workspace and writes only its own root. If
a page does not exist, the route opens that locale's home page.

## Network and security

The server binds only to `127.0.0.1` by default. `--host 0.0.0.0` exposes it to
the local network. Toudocu has no built-in TLS or authentication, so do not use
that mode on an untrusted network; the CLI prints a warning.

The main portal may check whether a newer stable release exists. This is the
only optional network request made by Toudocu in this mode. Add
`--no-update-check` to disable it.

## Static publishing is a different mode

There is no separate `preview` command. For publication, run `toudocu build`
and place the result on [static HTTP hosting](deployment.md). Static output has
no local API, editor, Git view, discussions, or roadmap writes.

## Related documents

- [UC-DOCS-03: Local server](../use-cases/serve-portal.md)
- [Viewing changes](documentation-changes.md)
- [Local discussions](../use-cases/UC-AGENT-FEEDBACK-01.md)
- [Editor HTTP API](../contracts/editor-http.md)
- [Changes HTTP API](../contracts/changes-http.md)
