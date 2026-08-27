<!-- toudocu
id: TASK-AGENT-012
status: in-progress
taskType: feature
priority: high
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-27
-->

# TASK-AGENT-012: Выделить Project Terminal в самостоятельную возможность

<!-- toudocu:section result -->
## Результат

Основной `serve` на loopback-адресе даёт разработчику один обычный
Project Terminal в корне репозитория. Structured Agent Console и терминал
работают независимо и не имитируют семантику друг друга.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Режим терминала запускал только Codex TUI и делил с Agent Session
взаимоисключающий жизненный цикл.

<!-- toudocu:section after -->
### Станет

Project Terminal по явному действию запускает стандартную командную оболочку платформы в
корне репозитория. Все последующие CLI и TUI пользователь запускает сам.
Toudocu не интерпретирует содержимое PTY, а терминал может
работать одновременно с Agent Session.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_pty*.go` и транспорт Agent Console;
- `web/src/features/agent-console/`, каталоги текстов и тесты интерфейса;
- контракт, архитектура, ADR и руководство локальной работы.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- несколько терминалов;
- сохранение и восстановление PTY после остановки `serve`;
- автоматическая привязка к задаче или разбор TUI в `AgentEvent`;
- настраиваемая команда старта из файлов репозитория.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Project Terminal доступен только в canonical loopback
  `serve` и по явному действию запускает стандартную командную оболочку в корне
  репозитория.
- [x] `AC-02` Ввод, вывод, изменение размера, прерывание, выход и остановка
  терминала остаются PTY-операциями без `AgentEvent`, согласований и привязки к задаче.
- [x] `AC-03` Agent Session и Project Terminal могут работать
  одновременно; их остановка и прерывание не влияют друг на друга.
- [x] `AC-04` В терминале есть одно действие запуска командной оболочки;
  отдельного действия для Codex нет, а браузер не передаёт исполняемый файл,
  аргументы, рабочий каталог и окружение.
- [x] `AC-05` ресурсы xterm остаются отдельным отложенно загружаемым пакетом и не
  попадают в статический портал.
- [x] `AC-06` После запуска PTY браузер передаёт фактический размер xterm;
  полноэкранный альтернативный буфер отображается, а не остаётся пустым.

<!-- toudocu:section plan -->
## План

1. Оставить в Project Terminal только запуск командной оболочки.
2. Убрать общую блокировку жизненных циклов с Agent Session.
3. Обновить действия, подписи и запасной путь в браузере.
4. Проверить серверную часть, интерфейс и документацию.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestProjectTerminalShell|TestAgentConsoleRuntimeIsolation'`
- `AC-02` → `go test ./internal/app -run 'TestAgentPTY'`
- `AC-03` → `go test ./internal/app -run 'TestProjectTerminalIndependent'`
- `AC-04` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'Portal and workspaces share visual language'`
- `AC-05` → `cd web && node --test --test-name-pattern='manifest separates static and serve assets' tests/generated-contract.test.mjs && npx playwright test tests/browser/runtime.spec.ts --grep 'Portal and workspaces share visual language'`
- `AC-06` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'Portal and workspaces share visual language'`
- `QUALITY` → `make check`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

Общие проверки качества и полного набора пока блокирует существующее рассогласование
`generated-contract`: тест ожидает точную строку `islandHost.unmountAll()`,
хотя версия файла из `HEAD` уже содержит `islandHost.unmountAll(["agent-console"])`.

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Обновить контракт Agent Console, ADR-009, ADR-010, границы доверия,
архитектурный обзор, модуль, сценарий и руководство локальной работы: Project
Terminal — обычный PTY, содержимое которого Toudocu не интерпретирует, а не запасной `AgentProvider`.
