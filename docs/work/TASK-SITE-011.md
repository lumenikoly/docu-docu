<!-- toudocu
id: TASK-SITE-011
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-010
-->

# TASK-SITE-011: Интегрировать острова React с мягкой навигацией

<!-- toudocu:section result -->
## Результат

Канонический `serve` безопасно монтирует автоматически и отложенно
активируемые острова React и меняет страницы через существующую мягкую
навигацию без утечек корней React и побочных эффектов.

<!-- toudocu:section scope -->
## Область изменения

- независимый от фреймворка IslandHost и жизненный цикл навигации в `web/`;
- точки монтирования в `internal/site/`, тесты и документация времени
  выполнения в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- React Router, новая версия `PageBootstrap` и миграция отдельной прикладной
  поверхности.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Независимый от фреймворка IslandHost выполняет `discover`,
  `activate`, `mount`, `unmount` и `unmountAll` и допускает не более
  одного корня React на экземпляр острова.
- [ ] `AC-02` После проверки целевой страницы отправляется
  `toudocu:pagebeforechange`, выполняется `unmountAll`, меняются компоновка
  и данные начальной загрузки, затем отправляется `toudocu:pagechange` и монтируются
  автоматически активируемые острова.
- [ ] `AC-03` Сохранены ограничение кеша, история, восстановление прокрутки,
  полный переход при ошибке, `PageBootstrap v1`, `window.ToudocuPage`,
  возможности, конечные точки и относительные пути.
- [ ] `AC-04` A→B, A→B→A и A→B→A→B не оставляют повторный корень или
  обработчик, устаревший диалог или данные начальной загрузки либо
  незавершённый `AbortController`.

<!-- toudocu:section plan -->
## План

1. Реализовать IslandHost без статического импорта React.
2. Добавить границу применения навигации и события жизненного цикла.
3. Проверить обнаружение, повторное монтирование, очистку и полный переход.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web test`
- `AC-02` → `npm --prefix web run test:browser`
- `AC-03` → `go test ./internal/site/... && npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `go test ./... && make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Граница времени выполнения браузерной части и MOD-SITE описывают IslandHost,
автоматическую и отложенную активацию и порядок применения мягкой навигации.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача меняет общий жизненный цикл существующей навигации без нового
пользовательского сценария.
