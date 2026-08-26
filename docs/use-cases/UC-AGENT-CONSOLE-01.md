<!-- toudocu
id: UC-AGENT-CONSOLE-01
status: planned
priority: high
module: MOD-AGENT-CONSOLE
updated: 2026-08-26
-->

# UC-AGENT-CONSOLE-01: Выполнить задачу с coding agent в Toudocu

Актор: разработчик, работающий с canonical loopback `serve`.

<!-- toudocu:section main-scenario -->
## Основной сценарий

1. Разработчик открывает Ready задачу и выбирает `Start work`.
2. Toudocu получает актуальный task context и запускает Codex app-server с
   пользовательским access preset.
3. Разработчик видит сообщения в Agent View и structured команды в Command
   Output одной Agent Session.
4. Он продолжает conversation, отправляет steering во время turn, отвечает на
   approvals или использует `Stop response`.
5. Через подготовленные действия он запускает Clarify, verification либо
   передаёт Agent Feedback; failed verification можно вернуть агенту.
6. Разработчик переходит между Tasks, Documentation, Changes и Editor, не
   завершая сессию.
7. После работы он выбирает `Stop agent`.

## Дополнительные пути

- Ask и другие read-only вопросы используют filesystem sandbox provider. Эта
  гарантия не распространяется автоматически на side effects внешних tools.
- Если structured provider несовместим, пользователь явно запускает Terminal
  Mode как PTY fallback.
- Если provider не поддерживает steering, сообщение ожидает завершения turn и
  отображается как отложенное.

<!-- toudocu:section postconditions -->
## Постусловия

Состояние задачи, результаты verification, Changes и Agent Feedback остаются в
своих источниках Toudocu. Agent Session завершена явно или продолжает работать
при навигации.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] Ready задача запускает одну Agent Session без ручного переноса контекста.
- [ ] Conversation, steering, interrupt, approvals, verification и Agent
  Feedback доступны через одну сессию.
- [ ] Command Output не принимает ввод и отражает structured события команд.
- [ ] Статическая сборка, переводы и любой non-loopback `serve` не запускают
  coding agent.

<!-- toudocu:section business-rules -->
## Бизнес-правила

- [BR-AGENT-CONSOLE-001](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-001-одна-сессия-имеет-один-источник-исполнения)
- [BR-AGENT-CONSOLE-002](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-002-команды-являются-наблюдаемой-проекцией)
- [BR-AGENT-CONSOLE-003](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-003-состояние-проекта-не-дублируется)
- [BR-AGENT-CONSOLE-004](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-004-запуск-ограничен-loopback)

<!-- toudocu:section implementation -->
## Реализация

- [FLOW-AGENT-CONSOLE](../flows/FLOW-AGENT-CONSOLE.md)
- [контракт Agent Console](../contracts/agent-console.md)
