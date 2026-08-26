<!-- toudocu
id: TASK-AGENT-001
status: draft
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
updated: 2026-08-25
-->

# TASK-AGENT-001: Интегрированная работа с AI-агентом в Toudocu

<!-- toudocu:section result -->
## Результат

Пользователь работает с AI coding agent непосредственно из canonical
`toudocu serve`: берёт готовую задачу в работу, общается с агентом, направляет
активную работу, запускает clarify и verification, обрабатывает Agent Feedback
и видит фактически выполняемые команды без переключения в отдельный терминал.

Codex получает structured integration через app-server. Command Output является
read-only проекцией structured execution events. Интерактивный PTY используется
только как fallback Terminal Mode.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Task Workspace остаётся read-only. Для работы с coding agent пользователь
переходит в отдельный терминал и вручную переносит Toudocu prompts, task ID,
feedback и результаты проверок между приложениями.

<!-- toudocu:section after -->
### Станет

В canonical loopback `serve` пользователь может одним действием начать Ready
задачу, запустить Codex, продолжать structured conversation, использовать
Ask/Clarify/Verify/Feedback и видеть Agent View вместе с Command Output.

Agent Session сохраняется при навигации между Tasks, Documentation, Changes и
Editor. `Stop response` прерывает текущий turn, а `Stop agent` завершает всю
сессию.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/`;
- `internal/site/`;
- `web/src/`;
- `web/package.json`;
- `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- Toudocu Studio;
- собственный LLM API и хранение API keys;
- multi-agent orchestration;
- несколько параллельных Agent Sessions;
- generic shell;
- repository-controlled agent executable или arguments;
- перенос Portal на SPA или React Router;
- сохранение hidden reasoning или полного AI transcript.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Все непосредственные дочерние задачи завершены, а обычная Ready
  задача запускает интегрированный agent workflow без ручного copy/paste.
- [ ] `AC-02` Codex использует structured app-server transport; Agent View и
  Command Output отображают одну сессию без второго Codex process.
- [ ] `AC-03` Пользователь может писать агенту во время и после turn, прерывать
  текущий turn, продолжать thread и отдельно завершать Agent Session.
- [ ] `AC-04` Verification, Agent Feedback и PTY fallback сохраняют существующие
  security и domain boundaries Toudocu; static, translations и LAN serve не
  получают agent execution.

<!-- toudocu:section plan -->
## План

1. Зафиксировать архитектурный и security contract.
2. Реализовать structured Codex provider и Agent Session.
3. Добавить loopback-only serve transport.
4. Реализовать Agent Console и Command Output.
5. Связать Tasks, verification и Agent Feedback с Agent Session.
6. Добавить PTY fallback для несовместимого structured provider.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go run ./cmd/toudocu task tree TASK-AGENT-001 ./docs --repository-root . && make browser-test`
- `AC-02` → `make web-check && make browser-test`
- `AC-03` → `make browser-test`
- `AC-04` → `make check && make browser-test`
- `ALL` → `make check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

После завершения дерева архитектура frontend/runtime, локальный workflow,
описание `serve`, Task Workspace, Agent Feedback и справочник возможностей
описывают интегрированную работу с внешним coding agent.
