<!-- toudocu
id: TASK-AGENT-007
status: done
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-006
updated: 2026-08-26
-->

# TASK-AGENT-007: Интегрировать Task Workspace с Agent Console

<!-- toudocu:section result -->
## Результат

Пользователь может из Task Workspace одним действием взять исполнимую Ready
задачу в работу и начать Codex session, а также использовать Ask, Clarify,
Explain blocker/problems и Ask what to do next без ручного ввода Toudocu
prompts.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Board/List/Tree показывают состояние задач только для чтения. Начало работы и
обращение к агенту требуют отдельного терминала.

<!-- toudocu:section after -->
### Станет

Canonical loopback `serve` добавляет actions по состоянию задачи. `Start work`
повторно проверяет readiness и digest, атомарно меняет `ready → in-progress` и
отправляет централизованный prepared action в Agent Session.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/task_ready.go`;
- `internal/app/agent_prompts.go`;
- `internal/app/agent_task.go`;
- `web/src/features/task-workspace/`;
- `web/src/features/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- drag-and-drop статусов;
- Jira-подобный workflow;
- автоматический выбор следующей задачи;
- автоматическое изменение task contract;
- multi-agent assignment.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` `StartTask` принимает только актуальную `Ready` +
  `readyForWork=true` задачу, сверяет digest/dependencies и атомарно меняет
  статус на `in-progress`.
- [x] `AC-02` Интерфейс показывает переданные сервером состояния поставщика и
  навыка Toudocu, а при конфликте — готовую CLI-инструкцию. Если перед запуском
  требуется подготовка, пользователь подтверждает её отдельным действием.
  Статус задачи меняется на `in-progress` только после успешной подготовки, а
  затем пользователь запускает работу одним действием `Start work`.
- [x] `AC-03` Ready/In progress/Waiting/Needs attention/Draft получают только
  допустимые для состояния actions.
- [x] `AC-04` Ask, Explain blocker/problems и Ask what to do next запускаются
  read-only, причём ограничение обеспечивается capability provider, а не только
  prompt; Clarify использует существующий `$toudocu clarify`.
- [x] `AC-05` Prepared actions формируются централизованным registry и не
  копируют полный TaskContext в prompt.

<!-- toudocu:section plan -->
## План

1. Добавить серверное действие `StartTask`.
2. Добавить реестр действий задачи.
3. Подключить состояние и подтверждённую подготовку навыка Toudocu.
4. Подключить действия к существующему Task Workspace.
5. Реализовать подготовленные запросы и вопросы без права записи.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestStartTask|TestStartTaskReadiness|TestStartTaskDigest'`
- `AC-02` → `make browser-test`
- `AC-03` → `make web-check && make browser-test`
- `AC-04` → `go test ./internal/app -run 'TestAgentTaskActions|TestAgentSessionQueuedReadOnlyPolicy' && make browser-test`
- `AC-05` → `go test ./internal/app -run 'TestAgentPromptRegistry'`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Work-item и local-workflow документация описывает интерактивные serve-only
actions, а static Task Workspace сохраняется read-only.
