<!-- toudocu
id: TASK-AGENT-016
status: ready
taskType: feature
priority: medium
module: MOD-AGENT-FEEDBACK
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
standards: STD-GO-001, STD-DOCS-001
dependsOn: TASK-AGENT-008, TASK-AGENT-014
updated: 2026-08-27
-->

# TASK-AGENT-016: Интегрировать durable обсуждение непосредственно в workflow задачи

<!-- toudocu:section result -->
## Результат

На странице задачи пользователь явно различает два сценария:

```text
Спросить
```

— одноразовый transient запрос текущему или внешнему coding agent;

и:

```text
Обсудить
```

— durable discussion, сохраняемый существующим модулем Agent Feedback.

Task discussion не создаёт новую систему комментариев и не дублирует
`Discussion`, `DiscussionMessage` или `AgentDelivery`.

Существующая Agent Feedback очередь остаётся единственным durable transport
обсуждений к development agent.

<!-- toudocu:section behavior-change -->

## Изменение поведения

<!-- toudocu:section before -->

### Было

Discussion можно создать через общий UI документации, но task workflow не
представляет обсуждение как самостоятельное действие задачи.

Пользователю легко смешать transient `Ask` в Agent Console с durable feedback,
который должен остаться в истории документа.

<!-- toudocu:section after -->

### Станет

Task Action Bar содержит отдельное действие:

```text
Обсудить
```

Оно открывает существующий Discussion composer для canonical task document.

После submit сообщение сохраняется обычным Agent Feedback API и получает
обычную `AgentDelivery`.

Дальнейшая обработка может выполняться:

* существующей активной Agent Console через `$toudocu feedback`;
* внешним development agent через существующий Toudocu skill/CLI workflow.

FIFO semantics не меняются.

<!-- toudocu:section scope -->

## Область изменения

* task page integration;
* `web/src/features/discussions/`;
* task-actions presentation;
* existing Agent Feedback UI hooks;
* Agent Console integration;
* локализация и browser tests;
* documentation workflow.

<!-- toudocu:section out-of-scope -->

## Не входит в задачу

* новый task comment entity;
* отдельный task discussion backend;
* новый queue;
* обход `AgentDelivery`;
* task-specific изменение FIFO;
* автоматический запуск агента после submit;
* автоматическое применение change request;
* закрытие discussion ответом агента;
* перенос discussion content в Agent Session history.


## Требования к реализации

### 1. Использовать существующую domain model

Для задачи используются существующие:

* `Discussion`;
* `DiscussionMessage`;
* `DocumentAnchor`;
* `AgentDelivery`.

Target:

```text
kind=document
path=<canonical task document>
```

Не добавлять `TaskDiscussion`.

### 2. Отделить transient Ask от durable Discuss

UI и документация должны явно различать:

#### `Спросить`

* transient conversation;
* не создаёт Discussion;
* не создаёт AgentDelivery;
* ответ находится в Agent Console либо внешнем agent session.

#### `Обсудить`

* durable message;
* сохраняется в Agent Feedback;
* остаётся доступным после завершения Agent Session;
* создаёт AgentDelivery по существующим правилам.

### 3. Composer

`Обсудить` с task page открывает существующий Discussion composer уже с
правильным document target.

Пользователь выбирает существующий intent:

* `question`;
* `change_request`.

Правила разрешений не меняются:

* `question` не разрешает изменение файлов;
* `change_request` разрешает только существующую область изменения.

### 4. Whole-task и selection discussion

Действие в Task Action Bar создаёт discussion для task document в целом.

Если пользователь выделил текст task document и использует существующее
контекстное действие обсуждения, используется текущая anchor/range semantics.

Не создавать второй формат anchors специально для задач.

### 5. Обработка активным агентом

Если есть active Agent Session, UI может предложить:

```text
Обработать с активным агентом
```

Действие отправляет существующий prepared:

```text
$toudocu feedback
```

и не извлекает `DiscussionMessage` напрямую в Agent Console prompt.

Skill продолжает использовать существующие:

```text
toudocu agent next
toudocu agent respond
```

Таким образом Agent Console не обходит Agent Feedback transport.

### 6. Обработка внешним агентом

Для внешнего development agent предоставить копируемую canonical instruction,
которая направляет агента в существующий feedback workflow.

Не сериализовать discussion вручную в новый task handoff protocol.

Внешний агент должен получить delivery через существующий Toudocu skill/CLI,
чтобы сохранить:

* FIFO;
* locking;
* idempotent response;
* permission boundaries.

### 7. Queue semantics

Task page не фильтрует и не переупорядочивает Agent Feedback queue.

Если oldest pending delivery относится к другому документу, `$toudocu feedback`
обрабатывает его первым согласно существующему бизнес-правилу.

Не вводить task-specific bypass.

### 8. Session independence

Discussion существует независимо от Agent Session lifecycle.

Stop agent, crash или смена provider:

* не удаляют discussion;
* не удаляют pending delivery;
* не закрывают discussion.

<!-- toudocu:section acceptance-criteria -->

## Критерии приёмки

* [ ] `AC-01` Task page предоставляет отдельные `Спросить` и `Обсудить` с
  различимыми semantics.
* [ ] `AC-02` `Обсудить` использует существующие Discussion и AgentDelivery без
  нового task-specific persistence.
* [ ] `AC-03` Whole-task discussion использует canonical document target, а
  selection discussion — существующий DocumentAnchor.
* [ ] `AC-04` `question` и `change_request` сохраняют существующие permission
  boundaries.
* [ ] `AC-05` `Обработать с активным агентом` использует `$toudocu feedback`
  и существующие `agent next/respond`, не передавая сообщение напрямую.
* [ ] `AC-06` Внешний agent workflow также не обходит FIFO AgentDelivery.
* [ ] `AC-07` Task page не создаёт отдельную очередь и не меняет глобальный FIFO.
* [ ] `AC-08` Discussion и pending delivery переживают stop/crash Agent Session.
* [ ] `AC-09` После browser soft navigation discussion state остаётся
  authoritative server state без duplicate composer/listeners.

<!-- toudocu:section plan -->

## План

1. Добавить `Обсудить` в task page presentation.
2. Связать его с существующим Discussion composer.
3. Проверить whole-document и selection targets.
4. Добавить обработку через active Agent Console.
5. Добавить внешний feedback instruction.
6. Проверить сохранение FIFO и permission semantics.
7. Добавить browser/integration tests.

<!-- toudocu:section verification -->

## Проверка

* `AC-01` → `go test ./internal/app -run 'TestAgentFeedback' && make browser-test`
* `AC-02` → `go test ./internal/app -run 'TestAgentFeedback' && make browser-test`
* `AC-03` → `go test ./internal/app -run 'TestAgentFeedback' && make browser-test`
* `AC-04` → `go test ./internal/app -run 'TestAgentFeedback' && make browser-test`
* `AC-05` → `go test ./internal/app -run 'TestAgentConsoleFeedback' && make browser-test`
* `AC-06` → `go test ./internal/app -run 'TestAgentFeedbackFIFO|TestAgentFeedbackDelivery'`
* `AC-07` → `go test ./internal/app -run 'TestAgentFeedbackFIFO|TestAgentFeedbackDelivery'`
* `AC-08` → `go test ./internal/app -run 'TestAgentFeedbackSessionIndependence'`
* `AC-09` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task discussion'`
* `QUALITY` → `make check`
* `ALL` → `make check && make browser-test`
* `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->

## Влияние на документацию

Обновить:

* `MOD-AGENT-FEEDBACK`;
* `MOD-AGENT-CONSOLE`;
* `FLOW-AGENT-FEEDBACK`;
* task workflow;
* local workflow;
* `UC-AGENT-FEEDBACK-01`;
* `UC-AGENT-CONSOLE-01`.

Документация должна явно фиксировать:

```text
Ask = transient agent conversation
Discuss = durable Agent Feedback
```
