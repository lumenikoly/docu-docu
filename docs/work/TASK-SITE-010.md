<!-- toudocu
id: TASK-SITE-010
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-009
-->

# TASK-SITE-010: Ввести общую основу React/Base UI

<!-- toudocu:section result -->
## Результат

В Toudocu существует проверяемый повторно используемый слой React 19 и Base UI,
не привязанный к конкретной крупной поверхности.

<!-- toudocu:section scope -->
## Область изменения

- зависимости React/Base UI, `ui`, `docs-ui`, тесты и лёгкая страница
  разработки компонентов в `web/`;
- границы общего слоя в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- миграция прикладных поверхностей, Storybook, рабочее пространство npm и преждевременное
  выделение пакетов.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Подключены React 19, React DOM, Base UI, Vitest, React Testing
  Library и user-event с зафиксированными версиями и лицензиями.
- [ ] `AC-02` `ui` предоставляет Button, IconButton, Badge, Separator,
  Spinner, EmptyState, Diagnostic, Dialog, Tabs, Tooltip, Popover, Menu и
  Select; сложные элементы используют Base UI.
- [ ] `AC-03` `ui` не знает `PageBootstrap`, HTTP и ключи локализации;
  `docs-ui` получает переводчик и модель представления через параметры, не
  делает запросы к серверной части и не читает `window.ToudocuPage`.
- [ ] `AC-04` `web/dev/ui.html` заменяет Storybook; тесты покрывают
  клавиатуру, фокус, Escape, вкладки, диалоги, меню, подписи и уменьшение
  движения.

<!-- toudocu:section plan -->
## План

1. Подключить минимальные рабочие и тестовые зависимости.
2. Создать границы `ui`/`docs-ui` и общие элементы.
3. Добавить страницу разработки и тесты компонентов.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web run build`
- `AC-02` → `npm --prefix web test`
- `AC-03` → `npm --prefix web run typecheck && npm --prefix web test`
- `AC-04` → `npm --prefix web test && npm --prefix web run test:browser`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Руководство по разработке и границы модулей объясняют зависимости,
`ui`/`docs-ui`, локализацию и страницу разработки.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача создаёт общий компонентный фундамент без нового пользовательского пути.
