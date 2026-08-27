<!-- toudocu
id: TASK-AGENT-014
status: ready
taskType: feature
priority: high
module: MOD-SITE
useCase: UC-AGENT-CONSOLE-01
parentTask: TASK-AGENT-001
standards: STD-GO-001, STD-DOCS-001
dependsOn: TASK-AGENT-013
updated: 2026-08-27
-->

# TASK-AGENT-014: Добавить единые действия непосредственно на страницу задачи

<!-- toudocu:section result -->
## Результат

Страница work-item становится основной рабочей поверхностью задачи.

Пользователь видит рядом с task status только действия, допустимые в текущем
`workspaceState`, и может выполнять их без возврата в Board/List/Tree.

Task Workspace и task detail page используют один и тот же server-driven action
contract и один frontend controller.

Интерфейс не дублирует state machine Toudocu и не определяет допустимость
действий самостоятельно.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Agent actions расположены преимущественно в Board/List/Tree.

На task detail page пользователь видит Markdown задачи, но для начала работы,
Ask или Clarify возвращается в Task Workspace либо открывает Agent Console
вручную.

В Task Workspace одновременно выводится несколько отдельных кнопок.

Для `Ask` используется `window.prompt()`.

<!-- toudocu:section after -->
### Станет

Каждая canonical task page в loopback `serve` получает Task Action Bar.

Пример для `ready`:

```text
TASK-X
Ready · High

[Взять в работу] [Спросить] [Уточнить] [...]
```

Пример для `in-progress` без active Agent Session:

```text
[Продолжить работу] [Спросить] [Уточнить] [...]
```

Пример при active Agent Session этой задачи:

```text
[Открыть агента] [Спросить] [Уточнить] [...]
```

Board/List/Tree остаются обзорными поверхностями и показывают компактный primary
action плюс overflow вместо длинного набора одинаковых кнопок.

<!-- toudocu:section scope -->

## Область изменения

* Go presentation для work-item document page;
* `internal/app/task_site.go`;
* presentation-safe task action bootstrap;
* `web/src/features/task-workspace/`;
* Portal soft-navigation lifecycle;
* стили и локализация;
* browser tests.

<!-- toudocu:section out-of-scope -->

## Не входит в задачу

* перенос Task Workspace на React;
* SPA;
* React Router;
* отдельная страница Task Studio;
* drag-and-drop status;
* редактирование task Markdown из action bar;
* внешний handoff content;
* Project Terminal;
* новый discussion store;
* собственная task comments система.


## Требования к реализации

### 1. Progressive enhancement

Canonical Markdown остаётся основным содержимым task page.

Go presentation layer добавляет semantic action container и page-local
presentation data только для документа, связанного с `WorkItem`.

Browser не определяет task ID, status, digest или действия парсингом текста
Markdown или DOM.

Без JavaScript task page остаётся обычной читаемой страницей документации.

### 2. Shared task action controller

Выделить общий frontend controller:

```text
web/src/features/task-actions/
```

Он отвечает за:

* загрузку authoritative action projection;
* выполнение action;
* выбор delivery;
* текстовый input;
* typed error presentation;
* открытие Agent Console;
* обновление action projection после изменения task state.

Task Workspace не должен содержать собственный HTTP implementation task actions.

Board/List/Tree и task page используют один controller/API client.

### 3. Primary action

Server projection определяет порядок и primary action.

Frontend не содержит `switch(workspaceState)` для выбора бизнес-действия.

Ожидаемое primary поведение:

* `ready` → `start-work`;
* `in-progress` → `continue-work` либо `open active agent`;
* `waiting` → `explain-blocker`;
* `needs-attention` → `clarify`;
* `draft` → `clarify`;
* `blocked` → `explain-blocker`.

Для `done`, `cancelled`, `archive` destructive или misleading primary action не
показывать.

### 4. Delivery chooser

Если action поддерживает несколько delivery modes, primary button использует
предпочтительный доступный вариант, а adjacent menu позволяет выбрать другой.

Например:

```text
[Взять в работу] [▼]

В Agent Console · Codex
Скопировать для внешнего агента
```

Provider name является presentation текущего Agent Console setup, а не частью
action ID.

Если Agent Console недоступна, кнопка handoff остаётся доступной.

Если Agent Console занята другой задачей, соответствующий option disabled с
готовой серверной причиной; handoff не блокируется.

### 5. Ask composer

Удалить `window.prompt()`.

`Ask` открывает нормальный modal/dialog:

```text
Спросить о TASK-X

[ textarea ]

[В Agent Console]
[Для внешнего агента]
```

Использовать стандартный доступный dialog pattern:

* корректный focus trap;
* Escape;
* восстановление focus после закрытия;
* label для textarea;
* submit по явному действию;
* состояние pending;
* отображение typed server error.

Не вводить новый UI framework только ради dialog.

### 6. Task Workspace

Board:

* primary action показывается непосредственно;
* дополнительные действия находятся в overflow menu.

List:

* action column остаётся компактной;
* не выводить ряд из нескольких кнопок.

Tree:

* actions не должны визуально конкурировать с hierarchy;
* primary/overflow располагаются после основной task information.

Все представления используют одинаковый action state.

### 7. Обновление после mutation

После успешного `start-work` browser не меняет status оптимистически.

Он получает новый authoritative task/action snapshot и обновляет UI из него.

Task Workspace и открытая task page после soft navigation должны показывать
одинаковое состояние.

Если документ был изменён извне, stale digest приводит к typed conflict и
reload authoritative projection.

### 8. Agent Console integration

После успешной console delivery:

* Agent Console открывается или получает focus;
* task page остаётся текущей основной страницей;
* soft navigation не завершает session.

Если session уже принадлежит этой задаче, `Открыть агента` только открывает
существующую sidebar и не отправляет дополнительный prompt.

<!-- toudocu:section acceptance-criteria -->

## Критерии приёмки

* [ ] `AC-01` Каждая canonical work-item page в loopback `serve` показывает
  server-resolved Task Action Bar.
* [ ] `AC-02` Frontend не вычисляет допустимые actions из machine status,
  readiness или dependencies.
* [ ] `AC-03` Task Workspace и task detail используют один HTTP client/controller
  и не содержат дублирующих action handlers.
* [ ] `AC-04` Primary action соответствует server projection, дополнительные
  действия доступны через компактный overflow.
* [ ] `AC-05` `Ask` использует доступный dialog/composer вместо
  `window.prompt()`.
* [ ] `AC-06` Active Agent Session этой задачи отображается как
  `Открыть агента`; новая session или duplicate continue prompt не создаются.
* [ ] `AC-07` Session другой задачи блокирует только Agent Console delivery и
  не блокирует остальные разрешённые delivery modes.
* [ ] `AC-08` После mutation UI перечитывает authoritative action/task snapshot,
  а stale digest корректно обрабатывается как conflict.
* [ ] `AC-09` Soft navigation не оставляет stale listeners, duplicate handlers
  или duplicate dialogs.
* [ ] `AC-10` Keyboard navigation, focus management и screen-reader labels
  соответствуют доступному dialog/menu pattern.

<!-- toudocu:section plan -->

## План

1. Добавить task action presentation data на work-item page.
2. Выделить shared task-actions controller.
3. Перевести Task Workspace с прямых fetch handlers на controller.
4. Добавить Task Action Bar.
5. Добавить primary + overflow presentation.
6. Заменить `window.prompt()` на Ask composer.
7. Подключить active-session presentation.
8. Проверить soft navigation и accessibility.

<!-- toudocu:section verification -->

## Проверка

* `AC-01` → `make web-check && make browser-test`
* `AC-02` → `make web-check && make browser-test`
* `AC-03` → `make web-check && make browser-test`
* `AC-04` → `make web-check && make browser-test`
* `AC-05` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task ask'`
* `AC-06` → `go test ./internal/app -run 'TestTaskActionsHTTP|TestTaskActionSessionBinding' && make browser-test`
* `AC-07` → `go test ./internal/app -run 'TestTaskActionsHTTP|TestTaskActionSessionBinding' && make browser-test`
* `AC-08` → `go test ./internal/app -run 'TestTaskActionsHTTP|TestTaskActionSessionBinding' && make browser-test`
* `AC-09` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task actions soft navigation'`
* `AC-10` → `cd web && npx playwright test tests/browser/runtime.spec.ts --grep 'task action accessibility'`
* `QUALITY` → `make check`
* `ALL` → `make check && make browser-test`
* `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->

## Влияние на документацию

Обновить:

* Task Workspace guide;
* local workflow;
* `UC-AGENT-CONSOLE-01`;
* `MOD-SITE`;
* frontend runtime boundary;
* screenshots/описание task workflow при необходимости.

Документация должна считать task detail page основной рабочей поверхностью
конкретной задачи, а Task Workspace — поверхностью поиска, выбора и обзора.
