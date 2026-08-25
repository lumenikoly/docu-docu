<!-- toudocu
id: TASK-SITE-017
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-016
-->

# TASK-SITE-017: Завершить миграцию и удалить прежнюю UI-основу

<!-- toudocu:section result -->
## Результат

В рабочей версии существует только новая архитектура браузерной части;
временные мосты и прежняя UI-основа удалены, а `static`, `serve` и выпуск проходят
полный интеграционный цикл.

<!-- toudocu:section scope -->
## Область изменения

- удаление прежней реализации в `web/` и `internal/site/`;
- полная проверка через `Makefile` и итоговая документация в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- новые возможности, сохранение мостов совместимости и документация переходного
  состояния.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Удалены вспомогательные DOM-компоненты, ручные
  Dialog/Tabs/Tooltip/Menu, неиспользуемые селекторы, прежние переменные,
  Unicode-иконки интерфейса, устаревшие CSS, сборка приложения через esbuild и
  мосты совместимости.
- [x] `AC-02` `static` в корне и во вложенном пути не загружает React на
  обычной странице; поиск, темы и Mermaid работают.
- [x] `AC-03` `serve` проходит мягкую навигацию, жизненный цикл островов,
  Roadmap, Discussions, Editor, Changes, проверку обновлений и изоляцию
  переводов и API Docs.
- [x] `AC-04` Бинарный файл без Node.js выполняет `check`, `build` и
  `serve`; документация описывает только итоговую архитектуру.

<!-- toudocu:section plan -->
## План

1. Удалить остатки прежней основы и мосты совместимости.
2. Проверить критические пути `static` и `serve`.
3. Выполнить проверку выпуска без Node.js.
4. Обновить архитектуру, модули, руководство, справочник и затронутые
   типизированные документы.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `make web-check && git diff --exit-code -- internal/site/assets/generated`
- `AC-02` → `make browser-test`
- `AC-03` → `make browser-test && go test ./...`
- `AC-04` → `go test ./... -run TestReleasedBinaryWithoutNodeRuntime && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `make check && make web-check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Обновляются граница браузерной среды, компоненты времени выполнения, MOD-SITE,
руководство по разработке, каталог возможностей и все затронутые сценарии,
процессы, экраны и контракты; переходные оговорки удаляются.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача завершает техническую миграцию существующих сценариев и не добавляет новый пользовательский путь.
