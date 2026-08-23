<!-- toudocu
architectureQuestion: Где проходят границы доверия?
-->

# Toudocu Trust Boundaries

Markdown, links, assets, and Mermaid source are treated as untrusted data;
repository root and the selected output/report paths define the filesystem
boundary; work item commands are treated as trusted code only after separate,
explicit authorization to run them. Before the CLI starts, the release
installer separately trusts GitHub Release as the single remote source of the
binary and checksum.

## Scope

This answer lists the architectural trust zones. Concrete CLI errors and
Markdown rules are defined in the [CLI contract](../contracts/cli.md) and
[MOD-MARKDOWN](../modules/markdown.md).

## Untrusted content

Text and metadata are escaped during HTML rendering. Active local assets,
dangerous protocols, and links that escape the repository root are blocked.
Mermaid runs in the browser with a pinned bundle and strict configuration
without becoming a source of requirements.

## Filesystem boundary

The input directory, repository root, output, and report are normalized before
operations. Symbolic links cannot replace the cleanup or write boundary, and
generated output never becomes a documentation source.

The skill target is additionally confined to the canonical project root or
user home and cannot equal or escape that boundary. Symlink/reparse components,
symlinks inside a managed package, and non-regular files block the lifecycle.
Manifest schema v1 and its SHA-256 table distinguish an unchanged managed copy
from unmanaged or locally modified content; a conflict is never replaced
automatically.

In `serve`, the editor workspace additionally accepts only a canonical relative
POSIX path to a regular `.md`, `.yaml`, `.yml`, or `.json` file inside docs
root. Hidden/excluded/output paths, traversal, encoded remnants, and every
detected symlink/reparse component are blocked. SHA-256 CAS and atomic replace
protect against accidentally losing a concurrent change. An intentional,
privileged local race that replaces a directory is outside the threat model of
a trusted working copy.

Review endpoints accept only regular paths relative to the repository.
Absolute paths, `..`, percent-encoded remnants, `.git`, special or binary
files, and symbolic links are rejected. A UTF-8 snapshot is limited to 2 MiB.
State uses private `0700`/`0600` permissions; corrupt storage is not overwritten
automatically, and concurrent writes use a lock plus a version check.

## Serve HTTP boundary

An editor write requires a JSON content type, an exact action header, and a
same-origin browser context, does not enable CORS, and limits the body/content.
These guards protect against a cross-origin browser page but do not authenticate
a direct HTTP client. An explicitly selected non-loopback listener therefore
places reachable LAN clients inside the trust boundary; the CLI retains a
warning about the absence of TLS and authorization. Locale routes are confined
to `/_toudocu/locales/<locale>/` and serve only generated read-only snapshots.
They do not redirect to editor, changes, workspace, API docs, or the canonical
API; the server computes target URLs from allowed profiles and mounts.

Canonical API docs load only embedded Swagger UI and same-origin validated
specs. CSP prohibits external script/style/connect targets, and browser Try it
out is limited to `GET`/`HEAD`; the UI does not weaken the guards of the APIs
themselves.

The version check exists only in canonical `serve` and contacts a fixed HTTPS
GitHub Releases API without credentials. The server limits response time and
size, rejects redirects, accepts only a stable `X.Y.Z` tag, constructs the
official release URL itself, and caches the status. The browser sees only a
same-origin endpoint; the remote body never becomes HTML, executable code, or
a general-purpose URL. `--no-update-check` disables the capability and makes
the endpoint unavailable, while static and translation portals do not create
this boundary.

Comments can be changed only when `target=working-tree`. A comparison ending at
a commit or the index is read-only.

## Execution boundary

Documentation Changes invokes the installed `git` directly as an argument
array with `--no-ext-diff`, `--no-textconv`, `--no-color`, and NUL-separated
path output. It does not run hooks, a shell, fetch, checkout, or index changes.
The revision is validated, the blob is read from the object database, and the
HTTP path must match a change-set entry within documentation roots. Old
Markdown passes through the same sanitization policy and receives no editor or
network privileges.

Discussions use the same boundary for allowed files across the repository. The
feedback CLI reads and writes only local discussion state; it does not start an
agent, contact an AI model, or run repository commands.

Ordinary `check`, `build`, `serve`, editor API, `search`, `task ready`,
`task candidates`, and `task context` do not run commands from Markdown.
Execution appears only in `task verify --run` after the task-local validation
gate; authorization rules are described in [MOD-CLI](../modules/cli.md).

## Release bootstrap boundary

POSIX and PowerShell installers run before the Go CLI with the current user's
permissions. They download the exact selected binary and `checksums.txt` from
one HTTPS GitHub Release, require exactly one matching SHA-256 entry, and check
the version before replacement. The binary and checksum share one trust root:
this check detects corruption but does not replace an independent signature.

Installation does not receive `sudo`: writes are limited by default to a user
install dir and one idempotent `PATH` entry. An explicit
`TOUDOCU_INSTALL_DIR` may select any writable directory and does not change
the profile. Download, verification, and staging finish before replacement; a
failure does not damage an already installed binary. Direct `curl | sh` and
`irm | iex` deliberately add the remote installer to the user's trust boundary.

## Embedded skill boundary

The skill package is compiled into the binary and needs no network access after
installation. The bundle validator permits only regular relative paths and
limits both individual file size and total package size; `SKILL.md` must declare
the expected name. The lifecycle does not invoke a shell or execute embedded
scripts.

Writes are prepared in a sibling stage, and the manifest is created last. For
update and uninstall, the target is first moved atomically to a unique backup
and checked against the snapshot again. An error or concurrent replacement
causes rollback; if restoration is impossible, the backup is preserved and its
exact path is reported to the user.
