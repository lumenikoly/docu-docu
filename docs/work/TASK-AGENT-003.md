<!-- toudocu
id: TASK-AGENT-003
status: done
taskType: maintenance
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-002
updated: 2026-08-25
-->

# TASK-AGENT-003: Добавить structured AgentProvider и Codex adapter

<!-- toudocu:section result -->
## Результат

Go runtime предоставляет независимые от конкретного агента контракты
`AgentProvider`, `AgentEvent` и `AgentCapabilities`. Первый адаптер обнаруживает
установленный Codex, запускает один `codex app-server`, выполняет protocol
initialization, создаёт thread и turns и преобразует события Codex в общую
модель.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_provider.go`;
- `internal/app/agent_codex.go`;
- `internal/app/agent_events.go`;
- `internal/app/agent_codex_test.go`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- HTTP/WebSocket API;
- React UI;
- Task Workspace actions;
- PTY fallback;
- verification;
- Agent Feedback UI.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Codex обнаруживается без shell через доверенный executable lookup,
  а browser или repository не могут передать executable, argv или cwd.
- [x] `AC-02` Общий `AgentProvider` описывает lifecycle сессии и turn,
  сообщения, steering, interrupt, approvals, stop и capabilities без
  Codex-specific request или response types.
- [x] `AC-03` Сообщения агента, command execution, output, file changes,
  approvals и turn lifecycle преобразуются в стабильные `AgentEvent`.
- [x] `AC-04` Fake app-server позволяет полностью тестировать provider без сети,
  Codex account и OpenAI backend.
- [x] `AC-05` Общая модель настроек знает только provider, access preset,
  effective access и capabilities; модель, agent, variant и другие параметры
  запуска остаются provider-specific и не расширяют общий lifecycle interface.

<!-- toudocu:section plan -->
## План

1. Ввести общий AgentProvider contract, capabilities и normalized event model.
2. Реализовать Codex app-server stdio client.
3. Добавить lifecycle thread/turn и capability negotiation.
4. Добавить fake provider process и тесты normalization.
5. Отделить provider-specific launch preferences от общей access semantics.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestCodexProviderDetect|TestCodexProviderInvocation'`
- `AC-02` → `go test ./internal/app -run 'TestAgentProviderContract|TestCodexAppServerLifecycle|TestCodexProtocolCompatibility'`
- `AC-03` → `go test ./internal/app -run 'TestCodexAgentEvents'`
- `AC-04` → `go test ./internal/app -run 'TestFakeCodexAppServer'`
- `AC-05` → `go test ./internal/app -run 'TestAgentProviderPreferences|TestAgentCapabilities'`
- `ALL` → `go test ./...`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Архитектурное описание Agent Console и frontend runtime фиксирует общий
provider contract, нормализованные события и Codex как первый adapter.
