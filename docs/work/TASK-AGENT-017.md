<!-- toudocu
id: TASK-AGENT-017
status: ready
taskType: feature
priority: medium
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
standards: STD-GO-001, STD-DOCS-001
dependsOn: TASK-AGENT-012, TASK-AGENT-015
updated: 2026-08-27
-->

# TASK-AGENT-017: Связать внешний task handoff с Project Terminal

<!-- toudocu:section result -->
## Результат

Пользователь может одним явным workflow подготовить действие задачи для
внешнего coding agent и перейти в встроенный Project Terminal:

```text
Скопировать и открыть Terminal
```

Toudocu:

1. получает authoritative handoff;
2. копирует его в clipboard;
3. открывает или фокусирует Project Terminal.

После этого пользователь сам запускает нужный CLI/TUI и вставляет handoff.

Project Terminal не становится AgentProvider, не получает task binding и не
интерпретирует вставленный текст.

<!-- toudocu:section behavior-change -->

## Изменение поведения

<!-- toudocu:section before -->

### Было

Handoff можно скопировать, а Project Terminal существует как отдельная
независимая возможность.

Пользователь вручную открывает terminal после copy.

<!-- toudocu:section after -->

### Станет

Для handoff-capable action доступно:

```text
Скопировать
Скопировать и открыть Terminal
```

Второе действие объединяет только навигационные шаги UI.

Оно не выполняет prompt, не пишет его в PTY и не запускает coding agent.

<!-- toudocu:section scope -->

## Область изменения

* `web/src/features/`;
* Project Terminal public frontend event/API;
* существующий Project Terminal lifecycle;
* локализация;
* browser tests;
* local workflow documentation.

<!-- toudocu:section out-of-scope -->

## Не входит в задачу

* auto-paste;
* synthetic keyboard events;
* запись handoff в PTY stdin;
* запуск `codex`, `claude`, `opencode` или другой AI CLI;
* определение установленного external agent;
* provider detection для Project Terminal;
* разбор terminal output;
* TUI automation;
* task-bound terminal;
* несколько Project Terminals;
* изменение shell command;
* repository-controlled shell configuration.


## Требования к реализации

### 1. Использовать существующий handoff

Project Terminal integration не строит prompt или context самостоятельно.

Она получает готовый `TaskActionHandoff` из реализации `TASK-AGENT-015`.

Не создавать `terminalPromptBuilder`.

### 2. Порядок действий

Для `Скопировать и открыть Terminal`:

1. выполнить/подготовить task action на server;
2. получить итоговый handoff;
3. попытаться скопировать `handoff.text`;
4. при успешном copy открыть/focus Project Terminal;
5. показать краткое подтверждение, что handoff скопирован.

Для `start-work` server mutation выполняется только один раз через общий Task
Action coordinator.

### 3. Clipboard failure

Если clipboard write не удался:

* UI не сообщает `Copied`;
* показывает selectable handoff;
* предлагает `Copy again`;
* отдельно позволяет открыть Terminal вручную.

Task mutation, уже успешно выполненная server action, не откатывается.

### 4. Terminal lifecycle

Если Project Terminal уже активен:

```text
Скопировать и открыть Terminal
```

только открывает/focus существующий terminal.

Если terminal ещё не запущен, то же пользовательское действие является явным
разрешением запустить стандартную shell через существующий trusted Project
Terminal service.

Не создавать второй PTY endpoint или task-specific terminal process.

### 5. Никакой автоматической вставки

После открытия terminal Toudocu не должен:

* писать handoff через WebSocket PTY input;
* симулировать Ctrl+V/Cmd+V;
* использовать bracketed paste;
* эмулировать keyboard events;
* автоматически добавлять Enter.

Это остаётся сознательным действием пользователя.

### 6. Никакого автоматического запуска агента

Toudocu не выполняет:

```text
codex
claude
opencode
```

и не пытается угадать, какой CLI нужен пользователю.

Пользователь сам запускает нужный development environment.

### 7. Independence

Agent Session и Project Terminal остаются независимыми.

Допустимый сценарий:

```text
Agent Console: TASK-A
Project Terminal: пользователь вручную работает с TASK-B
```

Toudocu не считает terminal session агентом TASK-B и не синхронизирует его
lifecycle с Task Action state.

### 8. Lazy assets

Интеграция не должна приводить к eager loading xterm в обычной task page.

Terminal assets загружаются только при фактическом открытии Project Terminal по
существующему lazy-loading contract.

<!-- toudocu:section acceptance-criteria -->

## Критерии приёмки

* [ ] `AC-01` Handoff-capable task actions предоставляют
  `Скопировать и открыть Terminal`.
* [ ] `AC-02` Используется существующий `TaskActionHandoff`; terminal layer не
  содержит prompt/context builder.
* [ ] `AC-03` При успешном clipboard write Project Terminal открывается или
  получает focus.
* [ ] `AC-04` Clipboard failure отображает handoff вручную и не выдаётся за
  успешное копирование.
* [ ] `AC-05` Handoff никогда автоматически не передаётся в PTY stdin и не
  вставляется synthetic keyboard action.
* [ ] `AC-06` Toudocu не запускает AI CLI и не определяет provider для Project
  Terminal.
* [ ] `AC-07` Используется один существующий Project Terminal lifecycle без
  task-specific PTY и без связи с Agent Session.
* [ ] `AC-08` Agent Session и Project Terminal продолжают работать независимо.
* [ ] `AC-09` xterm assets остаются lazy и не загружаются до открытия Terminal.
* [ ] `AC-10` `start-work` через этот workflow изменяет task state ровно один
  раз через общий Task Action coordinator.

<!-- toudocu:section plan -->

## План

1. Добавить frontend action `Copy & open terminal`.
2. Использовать существующий handoff result.
3. Подключить clipboard success/failure UX.
4. Добавить публичное frontend действие open/focus Project Terminal.
5. Переиспользовать существующий terminal start lifecycle.
6. Проверить отсутствие PTY writes и AI CLI launch.
7. Проверить независимость Agent Session и Project Terminal.
8. Проверить lazy loading terminal assets.

<!-- toudocu:section verification -->

## Проверка

* `AC-01` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff terminal'`
* `AC-02` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff terminal'`
* `AC-03` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff terminal'`
* `AC-04` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task handoff terminal'`
* `AC-05` → `go test ./internal/app -run 'TestProjectTerminalTaskHandoffIsolation' && make browser-test`
* `AC-06` → `go test ./internal/app -run 'TestProjectTerminalTaskHandoffIsolation' && make browser-test`
* `AC-07` → `go test ./internal/app -run 'TestProjectTerminalIndependent'`
* `AC-08` → `go test ./internal/app -run 'TestProjectTerminalIndependent'`
* `AC-09` → `cd web && node --test --test-name-pattern='manifest separates static and serve assets' tests/generated-contract.test.mjs`
* `AC-10` → `go test ./internal/app -run 'TestTaskActionHandoffStartWork'`
* `QUALITY` → `make check`
* `ALL` → `make check && make browser-test`
* `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->

## Влияние на документацию

Обновить:

* `MOD-AGENT-CONSOLE`;
* ADR Project Terminal;
* Agent Console contract;
* local workflow;
* agent workflows;
* `UC-AGENT-CONSOLE-01`.

Документация должна прямо фиксировать:

```text
Project Terminal = обычная пользовательская shell
Task handoff = переносимый текст
Copy & open = UI convenience

Copy & open ≠ auto-paste
Copy & open ≠ agent execution
Project Terminal ≠ AgentProvider
```
