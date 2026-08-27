<!-- toudocu
id: TASK-AGENT-008
status: done
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-007
updated: 2026-08-25
-->

# TASK-AGENT-008: Интегрировать verification и Agent Feedback

<!-- toudocu:section result -->
## Результат

Пользователь завершает основной task/agent feedback loop внутри Toudocu:
явно запускает task verification, видит отдельный Verification Output,
передаёт failed verification активному агенту и обрабатывает существующие
discussion deliveries без copy/paste.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

`task verify --run` запускается из CLI, а Agent Feedback требует отдельного
`$toudocu feedback` в интерфейсе внешнего агента.

<!-- toudocu:section after -->
### Станет

Agent Console запускает verification через существующий Go service после
явного подтверждения, показывает его отдельно от agent Command Output и
позволяет явно передать failure активному Codex thread.

Pending Agent Feedback обрабатывается той же существующей очередью через
prepared `$toudocu feedback`.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/task_verify.go`;
- `internal/app/agent_prompts.go`;
- `web/src/features/`;
- `web/src/features/discussions/`;
- `web/src/features/changes/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- автоматический запуск verification;
- автоматическое исправление failure;
- изменение FIFO AgentDelivery;
- второй feedback protocol;
- использование agent command events как verification result.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Verify требует отдельного user confirmation и вызывает
  существующую Go-семантику `task verify --run`, а не просит AI организовать
  verification.
- [x] `AC-02` Verification Output визуально и в state-model отделён от Codex
  Command Output.
- [x] `AC-03` Failed verification можно явно передать active agent новым turn,
  но исправление не запускается автоматически.
- [x] `AC-04` Process feedback отправляет `$toudocu feedback`, после чего skill
  использует существующие `agent next/respond`; Agent Feedback semantics не
  меняются.
- [x] `AC-05` Refresh diff и contextual Ask над command/output используют
  structured Agent Composer и существующие permission boundaries.
- [x] `AC-06` Пользователь может выполнить `Copy command`, `Ask about command`,
  `Copy selection` и `Ask about selection`; эти действия не отправляют данные в
  stdin процесса и передают вопросы только через Agent Composer.

<!-- toudocu:section plan -->
## План

1. Подключить verification service к Agent Console.
2. Добавить Verification Output.
3. Добавить failure-to-agent action.
4. Связать active Agent Session с существующим Agent Feedback.
5. Добавить remaining contextual Toudocu actions.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestAgentConsoleVerificationAuthorization|TestAgentConsoleVerificationRun|TestTaskVerify'`
- `AC-02` → `make web-check && make browser-test`
- `AC-03` → `go test ./internal/app -run 'TestAgentConsoleVerificationFailureToAgent' && make browser-test`
- `AC-04` → `go test ./internal/app -run 'TestAgentFeedback' && make browser-test`
- `AC-05` → `make browser-test`
- `AC-06` → `make web-check && make browser-test`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Agent Feedback, task verification, Changes и local workflow описывают единый
интерактивный цикл человека, Toudocu и coding agent.
