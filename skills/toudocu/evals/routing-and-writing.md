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
