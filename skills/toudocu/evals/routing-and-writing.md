# Toudocu skill evaluation cases

Use these cases when changing the skill description, routing table, or writing
rules. They are maintainer tests, not runtime instructions.

## Trigger evaluation

Use [`trigger-prompts.csv`](trigger-prompts.csv) as the routing dataset. Run each
prompt in a clean context with the same model, repository fixture, and available
skills. Repeat each case at least three times because skill selection is not
fully deterministic. Record the model version, skill checksum, invocation result,
and any unexpected side effects.

A routing run passes when:

- a `should_trigger=true` case loads this skill in at least two of three runs;
- a `should_trigger=false` case loads this skill in no more than one of three
  runs;
- explicit `$toudocu` workflows select only the requested operation;
- no case infers `$toudocu init` or executes `task verify --run` without the
  required explicit authorization.

When the description changes, compare the same prompt set before and after the
change. Add real false positives, false negatives, and ambiguous requests to the
dataset instead of weakening the expected boundary.

## Workflow routing

These behavioral cases inspect the execution transcript, not only the final
answer. Do not add them to `trigger-prompts.csv`; that file tests skill
activation, not tool order.

### Implement a Ready task

Input:

> Implement TASK-CLI-014.

Expected transcript:

- the skill activates;
- `task context --format json` runs before broad search across canonical
  documentation;
- the returned documents and relationships form the starting map;
- repository or code search may inspect source code afterward.

### Continue a bug investigation

Input:

> Continue investigating BUG-CLI-007.

Expected transcript:

- `task context --format json` first loads the existing Ready+ work item;
- manual traversal of canonical documentation does not replace that command.

### Prepare context and a verification plan

Input:

> Prepare the context for TASK-CLI-014 and show its verification plan without
> running commands.

Expected transcript:

1. `task context --format json`;
2. `task verify --dry-run --format json`.

The agent does not run `task verify --run`.

### Find documentation

Input:

> Find the Toudocu documentation describing authentication configuration.

Expected transcript:

- `toudocu search` runs before any broad raw search across canonical
  documentation;
- broad `rg` does not replace `toudocu search`;
- identified files may then be read directly.

### Review a known file

Input:

> Review `docs/reference/configuration.md`.

Expected transcript:

- the agent may read the named file directly;
- it does not run `toudocu search` only to satisfy Toudocu-first routing.

### Find code consumers

Input:

> For TASK-CLI-014, find all Go consumers of the API mentioned in its context.

Expected transcript:

1. `task context --format json`;
2. repository or code search for Go consumers.

Toudocu maps the task context but does not replace source-code search.

A workflow case passes only when the required Toudocu command actually runs
before an equivalent broad raw search across canonical documentation. Direct
reading of a known file and later source-code search remain allowed. The
transcript must not contain irrelevant Toudocu commands, an unauthorized
`task verify --run`, or use of configured translation roots as canonical
documentation or work-item context.

## Clarification workflow

### Case 1 — Existing task

Input:

> `$toudocu clarify TASK-CLI-014`

Expected transcript:

- the skill activates and selects `references/clarify.md`;
- `task context --format json` runs before broad documentation search;
- implementation code does not change;
- `task verify --run` does not run.

### Case 2 — Free-form feature

Input:

> `$toudocu clarify the authentication configuration redesign`

Expected transcript:

- `toudocu search` runs before broad raw search across canonical documentation;
- the returned documents form the initial context;
- repository code is inspected when necessary;
- the user is not asked to supply facts available from the repository.

### Case 3 — Fact versus decision

The fixture defines an existing default in authoritative configuration and a
proposal that may change it.

Expected transcript:

- the agent finds and states the current default without asking the user;
- the desired future default remains a decision put to the user.

### Case 4 — Whole frontier

The fixture exposes several independent decisions with settled prerequisites.

Expected transcript:

- every currently independent decision appears in the same round;
- there is no artificial `1–3` or other numerical question limit;
- no available question is deferred merely because of the round size.

### Case 5 — Dependent question

Decision B can be formulated only after decision A is answered.

Expected transcript:

- A appears in the first round;
- B does not appear in that round;
- B appears in a later frontier only when A makes it relevant.

### Case 6 — Pruned branch

The user's answer to an upstream decision makes a downstream branch
irrelevant.

Expected transcript:

- the branch is removed when the frontier is recomputed;
- no question from that branch is ever asked.

### Case 7 — Recommendation

Expected transcript:

- each decision with a defensibly preferred option includes a recommendation
  and a short reason;
- the recommendation is not treated as the user's answer and does not resolve
  the decision automatically.

### Case 8 — Large frontier

The fixture has more than three independent frontier decisions.

Expected transcript:

- the round contains the complete frontier rather than only three questions.

### Case 9 — Shared understanding

After the frontier becomes empty, expected transcript:

- the agent summarizes the confirmed design, exclusions, constraints, open
  questions, and affected canonical documents;
- the agent waits for explicit user confirmation;
- clarification is not declared complete and implementation does not begin
  before confirmation.

### Case 10 — Correction after summary

The user corrects a decision in the shared-understanding summary.

Expected transcript:

- the design tree is updated and the frontier is recomputed;
- any newly reachable decisions are interviewed;
- implementation does not begin.

### Case 11 — Documentation persistence

After confirmation, a durable decision belongs to an existing canonical
source.

Expected transcript:

- the existing source of truth is updated and receives the exact durable
  constraints;
- `CONTEXT.md`, `CONTEXT-MAP.md`, `clarification.md`, and `decision-log.md` are
  not created without a separate product reason;
- the ordinary documentation check runs after the mutation.

### Case 12 — ADR gate

The fixture contains four decisions: one satisfies all three ADR criteria; the
others respectively lack hard-to-reverse cost, surprise without context, or a
real trade-off.

Expected transcript:

- an ADR is permitted only for the decision satisfying all three criteria;
- no ADR is created for any of the other three decisions.

## Reader-first writing

### Mixed-language prose

Input:

> Typed transport преобразует backend error payload в предсказуемую frontend
> ошибку и предоставляет recovery action.

Expected properties:

- the output uses idiomatic Russian prose;
- it explains the server response, client error, and available next action;
- it keeps an exact code token only when needed for traceability;
- it does not invent current behavior or a recovery path.

### Diagram labels

Input labels:

```text
Resolve event: JOIN_LINK
canJoin = true?
REGISTER
```

Expected properties:

- visible labels are written in the document language;
- the decision is a natural question about the business condition;
- `JOIN_LINK` or `REGISTER` appears only after a human-readable meaning when its
  exact identity matters;
- Mermaid node IDs and syntax remain unchanged.

### Truth states

Input evidence says that a recovery action is required but missing for two error
paths.

Expected properties:

- the required behavior and current gaps are separate statements;
- the output does not say the recovery behavior is fully implemented;
- issue or requirement IDs follow the explanation rather than replacing it.
