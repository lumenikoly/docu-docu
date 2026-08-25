<!-- toudocu
id: TASK-SITE-008
status: ready
taskType: maintenance
priority: high
module: MOD-SITE
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-24
parentTask: TASK-SITE-006
dependsOn: TASK-SITE-007
-->

# TASK-SITE-008: Перевести сборку браузерной части и ресурсов на Vite 8

<!-- toudocu:section result -->
## Результат

Vite 8 собирает браузерные ресурсы вместо сборки приложения через esbuild,
сохраняя цепочку «исходники → рабочие ресурсы → зафиксированные сгенерированные
ресурсы → `go:embed` → один бинарный файл» и поведение Portal.

<!-- toudocu:section scope -->
## Область изменения

- зависимости, конфигурация Vite и скрипты упаковки в `web/`;
- ресурсы в `internal/site/`, команды в `Makefile`, CI и упаковка выпуска в
  `.github/`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- перенос интерфейса на React, визуальный редизайн, Editor, Changes,
  Discussions и Roadmap.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [ ] `AC-01` Vite 8 собирает несколько точек входа через
  `build.rolldownOptions` и создаёт воспроизводимый манифест Vite.
- [ ] `AC-02` Отдельный `package-assets.mjs` вычисляет транзитивное замыкание
  ресурсов для `static` и `serve`, создаёт манифест Toudocu, SHA-256 и JSON лицензий,
  сохраняя уведомления Mermaid и Swagger UI.
- [ ] `AC-03` Ресурсы `static` и `serve` изолированы, вложенный URL не зависит от
  абсолютных `/assets/...`, а `appearance.js` подключается раньше CSS.
- [ ] `AC-04` Сгенерированные ресурсы воспроизводимы, прежняя сборка
  приложения через esbuild удалена, а выпуск и работа программы не требуют
  Node.js.

<!-- toudocu:section plan -->
## План

1. Ввести сборку нескольких точек входа через Vite.
2. Вынести упаковку, замыкание ресурсов, лицензии и хеши.
3. Перевести встраивание, Makefile, CI и упаковку выпуска; удалить прежний
   скрипт сборки.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `npm --prefix web run build`
- `AC-02` → `make web-check`
- `AC-03` → `make browser-test`
- `AC-04` → `make web && git diff --exit-code -- internal/site/assets/generated && make build`
- `ALL` → `make web-check && make browser-test && make build`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `go vet ./... && go test ./... && go test -race ./... && make web-check`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

Архитектура ресурсов, руководство по разработке, упаковка выпуска и уведомления
о лицензиях описывают Vite и новый манифест.

<!-- toudocu:section use-case-omission-reason -->
## Обоснование отсутствия сценария

Меняется инфраструктура сборки при сохранении пользовательского поведения.
