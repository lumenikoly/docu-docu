# API and programmatic interface map

Use this table to choose an interface Toudocu actually provides. OpenAPI 3.1.0
defines exact Editor, Changes, and agent feedback HTTP fields; adjacent Markdown explains
behavior and security boundaries.

| Interface | Availability | Purpose | Read and write boundary | Details |
|---|---|---|---|---|
| CLI | Installed binary; `go run ./cmd/toudocu` in the source repository | Check, build, `serve`, search, changes, and work items | Most commands are read-only. `build` writes output, task creation and archiving move explicitly selected files, and `task verify --run` starts authorized commands | [CLI contract](../contracts/cli.md) |
| Go API | Root package of this source module | Embed the model, generator, work-item operations, and changes without another process | Side effects depend on the called function and are not hidden behind one universal entry point | [Public Go API](features.md#public-go-api) |
| JSON reports | `--format json`, `--report`, and `report.json` | CI, agents, and integrations | A report does not change Markdown; the CLI writes only an explicit report or output path | [CLI schemas](../contracts/cli.md#json-results) |
| Editor HTTP API | Main `toudocu serve` portal only | List, read, preview, validate, create, and safely save files; add one `DLV-*` | Allowed `.md`, `.yaml`, `.yml`, and `.json` inside the canonical documentation root | [OpenAPI](../contracts/editor.openapi.yaml), [behavior](../contracts/editor-http.md) |
| Version HTTP API | Main `serve` without `--no-update-check` | Compare the running version with the latest stable release | The browser calls the local server; Go makes one bounded read-only GitHub request | [Behavior](../contracts/editor-http.md#version-check) |
| Changes HTTP API | Any locale portal in `serve` | Return `ChangeSetReport` and specialized documentation views | Reads local Git and the active locale root; changes nothing | [OpenAPI](../contracts/changes.openapi.yaml), [behavior](../contracts/changes-http.md) |
| Agent feedback HTTP API | Main `serve` and the working Git diff only | Store discussion state, queue messages automatically, edit them before retrieval, and show responses | Changes local state outside the repository only; `document` and `file` targets are limited to safe paths | [OpenAPI](../contracts/agent-feedback.openapi.yaml), [architecture](../architecture/agent-feedback-delivery.md) |
| Agent feedback commands | Binary or source repository | Retrieve the oldest queue entry and return one structured response | Read and write local state only; start neither an agent, a language model, nor Git | [CLI contract](../contracts/cli.md) |
| HTTP API reference | `/_toudocu/api-docs/` in the main `serve` portal only | Browse all three OpenAPI contracts and try safe requests | Same-origin; Try it out permits only `GET` and `HEAD`; no CDN | [Screen](../screens/SC-SITE-API-DOCS.md) |
| Manual rebuild | Main `serve` only | Reread documents and rebuild the model, HTML, and search | Reads canonical sources and writes derived output only; does not change Markdown | [Editor OpenAPI](../contracts/editor.openapi.yaml) |

The `toudocu` import path is intended for the source tree or an explicit local
`replace`. The CLI remains the public distribution interface.

Translation portals have no Editor, API reference, or discussion writes. No HTTP
interface writes Git, invokes a system shell, or runs work-item verification.
