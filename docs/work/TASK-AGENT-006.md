<!-- toudocu
id: TASK-AGENT-006
status: ready
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-005
updated: 2026-08-26
-->

# TASK-AGENT-006: Добавить structured Agent Console

<!-- toudocu:section result -->
## Результат

Пользователь видит в `serve` Agent View и Command Output одной structured Agent
Session, пишет агенту сообщения, направляет активный turn, обрабатывает
approvals и может отдельно остановить response или всю session. Frontend
работает только с общими `AgentEvent`, capabilities и lifecycle actions.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Toudocu не показывает выполняющего работу coding agent и не предоставляет
интерфейс продолжения его conversation.

<!-- toudocu:section after -->
### Станет

Activated React island Agent Console показывает Agent View и Command Output
одновременно на desktop и как две вкладки на узком экране. Conversation
управляется structured protocol, а Command Output остаётся read-only.

<!-- toudocu:section scope -->
## Область изменения

- `web/src/features/`;
- `web/src/core/react/`;
- `internal/site/templates/`;
- `internal/site/i18n/`;
- frontend и browser tests.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- Task status mutation;
- verification execution;
- Agent Feedback delivery;
- PTY/xterm;
- generic terminal;
- разбор TUI конкретного provider;
- provider-specific protocol или настройки запуска во frontend.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Agent View и Command Output одновременно работают на desktop, а
  на узком экране доступны через accessible tabs.
- [ ] `AC-02` Composer отправляет новый turn при idle и steering при active
  turn; queued fallback явно виден пользователю.
- [ ] `AC-03` Approvals, `Stop response` и `Stop agent` используют structured
  controls и не эмулируют terminal input.
- [ ] `AC-04` Command cards связаны с live Command Output одного item; output
  ограничивается и корректно показывает truncation.
- [ ] `AC-05` Agent Session продолжает работать после soft navigation/reload, а
  Agent Console reconnects через существующий IslandHost lifecycle.
- [ ] `AC-06` Command Output не загружает xterm и не предоставляет shell input.
- [ ] `AC-07` Перед первым включением `Full access` интерфейс объясняет, что это
  запрос максимального режима конкретного provider, запрашивает подтверждение
  для репозитория и отдельно показывает requested preset и фактический доступ
  без обещания обойти managed или explicit deny policies.
- [ ] `AC-08` Agent View показывает live-карточку изменённых файлов и действие
  `Open Changes`; события агента не подменяют authoritative Changes Toudocu.
- [ ] `AC-09` Command Output показывает рабочий каталог при необходимости,
  running/completed state, exit code, duration и approval state, безопасно
  удаляет ANSI control sequences и сохраняет только ограниченное форматирование.
- [ ] `AC-10` Интерфейс автоматически читает состояние Toudocu skill, но
  запускает install или update только отдельным подтверждённым действием через
  существующий Go installer. `outdated` без compatibility contract требует
  обновления, а конфликтные или unsafe состояния не заменяются автоматически.
- [ ] `AC-11` Выбор provider, Agent View, Command Output, approvals, composer и
  lifecycle controls используют только нормализованные события и capabilities;
  добавление нового structured provider не требует provider-specific frontend.

<!-- toudocu:section plan -->
## План

1. Реализовать activated Agent Console island.
2. Добавить Agent View и composer.
3. Добавить Command Output и command correlation.
4. Добавить approvals и lifecycle controls.
5. Добавить reconnect и responsive presentation.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `make web-check && make browser-test`
- `AC-02` → `make web-check && make browser-test`
- `AC-03` → `make browser-test`
- `AC-04` → `make web-check && make browser-test`
- `AC-05` → `make browser-test`
- `AC-06` → `make web-check && make browser-test`
- `AC-07` → `make web-check && make browser-test`
- `AC-08` → `make web-check && make browser-test`
- `AC-09` → `make web-check && make browser-test`
- `AC-10` → `make web-check && make browser-test`
- `AC-11` → `make web-check && make browser-test`
- `ALL` → `make web-check && make browser-test && go test ./...`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Frontend runtime и local workflow описывают Agent View, Command Output,
structured controls и lifecycle island.
