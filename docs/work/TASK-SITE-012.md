<!-- toudocu
id: TASK-SITE-012
status: done
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-011
-->

# TASK-SITE-012: Перевести Discussions на отложенно активируемый остров React

<!-- toudocu:section result -->
## Результат

Интерфейс Discussions и Agent Feedback использует React/Base UI, но
каноническая страница не загружает React до первого реального открытия
обсуждений.

<!-- toudocu:section scope -->
## Область изменения

- DiscussionPanel, DiscussionComposer, DiscussionState и отложенная активация в `web/`;
- точка монтирования в `internal/site/`, тесты и документы в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- изменение семантики серверной части для Discussion, AgentDelivery, якорей,
  состояния сообщений, аренд и ответов; новый источник истины в браузере и
  полная миграция Changes.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Discussions загружается и монтируется только после реального открытия.
- [x] `AC-02` Компоненты сохраняют семантику серверной части, состояния,
  якоря, аренды, ответы и ошибки без второго источника истины.
- [x] `AC-03` Компоненты и API пригодны для повторного использования внутри Changes.
- [x] `AC-04` Клавиатура, фокус, закрытие, повторное открытие и мягкая
  навигация не оставляют устаревшее состояние или обработчики.

<!-- toudocu:section plan -->
## План

1. Выделить повторно используемые модель представления и компоненты обсуждений.
2. Подключить отложенный остров к существующему серверному контракту.
3. Проверить жизненный цикл, доступность и API повторного использования.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web run test:browser`
- `AC-02` → `go test ./... && npm --prefix web test`
- `AC-03` → `npm --prefix web run typecheck && npm --prefix web test`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Документы Agent Feedback, граница времени выполнения и MOD-SITE отражают
отложенную активацию и повторно используемый API Discussions.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача сохраняет существующий сценарий обсуждений и меняет только браузерную реализацию.
