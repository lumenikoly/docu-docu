<!-- toudocu
id: TASK-SITE-013
status: ready
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-011
-->

# TASK-SITE-013: Перевести элементы Roadmap на остров React

<!-- toudocu:section result -->
## Результат

Операция «Добавить результат» на Roadmap использует автоматически монтируемый
React island и Base UI при неизменном ограниченном контракте записи.

<!-- toudocu:section scope -->
## Область изменения

- остров Roadmap, форма и диалог в `web/`;
- точка монтирования в `internal/site/`, тесты и документы в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- новые возможности записи и изменение `suggestedId`, `digest`,
  `expectedDigest`, `stale_digest`, `roadmap-add` или `X-Toudocu-Action`.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Автоматически монтируемый остров загружает состояние и
  сохраняет ограниченный контракт записи без новых операций.
- [ ] `AC-02` Проверка ввода, успешное добавление и устаревший `digest` понятны;
  состояние формы сохраняется при исправимой ошибке.
- [ ] `AC-03` Диалог доступен с клавиатуры, корректно управляет фокусом и
  работает на мобильном размере.
- [ ] `AC-04` Повторное монтирование при мягкой навигации не оставляет
  повторный корень, обработчики или устаревшую форму; DOM-реализация удалена.

<!-- toudocu:section plan -->
## План

1. Перенести текущую форму и состояния в React/Base UI.
2. Подключить автоматически монтируемый остров к неизменному контракту конечной точки.
3. Проверить ошибки, доступность, мобильный режим и повторное монтирование;
   удалить DOM-реализацию.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./... && npm --prefix web test`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Процесс и экран Roadmap, MOD-SITE и граница времени выполнения описывают
автоматически монтируемый остров при неизменном контракте записи.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача сохраняет существующую операцию Roadmap и меняет только её реализацию в
интерфейсе.
