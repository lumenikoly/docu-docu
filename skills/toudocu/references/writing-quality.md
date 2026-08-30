# Reader-first writing gate

Use this reference whenever the work creates or revises reader-facing prose,
headings, lists, tables, diagram labels, messages, or translations. Apply it
after collecting evidence and before the semantic and structural gates.

Return `PASS` only when every applicable check passes. Otherwise return
`NEEDS_REWORK` with the failed `WRITE-*` codes and a concrete correction for
each failure.

## Select the reader and language

- `WRITE001`: identify the intended reader, the useful question or task, and
  the understanding or decision the document should provide. Do not write for
  the template, validator, or implementation alone.
- `WRITE002`: use one primary natural language for each document, paragraph,
  table label, and diagram label. For an existing canonical document, preserve
  its established language unless the user or project policy requires a
  migration. For a new document, use the active configured locale. During
  translation, use only the selected target locale.
- Preserve exact commands, flags, paths, filenames, URLs, stable IDs, API
  fields, enum values, event names, protocol and format names, and official
  product or library names. Do not translate or inflect text inside code spans.

## Put meaning before code terminology

- `WRITE003`: introduce an uncommon technical term through its meaning in the
  current context. Add the exact code token after that explanation only when it
  helps the reader trace the statement to implementation or a contract.
- Prefer the pattern ``plain-language concept (`CODE_NAME`)`` or a natural
  equivalent in the target language. Do not make a raw identifier carry the
  sentence by itself.
- Widely understood names such as HTTP, JSON, Git, CLI, API, and URL may remain
  unchanged when the intended audience knows them. Explain domain-specific
  acronyms, internal events, state names, fields, and error codes on first use.
- Remove an identifier when it adds no useful precision. Documentation is not a
  dump of every symbol that appears in code.

## Write complete, concrete statements

- `WRITE004`: make every important statement clear about who or what acts, what
  happens, under which relevant condition, and what result or consequence
  follows. Use active voice and present tense where they are natural in the
  target language.
- `WRITE005`: replace vague verbs and noun piles with concrete behavior. Words
  such as *handles*, *supports*, *provides*, *state*, *context*, *flow*, or
  *integration* require an object and an observable meaning; they must not hang
  in the text without explaining what changes for the system or reader.
- `WRITE006`: lead with the conclusion or rule, then add evidence, exceptions,
  and implementation detail. Keep one central point per paragraph. Remove
  filler, promotional language, repetition, and background that does not help
  the intended reader decide or act.

A technical paragraph does not need to mention an end user in every sentence.
It does need to explain why the technical fact matters: a boundary it enforces,
a failure it prevents, a decision it supports, or a behavior it produces.

## Separate facts, requirements, plans, and gaps

- `WRITE007`: state whether a claim describes verified current behavior, a
  required or expected behavior, a planned change, or a known gap. Do not blend
  these states into a sentence that sounds more complete than the evidence.
- Use explicit target-language markers when the distinction matters, such as
  equivalents of *currently*, *the contract requires*, *planned*, *not yet
  implemented*, or *known limitation*.
- State material conditions, exclusions, failure modes, and recovery paths.
  Do not hide a contradiction or incomplete implementation behind a generic
  summary.
- Keep stable issue, requirement, or rule IDs when they help the reader verify a
  gap or claim, but explain the finding before listing the IDs.

## Write idiomatically in every supported language

- `WRITE008`: express the meaning in idiomatic target-language grammar and
  terminology. Translate meaning, not English word order or source sentence
  structure.
- Avoid mixed-language hybrids, unnecessary transliteration, and untranslated
  common concepts when the target language has a clear established term. For
  example, avoid constructions such as `frontend ошибка`, `persisted-фаза`,
  `recovery action`, or `dev mode` inside otherwise Russian prose.
- Keep an English or code term when it is an official name, exact interface
  value, or genuinely established term for the intended audience. Surround it
  with natural target-language prose and explain it when it is not obvious.
- Follow an explicit project glossary when it is consistent with the product
  UI, contracts, and audience. Do not perpetuate a one-off awkward phrase merely
  because it already appears in one file. Report material terminology conflicts
  instead of silently choosing different meanings in different documents.

## Treat diagrams and tables as prose

- `WRITE009`: make every reader-facing Mermaid label, edge label, note, table
  heading, and descriptive cell understandable without reconstructing variable
  names or source code.
- In a user or business flow, label nodes with user actions, business events,
  visible states, or meaningful system outcomes. Phrase decision nodes as
  natural questions and branches as outcomes such as *yes*, *no*, *allowed*, or
  *rejected* in the target language.
- Do not use assignment or comparison syntax such as `canJoin = true?` as a
  reader-facing decision unless the document explicitly explains code logic.
  Prefer a question such as “Is joining allowed?” in the document language.
- Keep Mermaid keywords, node IDs, shape syntax, and exact protocol tokens
  unchanged. Translate the visible labels, notes, and arrow text. When an event
  or command name is contract-relevant, place it after the human meaning, for
  example `Create the participant (REGISTER)`.
- In a technical component or sequence diagram, exact component names may remain
  visible. Arrows and notes must still describe the action or result in natural
  language.

## Make procedures and failures actionable

- `WRITE010`: write procedural steps as ordered actions with clear actors,
  prerequisites, and an observable result when verification is not obvious.
  Keep one primary action per step.
- When documenting an error or degraded state, explain what happened, who or
  what is affected, what the system does next, and what action can continue or
  recover the process. Expose internal details only when the intended reader
  needs them.

## Final naturalness pass

Before returning `PASS`:

1. Read every changed paragraph and visible diagram label as continuous prose in
   the target language.
2. Rewrite any sentence that sounds translated, requires unexplained code
   knowledge, or contains a term with no clear role in the statement.
3. Check every code token: keep it exact, explain it when needed, and remove it
   when it adds no value.
4. Confirm that current behavior, requirements, plans, and gaps remain distinct.
5. Confirm that every material claim is supported by repository evidence or is
   explicitly marked as unknown.

## Examples

### Prose

Poor Russian documentation:

> Typed transport преобразует backend error payload в предсказуемую frontend
> ошибку. Пользователь получает безопасное сообщение и recovery action, а
> технические details не становятся UI contract. Loading, empty, success и
> error states принадлежат feature UI. Agentation toolbar подключается только в
> dev mode и не является production runtime dependency.

Better Russian documentation:

> Типизированный транспорт преобразует ответ серверной части в известную
> клиентскую ошибку. Целевое правило — показать человеку безопасное сообщение и
> понятное действие для продолжения, не раскрывая технические детали. Сейчас оно
> выполнено не везде: часть ошибок не даёт пути назад, окончательный ответ
> 404/410 активной игры может оставить загрузку без конца, а режим без сети
> объясняется неполно (`UI-003`–`UI-005`, `RT-015`, `UI-022`). Панель Agentation
> подключается только в режиме разработки и не входит в рабочую сборку продукта.

The improved version names the mechanism in natural language, explains the
reader-visible rule, separates the current gaps from the target behavior, and
keeps exact identifiers only as traceability evidence.

### Diagram labels

| Avoid | Prefer |
|---|---|
| `Resolve event: JOIN_LINK` | `Проверить событие по ссылке` |
| `canJoin = true?` | `Подключение разрешено?` |
| `REGISTER` | `Создать участника (REGISTER)` |
| `Saved player identity?` | `Для события уже сохранён этот участник?` |
| `RESUME saved identity` | `Восстановить сохранённого участника (RESUME)` |
| `Save participantId, reconnect token and access token` | `Сохранить идентификатор, секрет восстановления и доступ` |

Keep `JOIN_LINK`, `REGISTER`, `RESUME`, or another exact token visible only when
its contract identity matters to the reader. The node ID may remain technical;
the visible label must communicate the human meaning.
