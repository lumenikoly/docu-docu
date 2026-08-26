<!-- toudocu
id: TASK-AGENT-011
status: done
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-005
updated: 2026-08-26
-->

# TASK-AGENT-011: Завершить transport Agent Console для frontend

<!-- toudocu:section result -->
## Результат

Основной loopback `serve` предоставляет Agent Console все нормализованные
сведения и действия, необходимые интерфейсу, не раскрывая протокол поставщика и
не передавая браузеру право выбирать процесс, рабочий каталог или политику
доступа.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Реализованная основа передаёт события и базовый снимок сессии, но теряет часть
метаданных команд и не описывает разрыв replay, актуальные approvals, отмену
очереди, настройку следующей сессии и подготовку Toudocu skill.

<!-- toudocu:section after -->
### Станет

Версионированный HTTP- и WebSocket-контракт передаёт полную нормализованную
проекцию Agent Console и сериализует все изменяющие действия на сервере.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/`;
- `internal/site/bootstrap.go`;
- `docs/contracts/`;
- `docs/architecture/`;
- Go tests.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- React Agent Console;
- Task status mutation и prepared task actions;
- verification и Agent Feedback;
- PTY, generic terminal и provider-specific browser protocol;
- автоматическая замена конфликтного или unsafe Toudocu skill.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Setup projection сообщает доступные providers, выбранный provider,
  preference следующей сессии, состояние Toudocu skill и готовые безопасные
  diagnostics; browser не выводит эти сведения из DOM или provider-specific
  полей.
- [x] `AC-02` Первое включение `full-access` требует подтверждения для
  canonical repository root. Preference хранится в user-local server state, а
  активная сессия неизменно сообщает отдельно requested preset и доказанный
  effective access.
- [x] `AC-03` Командные события одного стабильного item ID передают command,
  рабочий каталог при необходимости, running/completed state, exit code,
  duration, approval state и явное truncation с доступным хвостом.
- [x] `AC-04` Каждый pending envelope имеет стабильный ID, текст, состояние,
  нормализованную причину и FIFO position. Structured cancel изменяет очередь
  только на сервере, а `not-sent` никогда не отправляется автоматически.
- [x] `AC-05` Снимок сессии содержит только актуальные pending approvals со
  стабильными IDs, а повторный ответ по одному ID идемпотентен.
- [x] `AC-06` Stop без разрешения удалить pending возвращает typed conflict с
  отдельным числом `queued` и `not-sent`. Подтверждённый stop удаляет все pending
  на момент сериализованной обработки, а после начала stop новые сообщения
  отклоняются как `session_stopping`.
- [x] `AC-07` Reconnect явно сообщает нормализованный bounded replay gap и
  границы пропуска, достаточные frontend для однократного отображения. Gap не
  меняет статус сессии на failed.
- [x] `AC-08` Read-only skill status выполняется автоматически, а install или
  update запускается только отдельным подтверждённым действием через
  существующий Go installer. `outdated` без compatibility metadata требует
  update; `modified`, `unmanaged`, `newer-than-bundle` и `unsafe-path` не
  заменяются автоматически.
- [x] `AC-09` OpenAPI точно описывает реализованные endpoints, action headers,
  ответы, snapshots и WebSocket messages; API по-прежнему доступен только в
  canonical loopback `serve`.
- [x] `AC-10` Start отклоняется для любой failed session, пока подтверждённый
  cleanup не удалит прежнюю сессию и её `not-sent`; после cleanup новая сессия
  создаёт новый provider thread.

<!-- toudocu:section plan -->
## План

1. Дополнить нормализованные setup, session, command и approval projections.
2. Добавить отмену pending envelope и typed stop conflict.
3. Добавить явную семантику replay gap.
4. Подключить preference и подтверждённые операции существующего skill installer.
5. Обновить OpenAPI, сопроводительный контракт и тесты границы доверия.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app ./internal/site -run 'TestAgentConsoleSetup|TestAgentConsoleBootstrap'`
- `AC-02` → `go test ./internal/app -run 'TestAgentConsoleAccessPreference|TestAgentEffectiveAccessSummary'`
- `AC-03` → `go test ./internal/app -run 'TestAgentConsoleCommandProjection|TestAgentConsoleBufferLimit'`
- `AC-04` → `go test ./internal/app -run 'TestAgentPendingCancel|TestAgentSessionQueue'`
- `AC-05` → `go test ./internal/app -run 'TestAgentApprovalSnapshot|TestAgentApprovalIdempotency'`
- `AC-06` → `go test ./internal/app -run 'TestAgentSessionStopConflict|TestAgentSessionStopping'`
- `AC-07` → `go test ./internal/app -run 'TestAgentConsoleReplayGap|TestAgentConsoleReconnect'`
- `AC-08` → `go test ./internal/app -run 'TestAgentConsoleSkillSetup'`
- `AC-09` → `go test ./internal/app -run 'TestAgentConsoleLoopback|TestAgentConsoleOrigin'`
- `AC-10` → `go test ./internal/app -run 'TestAgentFailedSessionRequiresCleanup'`
- `ALL` → `go test ./...`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Agent Console contract и trust-boundary документация описывают расширенную
проекцию setup/session, новые structured actions и сохранённую loopback boundary.
