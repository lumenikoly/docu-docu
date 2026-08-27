# ТЗ: интегрированная работа с AI-агентом в Toudocu

## 1. Результат

Добавить в canonical `toudocu serve` интегрированный workflow работы человека с AI coding agent.

Первым deeply-integrated provider является Codex.

После доработки пользователь должен иметь возможность прямо из Toudocu:

* взять `Ready` задачу в работу;
* запустить Codex для выбранной задачи;
* видеть structured состояние работы агента;
* одновременно видеть выполняемые агентом команды и их live output;
* писать Codex сообщения;
* направлять уже выполняющуюся работу дополнительными сообщениями;
* остановить текущий ответ/turn без завершения Agent Session;
* полностью остановить агента;
* задать вопрос по задаче;
* запустить `$toudocu clarify`;
* использовать заранее подготовленные Toudocu actions;
* передать существующий Agent Feedback;
* запустить verification задачи;
* передать failed verification обратно агенту;
* переходить между Tasks, Documentation, Changes и Editor без остановки Agent Session.

Для обычной работы пользователь не должен переключаться в отдельный терминал и вручную переносить Toudocu prompts.

Toudocu не становится AI model provider и не вызывает LLM API самостоятельно.

---

# 2. Основной принцип

Toudocu отвечает за:

```text
project model
task state
task readiness
task context
verification contract
Changes
documentation
Agent Feedback
agent session lifecycle
prepared actions
integrated agent UI
```

AI agent отвечает за:

```text
reasoning
repository exploration
implementation
code editing
tool execution
dialog with user
```

Toudocu не реализует собственный agent loop.

---

# 3. Целевая архитектура

```text
                       TOUDOCU SERVE

       Tasks        Docs       Changes       Editor
         │            │            │            │
         └────────────┴──────┬─────┴────────────┘
                             │
                       Agent Console
                             │
                ┌────────────┴────────────┐
                │                         │
                ▼                         ▼
          Agent View                Command Output
                │                         │
                └────────────┬────────────┘
                             │
                       CodexProvider
                             │
                      Codex app-server
                             │
                    repository / tools
```

Codex app-server является единственным основным Codex process.

Не запускать второй Codex TUI только ради отображения команд.

---

# 4. Два одновременных представления

На desktop Agent Console показывает:

```text
┌──────────────────────────────┬──────────────────────────────┐
│ Agent                        │ Command Output               │
│                              │                              │
│ Codex · Working              │ go test ./internal/...      │
│                              │                              │
│ Reading task context         │ ok   internal/app           │
│                              │                              │
│ Modified 3 files             │ git diff --stat             │
│                              │ ...                          │
│ [■ Stop response]            │                              │
│                              │                              │
│ > Ask Codex...               │                              │
└──────────────────────────────┴──────────────────────────────┘
```

На узком экране:

```text
[Agent] [Command Output]
```

Оба представления относятся к одной Agent Session.

---

# 5. Agent View

Agent View отображает нормализованные structured events provider.

Минимально:

```text
user message
agent message
turn status
command execution
file changes
approval request
turn completion
turn failure
session error
```

Не отображать hidden chain-of-thought.

Не сохранять reasoning tokens.

Использовать только:

* пользовательские сообщения;
* ответы агента;
* provider-supported status;
* observable tool activity;
* approvals;
* результаты выполнения.

---

# 6. Command Output

`Command Output` — structured представление выполняемых агентом команд.

Это не terminal emulator.

Для каждой команды показывать:

```text
command
working directory при необходимости
live stdout/stderr
running/completed state
exit code
duration
approval state
```

Пример:

```text
go test ./internal/app/...

ok      toudocu/internal/app     3.14s

Exit 0 · 3.14s
```

Следующая команда:

```text
npm run test

> vitest run
✓ 84 tests passed

Exit 0 · 4.8s
```

---

# 7. Command Output строится из structured events

Источник:

```text
commandExecution started
commandExecution output delta
commandExecution completed
```

или эквивалентные события актуального Codex app-server protocol.

Не использовать:

```text
PTY output
Codex TUI parsing
ANSI screen scraping
```

как основной structured transport.

---

# 8. Command Output не принимает ввод

В structured Codex mode `Command Output` не имеет:

```text
$ _
```

и не позволяет:

* вводить shell-команды;
* писать сообщения Codex;
* отвечать на approval;
* посылать Ctrl+C;
* управлять task;
* определять lifecycle агента через вывод.

Единственный пользовательский input channel к Codex — Agent Composer и structured controls.

---

# 9. Реализация Command Output

Реализовать обычным React component.

Например:

```text
CommandOutput.tsx
CommandExecution.tsx
CommandStream.tsx
```

Использовать:

```text
<pre>
monospace typography
virtualized/limited output при необходимости
```

Не использовать `xterm.js` для structured Command Output без технической необходимости.

Если output содержит ANSI control sequences:

* поддержать безопасное ограниченное ANSI formatting;
* либо корректно удалить неподдерживаемые control sequences.

Не интерпретировать terminal cursor-control semantics как UI-команды.

---

# 10. Ограничение command output

Не хранить неограниченный stdout/stderr.

Для каждой команды и для session в целом установить разумные пределы.

При обрезании явно показывать:

```text
Output truncated
```

Последний доступный хвост должен сохраняться для reconnect browser.

---

# 11. Command card

В Agent View команда также показывается компактной structured card:

```text
Command

go test ./internal/app/...

● Running
```

После завершения:

```text
Command

go test ./internal/app/...

✓ Exit 0 · 3.14s
```

Клик по card:

* фокусирует соответствующий блок в `Command Output`;
* не запускает команду повторно.

---

# 12. Correlation

Structured Command Card и Command Output связываются одним internal item ID.

```text
CommandStarted(item=42)
        │
        ├── Agent View
        └── Command Output

CommandOutput(item=42)
        └── Command Output

CommandCompleted(item=42)
        ├── Agent View
        └── Command Output
```

---

# 13. Agent Composer

В Agent View постоянно доступен:

```text
┌─────────────────────────────────────┐
│ Ask Codex...                        │
│                                     │
│ [Send]          [Actions ▾]         │
└─────────────────────────────────────┘
```

Agent Composer — единственный обычный пользовательский input channel structured session.

---

# 14. Сообщение при idle

Если Codex thread существует, но active turn отсутствует:

```text
user message
     ↓
turn/start
```

Создать новый turn в том же thread.

Не запускать новый Codex process.

---

# 15. Сообщение во время active turn

Если turn выполняется и provider поддерживает steering:

```text
user message
     ↓
turn/steer
```

Пример:

```text
User:
Также проверь Windows.
```

Сообщение направляет текущую работу агента.

Не останавливать turn автоматически.

---

# 16. Steering fallback

Если steering недоступен или его нельзя безопасно выполнить:

1. сообщение пользователя не теряется;
2. оно помещается в локальную FIFO очередь Agent Session;
3. UI показывает:

```text
Queued
```

4. после завершения текущего turn оно отправляется новым turn.

Не эмулировать steering через stdin или PTY.

---

# 17. Stop response

Добавить:

```text
[■ Stop response]
```

При active turn выполнить structured interrupt.

Логически:

```text
turn/interrupt
```

После interrupt:

```text
Codex process      жив
Codex thread       жив
Agent Session      жива
task               остаётся in-progress
```

Пользователь может сразу продолжить conversation.

---

# 18. Stop agent

Отдельное действие:

```text
⋯
Stop agent
```

Оно:

1. interrupt active turn при необходимости;
2. завершает provider session;
3. завершает child process;
4. закрывает transport;
5. освобождает buffers;
6. переводит Agent Session в `exited`.

Статус task автоматически не менять.

---

# 19. Stop response и Stop agent различаются

Зафиксировать:

```text
Stop response
→ остановить текущую работу/ответ

Stop agent
→ завершить всю Agent Session
```

Не объединять их одним действием.

---

# 20. Approvals

Codex approvals обрабатывать structured способом.

Пример:

```text
Approval required

Command:
go test ./...

[Decline] [Approve]
```

Ответ передаётся через Codex app-server protocol.

Не:

* искать `[y/N]` в output;
* отправлять символ `y`;
* парсить TUI;
* использовать Command Output как control channel.

---

# 21. File changes

Structured Codex file-change events показывать как live activity:

```text
Files changed

internal/app/task.go
web/src/features/agent-console/AgentConsole.tsx
docs/guides/work-items.md

[Open Changes]
```

Эти события не являются authoritative diff.

Authoritative состояние изменений остаётся за Toudocu Changes.

---

# 22. Codex transport

Основной transport:

```text
codex app-server
```

Toudocu управляет lifecycle:

```text
initialize
initialized

thread/start

turn/start
turn/steer
turn/interrupt

approval responses

notifications
```

Использовать актуальную supported protocol surface.

Provider-specific wire format изолировать внутри `CodexProvider`.

---

# 23. Browser не знает Codex protocol

Граница:

```text
Browser
   │
   │ Toudocu Agent protocol
   ▼
Toudocu Go
   │
   │ Codex app-server protocol
   ▼
Codex
```

Frontend работает только с нормализованными Toudocu objects/events.

---

# 24. AgentProvider

Ввести:

```text
AgentProvider
```

Логически:

```text
Detect()
StartSession()
ResumeSession()

SendMessage()
Steer()
InterruptTurn()

RespondToApproval()

StopSession()

Capabilities()
```

---

# 25. Provider capabilities

Поддержать:

```text
structuredConversation
steering
interrupt
commandStreaming
fileChangeEvents
approvals
sessionResume
commandOutput
rawPTY
```

Для Codex structured provider:

```text
structuredConversation = true
commandStreaming       = true
commandOutput          = true
fileChangeEvents       = true
approvals              = true
interrupt              = true
```

`steering` определяется реально поддерживаемой установленной версией.

---

# 26. AgentEvent

Provider events нормализовать в:

```text
AgentEvent
```

Минимальные виды:

```text
AgentMessageStarted
AgentMessageDelta
AgentMessageCompleted

CommandStarted
CommandOutput
CommandCompleted

FilesChanged

ApprovalRequested

TurnStarted
TurnCompleted
TurnInterrupted
TurnFailed

SessionError
SessionExited
```

---

# 27. Agent Session

Добавить ephemeral:

```text
AgentSession
```

Минимально:

```text
sessionID
provider
providerSessionID
taskID
startedAt

state
transport

requestedAccess
effectiveAccess

queuedMessages
```

Состояния:

```text
starting
idle
running
waiting-for-approval
interrupting
failed
exited
```

---

# 28. Один Codex thread на session

```text
AgentSession
    │
    └── Codex thread
          ├── Turn 1
          ├── steer
          ├── Turn 2
          ├── interrupt
          └── Turn 3
```

Не запускать новый process для каждого пользовательского сообщения.

---

# 29. Одна активная Agent Session

Первая версия поддерживает максимум одну активную Agent Session на один `serve`.

Не реализовывать:

```text
parallel agents
agent pools
multi-agent orchestration
```

---

# 30. Codex access presets

Поддержать локальные launch presets:

```text
Codex defaults
Full access
```

---

# 31. Codex defaults

В режиме:

```text
Codex defaults
```

Toudocu не должен без необходимости переопределять:

```text
approval policy
sandbox
model
reasoning settings
provider credentials
```

Использовать обычные Codex defaults/configuration.

---

# 32. Full access

`Full access` является structured эквивалентом привычного:

```text
codex --yolo
```

Для structured thread запросить эквивалентную семантику:

```text
approvalPolicy = never
sandbox = dangerFullAccess
```

Точное provider representation изолировать внутри `CodexProvider`.

---

# 33. Full access UI

Показывать явно:

```text
Full access

Codex runs without normal approval prompts
and filesystem sandbox restrictions.
```

Не использовать эвфемизмы:

```text
Fast
Easy
Automatic
```

Первое включение требует подтверждения.

---

# 34. Access preset является user preference

Не хранить его в:

```text
.toudocu/config.yml
```

Repository не должен иметь права включать Full access.

Хранить в локальном пользовательском состоянии Toudocu.

---

# 35. Requested и effective access

Различать:

```text
requestedAccess
effectiveAccess
```

Если provider сообщает effective settings, показывать их.

При расхождении не утверждать, что requested mode действительно применён.

---

# 36. Task Workspace

Существующие:

```text
Board
List
Tree
filters
search
Current work
```

не переписывать без необходимости.

Добавить serve-only actions.

---

# 37. Ready task

Для:

```text
status = ready
readyForWork = true
```

показать:

```text
[Start work]
[Ask]
[Clarify]
```

---

# 38. In progress

Показать:

```text
[Open agent]
[Ask]
[Clarify]
[Verify]
[Changes]
```

---

# 39. Waiting

Показать:

```text
[Explain blocker]
[Ask]
[Clarify]
```

---

# 40. Needs attention

Показать:

```text
[Explain problems]
[Clarify]
[Edit task]
```

---

# 41. Draft

Показать:

```text
[Ask]
[Clarify]
[Edit task]
```

---

# 42. StartTask

Добавить server-side domain operation:

```text
StartTask(taskID, expectedDigest)
```

Она:

1. перечитывает модель;
2. находит task;
3. требует `status=ready`;
4. проверяет `readyForWork`;
5. проверяет dependencies;
6. проверяет digest task-файла;
7. атомарно меняет:

```text
ready → in-progress
```

8. перестраивает модель;
9. запускает или открывает Agent Session;
10. отправляет prepared work action.

Browser не редактирует task Markdown самостоятельно.

---

# 43. Один клик

Если:

```text
Codex найден
Toudocu skill установлен
access preset выбран
```

действие:

```text
[Start work]
```

должно выполнить основной workflow одним пользовательским действием.

---

# 44. Prompt/Action Registry

Добавить:

```text
AgentPromptRegistry
```

Browser передаёт:

```text
action
taskID
optional userText
optional context
```

Prompt формируется централизованно.

Не собирать его из DOM.

---

# 45. Start work action

Смысл сообщения:

```text
Work on TASK-SITE-021 using the Toudocu workflow.

Start from the authoritative Toudocu task context.
Follow the documented scope and acceptance criteria.
Do not change the task contract unless I explicitly approve it.
Do not mark the task done automatically.
```

Не вставлять полный TaskContextReport в prompt.

Agent должен использовать Toudocu workflow.

---

# 46. Ask

Кнопка:

```text
[Ask]
```

открывает:

```text
Ask about TASK-SITE-021

[ Why is this dependency required? ]
```

Выполнить отдельным read-only turn.

---

# 47. Read-only question actions

Для:

```text
Ask
Explain blocker
Explain problems
Ask what to do next
```

при наличии provider capability устанавливать read-only execution boundary.

Не полагаться только на prompt:

```text
Do not modify files.
```

---

# 48. Clarify

```text
[Clarify]
```

поддерживает optional focus.

Например:

```text
Clarify TASK-SITE-021

Focus:
[ backward compatibility ]
```

Отправлять существующий workflow:

```text
$toudocu clarify TASK-SITE-021 — backward compatibility
```

или:

```text
$toudocu clarify TASK-SITE-021
```

Clarify не начинает implementation автоматически.

---

# 49. Clarify conversation

Вопросы и ответы проходят через Agent View:

```text
Codex:
Should v1 compatibility remain?

User:
No.
```

Если во время активной работы пользователь добавляет:

```text
Не рассматривай обратную совместимость —
мы уже решили её удалить.
```

использовать steering либо очередь fallback.

---

# 50. Ask what to do next

Добавить в Task Workspace:

```text
[Ask what to do next]
```

Agent использует:

```text
task candidates
readiness
dependencies
current repository state
```

и объясняет выбор.

Действие read-only.

Сам Toudocu по-прежнему не выбирает следующую задачу автоматически.

---

# 51. Verification

`Verify` не является AI prompt.

Использовать существующий Toudocu verification service напрямую.

```text
[Verify]
```

→

```text
Run verification commands declared by TASK-SITE-021?

[Cancel] [Run]
```

Только после подтверждения выполнить semantics:

```text
task verify --run
```

---

# 52. Verification Output

Результат verification показывать отдельной секцией:

```text
Verification Output
```

а не смешивать с `Command Output` Codex.

Например:

```text
Verification

AC-01 ✓
go test ./internal/app/...
exit 0

AC-02 ✗
npm run test:browser
exit 1
```

Так пользователь всегда понимает источник execution.

---

# 53. Command Output и Verification Output — разные источники

Зафиксировать:

```text
Command Output
→ команды, запущенные AI agent

Verification Output
→ команды, запущенные Toudocu verification service
```

Можно визуально использовать один общий компонент отображения process output, но source/state должны оставаться разными.

---

# 54. Failed verification

После ошибки:

```text
Verification failed

AC-03
npm run test:browser
exit 1

[Ask agent to fix]
```

Создать новый Codex turn:

```text
Investigate the failed verification for TASK-SITE-021.

Use the current Toudocu verification result.
Fix the implementation only within the existing task scope.
Do not weaken tests, acceptance criteria or verification commands.
```

Не запускать исправление автоматически.

---

# 55. Agent Feedback

Существующий Agent Feedback contract не менять.

Остаётся:

```text
Discussion
   ↓
AgentDelivery
   ↓
agent next
   ↓
agent respond
```

При активном агенте:

```text
[Process with active agent]
```

отправляет:

```text
$toudocu feedback
```

через Agent Composer.

---

# 56. Command Output contextual actions

Для command execution:

```text
go test ./...

FAIL TestFoo
...
```

добавить:

```text
[Ask about command]
[Copy command]
```

Для выделенного output:

```text
[Ask Codex about selection]
[Copy]
```

`Ask` всегда использует structured Agent Composer.

Не отправлять выбранный текст через process stdin.

---

# 57. Обычный разговор

Пользователь может просто написать:

```text
Не меняй публичный API.
```

Toudocu делает:

```text
active turn
→ steer

idle
→ new turn
```

Это основной interactive UX.

---

# 58. Actions menu

Agent Composer:

```text
Actions
├── Ask about task
├── Clarify
├── Process feedback
└── Refresh diff
```

Не создавать десятки отдельных command buttons.

---

# 59. Refresh diff

Отправляет:

```text
$toudocu refresh diff
```

и сохраняет существующую permission semantics skill.

---

# 60. Agent Console frontend

Добавить:

```text
web/src/features/agent-console/
```

Пример:

```text
AgentConsole.tsx
AgentView.tsx
CommandOutput.tsx
CommandExecution.tsx
PromptComposer.tsx
CommandCard.tsx
ApprovalCard.tsx
FileChangeCard.tsx
ProviderSetup.tsx
api.ts
model.ts
index.tsx
```

Использовать существующие:

```text
IslandHost
React
Base UI
design
ui
docs-ui
PageBootstrap v1
```

---

# 61. Task Workspace не переносить целиком в React

Board/List/Tree могут остаться framework-free.

Task action отправляет application event:

```text
toudocu:agentaction
```

с:

```text
action
taskID
```

Agent Console island активируется при необходимости.

---

# 62. Soft navigation

Agent Session остаётся server-side при переходах:

```text
Tasks
→ Documentation
→ Changes
→ Editor
```

При:

```text
toudocu:pagebeforechange
```

Agent Console UI размонтируется.

Agent process продолжает работу.

После:

```text
toudocu:pagechange
```

Agent Console reconnects.

---

# 63. Reconnect buffers

Server хранит ограниченные:

```text
structured AgentEvent buffer
Command Output buffer
queued messages
```

После reconnect UI восстанавливает:

* состояние session;
* последние agent messages/events;
* активную команду;
* последний output;
* queued messages.

Не сохранять полный transcript бессрочно.

---

# 64. Fallback через Project Terminal

Если structured Codex integration недоступна:

```text
Structured integration unavailable

[Open Terminal]
```

Нужную CLI пользователь запускает сам. Терминал не является fallback provider.

---

# 65. Project Terminal

Самостоятельная локальная возможность называется:

```text
Project Terminal
```

и использует настоящий terminal emulator.

Схема:

```text
Toudocu
   │
   ▼
PTY
   │
   ▼
стандартная командная оболочка платформы
```

В этом режиме терминал действительно интерактивен:

* keyboard input;
* обычные CLI и TUI;
* Ctrl+C;
* approvals через TUI запущенного агента;
* обычное terminal interaction.

---

# 66. xterm.js

`@xterm/xterm` и PTY нужны только для `Project Terminal`.

Не грузить xterm bundle в основной structured Codex mode.

Это сохраняет structured Agent Console легче и исключает путаницу между:

```text
Command Output
```

и:

```text
Project Terminal
```

---

# 67. Запуск CLI в Project Terminal

Project Terminal запускает только стандартную командную оболочку. `codex`,
`opencode`, `claude` и нужные им аргументы пользователь вводит сам.

---

# 68. Запрет arbitrary command

Не поддерживать:

```yaml
agent:
  command: ...
  args: ...
```

в project config.

Browser не передаёт:

```text
executable
argv
cwd
environment
```

Invocation определяет trusted provider adapter.

---

# 69. Provider detection

Codex искать через Go:

```text
exec.LookPath("codex")
```

Shell alias не использовать как источник конфигурации.

---

# 70. Skill integration

Перед интегрированным Toudocu workflow проверить существующий installed skill.

Если skill отсутствует:

```text
Toudocu skill is required.

[Install & continue]
```

Использовать существующий Go skill installer.

Не выполнять установку через shell.

---

# 71. Credentials

Codex использует собственную authentication/config environment.

Toudocu:

* не хранит Codex/OpenAI API keys;
* не передаёт environment browser;
* не показывает credentials;
* не реализует собственный Codex login flow.

---

# 72. Agent Console API

Минимально добавить:

```text
GET  /_toudocu/api/agent-console/
POST /_toudocu/api/agent-console/start
POST /_toudocu/api/agent-console/stop
POST /_toudocu/api/tasks/{taskID}/start
GET  /_toudocu/api/agent-console/ws
```

---

# 73. Agent WebSocket

Через WebSocket передавать:

```text
AgentEvent
session state
Command Output chunks
queued message state
approval requests

user message
steering
interrupt
approval response
```

Не передавать arbitrary process invocation.

---

# 74. PageBootstrap

Аддитивно добавить:

```text
capabilities.agentConsole
endpoints.agentConsole
```

Go остаётся владельцем разрешения feature.

---

# 75. Loopback-only

Agent execution разрешён только в canonical:

```text
127.0.0.1
::1
```

При:

```text
--host 0.0.0.0
```

установить:

```text
capabilities.agentConsole = false
```

и не регистрировать доступные agent execution endpoints.

---

# 76. Static build

`toudocu build` не содержит:

```text
Agent Console runtime
Codex app-server integration
PTY
xterm
Agent WebSocket
task start mutation API
```

Static Task Workspace остаётся read-only.

---

# 77. Translation portals

Agent Console отсутствует в translation portals и `serve`, запущенном непосредственно для translation root.

---

# 78. Cleanup

При завершении `serve`:

1. interrupt active turn;
2. завершить Agent Session;
3. завершить provider child process;
4. закрыть WebSocket;
5. удалить ephemeral buffers.

Orphan agent processes не допускаются.

---

# 79. Что не делать

Не добавлять в эту версию:

* Toudocu Studio;
* direct OpenAI API integration;
* API key settings;
* собственный LLM conversation engine;
* второй Codex process ради Command Output;
* PTY в structured Codex mode;
* parsing Codex TUI;
* repository-controlled agent launch flags;
* arbitrary provider executable;
* multi-agent orchestration;
* несколько параллельных Agent Sessions;
* persisted chain-of-thought;
* полный persisted AI transcript;
* автоматическое завершение задачи;
* автоматическое исправление failed verification;
* второй Agent Feedback mechanism;
* browser implementation task readiness/context/verify.

---

# 80. Go-компоненты

Добавить логически:

```text
AgentProvider
CodexProvider
PTYProvider
CodexAppServerClient

AgentSessionManager
AgentSession
AgentEvent

AgentPromptRegistry
AgentPreferenceStore

TaskStartService
AgentConsoleHTTP
```

Не выполнять несвязанный большой package refactor.

---

# 81. Тестовый provider

Tests не требуют:

```text
реальный Codex account
сеть
OpenAI backend
```

Создать fake app-server process.

Он должен эмулировать:

```text
agent messages
command started
command output
command completed
file changes
approval
steering
interrupt
turn completion
failure
```

---

# 82. Go tests

Проверить:

* Codex detection;
* app-server initialization;
* session lifecycle;
* idle message → new turn;
* active message → steering;
* steering fallback queue;
* queue ordering;
* turn interrupt;
* продолжение после interrupt;
* full Agent Session stop;
* AgentEvent normalization;
* command correlation;
* Command Output ordering;
* output truncation;
* approvals;
* reconnect;
* Ready → In progress;
* stale task digest;
* Full access mapping;
* read-only question actions;
* loopback-only restriction;
* arbitrary argv rejection.

---

# 83. Frontend tests

Проверить:

* Agent + Command Output одновременно;
* narrow-screen tabs;
* live agent message;
* live Command Output;
* command card → matching output;
* idle Send;
* active Send;
* queued message;
* Stop response;
* continue after interrupt;
* Stop agent;
* approval dialog;
* Ask about command;
* Ask about output selection;
* Ask;
* Clarify;
* Start work;
* Verify;
* Verification Output;
* failed verify → Ask agent to fix;
* reconnect after soft navigation;
* Full access indicator.

---

# 84. Project Terminal tests

Отдельно проверить:

```text
loopback serve
→ Project Terminal available
```

В Project Terminal проверить:

* xterm mount;
* PTY input/output;
* resize;
* Ctrl+C;
* stop process;
* стандартную командную оболочку;
* отображение полноэкранного альтернативного буфера после начального resize;
* одновременную работу с structured Agent Session.

Не смешивать эти тесты с structured Codex tests.

---

# 85. Browser acceptance flow

Основной сценарий:

```text
Task Workspace
      ↓
TASK-X Ready
      ↓
Start work
      ↓
Ready → In progress
      ↓
Codex structured session
      ↓
prepared task action
      ↓
Agent View streams response
      ↓
Command starts
      ↓
Command Output streams result
      ↓
User:
"Также проверь Windows"
      ↓
steering
      ↓
Codex adjusts work
      ↓
User:
Stop response
      ↓
turn interrupted
      ↓
User:
"Сначала исправь тест"
      ↓
new turn
      ↓
Changes
      ↓
Verify
      ↓
Verification Output
      ↓
failure
      ↓
Ask agent to fix
```

Внешний terminal не требуется.

---

# 86. Документация

Обновить:

```text
architecture/frontend-runtime-boundary.md
architecture/trust-boundaries.md
modules/site.md
modules/MOD-AGENT-FEEDBACK.md
guides/work-items.md
guides/agent-workflows.md
guides/local-workflow.md
reference/features.md
```

Добавить:

```text
MOD-AGENT-CONSOLE
UC-AGENT-CONSOLE-01
FLOW-AGENT-TASK-WORKFLOW
```

Добавить ADR со следующим решением:

```text
Codex app-server
→ structured control transport

Agent View
→ conversation and lifecycle

Command Output
→ structured projection of agent command execution

Project Terminal
→ PTY со стандартной командной оболочкой; Toudocu не интерпретирует вывод
```

---

# 87. Критерии приёмки

## Task workflow

* `Ready` task берётся в работу одной кнопкой.
* Go повторно проверяет readiness.
* `ready → in-progress` выполняется server-side.
* Codex session запускается автоматически.
* Prepared action отправляется автоматически.

## Conversation

* Пользователь пишет Codex прямо в Toudocu.
* Idle session создаёт новый turn.
* Active turn использует steering при поддержке.
* При невозможности steering сообщение не теряется.
* Queued message запускается после текущего turn.

## Stop

* `Stop response` прерывает только текущий turn.
* Agent Session остаётся активной.
* После interrupt conversation можно продолжить.
* `Stop agent` завершает всю Agent Session.

## Structured integration

* Agent lifecycle определяется structured protocol.
* Approvals structured.
* Commands structured.
* File-change activity structured.
* Completion не определяется анализом process output.

## Command Output

* Command Output отображается одновременно с Agent View на desktop.
* Он показывает команды и live output той же Codex session.
* Второй Codex process не запускается.
* Command Output не принимает терминальный ввод.
* Command Output не является control transport.
* Command Card и output корректно связаны.
* Пользователь может спросить Codex о команде или выделенном output.

## Project Terminal

* Project Terminal существует как самостоятельная локальная возможность.
* Только Project Terminal использует PTY/xterm.
* Project Terminal и structured Agent Session могут работать одновременно.
* Вывод PTY не превращается в `AgentEvent`.

## Full access

* Поддерживаются `Codex defaults` и `Full access`.
* Full access соответствует семантике `codex --yolo`.
* Repository не может включить Full access.
* Выбор хранится локально.
* UI явно сообщает об отключении обычных sandbox/approval restrictions.

## Verification

* Verification запускается Toudocu напрямую.
* Требуется отдельное user confirmation.
* Agent Command Output и Verification Output не смешиваются.
* Failed verification не исправляется автоматически.
* Failure можно явно отправить active agent.

## Security

* Agent execution только loopback.
* `0.0.0.0` не предоставляет agent execution.
* Static portal не содержит agent runtime.
* Translation portal не содержит agent runtime.
* Browser не передаёт executable/argv/cwd.
* Toudocu не вызывает shell для Codex structured provider.
* Child process завершается вместе с `serve`.

## Existing architecture

* Portal не становится SPA.
* `serve.ts` остаётся framework-free.
* Agent Console использует существующий IslandHost.
* Task/document semantics остаются в Go.
* Agent Feedback semantics не меняется.
* Changes остаётся authoritative diff.
* Task verification остаётся authoritative verification contract.

---

# 88. Целевой UX

```text
TASK-SITE-021 · In progress

┌──────────────────────────────┬──────────────────────────────┐
│ Codex · Full access          │ Command Output              │
│                              │                              │
│ ● Working                    │ go test ./...               │
│                              │                              │
│ Editing agent_console.go     │ ok   internal/app           │
│                              │                              │
│ [■ Stop response]            │ npm test                    │
│                              │ ✓ 84 tests                  │
│ > Также проверь Windows      │                              │
│                        Send  │                              │
├──────────────────────────────┴──────────────────────────────┤
│ Ask · Clarify · Verify · Changes · Process feedback       │
└─────────────────────────────────────────────────────────────┘
```

Главный UX-инвариант:

> Пользователь управляет Codex через structured Agent View и единый Agent Composer, а Command Output одновременно показывает фактически выполняемые команды и их вывод.

Главный архитектурный инвариант:

> Command Output является доступной только для чтения проекцией структурированных событий выполнения. Project Terminal является отдельным PTY, содержимое которого Toudocu не интерпретирует, и не реализует `AgentProvider`.
