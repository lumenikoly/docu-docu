<!-- toudocu
id: MOD-AGENT-CONSOLE
status: in-progress
updated: 2026-08-28
-->

# MOD-AGENT-CONSOLE: Интегрированная работа с coding agent

Модуль управляет одной Agent Session в основном `serve` на loopback-адресе,
связывает её с задачами, проверкой и Agent Feedback, а также даёт доступ к
независимому Project Terminal. Общий контракт поставщика пока реализует только Codex;
другие структурированные интеграции остаются планом.

<!-- toudocu:section code-location -->
## Расположение в коде

- `internal/app/agent_provider.go`, `agent_events.go`, `agent_session.go` и
  `agent_codex.go` — общие контракты, события, жизненный цикл и Codex adapter;
- `internal/app/agent_console_http.go` и `agent_pty*.go` — локальный transport и PTY;
- `web/src/features/agent-console/` — Agent Console и Project Terminal в браузере.

<!-- toudocu:section boundaries -->
## Границы

Toudocu владеет жизненным циклом сессии и безопасной передачей данных браузеру.
Отдельный прикладной слой владеет действиями задачи и передаёт Agent Console
только готовую инструкцию. Поставщик владеет рассуждением, исследованием
репозитория, изменениями и выполнением инструментов. Модуль не вызывает LLM API, не хранит
ключи API и не реализует цикл работы агента.

<!-- toudocu:section business-rules -->
## Бизнес-правила

### BR-AGENT-CONSOLE-001: Одна сессия имеет один источник исполнения

Agent View и Command Output строятся из событий одной provider session. Для
Codex она соответствует одному process и одному thread; второй Codex process
для вывода команд не запускается. Общий контракт не требует, чтобы каждый
provider был дочерним процессом Toudocu.

### BR-AGENT-CONSOLE-002: Команды являются наблюдаемой проекцией

Command Output доступен только для чтения. Ввод, steering, interrupt, stop и
approvals проходят через Agent Composer и structured controls.

### BR-AGENT-CONSOLE-003: Состояние проекта не дублируется

Задача, verification contract, Changes и Agent Feedback остаются источниками
истины своих модулей. Agent Session хранит только ссылки и ограниченный буфер
событий, нужный для reconnect.

### BR-AGENT-CONSOLE-004: Запуск ограничен loopback

Агент запускается только в основном `serve` на loopback-адресе. Статическая
сборка, переводы и любой non-loopback `serve` не получают эту возможность.

### BR-AGENT-CONSOLE-005: Жизненный цикл сессии не зависит от provider

Один manager сериализует изменения одной активной Agent Session, управляет её
ограниченной FIFO и неизменяемой связью с задачей. Сообщения `not-sent` остаются
только в failed session и удаляются вместе с ней. Adapter реализует только
собственный transport, access mapping, возможности и
подтверждённую остановку принадлежащего ему исполнения.

Новая модель разрешений provider не изменяет общий session lifecycle. Интерфейс
принимает решения по фактическим возможностям созданной сессии, а не по имени
provider или его внутренним protocol fields.

### BR-AGENT-CONSOLE-006: Запрошенный доступ не подменяет фактический

Режим `Full access` является намерением пользователя, а не обещанием снять все
ограничения. Agent Session показывает его отдельно от сводки фактического
доступа. Только adapter может подтвердить отсутствие релевантных ограничений;
managed и explicit deny policies не обходятся.

<!-- toudocu:section invariants -->
## Инварианты

- навигация между Tasks, Documentation, Changes и Editor не завершает сессию;
- `Stop response` не завершает сессию, `Stop agent` завершает;
- filesystem-read-only action доступен только при session capability
  `ReadOnlyTurns`; она не обещает отсутствие side effects во внешних tools;
- queued messages не превращаются в steering, а новая сессия не наследует
  `not-sent` предыдущей;
- hidden reasoning и неограниченный command output не сохраняются;
- executable, argv и access policy не читаются из репозитория или браузера;
- Project Terminal является явно открываемым PTY, содержимое которого Toudocu
  не интерпретирует; это не структурированный transport и не `AgentProvider`.
- Project Terminal запускает стандартную командную оболочку платформы, лениво
  загружает отдельный фрагмент xterm и имеет независимый от AgentSession жизненный цикл.
- проекция действий задачи передаёт состояние и отношение активной Agent
  Session; интерфейс не угадывает занятость по последнему нажатию.
- ручной запуск создаёт сессию без привязки к открытой задаче; привязку задаёт
  только серверное действие задачи;
- переход к новой активной сессии очищает представление предыдущего запуска.

<!-- toudocu:section stable-interfaces -->
## Стабильные интерфейсы

- [контракт Agent Console](../contracts/agent-console.md);
- [контракт действий задачи](../contracts/task-actions.md);
- общий `AgentProvider` и поток `AgentEvent`;
- [ADR-009](../decisions/ADR-009.md);
- [ADR-010](../decisions/ADR-010.md);
- [ADR-011](../decisions/ADR-011.md).

<!-- toudocu:section related-use-cases -->
## Связанные сценарии

- [UC-AGENT-CONSOLE-01](../use-cases/UC-AGENT-CONSOLE-01.md)
