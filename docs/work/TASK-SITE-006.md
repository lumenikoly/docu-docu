<!-- toudocu
id: TASK-SITE-006
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
-->

# TASK-SITE-006: Новая UI-система Toudocu

<!-- toudocu:section result -->
## Результат

Toudocu использует единый долгосрочный фундамент интерфейса: Go создаёт
статический HTML, семантическая дизайн-система задаёт визуальный контракт,
React реализует прикладные поверхности, Base UI — сложные элементы
взаимодействия, а Vite собирает браузерные ресурсы. Portal не становится SPA,
а `appearance.ts`, `portal.ts` и `serve.ts` не зависят от React.
`PageBootstrap schema v1`, мягкая навигация, общие каталоги локализации и
размещение во вложенном URL сохраняются. Обычные страницы не загружают React,
а выпущенный бинарный файл не требует Node.js.

<!-- toudocu:section scope -->
## Область изменения

- координация миграции `web/` и встраивания ресурсов через `internal/site/`;
- итоговая документация в `docs/` и интеграционные проверки из `Makefile`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- SPA, React Router и перенос правил Markdown, маршрутизации или безопасности из Go;
- Node.js как зависимость пользователя во время работы.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Все непосредственные дочерние задачи завершены, а старая и новая UI-основы не существуют параллельно.
- [x] `AC-02` Обычная статическая страница не загружает React; Portal
  работает в корне и во вложенном URL.
- [x] `AC-03` Выпущенный бинарный файл выполняет `check`, `build` и
  `serve` без Node.js.
- [x] `AC-04` Документация описывает итоговую архитектуру; `make check`,
  браузерные тесты и проверка выпуска проходят.

<!-- toudocu:section plan -->
## План

1. Завершить архитектурный, сборочный и общий UI-фундамент.
2. Завершить ветви островов, Editor и Changes.
3. Применить визуальный язык, удалить прежнюю основу и сверить документацию.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go run ./cmd/toudocu task tree TASK-SITE-006 ./docs --repository-root . && make web-check && make browser-test`
- `AC-02` → `make browser-test`
- `AC-03` → `go test ./... -run TestReleasedBinaryWithoutNodeRuntime`
- `AC-04` → `make check && make browser-test && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `make check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

После завершения дерева архитектурные, модульные, справочные и пользовательские документы описывают только итоговую UI-систему. Черновик остаётся входом в декомпозицию, а не источником текущего состояния.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача координирует техническую миграцию нескольких существующих поверхностей и не вводит отдельный пользовательский сценарий.
