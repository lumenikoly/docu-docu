<!-- toudocu
id: TASK-AGENT-005
status: done
taskType: maintenance
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-004
updated: 2026-08-25
-->

# TASK-AGENT-005: Добавить безопасный serve transport Agent Console

<!-- toudocu:section result -->
## Результат

Canonical loopback `serve` предоставляет локальный API и WebSocket для Agent
Session. Browser получает только нормализованные session/events и разрешённые
actions, а Codex app-server остаётся скрыт за Go boundary.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/server.go`;
- `internal/app/agent_console_http.go`;
- `internal/app/agent_console_http_test.go`;
- `internal/app/agent_events.go`;
- `internal/app/agent_provider.go`;
- `internal/app/agent_session.go`;
- `internal/app/types.go`;
- `internal/site/bootstrap.go`;
- `internal/site/bootstrap_test.go`;
- `docs/contracts/`;
- `docs/architecture/trust-boundaries.md`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- React Agent Console;
- PTY;
- arbitrary process execution;
- доступ из LAN;
- static или translation agent runtime.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Agent execution API существует только в canonical loopback
  `serve`; `0.0.0.0`, static и translation runtime его не получают.
- [x] `AC-02` `PageBootstrap v1` аддитивно получает capability/endpoint Agent
  Console, а browser не определяет permission по DOM или URL.
- [x] `AC-03` WebSocket передаёт `AgentEvent`, Command Output, session state,
  messages, steering, interrupt и approval responses без provider-specific
  protocol в browser.
- [x] `AC-04` API не принимает executable, argv, cwd или environment и проверяет
  same-origin/action boundary.
- [x] `AC-05` Ограниченные event/output buffers позволяют browser reconnect без
  бессрочного хранения transcript.

<!-- toudocu:section plan -->
## План

1. Добавить capability и endpoint в bootstrap.
2. Добавить session HTTP operations.
3. Добавить bidirectional WebSocket.
4. Добавить loopback/runtime guards.
5. Добавить reconnect buffers и cleanup.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestAgentConsoleLoopback|TestAgentConsoleRuntimeIsolation'`
- `AC-02` → `go test ./internal/site ./internal/app -run 'TestAgentConsoleBootstrap'`
- `AC-03` → `go test ./internal/app -run 'TestAgentConsoleWebSocket'`
- `AC-04` → `go test ./internal/app -run 'TestAgentConsoleRejectsProcessArguments|TestAgentConsoleOrigin'`
- `AC-05` → `go test ./internal/app -run 'TestAgentConsoleReconnect|TestAgentConsoleBufferLimit'`
- `ALL` → `go test ./...`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Serve, HTTP contract и trust-boundary документация получает фактические
capabilities, endpoints и loopback restrictions Agent Console.
