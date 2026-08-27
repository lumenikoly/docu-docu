<!-- toudocu
id: TASK-AGENT-002
status: done
taskType: documentation
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
updated: 2026-08-25
-->

# TASK-AGENT-002: Зафиксировать контракт интеграции AI-агента

<!-- toudocu:section result -->
## Результат

Документация Toudocu однозначно фиксирует границу между Toudocu и coding
agent: Codex app-server является structured transport, Agent View отвечает за
conversation/lifecycle, Command Output показывает structured command execution,
а интерактивный PTY разрешён только как fallback Terminal Mode.

<!-- toudocu:section scope -->
## Область изменения

- `docs/architecture/`;
- `docs/decisions/`;
- `docs/modules/`;
- `docs/use-cases/`;
- `docs/flows/`;
- `docs/guides/`;
- `docs/reference/`;
- `docs/contracts/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- Go implementation;
- browser implementation;
- запуск Codex;
- изменение существующего Agent Feedback transport.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` ADR фиксирует Codex app-server как основной structured transport,
  Command Output как read-only projection и PTY только как fallback.
- [x] `AC-02` Trust boundary фиксирует loopback-only agent execution,
  user-local Full access preference и запрет repository-controlled executable,
  argv и access policy.
- [x] `AC-03` Модуль, use case и flow описывают Start work, conversation,
  steering, interrupt, approvals, verification и Agent Feedback без создания
  второго source of truth.
- [x] `AC-04` Документация сохраняет существующие границы React islands,
  `PageBootstrap v1`, MOD-SITE, Changes и MOD-AGENT-FEEDBACK.

<!-- toudocu:section plan -->
## План

1. Добавить ADR интеграции coding agent.
2. Добавить модуль Agent Console, пользовательский сценарий и flow.
3. Обновить frontend runtime и trust boundaries.
4. Обновить local workflow, work items и справочник возможностей.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `rg -n 'app-server|Command Output|PTY' docs/decisions docs/contracts`
- `AC-02` → `rg -n 'loopback|Full access|executable|argv' docs/architecture docs/contracts`
- `AC-03` → `rg -n 'Start work|steering|interrupt|verification|Agent Feedback' docs/modules docs/use-cases docs/flows`
- `AC-04` → `rg -n 'PageBootstrap v1|MOD-SITE|MOD-AGENT-FEEDBACK|Changes' docs/architecture docs/modules docs/use-cases docs/flows`
- `ALL` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Создаются и обновляются архитектурные, модульные и пользовательские документы,
которые становятся контрактом для последующих дочерних задач.
