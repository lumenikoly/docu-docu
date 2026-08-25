<!-- toudocu
id: TASK-SITE-015
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-012
-->

# TASK-SITE-015: Перевести рабочую область Changes на React

<!-- toudocu:section result -->
## Результат

Оболочка рабочей области Changes работает на React/Base UI и переиспользует
новый интерфейс Discussions, сохраняя вычисление Git и различий на стороне Go.

<!-- toudocu:section scope -->
## Область изменения

- элементы управления, список файлов, оболочка деталей и интерфейс проверки в `web/`;
- интеграция Changes в `internal/site/`, тесты и документы в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- перенос Git, семантического и отрисованного сравнения в React, переписывание
  CodeMirror Merge и изменение Changes API, якорей проверки или семантики Git.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` React управляет диапазоном, фильтрами, списком файлов, оболочкой
  деталей, уведомлениями, мобильной панелью файлов, редактором проверки, выбором
  связанного файла и сообщениями о состоянии.
- [ ] `AC-02` Changes переиспользует компоненты и API Discussions из
  TASK-SITE-012 без второй реализации.
- [ ] `AC-03` `ChangeSetReport`, Changes API, семантическое и отрисованное
  сравнение, CodeMirror Merge, якоря проверки и семантика Git сохраняются.
- [ ] `AC-04` Диапазоны, фильтры, выбор файла, проверка, ошибки, мобильный режим
  и очистка покрыты тестами; прежняя оболочка удалена.

<!-- toudocu:section plan -->
## План

1. Перенести оболочку и локальное состояние в React.
2. Встроить общий интерфейс Discussions и сохранить API и модель представления.
3. Проверить сравнение, проверку изменений, мобильный режим и очистку; удалить
   DOM-оболочку.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web test && npm --prefix web run test:browser`
- `AC-02` → `npm --prefix web test`
- `AC-03` → `go test ./... && npm --prefix web run test:browser`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Документы Changes и проверки, компоненты времени выполнения и MOD-SITE
разделяют оболочку React и неизменные контракты Go, Git и сравнения.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача сохраняет существующий сценарий Changes и меняет его браузерную реализацию.
