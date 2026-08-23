<!-- toudocu
id: CON-CHANGES-HTTP-V1
status: done
updated: 2026-08-23
-->

# Changes HTTP API: Behavior and boundaries


[OpenAPI 3.1.0](changes.openapi.yaml) defines exact routes, parameters,
response codes, and JSON schemas. This document describes what an HTTP schema
cannot express: version sources and Git read boundaries. Documentation
discussions have a separate
[agent feedback API](agent-feedback.openapi.yaml).

In the main `serve`, the specification is available through the built-in
Swagger UI.

## Availability

The Changes API exists only under `serve` and does not write anything itself.
When a translation root is served directly, the API compares that root, but
translations attached to the main portal do not receive a separate API.

Go includes the screen only when Changes is available and passes a same-origin
API address to the browser in the page's JSON block. A static build contains
neither the address nor client code.

The agent feedback API is available only in the main `serve` and does not
depend on the selected Git range: its target always points to current Markdown
in the canonical documentation root.

## Git comparison

The API reads local commits, the index, and the working tree. It does not run
`fetch` or `checkout` and does not change Git. `branchBase` computes the merge
base between the selected branch and `HEAD`; when `base` is also present, both
refs must resolve locally.

The ETag depends on the filtered result, selected range, current documentation
version, `HEAD`, Git status, and resolved refs. An index or working-tree change
therefore cannot return the old report.

The summary does not contain the complete patch. Details, full content, and
rendered Markdown are built only after a specific file is selected.

## Safe file reads

The ordinary Changes API accepts only a relative POSIX path inside an allowed
documentation root. It rejects an absolute path, `..`, backslashes, `.git`,
symbolic-link escapes, and every path outside that root.

The server chooses the response type, adds `X-Content-Type-Options: nosniff`,
and applies a strict content policy. Only the safe renderer converts Markdown
to HTML. SVG cannot execute scripts, load external resources, or apply inline
styles.

A Semantics, Mermaid, OpenAPI, screen-map, or asset error remains a diagnostic
for that view and does not hide the exact Git patch.

## Repository projection for Changes

`/_toudocu/api/changes/review/repository/` separately lists tracked and new
non-ignored files across the repository for the Changes interface. It uses the
same read-only Git mode but does not alter public `ChangeSetReport`, the
ordinary `changes` command, or the Go API.

Old and current full content and the patch load only on request. Go validates
the path, regular-file type, UTF-8, absence of NUL, binary format, and 2 MiB
limit. For Markdown with a rendered comparison available, the same response
contains safe HTML for the old and current versions. The interface therefore
does not need two extra requests for its initial Before and after display.

These routes do not write discussions. Fields and HTTP statuses are defined in
[Changes OpenAPI](changes.openapi.yaml), while local conversation writes use
the [agent feedback OpenAPI](agent-feedback.openapi.yaml).

## Related documents

- [Why OpenAPI is separate from behavior](../decisions/ADR-004.md)
- [MOD-CHANGES](../modules/MOD-CHANGES.md)
- [MOD-AGENT-FEEDBACK](../modules/MOD-AGENT-FEEDBACK.md)
- [Delivering requests to an agent](../architecture/agent-feedback-delivery.md)
- [Comparison architecture](../architecture/documentation-changes.md)
- [ChangeSetReport fields](../reference/changes-report.md)
