# Clarify before implementation

Use this workflow only when the user explicitly invokes `$toudocu clarify` or
clearly asks Toudocu to interview them until a plan, feature, design, migration,
task, architecture change, contract, or other proposed behavior is sufficiently
defined.

Clarification exists to remove meaningful ambiguity before implementation.
Investigate facts yourself. Put genuine decisions to the user. Treat the subject
as a design tree, work through every reachable decision frontier, preserve the
confirmed durable result in its proper source of truth, and stop before
implementation.

Clarification is not a Go CLI command. It permits read-only investigation,
questions, and only the canonical documentation mutations and ordinary
documentation checks defined below. By default, mutations wait for the
shared-understanding gate; the user may explicitly request immediate
persistence of one settled decision.

It does not authorize implementation code changes, product migration,
refactoring, arbitrary repository commands, Git writes, publishing,
`task verify --run`, Agent Feedback transport, or unrelated cleanup.

## Establish the subject

Determine what the user is trying to make sufficiently clear to continue.

Use the subject already present in the request. If the user named a `TASK-*`,
`BUG-*`, document, feature, behavior, migration, or other concrete target, start
there.

Do not ask the user to restate information that is already clear from their
request or available from project evidence.

Do not create a work item merely because clarification has started.

## Build context before interviewing

Research the subject before constructing the first decision frontier.

For Toudocu-managed documentation, prefer Toudocu's structured discovery over
generic filesystem search when it covers the required context.

For an existing Ready+ work item, the first documentation-context command must
be:

```bash
toudocu task context <ID> <docs-root> \
  --repository-root <repository-root> \
  --format json
```

For a free-form subject, discover relevant source documents first with:

```bash
toudocu search "<query>" <docs-root> --format json
```

When the subject concerns current repository changes, use the applicable
read-only `toudocu changes`, `toudocu changes file`, or `toudocu task changes`
command.

Use returned documents, entities, dependencies, related standards, runbooks,
screens, flows, and diagnostics as the initial investigation map.

Follow [workflows.md](workflows.md) for repository conventions, exact CLI
usage, documentation roots, repository root, translation boundaries, and
read-only command behavior.

Use the returned relationships and paths to identify the minimum canonical
documents that need full inspection. Then inspect the minimum relevant code,
tests, contracts, configuration, CI, UI, history, or other repository evidence
needed to understand the subject.

Do not begin by grepping the entire documentation tree when Toudocu can provide
the relevant document or task context directly. Generic repository search
remains appropriate for implementation evidence, unsupported file types, or
questions outside Toudocu's model.

Expand the investigation only when a current question cannot be answered
reliably from the evidence already collected.

## Facts are the agent's job

Separate facts from decisions throughout the session.

A **fact** is something authoritative project evidence can reasonably establish:
current implementation, existing documented behavior, configured defaults,
known IDs and relationships, present constraints, tests, interfaces, or current
repository state.

A **decision** chooses desired behavior, policy, scope, compatibility,
terminology, ownership, trade-offs, priorities, or another future direction
that project evidence cannot decide for the user.

Investigate facts. Ask the user for decisions.

Never ask the user:

- what the current implementation does when the repository can establish it;
- where behavior is documented when Toudocu can locate it;
- which typed entities or relationships already exist when Toudocu can report
  them;
- what a configured default or public interface currently is when the source
  establishes it;
- to repeat a decision already settled in the current session.

Do not turn incomplete investigation into a design question.

If a user's claim about current behavior conflicts with authoritative evidence,
state the claim, show what the evidence establishes, and ask whether the claim
is a new requirement or a misunderstanding. Do not assume that current code is
the correct future behavior or that the user's assumption is a current fact.

If the user explicitly delegates a decision, investigate the relevant facts,
choose the recommended option, state its material trade-off, and treat that
choice as confirmed within the delegated scope.

When exploration can run independently of existing questions, do not stall the
whole interview. Questions that depend on an unresolved fact wait; independent
questions remain eligible for the current frontier.

## Maintain a design tree

Model the clarification internally as a design tree.

Do not persist the tree merely to run the workflow. In particular, do not
create `design-tree.json`, `clarification.json`, `CONTEXT.md`, or
`CONTEXT-MAP.md`.

Each unresolved decision may unlock further decisions. A decision can also make
entire downstream branches irrelevant.

The **frontier** is every unresolved decision whose prerequisites are already
settled and whose question can therefore be answered meaningfully now.

A **round** is the complete current frontier.

Ask the whole frontier in the round. Do not impose an arbitrary maximum number
of questions, either per round or for the whole session.

If the current frontier contains twelve independent decisions, ask those twelve.
If the subject ultimately requires thirty decisions, continue until all
material branches have been resolved or deliberately left open.

Question count is not a completion criterion.

Never move a question into the current round when its useful answer depends on
another unresolved question. It belongs to a later frontier.

Prefer an upstream decision when its answer may remove several downstream
branches.

After every user response:

1. record which decisions are now settled;
2. apply answers to the design tree;
3. remove branches made irrelevant;
4. identify facts that must now be investigated;
5. identify decisions whose prerequisites are now settled;
6. recompute the complete frontier;
7. ask the next round.

Continue until the frontier is empty.

## Make rounds easy to answer

Number every question and give it a short title.

Each question must identify one real decision and explain enough context for
the user to understand its consequence.

When there is a defensible preferred answer, give the recommendation explicitly
after the question.

Use this shape:

```text
Q1 — Backward compatibility

Should the previous documentation format remain supported after migration?

Recommendation: No. Require migration to the current format before normal
operations, because maintaining two documentation contracts would create a
permanent compatibility surface.
```

A recommendation is advice, not a hidden default. The user still decides.

When several alternatives matter, show the meaningful alternatives and their
material trade-offs. Do not manufacture options merely to make a question look
structured.

Questions in the same frontier may be grouped under topical headings such as
compatibility, migration, runtime behavior, user experience, security, or
operations. Grouping must not omit questions from the frontier.

Do not:

- ask one question at a time when several independent frontier decisions are
  already answerable;
- dump downstream speculative questions whose prerequisites are still open;
- ask vague prompts such as "anything else?";
- ask several phrasings of the same decision;
- ask the user to choose facts that should be researched;
- hide a material downside of your recommendation;
- treat existing implementation as mandatory future behavior.

## Stress-test language and boundaries

Challenge ambiguity that could change the resulting design.

When the project has an authoritative glossary, use its vocabulary. If the user
uses a term differently from the established meaning, surface the conflict and
clarify which meaning is intended.

When a term is overloaded or vague, propose a more precise term when doing so
would remove a real ambiguity. Do not create terminology for ordinary technical
concepts that already have clear names.

Cross-reference important claims with the code and canonical documentation. If
the user describes behavior that contradicts current evidence, say what the
evidence shows and clarify whether the current behavior or the newly stated
intent should govern the change.

Use concrete scenarios when they expose ambiguity more effectively than
abstract questions. In particular, probe relevant boundary and failure cases
rather than clarifying only the happy path.

Depending on the subject, this may include:

- unsupported behavior;
- backward compatibility;
- partial or stale state;
- empty or missing data;
- cancellation and retry behavior;
- source-of-truth ownership;
- permissions and trust boundaries;
- failure and recovery behavior;
- lifecycle or deletion;
- migration and rollback;
- numerical limits or defaults;
- ordering or consistency guarantees;
- negative requirements;
- behavior across configured locales;
- conflicts between related entities.

These are prompts for reasoning, not a mandatory checklist. Ask only what is
material to the current design tree.

## Preserve decisions in Toudocu sources

Do not reproduce the original `grill-with-docs` paper trail literally.

Do not create `CONTEXT.md`, `CONTEXT-MAP.md`, `clarification.md`,
`decision-log.md`, a clarification transcript, or a generic decision ledger
merely because this workflow ran.

Toudocu-managed documentation already has purpose-specific sources of truth.
Preserve a confirmed durable decision in the document whose purpose owns that
decision.

Typical mappings are:

- observable actor behavior → the applicable `UC-*`, guide, or public contract;
- reusable significant process → `FLOW-*` when a distinct flow is justified;
- stable component ownership, behavior, or boundary → architecture or `MOD-*`;
- public CLI, API, configuration, data, or schema behavior → its canonical
  contract or reference;
- user-visible screen behavior → the applicable `SC-*`, `TR-*`, use case, or
  screen model;
- acceptance or verification requirement → the existing qualifying `TASK-*` or
  `BUG-*`;
- enforceable project rule → the appropriate `STD-*` when it truly is a
  standard;
- operational procedure → the appropriate `RB-*`;
- durable architectural decision → `ADR-*` when the ADR gate passes;
- project-specific terminology → the existing authoritative glossary, if the
  project already maintains one;
- temporary implementation detail → normally no permanent documentation.

Apply [document-model.md](document-model.md) before choosing or creating a typed
document.

Update an existing source of truth instead of creating a duplicate.

Do not create a glossary just because a term was clarified.

Do not create a typed document merely to store an interview answer.

Do not create a task merely to preserve decisions from the session.

Unlike the original `grill-with-docs` workflow, do not leave a durable,
implementation-relevant decision only in conversation when an existing
canonical Toudocu document owns it. This includes exact defaults, limits,
ordering guarantees, compatibility rules, negative requirements, failure
behavior, and other constraints that would be weakened if later reconstructed
from memory.

## Offer ADRs sparingly

A decision qualifies for an `ADR-*` only when all three conditions are true:

1. **Hard to reverse** — changing the decision later would have meaningful
   technical, product, migration, compatibility, or operational cost.
2. **Surprising without context** — a future maintainer could reasonably see
   the resulting system and wonder why this approach was chosen.
3. **Real trade-off** — credible alternatives existed and the project selected
   one for a meaningful reason.

If any condition is absent, do not create an ADR.

An ADR is not required merely because a decision is important.

Easy-to-reverse choices should simply be changed when necessary. Obvious choices
do not need an explanation document. A decision with no credible alternative
does not need a record pretending that a trade-off occurred.

When an ADR qualifies, follow Toudocu's `decisions/*.md` and `ADR-*` model.
Record the decision and the context needed to understand why it was made. Keep
rejected alternatives or consequences only when they are useful to a future
reader.

The ADR must describe a decision actually made during the clarification, not a
retrospective justification invented from the implementation.

## Do not persist a half-understood design

The conversation may settle individual decisions before the complete design
tree is understood, but canonical Toudocu documents are sources of truth rather
than interview scratch space.

Do not continuously rewrite canonical documentation after every answer merely
to mirror the conversation.

Keep the settled decisions in the active clarification context while the
frontier remains open.

If the user explicitly asks to persist an already settled decision before the
session ends, update only the appropriate canonical source and apply the
ordinary documentation gates.

Otherwise wait for the shared-understanding gate before applying the complete
confirmed documentation change.

This deliberately differs from the original `grill-with-docs` behavior of
writing glossary terms immediately: Toudocu has richer canonical document types,
and partial design state should not temporarily masquerade as accepted project
documentation.

## Reach shared understanding

An empty frontier means there are no currently unresolved reachable decisions.
It does not by itself authorize action.

Re-evaluate the original clarification goal before closing the session.

Check that every material ambiguity discovered during the interview is one of:

- resolved by project evidence;
- decided by the user;
- explicitly delegated to the agent and resolved;
- deliberately left open;
- made irrelevant by another decision.

Then present a compact shared-understanding summary.

Include:

- the important confirmed decisions;
- important exclusions and negative requirements;
- material constraints on implementation;
- any questions deliberately left open;
- the canonical documents that should change, if any.

Do not reproduce the whole interview.

Ask the user to confirm or correct this understanding.

If the correction changes an existing decision or unlocks another branch,
update the design tree, recompute the frontier, and continue interviewing.

Clarification is complete only when:

1. the frontier is empty; and
2. the user confirms the shared understanding.

## Apply the confirmed documentation

After shared understanding is confirmed, preserve the durable result in the
appropriate canonical Toudocu sources.

Before changing reader-facing source documentation, apply
[writing-quality.md](writing-quality.md) and
[semantic-gate.md](semantic-gate.md), plus any applicable architecture, screen,
document, or work-item gate.

Make the smallest set of documentation changes that fully preserves the
confirmed design.

Distinguish clearly between:

- current behavior;
- newly confirmed requirements;
- planned implementation;
- known gaps or unresolved decisions.

Do not describe planned behavior as already implemented.

Do not invent implementation details, statuses, dates, relationships,
verification commands, or other facts that the interview did not establish.

If the session produced no durable documentation change, do not manufacture one.

Run the ordinary project documentation check after source-documentation
mutations.

Use strict mode only when repository policy, CI, or the user requires it.

Do not build the portal unless requested or needed to verify the result.

## Stop before implementation

Clarification does not include implementation.

Do not edit implementation code.

Do not run `task verify --run`.

Do not begin migration, refactoring, cleanup, or another implementation action
just because the user confirmed the design.

Do not silently transition from clarification into ordinary implementation.

Finish by stating:

- what was clarified;
- which canonical documents changed, if any;
- which important questions remain deliberately open;
- which documentation validation ran;
- that implementation has not started.

A later implementation request starts the ordinary Toudocu workflow and uses
the confirmed canonical documentation as evidence.
