<!-- toudocu
id: TASK-AGENT-013
status: done
taskType: feature
priority: high
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
standards: STD-GO-001, STD-DOCS-001
dependsOn: TASK-AGENT-007, TASK-AGENT-011
updated: 2026-08-27
-->

# TASK-AGENT-013: Отделить действия задачи от способа доставки агенту

<!-- toudocu:section result -->
## Результат

Toudocu получает единый серверный контракт действий над задачей.

Доступность действия определяется состоянием задачи и доменными правилами
Toudocu, а способ выполнения выбирается отдельно:

- через существующую structured Agent Console;
- через переносимый handoff для внешнего coding agent.

Один и тот же semantic action, например `start-work`, `ask` или `clarify`,
не имеет отдельных Codex/OpenCode/terminal-вариантов и не дублирует prompt,
валидацию readiness или правила состояния задачи.

Task Markdown остаётся единственным источником долговременного состояния задачи.
Agent Session, Task Workspace, task detail page и handoff не создают собственных
копий task status или task contract.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

`TaskWorkspaceItem.AgentActions` заполняется только при доступной
`Agent Console`.

Browser вызывает специальные Agent Console endpoints.

`start-work` непосредственно связан с запуском structured provider, а остальные
task actions автоматически создают или используют Agent Session.

При активной Agent Session действие другой задачи может попасть в существующую
сессию без достаточной проверки task binding.

Для продолжения уже начатой задачи после завершения Agent Session отдельного
semantic action нет.

<!-- toudocu:section after -->
### Станет

Backend предоставляет единый `Task Action` application layer.

Он:

1. перечитывает актуальную модель;
2. определяет `workspaceState`;
3. возвращает разрешённые действия;
4. проверяет action повторно непосредственно перед выполнением;
5. применяет необходимую task mutation;
6. передаёт результат выбранному delivery adapter.

Доступность task actions не зависит от наличия Codex, OpenCode или другой
structured integration.

Agent Console является одним из способов доставки, а не владельцем действий
задачи.

Внешний handoff является вторым способом доставки и не требует установленного
structured provider.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_prompts.go`;
- `internal/app/agent_task.go`;
- `internal/app/task_site.go`;
- `internal/app/agent_session.go`;
- `internal/app/agent_console_http.go`;
- новый небольшой application-layer код для task actions;
- `internal/site/bootstrap.go`;
- новый HTTP/OpenAPI contract task actions;
- frontend-модель Task Workspace;
- документация архитектурной границы.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- новый provider plugin framework;
- изменение `AgentProvider`;
- реализация OpenCode;
- изменение Project Terminal;
- автоматическое выполнение handoff во внешнем терминале;
- drag-and-drop task statuses;
- generic workflow engine;
- конфигурируемые пользователем действия задачи;
- repository-defined prompts;
- multi-agent assignment;
- несколько параллельных Agent Sessions;
- сохранение нового task state вне Markdown.

## Требования к реализации

### 1. Единый реестр действий

Существующий `agentTaskActions` преобразовать в единый закрытый серверный
реестр task actions.

Реестр должен содержать только необходимые серверу свойства, например:

- стабильный `id`;
- тип пользовательского ввода: `none` или `text`;
- policy для structured Agent Console;
- builder подготовленного prompt;
- состояния задачи, в которых действие допустимо;
- признак primary/secondary presentation при необходимости;
- признак изменения task state.

Browser не получает prompt template и не формирует prompt самостоятельно.

Пользовательский ввод не должен конкатенироваться в HTTP handler или frontend.
Формирование итоговой instruction должно происходить централизованно через
typed prompt builder.

Не создавать универсальный expression engine или DSL.

### 2. Server-driven action resolution

Добавить серверную функцию уровня приложения, которая по актуальным:

- `Model`;
- `WorkItem`;
- readiness;
- `workspaceState`;
- состоянию Agent Session;

строит presentation-safe список разрешённых действий.

Frontend не должен самостоятельно выводить допустимость action из `status`,
dependencies или DOM.

### 3. Матрица действий

Минимальная матрица:

#### `ready`

- `start-work`;
- `ask`;
- `clarify`.

#### `in-progress`

- `continue-work`;
- `ask`;
- `clarify`;
- `next`;
- `refresh-diff`.

#### `waiting`

- `ask`;
- `explain-blocker`.

#### `needs-attention`

- `ask`;
- `clarify`;
- `explain-problems`.

#### `draft`

- `ask`;
- `clarify`;
- `explain-problems`.

#### `blocked`

- `ask`;
- `clarify`;
- `explain-blocker`.

#### `done`

- `ask`.

#### `cancelled` и `archive`

Structured agent actions не предоставляются.

Durable discussion для этих состояний относится к отдельной задаче и не входит
в эту матрицу.

### 4. `continue-work`

Добавить отдельное действие `continue-work`.

Оно предназначено только для уже находящейся в `in-progress` задачи.

Если Agent Session отсутствует, delivery через Agent Console создаёт новую
task-bound provider session и просит агента заново прочитать актуальный
Toudocu task context и состояние репозитория.

Предыдущий provider thread автоматически не восстанавливается.

Если активная Agent Session уже принадлежит той же задаче, новая session и новый
`continue-work` turn не создаются. Server projection сообщает frontend, что
задача уже имеет active session и её следует открыть.

Если Agent Session принадлежит другой задаче, console delivery для выбранной
задачи запрещено с typed conflict. Handoff delivery при этом остаётся
доступным.

### 5. Delivery contract

Поддержать закрытый набор delivery types:

- `agent-console`;
- `handoff`.

Не кодировать provider в action ID.

Запрещены варианты вида:

- `start-work-codex`;
- `start-work-opencode`;
- `ask-terminal`.

Structured provider выбирается существующим Agent Console preference/session
механизмом.

### 6. HTTP contract

Добавить отдельный task-actions HTTP contract, не принадлежащий
provider-specific Agent Console API.

Целевой интерфейс:

```text
GET  /_toudocu/api/tasks/{taskID}/actions
POST /_toudocu/api/tasks/{taskID}/actions/{actionID}
```

`GET` возвращает authoritative projection:

```json
{
  "schemaVersion": 1,
  "task": {
    "id": "TASK-X",
    "status": "ready",
    "workspaceState": "ready",
    "digest": "..."
  },
  "actions": []
}
```

Каждое действие содержит только presentation-safe данные и доступные delivery
options.

`POST` принимает заранее определённый объект:

```json
{
  "delivery": "agent-console",
  "expectedDigest": "...",
  "input": {
    "text": "..."
  }
}
```

`expectedDigest` обязателен для действий, изменяющих task Markdown.

Browser не передаёт:

* executable;
* argv;
* cwd;
* environment;
* provider protocol;
* sandbox policy;
* произвольный prompt template.

После перевода frontend на новый контракт существующие дублирующие внутренние
task-action handlers удалить, а не оставлять постоянные compatibility wrappers.

### 7. `start-work`

`start-work` остаётся единственным действием, выполняющим переход:

```text
ready → in-progress
```

Перед изменением повторно проверяются:

* актуальный digest;
* machine status `ready`;
* `readyForWork`;
* dependencies;
* readiness contract.

Для `agent-console` до изменения task state должны быть разрешены все заранее
определимые ошибки:

* provider unavailable;
* session другой задачи;
* invalid preset/setup;
* невозможность создать session.

Для `handoff` structured provider вообще не проверяется.

После успешного перехода модель перечитывается, и дальнейший результат строится
уже из актуального `in-progress` состояния.

Не заявлять транзакционность между filesystem и внешним provider process,
которой фактически нет. Неопределённая provider delivery не должна маскироваться
автоматическим откатом task state.

### 8. Read-only semantics

Для `ask`, `explain-blocker`, `explain-problems`, `next` delivery через
Agent Console использует существующую session capability `ReadOnlyTurns`.

Если конкретная session не может технически обеспечить требуемый read-only
turn, console delivery этого действия недоступно.

Handoff не может гарантировать sandbox внешнего агента. Для него read-only
является явно обозначенной instruction, а не технической гарантией.

### 9. Security boundary

Task Action execution существует только в canonical loopback `serve`.

Static portal, translations и non-loopback `serve`:

* не получают mutation endpoints;
* не получают Agent Console delivery;
* не могут запускать внешний process;
* не могут переводить задачу в `in-progress`.

<!-- toudocu:section acceptance-criteria -->

## Критерии приёмки

* [x] `AC-01` Список task actions вычисляется сервером из `workspaceState` и
  больше не зависит от `model.agentConsoleEnabled`.
* [x] `AC-02` Один action registry формирует semantics и prompt для всех
  delivery modes; browser и HTTP handlers не содержат дублирующих prompt
  templates.
* [x] `AC-03` Поддерживается закрытый delivery contract
  `agent-console | handoff`; provider не кодируется в action ID.
* [x] `AC-04` `continue-work` корректно различает отсутствие session, active
  session той же задачи и session другой задачи.
* [x] `AC-05` Ни одно действие задачи не может быть отправлено в Agent Session,
  привязанную к другому `taskID`.
* [x] `AC-06` `start-work` повторно проверяет readiness и digest и одинаково
  применяет `ready → in-progress` для Agent Console и handoff delivery.
* [x] `AC-07` Read-only actions используют `ReadOnlyTurns` для structured
  delivery и не выдают instruction-only handoff за технически защищённый
  sandbox.
* [x] `AC-08` Новый HTTP contract является authoritative для task action
  projection и execution; после миграции дублирующие старые handlers удалены.
* [x] `AC-09` Static, translations и non-loopback serve не получают execution
  endpoints.
* [x] `AC-10` Все invalid state, stale digest, busy-other-task и unavailable
  delivery случаи возвращают typed ошибки и не выводятся frontend из строк
  exception.

<!-- toudocu:section plan -->

## План

1. Выделить semantic Task Action registry.
2. Реализовать resolver действий по `workspaceState`.
3. Добавить `continue-work`.
4. Выделить delivery selection из action semantics.
5. Исправить Agent Session task binding.
6. Перенести start-work mutation в общий coordinator.
7. Добавить HTTP/OpenAPI task-actions contract.
8. Перевести существующий Task Workspace на новый contract.
9. Удалить дублирующие старые task-action handlers.
10. Зафиксировать решение в ADR.

<!-- toudocu:section verification -->

## Проверка

* `AC-01` → `go test ./internal/app -run 'TestTaskActionResolver|TestTaskActionRegistry|TestTaskActionDelivery'`
* `AC-02` → `go test ./internal/app -run 'TestTaskActionResolver|TestTaskActionRegistry|TestTaskActionDelivery'`
* `AC-03` → `go test ./internal/app -run 'TestTaskActionResolver|TestTaskActionRegistry|TestTaskActionDelivery'`
* `AC-04` → `go test ./internal/app -run 'TestTaskContinueWork|TestTaskActionSessionBinding'`
* `AC-05` → `go test ./internal/app -run 'TestTaskContinueWork|TestTaskActionSessionBinding'`
* `AC-06` → `go test ./internal/app -run 'TestTaskActionStartWork|TestStartTaskReadiness|TestStartTaskDigest'`
* `AC-07` → `go test ./internal/app -run 'TestTaskActionReadOnlyDelivery'`
* `AC-08` → `go test ./internal/app -run 'TestTaskActionsHTTP' && make web-check`
* `AC-09` → `go test ./internal/app -run 'TestTaskActionsRuntimeIsolation'`
* `AC-10` → `go test ./internal/app -run 'TestTaskActionsHTTP'`
* `QUALITY` → `make check`
* `ALL` → `make check && make browser-test`
* `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->

## Влияние на документацию

Создать ADR о разделении Task Action semantics и delivery.

Обновить:

* `MOD-AGENT-CONSOLE`;
* контракт Agent Console;
* новый task-actions contract;
* frontend runtime boundary;
* trust boundaries;
* task workflow;
* work-items guide;
* local workflow.

Документация должна явно различать:

```text
Task Action
≠
Agent Provider
≠
Delivery
≠
Project Terminal
```
