<!-- toudocu
id: TASK-SITE-009
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-008
-->

# TASK-SITE-009: Ввести семантическую дизайн-систему и единый контракт иконок

<!-- toudocu:section result -->
## Результат

Portal, Editor и Changes используют единый семантический визуальный контракт;
внешний вид пока может оставаться близким к текущему.

<!-- toudocu:section scope -->
## Область изменения

- токены, темы, типографика, движение, компоновка и иконки в `web/`;
- контракт дизайн-системы в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- полный визуальный редизайн, удаление всех прежних иконок и публикация
  дизайн-системы как npm-пакетов.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Переменные `--td-*` покрывают поверхности, текст, границы,
  акцент, статусы, фокус, выделение, интервалы, радиусы, высоту элементов и
  движение.
- [ ] `AC-02` Сохранены роли шрифтов body/interface/heading/mono, темы
  classic/paper/terminal, цветовые схемы system/light/dark и плотность
  compact/comfortable.
- [ ] `AC-03` Тема и плотность меняют семантические токены без отдельных
  палитр для конкретных возможностей.
- [ ] `AC-04` Единый локальный контракт SVG-иконок на основе Lucide задаёт
  толщину линий, доступные подписи и `aria-hidden` для декоративных иконок.

<!-- toudocu:section plan -->
## План

1. Ввести минимальные семантические шкалы и существующие варианты оформления.
2. Перевести общие стили на `--td-*` без редизайна.
3. Добавить реестр иконок и контракт доступности.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `make web-check`
- `AC-02` → `make browser-test`
- `AC-03` → `make web-check && make browser-test`
- `AC-04` → `make web-check && make browser-test`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Руководство по разработке и MOD-SITE описывают токены, темы, плотность, роли
шрифтов и иконки.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача создаёт общий визуальный контракт без самостоятельного пользовательского сценария.
