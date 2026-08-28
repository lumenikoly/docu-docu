<!-- toudocu
id: TASK-AGENT-015
status: done
taskType: feature
priority: high
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
standards: STD-GO-001, STD-DOCS-001
dependsOn: TASK-AGENT-013, TASK-AGENT-014
updated: 2026-08-28
-->

# TASK-AGENT-015: Добавить handoff действий задачи для внешнего coding agent

<!-- toudocu:section result -->
## Результат

Любое подходящее Task Action можно выполнить без встроенного structured
provider.

Toudocu формирует готовый переносимый handoff:

- action instruction;
- идентификатор и состояние задачи;
- необходимый компактный task context;
- acceptance criteria;
- dependency/readiness summary;
- ссылки/пути на authoritative документы;
- команду получения полного актуального контекста.

Browser может скопировать handoff и передать его Codex CLI, Claude Code,
OpenCode, Zed agent, IDE agent или другому coding agent.

Handoff остаётся provider-agnostic.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Если пользователь не хочет или не может использовать Agent Console, ему нужно
самостоятельно:

- открыть задачу;
- найти task ID;
- сформулировать действие;
- найти acceptance criteria;
- найти зависимости;
- объяснить агенту Toudocu workflow.

<!-- toudocu:section after -->
### Станет

У действия появляется delivery:

```text
Для внешнего агента
```

Toudocu формирует один bounded Markdown handoff, который можно скопировать
одним действием.

Для `start-work` Toudocu сначала применяет обычную серверную проверку readiness
и перевод `ready → in-progress`, после чего handoff строится из обновлённого
состояния задачи.

<!-- toudocu:section scope -->

## Область изменения

* Task Action delivery implementation;
* task context selection;
* handoff builder;
* task-actions HTTP contract;
* `web/src/features/task-workspace/`;
* clipboard UX;
* локализация;
* unit/browser tests;
* документация agent-agnostic workflow.

<!-- toudocu:section out-of-scope -->

## Не входит в задачу

* запуск внешнего coding agent;
* обнаружение всех AI CLI;
* provider-specific handoff format;
* автоматическое открытие терминала;
* auto-paste;
* auto-run;
* чтение terminal output;
* task binding Project Terminal;
* отправка исходников всего репозитория в handoff;
* полный dump `TaskContextReport`;
* clipboard API на стороне Go.


## Требования к реализации

### 1. Server-side handoff builder

Handoff формируется сервером из того же semantic Task Action registry, который
использует Agent Console.

Browser передаёт только:

* `taskID`;
* `actionID`;
* typed user input;
* `expectedDigest`, если action изменяет task state;
* `delivery=handoff`.

Browser не формирует instruction из DOM.

### 2. Handoff contract

Добавить presentation-safe объект:

```json
{
  "schemaVersion": 1,
  "taskID": "TASK-X",
  "actionID": "start-work",
  "mediaType": "text/markdown",
  "text": "...",
  "truncated": false,
  "fullContextCommand": "toudocu task context TASK-X ./docs --repository-root . --format json"
}
```

Clipboard остаётся browser concern.

Go API никогда не пытается напрямую работать с системным clipboard.

### 3. Формат handoff

Handoff должен быть пригоден для прямой вставки в современный coding agent.

Пример структуры:

```md
# Toudocu task handoff

## Action

Continue work on `TASK-X`.

Use the authoritative Toudocu task contract and current repository state.
Do not change the task contract without explicit user approval.
Do not mark the task Done automatically.

## Task

- ID: `TASK-X`
- Status: `in-progress`
- Document: `docs/work/TASK-X.md`

## Goal

...

## Acceptance criteria

- ...

## Dependencies

- ...

## Required context

- ...

For the complete current context run:

`toudocu task context TASK-X ./docs --repository-root . --format json`
```

Не добавлять provider-specific инструкции.

### 4. Compact context

Не вставлять полный `TaskContextReport`.

Использовать только информацию, необходимую для старта действия:

* ID/title/type;
* status/workspace state;
* task document path;
* result/goal;
* scope;
* acceptance criteria;
* blocker/readiness issues;
* dependencies;
* critical required reads;
* verification summary или команды, если они относятся к действию.

Для состояний, поддерживаемых `BuildTaskContext`, использовать его как
authoritative источник данных.

Для `draft`, где полный task context contract недоступен, использовать:

* parsed WorkItem;
* task document;
* readiness issues.

Не читать произвольные source files только ради handoff.

### 5. Bounded output

Ввести единый именованный предел размера handoff.

Ориентироваться на существующие ограничения Agent Console message payload, а не
добавлять несвязанный magic number.

При превышении лимита:

1. сохранять action instruction;
2. сохранять task ID/path;
3. сохранять acceptance criteria;
4. сокращать вторичный context;
5. выставлять `truncated=true`;
6. обязательно оставлять `fullContextCommand`.

Не обрезать UTF-8 посередине rune.

### 6. Read-only handoff

Для read-only semantic actions handoff явно сообщает:

```text
This action is intended to be read-only.
Do not modify repository files.
Toudocu cannot enforce the permissions of an external agent.
```

UI не должен показывать внешнему handoff badge вроде `Read-only enforced`.

### 7. `start-work` handoff

Для `start-work`:

1. server повторно проверяет digest/readiness;
2. переводит задачу в `in-progress`;
3. перечитывает модель;
4. строит handoff из нового состояния;
5. возвращает новый task/action snapshot вместе с handoff.

Если readiness или digest изменились, handoff не создаётся и task state не
меняется.

Отказ browser Clipboard API после успешного server action не откатывает
task state.

### 8. Clipboard UX

Основное действие browser:

```text
Скопировать
```

использует стандартный `navigator.clipboard.writeText()` в user gesture.

Если Clipboard API недоступен или отклонён:

* handoff отображается в selectable textarea/code area;
* пользователь получает `Copy again`;
* UI не сообщает об успешном копировании;
* данные не теряются.

Не использовать deprecated `document.execCommand('copy')` как основной путь.

### 9. Security

Handoff не включает автоматически:

* environment variables;
* credentials;
* API keys;
* содержимое user-local Toudocu state;
* provider configuration;
* скрытые Agent Session events;
* неограниченный Git diff;
* произвольные файлы вне выбранного task context.

<!-- toudocu:section acceptance-criteria -->

## Критерии приёмки

* [x] `AC-01` Каждый поддерживаемый Task Action может использовать
  `delivery=handoff` без доступного structured provider.
* [x] `AC-02` Agent Console и handoff используют один semantic prompt/action
  registry.
* [x] `AC-03` Handoff строится сервером и содержит компактный достаточный
  контекст без полного dump `TaskContextReport`.
* [x] `AC-04` `draft` имеет корректный handoff fallback без вызова
  неподдерживаемого `BuildTaskContext`.
* [x] `AC-05` Handoff имеет bounded UTF-8-safe размер и всегда оставляет команду
  получения полного context при truncation.
* [x] `AC-06` Read-only handoff явно отличает instruction от технически
  enforced provider sandbox.
* [x] `AC-07` `start-work` handoff использует тот же readiness/digest gate и
  переводит задачу в `in-progress` до построения итогового handoff.
* [x] `AC-08` Clipboard failure не откатывает task mutation и предоставляет
  пользователю ручной selectable fallback.
* [x] `AC-09` Browser не строит handoff из DOM и не содержит копий action
  templates.
* [x] `AC-10` Handoff не содержит credentials, environment или неограниченный
  repository content.

<!-- toudocu:section plan -->

## План

1. Добавить `TaskActionHandoff`.
2. Реализовать общий handoff builder.
3. Выделить компактную projection TaskContext.
4. Добавить Draft fallback.
5. Добавить bounded rendering.
6. Подключить `delivery=handoff` к task-actions API.
7. Добавить clipboard/fallback UI.
8. Проверить start-work mutation и read-only wording.
9. Покрыть security и truncation tests.

<!-- toudocu:section verification -->

## Проверка

* `AC-01` → `go test ./internal/app -run 'TestTaskActionHandoff|TestTaskActionHandoffDraft'`
* `AC-02` → `go test ./internal/app -run 'TestTaskActionHandoff|TestTaskActionHandoffDraft'`
* `AC-03` → `go test ./internal/app -run 'TestTaskActionHandoff|TestTaskActionHandoffDraft'`
* `AC-04` → `go test ./internal/app -run 'TestTaskActionHandoff|TestTaskActionHandoffDraft'`
* `AC-05` → `go test ./internal/app -run 'TestTaskActionHandoffBounds'`
* `AC-06` → `go test ./internal/app -run 'TestTaskActionHandoffReadOnly'`
* `AC-07` → `go test ./internal/app -run 'TestTaskActionHandoffStartWork'`
* `AC-08` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff'`
* `AC-09` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff'`
* `AC-10` → `go test ./internal/app -run 'TestTaskActionHandoffIsolation'`
* `QUALITY` → `make check`
* `ALL` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff'`
* `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->

## Влияние на документацию

Обновить:

* agent workflows;
* local workflow;
* Task Workspace/task page guide;
* task-actions contract;
* trust boundaries;
* `MOD-AGENT-CONSOLE`;
* `UC-AGENT-CONSOLE-01`.

Документация должна явно описывать внешний handoff как первый класс
agent-agnostic workflow, а не как fallback только при сломанной Agent Console.
