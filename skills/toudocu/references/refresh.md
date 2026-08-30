# Documentation refresh

Use this workflow only when the user explicitly invokes `$toudocu refresh`
or `$toudocu refresh diff`. Both forms review documentation currency and
apply evidence-backed updates. They are skill workflows, not Toudocu Go CLI
commands.

Refresh never performs initialization. Do not create a new documentation tree,
install or update managed `AGENTS.md` guidance, or infer `$toudocu init`
from missing files. If no established documentation exists, stop and recommend
explicit init.

## Establish the review boundary

1. Read repository instructions, standards, runbooks, CI, task contracts, and
   documented Toudocu commands. Resolve the repository root, documentation
   directory, language, strict policy, and tracked portal convention.
   Use only the canonical documentation root. Exclude every configured
   translation root from repository search, inventory, semantic review,
   implementation evidence, and task context.
2. Resolve Toudocu as described in `SKILL.md` and run the existing ordinary
   project check with JSON output. Diagnostics describe structural state; they
   are not evidence that a claim is current or useful.
3. For `$toudocu refresh`, inventory every source Markdown document in the
   canonical documentation root and the current repository evidence relevant to
   it: code, tests, public interfaces, schemas, configuration, CI, requirements,
   ADRs, and confirmed user input. Never inventory a configured translation root.
4. For `$toudocu refresh diff`, require a Git worktree with a valid `HEAD`.
   Determine changed tracked paths from `git diff --name-only HEAD --` and
   untracked paths from `git ls-files --others --exclude-standard`. This includes
   staged and unstaged tracked changes. Do not compare with a merge-base or
   default branch. If Git or `HEAD` is unavailable, stop and recommend the full
   refresh instead of silently expanding scope.
5. In diff mode, begin with changed source documents and repository artifacts,
   then add documentation affected through local links, backlinks, stable IDs,
   task relationships, declared repository paths, and changed public behavior
   or interfaces. Exclude generated portals, build output, caches, and vendored
   artifacts as documentation sources even when they appear in the diff. Also
   exclude configured translation roots, even when their files are changed or
   untracked; translation review requires an explicit locale-specific request.
6. If the resulting diff set is empty, run the structural check, report that no
   semantic refresh candidates were found, and make no source changes.

## Review currency before editing

For every document in scope:

1. State its audience, durable question, and authoritative evidence.
2. Compare behavioral claims, interfaces, boundaries, statuses, relationships,
   examples, commands, and paths with current evidence.
3. Classify it as current, needs update, unverifiable, obsolete, duplicated, or
   misplaced. A Toudocu warning alone is not evidence for adding content.
4. Review whether terminology and explanations still match the product and the
   intended reader. Flag mixed-language fragments, unexplained identifiers,
   literal translations, and implementation-led paragraphs that obscure the
   actual behavior.
5. Distinguish documentation drift from an implementation or product conflict.
   Refresh updates documentation to established current truth; it does not
   change code to make a document true. Report ambiguous conflicts as unresolved
   findings and do not guess which side should win.

## Apply evidence-backed updates

1. Update the existing source of truth and every explicitly related document
   required to keep the model honest. Apply the document model, reader-first
   writing gate, and semantic gate from this skill. Rewrite unclear wording only
   when evidence fixes the meaning unambiguously.
2. Creating, deleting, renaming, or merging a document and changing a stable ID
   are allowed when current evidence makes the change unambiguous. Update every
   affected link, ID reference, task relationship, architecture overview entry,
   and generated route together. If evidence is incomplete, leave the source
   unchanged and report the unresolved finding.
3. Change the document update date, stored under `Last updated` or `Последнее
   обновление`, only when content or declared relationships actually change. Do
   not create date-only churn for a document that was merely reviewed. Never
   advance the runbook verification date, stored under `Last verified` or
   `Последняя проверка`, unless its procedure was actually verified.
4. Do not create statuses, entities, links, or procedures to silence a
   diagnostic. Do not create a `TASK-*` merely because refresh is running; use
   the normal task threshold.
5. Ensure `project.defaultLocale` and the active locale's complete built-in `sections` map
   exist. Use the selected `en`/`ru`
   [locale pack](../assets/locale-packs.md) when available; for another valid
   locale preserve an explicit one-time map. Update H1 only for existing
   built-in entry documents, never custom section manifests.
6. Never edit generated portal output as documentation. Preserve unrelated
   working-tree changes, including changes that existed before refresh.

## Review, validate, and publish

1. Complete the reader-first writing review and author semantic review for every
   changed source document.
2. Obtain independent semantic review wherever `semantic-gate.md` requires it.
   Give the reviewer updated drafts and raw evidence, not a desired verdict or a
   green Toudocu result. Resolve `NEEDS_REWORK` before continuing.
3. Run the ordinary project-wide check. Run strict validation only when project
   policy requires it. Fix errors at their sources and report any supported
   warnings left intentionally unresolved.
4. Rebuild a portal only when it is tracked or repository instructions require
   it. Reconfirm the exact safe output path before `--clean`, and build only
   after the writing, semantic, and structural gates pass.
5. Report the command mode, reviewed boundary, evidence used, current documents,
   changed/created/deleted/renamed documents, stable-ID migrations, unresolved
   findings, writing-gate result, semantic reviewer and verdict, structural
   diagnostics, date changes, and portal build result. State explicitly when
   refresh produces no source diff.
