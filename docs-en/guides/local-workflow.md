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

## Working with a development agent

Agent Console is available when `serve` uses a loopback address such as
`http://127.0.0.1:8080`. It works with installed Codex and OpenCode agents.

To work on a ready item:

1. Open Work and select an item that is ready to start.
2. Select the Agent Console control next to Start work.
3. If more than one agent is installed, choose one. Before the session starts,
   you can also choose an offered model, reasoning effort, and access mode.
4. Follow the response in Agent View. Command Output shows commands and their
   results but does not accept terminal input.
5. Add a follow-up instruction when needed. Stop response interrupts only the
   current response; Stop agent ends the whole session.

Full access expands the permissions requested from the selected agent. Toudocu
asks you to confirm it separately for the current project. Restrictions in the
agent's own settings still apply.

You can open the panel from the header on any page. Moving between Work,
Documentation, Changes, and Editor does not stop the session. On a wide screen,
you can resize both the panel and the left navigation; the browser remembers
those sizes. On a narrow screen, the panel uses the available width.

Work-item pages also provide short actions for asking a question, clarifying
requirements, continuing work, or updating documentation for current changes.
Toudocu prepares the relevant context automatically. Process with active agent
handles requests left in documentation discussions one at a time. If the agent
is already working on another item, a new session cannot start, but you can
still copy the prepared handoff.

### Working without Agent Console

Every work-item action can copy a prepared handoff for any external agent. If
automatic copying is unavailable, the portal displays the text for manual
copying.

The project terminal is a separate action that opens an ordinary shell in the
repository root. You can start `codex`, `opencode`, `claude`, Git, or another
program there. The terminal and Agent Console can run at the same time, and
stopping one does not stop the other.

Manual rebuild is useful when you want to reread documentation immediately. It
shows what is being rebuilt and does not reload the page until the operation
finishes.

## Available routes

- `/` — the main portal;
- `/_toudocu/editor/` — editor for allowed source files;
- `/changes/` — Git changes and local discussions;
- `/_toudocu/api-docs/` — Editor, Changes, and agent feedback HTTP API
  reference;
- `/_toudocu/api/tasks/{taskID}/actions` — actions available for a work item in
  the main loopback `serve` instance;
- `/_toudocu/locales/<locale>/` — a peer locale portal.

Each locale portal provides documentation, Editor, Changes, discussions,
manual rebuild, and the work-item workspace. It writes only to its own root.
Agent Console and work-item actions are available only in the main `serve`
instance. If a page does not exist, the route opens that locale's home page.

## Network and security

The server binds only to `127.0.0.1` by default. `--host 0.0.0.0` exposes it to
the local network. Toudocu has no built-in TLS or authentication, so do not use
that mode on an untrusted network; the CLI prints a warning.

Agent Console is available only on loopback. It remains disabled for every
other binding, including `--host 0.0.0.0`, regardless of user settings.

The main portal may check whether a newer stable release exists. This is the
only optional network request made by Toudocu in this mode. Add
`--no-update-check` to disable it.

## Static publishing is a different mode

There is no separate `preview` command. For publication, run `toudocu build`
and place the result on [static HTTP hosting](deployment.md). Static output has
no local API, editor, Git view, discussions, roadmap writes, or Agent Console.

## Related documents

- [UC-DOCS-03: Local server](../use-cases/serve-portal.md)
- [Viewing changes](documentation-changes.md)
- [Local discussions](../use-cases/UC-AGENT-FEEDBACK-01.md)
- [Agent Console](../reference/features.md#agent-console)
- [Editor HTTP API](../contracts/editor-http.md)
- [Changes HTTP API](../contracts/changes-http.md)
