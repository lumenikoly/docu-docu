# Explicit project initialization

Use this workflow only when the user explicitly invokes `$toudocu init` or
unambiguously asks to run the skill's init workflow. Do not infer initialization
from a missing `AGENTS.md` block, an ordinary documentation request, a check, or
another implicit skill trigger. The Toudocu Go CLI has no `init` command.

## Preflight before changing files

1. Read the applicable repository instructions, documentation conventions, CI
   commands, and existing Toudocu configuration.
2. Resolve the narrowest repository root that contains the documentation and
   every referenced scope path. Reuse the established documentation directory;
   otherwise use `<repository-root>/docs`.
3. Resolve the project locale from an existing valid `project.defaultLocale`, then
   repository instructions, then existing canonical documentation. Use `en`
   when none establishes a locale. Normalize an `en-*` locale to the `en`
   bundled templates and a `ru-*` locale to `ru`. For another valid locale, use
   the English templates only as structural scaffolding and continue only when
   the repository supplies a complete 13-title `project.sections` map. Author
   all headings and prose in the project locale. Managed agent instructions
   remain English and do not determine the documentation language.
4. Inspect the repository-root `AGENTS.md` for these exact markers:

   ```text
   <!-- toudocu:project-guidance:start -->
   <!-- toudocu:project-guidance:end -->
   ```

   Continue only when both markers are absent or each occurs exactly once in
   the correct order. Stop before writing when a marker is missing, duplicated,
   reversed, or nested. Also stop when an unmanaged instruction establishes a
   conflicting Toudocu trigger, language, or task-creation policy; show the
   conflict and request user direction.
5. Inspect `<docs-root>/architecture/` before validation. If it contains
   Markdown other than a structurally valid `architecture/overview.md`, treat
   it as legacy architecture: run the ordinary read-only check with JSON
   output, show its diagnostics, and stop without migrating or rewriting those
   files.
6. If the documentation directory already contains Markdown, run its ordinary
   read-only Toudocu check with JSON output. Errors block initialization, with
   one narrow exception: when `architecture/` contains no Markdown,
   `missing-architecture-overview` may be the only error and is the expected
   condition repaired by init. Warnings remain visible. Under established
   strict policy, only `missing-index`, `missing-project-locale`, and
   `incomplete-project-sections` may remain non-blocking because this workflow
   repairs them; every other warning blocks initialization.

## Apply the initialization

1. If the selected documentation directory has no `index.md`, create a neutral
   `<docs-root>/index.md` in the project locale. Base its title, description,
   and terminology on repository evidence. Apply the reader-first writing gate.
2. Resolve the complete section map in memory before creating built-in entry
   documents. Preserve an existing non-empty configured title; otherwise use
   the H1 of an entry document that existed before init; otherwise use the
   matching `en` or `ru` [locale pack](../assets/locale-packs.md). A project
   locale without a bundled pack must use the complete project-provided map
   established during preflight.
3. If `architecture/` contains no Markdown, create
   `<docs-root>/architecture/overview.md` using
   `assets/templates/<template-language>/architecture-overview.md` only as a
   structural cue. Set its H1 exactly to the resolved
   `project.sections.architecture` title; never copy the template H1 when it
   differs. For a locale without a bundled template, author headings and prose
   in the project locale rather than copying English template text. Keep the
   required `Architecture Overview` document type, state only the
   evidence-backed system boundary, leave the architecture-question map empty,
   and omit the optional context diagram unless repository evidence supports
   it. Do not create detailed architecture documents, typed entities,
   statuses, or relationships without evidence.
4. Create or complete `<repository-root>/.toudocu/config.yml` without removing
   existing `site`, `changes`, or `locales` settings. Set
   `documentationVersion: 3`, the resolved `project.defaultLocale`, and the already
   resolved 13 `project.sections` titles. Do not rewrite existing H1 headings.
5. Use `assets/project-guidance/en.md` for every project locale. The managed
   block is an agent instruction surface and therefore remains English. It must
   still require source documentation in the selected project language and
   isolate translation roots from ordinary work.
6. Upsert the complete asset into the repository-root `AGENTS.md`:
   - create the file with the block when it does not exist;
   - append the block after one blank line when both markers are absent;
   - replace from the start marker through the end marker when both occur once;
   - preserve every byte outside the managed block.
7. Do not create a `TASK-*` merely because init is running. Create one only when
   the user or existing repository policy explicitly requires it.

Do not install or refresh project guidance outside this explicit init workflow.
If a write fails, report every file already changed instead of claiming that
initialization completed atomically.

## Validate and report

1. Confirm that both managed markers occur exactly once and in the correct
   order.
2. Confirm that `<docs-root>/index.md` and
   `<docs-root>/architecture/overview.md` exist and that overview has document
   type `Architecture Overview`.
3. Confirm that `.toudocu/config.yml` contains `documentationVersion: 3`.
4. Complete the reader-first and semantic gates for the created documents.
5. Run the ordinary project-wide Toudocu check. Run an additional strict check
   only when strict validation is project policy.
6. Report the resolved repository root, documentation directory, project locale,
   template language, English guidance asset, created or updated files, errors,
   and remaining warnings.
