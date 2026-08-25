<!-- toudocu
id: TASK-SITE-016
status: ready
taskType: maintenance
priority: normal
module: MOD-SITE
standards: STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-013, TASK-SITE-014, TASK-SITE-015
-->

# TASK-SITE-016: Выполнить целевой визуальный редизайн

<!-- toudocu:section result -->
## Результат

Новая дизайн-система становится видимым единым интерфейсом профессионального
инструмента разработчика: плотным, но читаемым, с тонкими границами,
ограниченными тенями, умеренными радиусами, одним акцентом и приоритетом
клавиатурного управления.

<!-- toudocu:section scope -->
## Область изменения

- общее оформление, компоненты и состояния Portal, Editor и Changes в `web/`;
- описание визуального языка в `docs/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- изменение архитектуры, API или возможностей и типовая панель SaaS.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Заголовок, навигация, боковая панель, глобальный поиск, кнопки,
  списки выбора, вкладки, диалоги, формы и статусы используют единый визуальный язык.
- [ ] `AC-02` Task Workspace, Roadmap, Discussions, Editor и Changes
  согласованы, а Portal остаётся ориентированным на содержимое.
- [ ] `AC-03` Диагностика и состояния загрузки, пустого результата и ошибки
  понятны, доступны с клавиатуры и не зависят только от цвета.
- [ ] `AC-04` Мобильные состояния, темы, плотность и уменьшение движения проходят
  браузерные тесты без изменения поведения.

<!-- toudocu:section plan -->
## План

1. Применить типографику, компоновку, границы и акцент к общему оформлению.
2. Согласовать элементы управления и состояния перенесённых поверхностей.
3. Проверить темы, плотность, клавиатуру, уменьшение движения и мобильный режим.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web run test:browser -- --grep 'visual language'`
- `AC-02` → `npm --prefix web run test:browser -- --grep 'Portal and workspaces'`
- `AC-03` → `npm --prefix web run test:browser`
- `AC-04` → `npm --prefix web run test:browser`
- `ALL` → `make web-check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make web-check && go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Руководство по разработке и описания экранов отражают итоговый визуальный язык,
доступность и различие Portal и рабочих областей.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Задача согласует представление существующих сценариев и не добавляет новый путь.
