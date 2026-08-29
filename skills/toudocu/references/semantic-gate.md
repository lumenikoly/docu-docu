# Semantic gate

Use this gate before creating or changing any source document. Toudocu validates
the structure and explicit promises of a document; it does not prove that the
document is useful, necessary, current, readable, or true.

Apply [writing-quality.md](writing-quality.md) whenever the change affects
reader-facing language. The writing gate checks expression; this semantic gate
checks purpose, evidence, document boundaries, and truth.

## Prepare the document

1. State the intended audience, the durable question the document answers, and
   the understanding, decision, or action the reader should gain.
2. Select the target language from the existing document, active locale, or
   the explicit translation target. Identify any project glossary that is
   authoritative for this audience.
3. Check whether an existing document should be updated instead. Do not create
   a new file when it would duplicate another source of truth.
4. Choose the document type from its purpose. Do not choose a typed path merely
   to obtain IDs, portal features, or a green validation result.
5. Identify authoritative evidence in requirements, code, tests, UI, contracts,
   decisions, configuration, or confirmed user input. Mark unknowns instead of
   completing them with plausible content.
6. Draft from the evidence and apply the reader-first writing gate. Treat
   templates as removable scaffolding: keep only sections that communicate
   supported information, except fields required by a deliberately selected
   typed contract.
7. If a required typed field cannot be supported, stop and reconsider the
   document type or report the missing product intent. Do not invent an entity,
   relationship, status, date, transition, procedure, or verification
   command.

## Author review

Return `PASS` only when every answer is supported:

- What useful question, decision, or task does this document serve for its
  intended reader?
- Why is this the correct document type and source-of-truth location?
- Which evidence supports each important claim, boundary, status, and example?
- Does the document distinguish current behavior, requirements, plans, and
  known gaps accurately?
- Does it duplicate, contradict, or improperly absorb another document type?
- Can every entity, relationship, section, and status be explained without
  referring to a template or Toudocu diagnostic?
- Are uncertainty, exclusions, failure conditions, and intentionally absent
  sections honest?
- Did the reader-first writing gate pass for prose, tables, and diagrams?
- Can a reader find the main understanding quickly without reconstructing raw
  implementation details?

Otherwise return `NEEDS_REWORK`, revise the draft, and repeat both applicable
gates.

## Independent review

Require an independent semantic review for:

- every new typed document;
- changes to requirements, behavior, boundaries, contracts, architecture,
  decisions, risks, roadmap scope, or current status;
- stable IDs or relationships;
- task scope, acceptance criteria, or verification;
- flow topology, screen maps, or another machine-readable model.

A spelling, formatting, link repair, terminology normalization, or other
meaning-preserving change needs the author review and writing gate but not an
independent semantic review.

Use another agent or a human reviewer. Give the reviewer the draft and the raw
product or repository evidence, without the author's desired verdict or a
Toudocu result presented as proof of quality. Ask the reviewer to check both
semantic accuracy and whether the wording could cause a reasonable reader to
infer more than the evidence supports. The reviewer returns `PASS` or
`NEEDS_REWORK` with concrete reasons. If an independent reviewer is required
but unavailable, report the gate as pending and do not declare the document
complete.

## Document boundaries

Keep neighboring document types distinct:

- a module explains a stable ownership or behavior boundary, not every folder;
- a use case describes observable actor behavior, not an implementation call
  graph;
- a flow visualizes one reusable scenario, not every branch in source code;
- a screen map explains product-wide navigation, not the router catalog;
- an ADR records a durable decision that was actually made, not a retrospective
  justification invented for current code;
- a roadmap declares global outcomes, not every local checklist;
- a task defines authorized work and observable completion, not an aspirational
  description padded to satisfy its schema;
- status reports the current verified snapshot, not desired future state;
- reference material catalogs facts, while a guide helps a reader achieve a
  result.

For architecture changes, continue with
[architecture-gate.md](architecture-gate.md). For flows and screen maps,
continue with [screen-model.md](screen-model.md).
