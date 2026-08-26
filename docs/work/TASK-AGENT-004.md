<!-- toudocu
id: TASK-AGENT-004
status: done
taskType: maintenance
priority: high
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-003
updated: 2026-08-26
-->

# TASK-AGENT-004: Добавить общий слой Agent Session и Codex adapter

<!-- toudocu:section result -->
## Результат

Toudocu управляет одной активной Agent Session через общий слой, который не
зависит от модели разрешений конкретного поставщика. Слой задаёт жизненный цикл
сессии, последовательность операций, ограниченную очередь, показ
неотправленных сообщений, локальные пользовательские предпочтения и различие
между остановкой текущего ответа и всей сессии.

Codex остаётся первым adapter этого слоя. Только он преобразует режим запуска в
поля Codex app-server, хранит полную политику sandbox и восстанавливает её после
временно ограниченного turn.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_provider.go`;
- `internal/app/agent_codex.go`;
- `internal/app/agent_codex_test.go`;
- `internal/app/agent_session.go`;
- `internal/app/agent_preferences.go`;
- `internal/app/agent_session_test.go`;
- `internal/app/command_process_unix.go`;
- `internal/app/command_process_windows.go`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- browser UI и HTTP/WebSocket transport;
- prepared prompts и семантика Ask, Clarify, verification или Agent Feedback;
- действия Task Workspace и изменение статуса задачи;
- parallel agents и несколько одновременных Agent Sessions;
- persisted conversation или recovery history;
- repository-controlled agent configuration;
- универсальная модель sandbox или permissions конкретного provider;
- автоматическая установка или обновление Toudocu skill.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Один manager имеет не более одной активной Agent Session; её
  необязательный `taskID`, launch snapshot и provider остаются неизменными до
  завершения. Активная сессия одной задачи блокирует запуск другой.
- [x] `AC-02` Manager сериализует все изменяющие состояние операции. Сообщение
  в свободной сессии начинает turn; во время turn оно становится steering либо
  попадает в FIFO для отдельного следующего turn.
- [x] `AC-03` FIFO содержит не более 32 сообщений размером до 65 536 UTF-8
  bytes каждое. Переполнение отклоняет новое сообщение и не удаляет уже
  принятые данные.
- [x] `AC-04` Определённый отказ steering ставит сообщение в FIFO, а
  неопределённый результат доставки сохраняет его как `not-sent` без
  автоматической повторной отправки. Queued messages выполняются отдельными
  turns строго по порядку.
- [x] `AC-05` `Stop response` прерывает только текущий turn и сохраняет FIFO.
  После подтверждённого interrupt очередь продолжает работу отдельными turns.
  Неподтверждённый interrupt переводит сессию в `failed`, сохраняет pending как
  `not-sent` и запускает принудительный cleanup provider.
- [x] `AC-06` `Stop agent`, shutdown, startup rollback и аварийный cleanup
  используют один идемпотентный путь. Provider подтверждает прекращение
  принадлежащего ему исполнения; иначе сессия завершается ошибкой
  `provider_stop_unconfirmed`, и новая сессия не запускается.
- [x] `AC-07` Обычный stop с pending messages требует явного
  `discardPending=true`; forced cleanup подтверждения не требует. После crash
  сообщения `not-sent` доступны в failed session до её удаления; создание новой
  session очищает их без отдельного recovery store.
- [x] `AC-08` Пользователь выбирает только launch presets `default` и
  `full-access`. Предпочтение хранится вне репозитория отдельно для его
  canonical root; отсутствующий файл означает `default`, а весь повреждённый
  или несовместимый объект игнорируется без автоматической перезаписи.
- [x] `AC-09` Общий контракт разделяет возможности provider и фактические
  возможности session. Filesystem-read-only action разрешён только при session
  capability `ReadOnlyTurns`, которая гарантирует sandbox provider, но не
  отсутствие side effects внешних tools; prompt не заменяет эту гарантию.
- [x] `AC-10` Общий слой хранит только requested launch preset и
  `EffectiveAccessSummary {known, unrestricted}`. Значение `unrestricted=true`
  допустимо лишь когда adapter доказал отсутствие релевантных ограничений.
- [x] `AC-11` Codex `default` не передаёт access overrides, а `full-access`
  передаёт `sandbox=dangerFullAccess` и `approvalPolicy=never`. Requested и
  effective access не смешиваются.
- [x] `AC-12` Codex adapter хранит полную типизированную базовую sandbox policy.
  После filesystem-read-only turn его следующий `turn/start` передаёт точный
  неизменный базовый snapshot вместе с обычным сообщением. Если это невозможно,
  обычный turn отклоняется, а сессия объявляет `ReadOnlyTurns=false`.
- [x] `AC-13` Остановка Codex закрывает transport, ждёт внутренний grace period
  и при необходимости завершает всё дерево процессов через минимальную общую
  platform primitive.

<!-- toudocu:section plan -->
## План

1. Разделить общий контракт session, launch preset, turn policy и capabilities.
2. Реализовать сериализованный AgentSessionManager и bounded FIFO.
3. Добавить безопасное user-local хранилище preferences по canonical root.
4. Уточнить Codex mapping, effective snapshot и восстановление sandbox.
5. Вынести минимальную platform primitive завершения дерева процессов.
6. Проверить общий lifecycle и Codex-specific поведение fake app-server.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestAgentSessionLifecycle|TestAgentSessionTaskBinding'`
- `AC-02` → `go test ./internal/app -run 'TestAgentSessionMessages|TestAgentSessionOrdering'`
- `AC-03` → `go test ./internal/app -run 'TestAgentSessionQueueLimits'`
- `AC-04` → `go test ./internal/app -run 'TestAgentSessionSteeringQueue|TestAgentDeliveryUncertain'`
- `AC-05` → `go test ./internal/app -run 'TestAgentTurnInterrupt|TestAgentInterruptUnconfirmed'`
- `AC-06` → `go test ./internal/app -run 'TestAgentSessionStop|TestAgentSessionShutdown|TestAgentStopUnconfirmed'`
- `AC-07` → `go test ./internal/app -run 'TestAgentPendingDiscard|TestAgentFailedMessages'`
- `AC-08` → `go test ./internal/app -run 'TestAgentPreferenceStore|TestAgentInvalidPreferences'`
- `AC-09` → `go test ./internal/app -run 'TestAgentCapabilities|TestAgentReadOnlyCapability'`
- `AC-10` → `go test ./internal/app -run 'TestAgentEffectiveAccessSummary'`
- `AC-11` → `go test ./internal/app -run 'TestCodexDefaults|TestCodexFullAccess'`
- `AC-12` → `go test ./internal/app -run 'TestCodexReadOnlyTurn|TestCodexSandboxRestore'`
- `AC-13` → `go test ./internal/app -run 'TestCodexSessionStop|TestAgentProcessTree'`
- `ALL` → `go test ./...`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Контракт Agent Console и модуль Agent Console описывают общий session lifecycle,
capability-driven providers и изоляцию provider-specific access policy.
