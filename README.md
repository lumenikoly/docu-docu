# Toudocu

**English** | [Русский](README.ru.md)

[![CI](https://github.com/lumenikoly/toudocu/actions/workflows/test.yml/badge.svg)](https://github.com/lumenikoly/toudocu/actions/workflows/test.yml)
[![Docs contract](https://github.com/lumenikoly/toudocu/actions/workflows/docs.yml/badge.svg)](https://github.com/lumenikoly/toudocu/actions/workflows/docs.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lumenikoly/toudocu)](https://go.dev/)
[![golangci-lint](https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen)](https://golangci-lint.run/)
[![License](https://img.shields.io/github/license/lumenikoly/toudocu)](LICENSE)

**Documentation that lives next to your code — and stays useful to both people and AI agents.**

Toudocu is a local Go CLI for project documentation written in Markdown. It helps you create documentation for an existing codebase, validate its structure and relationships, keep it in sync with code changes, discuss updates, and publish a ready-to-use documentation portal.

Instead of introducing a separate CMS or a heavyweight documentation stack, Toudocu keeps ordinary Markdown in your repository as the source of truth. It adds a verifiable structure, a local workspace, and workflows for AI agents around those files.

**One binary. Markdown in Git. Verifiable documentation next to your code.**

[View the Toudocu documentation →](https://lumenikoly.github.io/toudocu/project-docs/)

[Install Toudocu →](#1-install-toudocu)

## What Toudocu gives you

### Create and maintain documentation with an AI agent

Toudocu ships with a skill for supported coding agents. An agent can inspect an existing repository, create a minimal documentation structure, and keep it up to date as the project changes.

```text
$toudocu init
$toudocu refresh
$toudocu refresh diff
```

The skill does not force a predefined architecture onto your project. The agent is expected to document what actually exists in the codebase and repository.

Toudocu is flexible and agent-agnostic: its CLI and Markdown workflow can be used with any AI agent or agent harness.

Bundled skill installation is provided for **Codex**, **Claude Code**, and **GitHub Copilot**.

### Verifiable documentation, not just a folder of Markdown files

```bash
toudocu check ./docs
```

Toudocu validates documentation as a connected project model:

- Markdown documents and links;
- stable IDs and relationships between entities;
- architecture documents;
- use cases and processes;
- Mermaid diagrams;
- standards and runbooks;
- roadmap entries and work items.

You can run the checks locally or use them as a CI gate.

`check` deliberately does not pretend that every aspect of writing quality can be validated automatically. It verifies formal contracts and relationships; semantic accuracy still belongs to authors and reviewers.

### A local workspace for documentation

```bash
toudocu serve ./docs
```

Toudocu starts a local portal where documentation is more than something you read.

It includes:

- navigation and search;
- a Markdown source editor;
- live preview;
- diagnostics after edits;
- the current Git changes;
- a local Swagger UI;
- discussions attached to selected documentation or any file in the working diff.

By default, the portal is available at `http://127.0.0.1:8080`.

### Ask questions and request changes directly from the docs

While reading a document, or reviewing any changed file in Changes, you can create a question or change request. Available UTF-8 text supports exact ranges; binary, large, and deleted files use a whole-file discussion.

For example:

> Does this still use the old authentication flow?

or:

> Update this section for the new API.

Toudocu saves the request to a local queue. The installed skill can receive it through the CLI, process it with project context, and send the answer back to the original discussion.

This makes documentation part of the normal agent workflow instead of a separate set of files that must first be located and explained manually.

`serve` does not start an agent on its own: handing the request to an agent remains under the user's control.

### Understand documentation changes in Git

```bash
toudocu changes ./docs
```

Toudocu reads the local Git state and shows what changed in the documentation.

Changes helps you:

- review the current documentation diff;
- inspect changes for a specific file;
- associate documentation changes with a work item;
- discuss any changed file before committing it.

The analysis does not modify the working tree, index, or Git history.

### Publish a static documentation portal

When the documentation is ready to share:

```bash
toudocu build ./docs --output ./site --clean
```

Toudocu builds a regular static HTTP portal.

You can publish the `site/` directory with GitHub Pages or another static hosting service. Neither Toudocu nor a separate backend is required on the production server.

The static build is read-only and does not include the local Editor, Changes view, or `serve` APIs.

### Make documentation part of the work itself

Toudocu can keep work items next to long-lived documentation and give an agent only the context required for a selected task.

```bash
toudocu task context TASK-ID ./docs
toudocu task changes TASK-ID ./docs
toudocu task verify TASK-ID ./docs --dry-run
```

Verification commands declared by a task are never run automatically. Executing them requires an explicit `--run`:

```bash
toudocu task verify TASK-ID ./docs --run
```

This lets documentation act not only as reference material, but also as verifiable context for doing the work.

### Maintain multiple languages without a second source of truth

The skill can maintain documentation translations:

```text
$toudocu translate en --all-stale
$toudocu translate diff
```

The canonical documentation directory remains the source of truth, while translations are maintained as read-only mirrors.

This lets you update the primary documentation with the project and track stale translations separately.

---

## Quick start

### 1. Install Toudocu

**Linux and macOS**

```sh
curl -fsSL https://github.com/lumenikoly/toudocu/releases/latest/download/install.sh | sh
```

**Windows PowerShell**

```powershell
irm https://github.com/lumenikoly/toudocu/releases/latest/download/install.ps1 | iex
```

Verify the installation:

```bash
toudocu version
```

The installer selects the appropriate binary for your operating system and architecture, downloads `checksums.txt`, and verifies SHA-256 before replacing the binary.

See the [installation guide](docs-en/guides/installation.md) for details.

### 2. Connect Toudocu to your AI agent

Open the root of your project and install the bundled skill.

For Codex:

```bash
toudocu skill install --agent codex
```

For Claude Code:

```bash
toudocu skill install --agent claude-code
```

For GitHub Copilot:

```bash
toudocu skill install --agent copilot
```

By default, the skill is installed for the current project. Add `--scope user` if you want to make it available across your projects.

### 3. Ask the agent to create the documentation

From the repository root:

```text
$toudocu init
```

The agent inspects the project and creates a minimal documentation structure based on what it actually finds in the repository.

### 4. Validate the result

```bash
toudocu check ./docs
```

### 5. Open the documentation

```bash
toudocu serve ./docs
```

You can now read and edit the docs, inspect changes, and leave requests for the agent directly from the local portal.

**That is all you need for the basic setup.**

---

## Updating

Re-run the installer from step 1 to update Toudocu, then update the skill from
your project root:

```bash
toudocu skill update --agent codex
```

Use the same `--agent` and `--scope` values you chose during installation. If
`toudocu check` reports `DOCS_MIGRATION_REQUIRED`, ask the agent to migrate the
canonical documentation:

```text
$toudocu migrate
```

---

## Using Toudocu without an AI agent

Toudocu also works as a regular CLI for Markdown documentation.

A minimal documentation tree looks like this:

```text
docs/
├── index.md
└── architecture/
    └── overview.md
```

`index.md` introduces the project to readers.

`architecture/overview.md` describes the system boundary and connects the other architecture documents.

From there, you can simply run:

```bash
toudocu check ./docs
toudocu serve ./docs
```

The skill is an optional workflow, not a required Toudocu dependency.

---

## Day-to-day work with an agent

### Review all documentation

```text
$toudocu refresh
```

The agent checks the documentation against the project again and updates what is genuinely stale.

### Review only the current changes

```text
$toudocu refresh diff
```

The agent analyzes changes relative to `HEAD`, including the index, working tree, and new untracked files, then checks the documentation affected by those changes.

This is useful before a commit or pull request.

### Update translations

```text
$toudocu translate en --all-stale
```

Or update translations affected by the current diff:

```text
$toudocu translate diff
```

`$toudocu init`, `$toudocu refresh`, and `$toudocu translate` are skill workflows executed by an AI agent.

They are not top-level Go CLI commands.

---

## Discussions with an agent

In `serve` mode, you can select a documentation fragment and create a question or change request.

When you save it, Toudocu creates an entry in the local queue.

You can edit or delete the request until an agent receives it.

The agent gets the next request with:

```bash
toudocu agent next --json
```

and sends a result back with:

```bash
toudocu agent respond --input response.json
```

In normal use, the installed skill handles these lower-level commands, so you do not have to manually copy discussion context between the portal and the agent.

---

## Core commands

| Goal | Command |
|---|---|
| Validate documentation | `toudocu check ./docs` |
| Open the local portal | `toudocu serve ./docs` |
| Build a static portal | `toudocu build ./docs --output ./site --clean` |
| Search text or IDs | `toudocu search "query" ./docs` |
| Review Git changes | `toudocu changes ./docs` |
| Review one changed file | `toudocu changes file PATH ./docs` |
| Create a neutral scaffold | `toudocu scaffold module MOD-PAYMENTS ./docs --title "Payments"` |
| Install or update the skill | `toudocu skill install\|update` |
| Get work-item context | `toudocu task context TASK-ID ./docs` |
| Match a task to the Git diff | `toudocu task changes TASK-ID ./docs` |
| Verify a task without running commands | `toudocu task verify TASK-ID ./docs --dry-run` |
| Run explicitly allowed task checks | `toudocu task verify TASK-ID ./docs --run` |

See [docs-en/contracts/cli.md](docs-en/contracts/cli.md) for the complete CLI contract.

---

## Publishing documentation

Build the static portal:

```bash
toudocu build ./docs --output ./site --clean
```

The generated `site/` directory can be served from any regular HTTP(S) static host.

For example:

- GitHub Pages;
- Cloudflare Pages;
- Netlify;
- your own web server;
- another static hosting provider.

Toudocu is not required on the production server.

Opening the generated `index.html` directly through `file://` is not supported. For local viewing, use:

```bash
toudocu serve ./docs
```

---

## Supported Markdown

Toudocu uses Goldmark 1.8.5 and one CommonMark/GFM parser across all commands.

Supported syntax includes:

- headings and paragraphs;
- links;
- emphasis and strikethrough;
- block quotes;
- regular lists and task lists;
- tables;
- autolinks;
- inline code and fenced code blocks;
- safe raster images.

Supported Mermaid diagram types are:

- `flowchart`;
- `stateDiagram-v2`;
- `sequenceDiagram`.

Toudocu deliberately excludes some extensions that make Markdown processing less predictable or harder to keep safe, including raw HTML, front matter, Markdown attributes, footnotes, and active SVG/XML/HTML resources.

See the [Markdown module](docs-en/modules/markdown.md) for details.

---

## Configuration

Toudocu works without a configuration file.

When needed, `.toudocu/config.yml` can configure the project language, portal appearance, translations, and Changes behavior.

For example:

```yaml
project:
  locale: en

site:
  title: My Project
  theme: classic
  colorScheme: system
  accent: indigo
```

See the [configuration reference](docs-en/reference/configuration.md) for all supported options.

---

## Local server security

`toudocu serve` is designed for local use.

By default, it listens only on:

```text
127.0.0.1:8080
```

It does not provide built-in TLS termination or user authentication, so it should not be exposed to an external network without a separate protection layer.

You can disable the optional stable-release update check with:

```bash
toudocu serve --no-update-check ./docs
```

---

## Public Go API

The root Go package exposes typed model, generator, and reporting operations.

The project module path is `toudocu`, so the API is primarily intended for programs built inside this source tree or projects using an explicit local `replace`.

For normal distribution and use, the CLI remains the supported interface.

---

## Developing Toudocu

Users of the released Toudocu binary do not need to build the project from source.

To work on Toudocu itself, you need Go 1.22 or newer.

```bash
git clone https://github.com/lumenikoly/toudocu.git
cd toudocu
go build -o toudocu ./cmd/toudocu
```

Common development commands:

```bash
make fmt
make fmt-check
make lint
make test
make web
make web-check
make browser-test
make check
make build
make docs
make docs-serve
make release
```

Node.js is only needed for development of Toudocu's browser-side code and is not required to use a released binary.

---

## Documentation

- [Features](docs-en/reference/features.md)
- [Installation](docs-en/guides/installation.md)
- [CLI commands](docs-en/contracts/cli.md)
- [Configuration](docs-en/reference/configuration.md)
- [Local workflow](docs-en/guides/local-workflow.md)
- [Documentation changes](docs-en/guides/documentation-changes.md)
- [AI agent workflows](docs-en/guides/agent-workflows.md)
- [Skill installation](docs-en/guides/skill-installation.md)
- [Work items](docs-en/guides/work-items.md)
- [Document types](docs-en/reference/document-types.md)
- [Toudocu source documentation](docs-en/index.md)
- [Contributing](CONTRIBUTING.md)

## License

Distribution terms are available in [LICENSE](LICENSE).

Licenses for bundled third-party components are listed in [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
