<!-- toudocu
id: TASK-AGENT-010
status: done
taskType: feature
priority: medium
module: MOD-SITE
useCase: UC-DOCS-03
parentTask: TASK-AGENT-001
dependsOn: TASK-AGENT-006
updated: 2026-08-29
-->

# TASK-AGENT-010: Добавить structured OpenCode provider

<!-- toudocu:section result -->
## Результат

Пользователь может выбрать OpenCode для существующего workflow в Task Workspace
и Agent Console. Toudocu запускает доверенный `opencode serve`, управляет одной
OpenCode session через HTTP API, читает поток событий server API и преобразует
наблюдаемую работу в те же `AgentEvent`, которые использует Codex adapter.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Structured workflow поддерживает только Codex. Для OpenCode доступен лишь
будущий общий PTY fallback без нормализованных событий и Command Output.

<!-- toudocu:section after -->
### Станет

В меню `Start work` можно выбрать OpenCode. Agent View, Command Output,
composer, verification и Agent Feedback работают через существующий
provider-agnostic frontend без отдельной реализации UI для OpenCode. Approvals
доступны только тогда, когда установленная версия OpenCode объявляет
совместимую structured capability.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_provider.go`;
- `internal/app/agent_events.go`;
- `internal/app/agent_opencode.go`;
- `internal/app/agent_opencode_test.go`;
- `internal/app/agent_preferences.go`;
- `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- изменение Task Workspace или Agent Console под protocol OpenCode;
- универсализация model, agent, variant, permissions или reasoning settings;
- OpenCode SDK dependency, если HTTP и SSE достаточно реализовать stdlib Go;
- одновременный запуск Codex и OpenCode;
- OpenCode TUI или PTY transport;
- repository-controlled executable, argv, cwd или access policy.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` OpenCode обнаруживается без shell, а Toudocu запускает один
  доверенный `opencode serve`; browser и repository не передают executable,
  argv, cwd или environment.
- [x] `AC-02` Provider создаёт новую OpenCode session, отправляет prompt,
  продолжает conversation и прерывает активную работу через HTTP API. После
  stop или crash новый запуск не выдаётся за продолжение прежней session.
- [x] `AC-03` Server event stream преобразует сообщения, команды и их output,
  изменения файлов и lifecycle в общие `AgentEvent` без provider-specific
  payload во frontend. Permission requests нормализуются только при наличии
  совместимой structured capability установленной версии OpenCode; иначе
  provider объявляет capability=false и UI не предлагает approval action.
- [x] `AC-04` Provider объявляет только реально поддержанные capabilities;
  недоступные steering, approvals или access mapping не эмулируются через
  prompt и честно отражаются в Agent Console.
- [x] `AC-05` `Default` сохраняет настройки OpenCode, а `Full access` включается
  только через доверенное provider-specific permission mapping и показывает
  фактически применённый режим.
- [x] `AC-06` Fake OpenCode server полностью проверяет HTTP lifecycle, поток
  событий, reconnect, abort и normalization без сети, credentials и AI backend.
- [x] `AC-07` Существующие Task Workspace и Agent Console tests проходят без
  provider-specific frontend branch для OpenCode.

<!-- toudocu:section plan -->
## План

1. Добавить обнаружение OpenCode и безопасный lifecycle `opencode serve`.
2. Реализовать HTTP session client и потребителя server event stream.
3. Нормализовать события и вычислить capabilities.
4. Добавить provider-specific access и launch preferences.
5. Добавить fake server и проверить существующий agent workflow.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestOpenCodeProviderDetect|TestOpenCodeProviderInvocation'`
- `AC-02` → `go test ./internal/app -run 'TestOpenCodeSessionLifecycle|TestOpenCodeInterrupt'`
- `AC-03` → `go test ./internal/app -run 'TestOpenCodeAgentEvents'`
- `AC-04` → `go test ./internal/app -run 'TestOpenCodeCapabilities'`
- `AC-05` → `go test ./internal/app -run 'TestOpenCodeAccessPreset|TestOpenCodePreferences'`
- `AC-06` → `go test ./internal/app -run 'TestFakeOpenCodeServer|TestOpenCodeReconnect'`
- `AC-07` → `make web-check && make browser-test`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Документация provider contract, Agent Console, local workflow, security boundary
и справочник возможностей описывает OpenCode как второй structured provider и
фиксирует provider-specific launch preferences.
