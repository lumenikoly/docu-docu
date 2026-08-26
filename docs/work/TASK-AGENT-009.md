<!-- toudocu
id: TASK-AGENT-009
status: ready
taskType: feature
priority: medium
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-006
updated: 2026-08-25
-->

# TASK-AGENT-009: Добавить Terminal Mode как PTY fallback

<!-- toudocu:section result -->
## Результат

Если установленный Codex не поддерживает совместимый structured app-server,
пользователь может запустить тот же agent workflow в явно обозначенном
Terminal Mode через настоящий PTY, не открывая внешний терминал.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Несовместимый structured Codex полностью блокирует встроенный agent workflow.

<!-- toudocu:section after -->
### Станет

Agent Console предлагает Terminal Mode fallback. Только этот режим загружает
xterm и запускает Codex TUI через PTY; structured session и PTY session для
одного агента одновременно не существуют.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_pty.go`;
- `internal/app/agent_pty_unix.go`;
- `internal/app/agent_pty_windows.go`;
- `web/src/features/`;
- `web/package.json`;
- frontend asset packaging и tests.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- generic shell;
- arbitrary terminal command;
- использование PTY при рабочем structured Codex;
- второй параллельный Codex process;
- provider plugin system.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Terminal Mode предлагается только при недоступной или
  несовместимой structured integration.
- [ ] `AC-02` PTY поддерживает interactive input/output, resize, interrupt и
  корректный stop на поддерживаемых платформах.
- [ ] `AC-03` `Full access` в Terminal Mode формируется доверенным provider как
  семантический `codex --yolo`; browser не передаёт literal argv.
- [ ] `AC-04` xterm/PTY assets не загружаются в structured mode и не входят в
  static runtime.
- [ ] `AC-05` Structured и Terminal Mode session для одного `serve` не могут
  работать одновременно.

<!-- toudocu:section plan -->
## План

1. Добавить platform PTY abstraction.
2. Добавить Terminal Mode frontend.
3. Связать fallback с существующей Agent Session lifecycle.
4. Добавить Full access mapping.
5. Проверить resource isolation и cleanup.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestAgentTerminalFallback' && make browser-test`
- `AC-02` → `go test ./internal/app -run 'TestAgentPTY'`
- `AC-03` → `go test ./internal/app -run 'TestAgentPTYFullAccess'`
- `AC-04` → `make web-check && make browser-test`
- `AC-05` → `go test ./internal/app -run 'TestAgentTransportExclusivity'`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Local workflow и Agent Console документация отличают structured Agent View +
Command Output от отдельного интерактивного Terminal Mode fallback.
