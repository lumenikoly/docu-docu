# Разработка интерфейса

Исходники TypeScript и CSS находятся в `web/`. Общие с серверным HTML
текстовые каталоги находятся в `internal/site/i18n/{en,ru}.json`, а серверные
оболочки страниц и рабочих поверхностей — в `internal/site/templates/`.
Небольшие серверные компоненты остаются в Go-коде представлений. В каталогах
нельзя хранить HTML: они содержат только текст.

Node.js нужен только разработчику, который меняет интерфейс. Обычный
пользователь и `go build ./...` используют уже собранные ресурсы из
`internal/site/assets/generated/`.

## Принятый цикл

Из корня репозитория:

```bash
make web
make web-check
make test
make build
```

Только для интерфейса, из `web/`:

```bash
npm ci
npm run typecheck
npm test
npm run build
npm run test:browser
```

Этот короткий цикл не заменяет Go vet, обычные и race-тесты и сборку бинарника.
Соответствующие цели Make выполняют их отдельно.

TypeScript работает в строгом режиме, а несколько точек входа JavaScript и CSS
собирает Vite 8. После Vite отдельный `package-assets.mjs` вычисляет по его
манифесту замыкания ресурсов для `static` и `serve`, добавляет закреплённые
Mermaid и Swagger UI и создаёт манифест Toudocu с SHA-256. Сборка также
сохраняет `vite-manifest.json` и `licenses.json` для проверки графа и лицензий.

Имена и содержимое производных ресурсов воспроизводятся: время и случайные
значения в них запрещены. После изменения `web/` нужно закоммитить обновлённый
`internal/site/assets/generated/`; CI повторит сборку и обнаружит расхождение.

## Что входит в какой режим

- `appearance.js` и `portal.js` нужны и статическому порталу, и `serve`;
- `appearance.js` загружается до CSS, чтобы сохранённая тема действовала с
  первого кадра;
- `serve.js`, `editor.js`, `changes.js`, диалог дорожной карты, CodeMirror и
  Swagger UI доступны только при `serve`.

Целевая миграция сохраняет `appearance.ts`, `portal.ts` и `serve.ts` без
статической зависимости от React, React DOM и Base UI. Editor и Changes
становятся отдельными React application roots. Roadmap загружается как eager
island, а Discussions — только после явного действия пользователя как activated
island. Обычная страница Portal не загружает React.

## Границы browser-кода

Исходники целевой системы делятся на внутренние каталоги `design`, `ui`,
`docs-ui`, `features` и `entries`; отдельные npm-пакеты и workspaces не нужны.
Импорты направлены только вверх по этому списку:

- `ui` использует `design`;
- `docs-ui` использует `ui` и `design`;
- `features` используют нижние UI-слои и разрешённую browser infrastructure из
  `core`;
- `entries` компонуют features и infrastructure.

`design`, `ui` и `docs-ui` не импортируют `PageBootstrap`, API clients,
product events, `window.ToudocuPage` или каталоги через `core`. Generic UI
получает строки через props. Documentation-oriented UI получает view models,
callbacks и `Translator`, но не обращается к endpoints самостоятельно.
Автоматическая проверка импортов защищает эту границу.

Используйте native HTML для обычных кнопок, ссылок, подписей, layout и
статических элементов. Подключайте Base UI, только когда нужен составной
accessibility contract: например dialog, menu, popover, tooltip, select,
combobox или сложные tabs.

Общий React-слой находится в `web/src/ui/`: собственные `Button`, `IconButton`,
`Badge`, `Separator`, `Spinner`, `EmptyState` и `Diagnostic` используют native
HTML, а `Dialog`, `Tabs`, `Tooltip`, `Popover`, `Menu` и `Select` предоставляют
составные элементы Base UI. `web/src/docs-ui/` принимает готовые view model,
callbacks и `Translator`; сетевой доступ остаётся в feature-слое.

Лёгкая галерея компонентов доступна во время `npm run dev` по адресу
`/dev/ui.html`. Она заменяет отдельный Storybook и не входит в ресурсы
выпускаемого портала.

## Визуальный контракт

Общие компоненты используют семантические переменные `--td-*` из
`web/src/styles/tokens.css`. Контракт охватывает поверхности, текст, границы,
акцент, состояния, фокус, выделение, интервалы, радиусы, высоту элементов и
движение. Прежние имена переменных остаются временными совместимыми алиасами к
`--td-*`; темы и новый общий CSS задают семантические переменные и не создают
отдельную палитру для конкретной возможности.

Роли шрифтов заданы переменными `--td-font-body`, `--td-font-interface`,
`--td-font-heading` и `--td-font-mono`. Атрибуты `data-site-theme` со значениями
`classic`, `paper` и `terminal`, `data-theme` для светлой или тёмной схемы и
`data-density` со значениями `comfortable` и `compact` меняют тот же набор
семантических токенов.

Новые общие компоненты берут локальные контуры Lucide из
`web/src/design/icons.ts`; прежние иконки можно переносить постепенно.
Добавляйте новую иконку в реестр и создавайте её через `createIcon`: без подписи она получает
`aria-hidden="true"`, а значимая самостоятельная иконка — `role="img"` и
`aria-label`. В кнопке доступное название принадлежит самой кнопке, поэтому
вложенная иконка остаётся декоративной. Толщина линии едина и задаётся стилем
`.ui-icon`; внешний пакет и загрузка из сети не нужны.

## React roots и islands

Editor и Changes владеют DOM только внутри выделенного root. Доступ к title,
focus, appearance и navigation проходит через ограниченные browser
infrastructure services, а не через произвольное изменение shell. CodeMirror и
CodeMirror Merge сами владеют document, selection, viewport, transactions и
diff state; не копируйте это чувствительное к задержкам состояние в React.

`web/src/core/react/island-host.ts` предоставляет независимые от React операции
`discover`, `activate`, `mount`, `unmount` и `unmountAll`. Registry загружает
feature-модули через dynamic import. Mount point хранит имя в
`data-td-island`, уникальный instance в `data-td-island-instance`, режим
`activated` при отложенном запуске и при необходимости ссылку
`data-td-island-model` на `application/json` с immutable view model. Не
помещайте туда runtime, capabilities, permissions, endpoints, locale, theme
или абсолютные пути. Ошибка одного island получает локальное состояние
`error` и не удаляет полезный Go-generated fallback.
Локализованный fallback размечается внутри mount point атрибутом
`data-td-island-error` и изначально скрыт. Feature регистрирует освобождение
созданного root через `onCleanup` до первого render, чтобы ошибка render не
оставила root для повторной активации.

Мягкая навигация полностью проверяет целевую страницу и `PageBootstrap` до
`toudocu:pagebeforechange`, затем освобождает islands, заменяет layout и
bootstrap, отправляет `toudocu:pagechange` и обнаруживает eager islands новой
страницы. При ошибке до commit boundary текущие roots сохраняются, а браузер
выполняет обычный полный переход.

Используйте `createRoot`, а не React SSR или hydration. Новый island не требует
ADR, пока соблюдает общий lifecycle, capability contract и не получает владение
маршрутизацией или domain model.

Проектную модель, классификацию документов, проверку путей, семантический diff
и решение о запуске команд нельзя переносить в TypeScript: это граница Go.

В Changes все редакторы используют координаты Unicode, начиная с единицы.
Подсветка закреплена для Go, Java, JavaScript, JSX, TypeScript и TSX. Остальные
корректные UTF-8 файлы показываются как обычный текст. Путь, выделенный текст,
контекст, пределы размера и перенос привязок проверяет Go.

В Editor React управляет workspace, а `CodeEditor` остаётся тонким адаптером
CodeMirror: создавайте экземпляр только для текущих `path` и `digest`,
передавайте изменения наружу и обязательно вызывайте `destroy()` в cleanup.
Не переносите selection, viewport, transactions или проверку путей в React;
сохранение, создание, validate и preview должны использовать существующий
Editor HTTP API и его action headers.

## Связанные документы

- [ADR-008: React-острова и Vite без превращения Portal в SPA](../decisions/ADR-008.md)
- [Граница Go и браузера](../architecture/frontend-runtime-boundary.md)
- [MOD-SITE](../modules/site.md)
- [Проверка изменений](testing.md)
