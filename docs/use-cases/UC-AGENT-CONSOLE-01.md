<!-- toudocu
id: UC-AGENT-CONSOLE-01
status: planned
priority: high
module: MOD-AGENT-CONSOLE
updated: 2026-08-30
-->

# UC-AGENT-CONSOLE-01: Выполнить задачу с coding agent в Toudocu

Актор: разработчик, работающий с canonical loopback `serve`.

<!-- toudocu:section main-scenario -->
## Основной сценарий

1. Разработчик открывает задачу со статусом Ready и запускает `Start work`
   через значок встроенного агента рядом с названием действия.
2. Toudocu получает актуальный task context и запускает Codex app-server с
   пользовательским access preset.
3. Страница остаётся открытой и показывает состояние Agent Session. По нажатию
   индикатора разработчик открывает Agent View и Command Output этой сессии.
4. Он продолжает conversation, отправляет steering во время turn, отвечает на
   approvals или использует `Stop response`.
5. Через подготовленные действия он запускает Clarify, verification либо
   передаёт Agent Feedback инструкцией `$toudocu feedback`; failed verification
   можно вернуть агенту.
6. Разработчик переходит между Tasks, Documentation, Changes и Editor, не
   завершая сессию.
7. После работы он выбирает `Stop agent`.

Для родительской задачи разработчик может выбрать `Complete task tree`.
Toudocu проверяет поддерево, запускает одну связанную Agent Session и после
каждого turn выбирает следующего готового потомка по зависимостям и `TASK-ID`.
Родитель выполняется после детей; интерфейс показывает текущую задачу и общий
счётчик. Подтверждение приостанавливает цикл, а остановка, ошибка или отсутствие
прогресса в трёх turn блокируют временную цель.

Вместо запуска Agent Console разработчик может скопировать у любого доступного
действия готовый handoff и передать его внешнему coding agent. Сервер включает
в него только краткую инструкцию действия; установленный навык получает
актуальный контекст из репозитория. Браузер отвечает только за копирование и
ручной fallback.

## Дополнительные пути

- Ask и другие read-only вопросы используют filesystem sandbox provider. Эта
  гарантия не распространяется автоматически на side effects внешних tools.
- Пользователь может явно открыть Project Terminal и работать в shell
  независимо от Agent Session.
- Если structured Codex не запускается, интерфейс предлагает открыть
  терминал; нужную CLI пользователь запускает самостоятельно.
- Если provider не поддерживает steering, сообщение ожидает завершения turn и
  отображается как отложенное.
- «Спросить» создаёт разовый разговор в Agent Console.

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
- [ ] Готовое дерево задач выполняется последовательно через общий provider
  contract и явно показывает завершение либо блокировку цели.

<!-- toudocu:section business-rules -->
## Бизнес-правила

- [BR-AGENT-CONSOLE-001](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-001-одна-сессия-имеет-один-источник-исполнения)
- [BR-AGENT-CONSOLE-002](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-002-команды-являются-наблюдаемой-проекцией)
- [BR-AGENT-CONSOLE-003](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-003-состояние-проекта-не-дублируется)
- [BR-AGENT-CONSOLE-004](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-004-запуск-ограничен-loopback)
- [BR-AGENT-CONSOLE-007](../modules/MOD-AGENT-CONSOLE.md#br-agent-console-007-цель-дерева-остаётся-общей-и-временной)

<!-- toudocu:section implementation -->
## Реализация

- [FLOW-AGENT-CONSOLE](../flows/FLOW-AGENT-CONSOLE.md)
- [контракт Agent Console](../contracts/agent-console.md)
