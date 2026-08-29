<!-- toudocu:project-guidance:start -->
## Toudocu

- Use `$toudocu` when a request explicitly concerns Toudocu or when a change
  affects documented behavior, CLI/API/schema/configuration, architecture,
  security, use cases, flows, screens, roadmap, or a task contract. Do not use
  it for code-only reading, formatting, or behavior-preserving refactoring with
  no documentation impact.
- Write canonical source documentation in the established document language;
  use the active configured locale for new documents and the selected target locale only for
  explicit translation work. Agent instructions remain English regardless of
  the documentation language.
- Write for a human reader before writing for the schema. State the useful
  question, conclusion, behavior, boundary, or action first. Use concrete
  actors, actions, conditions, and results instead of disconnected labels or
  implementation terminology.
- Use idiomatic target-language wording. Avoid mixed-language hybrids and
  literal translations. Preserve exact commands, paths, IDs, fields, enum
  values, protocol names, and official product names; explain an uncommon code
  token through its plain-language meaning before using the token alone.
- Keep verified current behavior, required behavior, planned behavior, and known
  gaps distinct. Never present a desired state as implemented or hide an
  incomplete path behind a generic summary.
- Treat headings, table cells, error text, and Mermaid labels as reader-facing
  prose. Keep Mermaid syntax and node IDs, but use natural-language actions,
  outcomes, and decision questions; add exact event or command names in
  parentheses only when traceability requires them.
- Treat the canonical documentation root as the only documentation and backlog
  source for ordinary work. Code, tests, contracts, and CI outside translation
  roots remain valid implementation evidence. Exclude configured translation
  roots from repository search, inventory, semantic review, implementation
  analysis, and task context, including translated work items.
- Read a translation root only for an explicit `$toudocu translate <locale>`,
  `$toudocu translate diff`, or a request to check, find, build, run, or inspect
  that locale. Process locales one at a time and do not add translation roots to
  ignore files.
- Use `$toudocu refresh` for a full evidence-based review of canonical source
  documentation and `$toudocu refresh diff` for staged, unstaged, and untracked
  changes relative to `HEAD` plus affected documents. These are mutating skill
  workflows, not Go CLI commands or initialization.
- Create a new `TASK-*` only when the user or repository policy explicitly
  requires one, or when substantial multi-step work needs durable scope,
  acceptance criteria, verification, or handoff. Do not create a task for every
  prompt.
- Update the existing source of truth when a change makes it inaccurate. Do not
  invent documents, statuses, relationships, procedures, or dates to
  satisfy a template or diagnostic, and never edit generated portal output as a
  source.
- Keep `architecture/overview.md` as the required system-boundary and question
  map. Every other Markdown file under `architecture/` must answer one explicit
  architectural question and be linked directly from the overview.
- Keep detailed interactions in `FLOW-*`, APIs and data formats in `CONTRACT`,
  factual catalogs in `REFERENCE`, operational procedures in `RUNBOOK`, decision
  rationale in `ADR`, and local ownership or rule boundaries in `MODULE`.
- Update this managed block only through an explicit `$toudocu init`.
<!-- toudocu:project-guidance:end -->
