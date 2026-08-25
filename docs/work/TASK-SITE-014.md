<!-- toudocu
id: TASK-SITE-014
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-010
-->

# TASK-SITE-014: Перевести рабочую область Editor на React

<!-- toudocu:section result -->
## Результат

Оболочка рабочей области Editor реализована на React/Base UI, а безопасная
серверная часть и семантика CodeMirror 6 полностью сохранены.

<!-- toudocu:section scope -->
## Область изменения

- оболочка рабочей области и состояния в `web/`;
- интеграция Editor в `internal/site/`, тесты и документы в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- переписывание CodeMirror 6, Editor HTTP API, SHA-256 CAS, атомарного
  сохранения, семантики предпросмотра, файловых ограничений и зависимость от
  IslandHost.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` React управляет оболочкой, деревом файлов, панелью инструментов,
  вкладками, конфликтами, диалогом создания, диагностикой, уведомлениями и
  адаптивным состоянием.
- [ ] `AC-02` CodeMirror остаётся императивной поверхностью внутри компонента
  React и корректно освобождает ресурсы.
- [ ] `AC-03` HTTP API, конфликты CAS, атомарное сохранение, предпросмотр и
  файловые ограничения сохраняют поведение и защитные проверки.
- [ ] `AC-04` Клавиатура, фокус, ошибки, создание и мобильный режим покрыты
  тестами; прежняя оболочка удалена.

<!-- toudocu:section plan -->
## План

1. Обернуть CodeMirror стабильным жизненным циклом React.
2. Перенести оболочку и состояние без изменения транспорта и серверной части.
3. Проверить сохранение, конфликт, предпросмотр, создание и доступность; удалить
   прежнюю оболочку.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web test && npm --prefix web run test:browser`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `go test ./... && npm --prefix web run test:browser`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Документы Editor, компоненты времени выполнения, MOD-SITE и руководство по
разработке разделяют оболочку React, императивный CodeMirror и серверный контракт.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача сохраняет существующий сценарий Editor и меняет только его оболочку.
