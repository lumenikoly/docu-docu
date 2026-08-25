<!-- toudocu
id: TASK-SITE-007
status: done
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-25
parentTask: TASK-SITE-006
-->

# TASK-SITE-007: Зафиксировать архитектуру новой основы интерфейса

<!-- toudocu:section result -->
## Результат

Принято архитектурное решение, которое разрешает React 19, Base UI 1 и Vite 8
и фиксирует их границы относительно Go Core, presentation layer, статического
Portal и режима `serve`.

<!-- toudocu:section scope -->
## Область изменения

- ADR об островах React и Vite, архитектурная карта, MOD-SITE и руководство по
  разработке браузерной части в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- рабочий код, браузерная сборка, визуальный редизайн и новый контракт времени
  выполнения.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` ADR закрепляет документационную и прикладную модель, маршрутизацию,
  `PageBootstrap` и решения безопасности за Go Core, а безопасный семантический
  HTML — за Go presentation code без обязательного рефакторинга пакетов.
- [x] `AC-02` `appearance.ts`, `portal.ts` и `serve.ts` остаются
  независимыми от фреймворка; React ограничен островами, Editor и Changes, а
  Base UI — сложными элементами взаимодействия.
- [x] `AC-03` Решение запрещает превращать Portal в SPA и добавлять в него React Router, задаёт контракт IslandHost,
  сохраняет `PageBootstrap v1`, интерфейс на основе возможностей, общую локализацию,
  вложенные URL, изоляцию ошибок islands и использование Vite только при сборке.
- [x] `AC-04` `docs/architecture/overview.md`, граница времени выполнения
  браузерной части, MOD-SITE и руководство по разработке согласованы с ADR-008
  и границами `design/ui/docs-ui`.

<!-- toudocu:section plan -->
## План

1. Создать ADR-008 с контекстом, решением и последствиями.
2. Обновить границу времени выполнения браузерной части и обзор архитектуры.
3. Согласовать MOD-SITE и руководство по разработке браузерной части.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-02` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-03` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `AC-04` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Создан ADR-008; обновлены граница времени выполнения браузерной части,
MOD-SITE, обзор архитектуры и руководство по разработке.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача принимает техническое архитектурное решение и не меняет самостоятельный пользовательский путь.
