# ТЗ: переход Toudocu на новую UI-систему

## 1. Цель

Перевести frontend Toudocu на новый долгосрочный UI-фундамент, пригодный:

* для дальнейшего развития Portal и `serve`;
* для более сложных интерактивных интерфейсов;
* для единого визуального языка Portal, Editor и Changes;
* для сохранения лёгкого статического портала без превращения его в SPA.

Миграция должна выполняться сразу в целевую архитектуру.

Совместимость со старым внутренним frontend API, CSS-классами, DOM-helper API и generated asset names сохранять не требуется.

---

# 2. Итоговое состояние

После доработки Toudocu должен использовать:

```text
Go
TypeScript
React 19.x
React DOM
Base UI 1.x
Vite 8.x
CodeMirror 6
Lucide icons
Vitest
React Testing Library
Playwright
```

При этом:

* Go остаётся владельцем документационной модели и generated HTML;
* Portal не становится SPA;
* React используется только для stateful application UI;
* `appearance.ts`, `portal.ts` и `serve.ts` остаются framework-free;
* React islands загружаются только при необходимости;
* Editor и Changes могут быть полноценными React workspace applications;
* существующая soft navigation остаётся владельцем canonical `serve` navigation;
* `PageBootstrap schema v1` остаётся runtime-контрактом frontend;
* существующие Go/browser i18n-каталоги остаются единственным источником пользовательских строк;
* production runtime не требует Node.js;
* `toudocu build` продолжает создавать автономный статический портал.

---

# 3. Архитектурные инварианты

## 3.1. Go остаётся владельцем документа

Go продолжает отвечать за:

* Markdown parsing;
* безопасный HTML rendering;
* typed documentation model;
* relations;
* navigation model;
* TOC;
* diagnostics;
* public routes;
* PageBootstrap;
* static portal generation;
* serve backend APIs.

React не должен:

* повторно разбирать Markdown;
* строить альтернативную документационную модель;
* вычислять typed relations независимо от Go;
* становиться источником canonical document HTML.

Целевая цепочка:

```text
Markdown
   ↓
Toudocu Go Core
   ↓
safe HTML + typed model + PageBootstrap
   ↓
Portal / React surfaces
```

---

## 3.2. Portal не становится SPA

Сохранить обычные public HTML routes:

```text
/index.html
/architecture/...
/modules/...
/use-cases/...
/flows/...
/screens/...
/quality/...
/runbooks/...
/work/...
```

Не вводить React Router для Portal.

Статический `build` должен оставаться обычным HTTP-сайтом, не требующим client-side routing.

---

## 3.3. Static HTML и application UI разделены

Использовать native HTML для:

* Markdown content;
* heading;
* ordinary links;
* navigation;
* TOC;
* footer;
* статических таблиц;
* статических metadata blocks;
* простого `<details>`.

Использовать React для:

* dialogs;
* menus;
* popovers;
* сложных tabs;
* Editor;
* Changes;
* Discussions;
* Roadmap write UI;
* сложных forms;
* stateful task UI;
* интерактивных workspace surfaces.

Правило:

> Static content → semantic Go-generated HTML. Stateful application UI → React.

---

# 4. Framework-free runtime entries

Следующие entrypoints обязаны оставаться без статической зависимости от React, ReactDOM и Base UI:

```text
appearance.ts
portal.ts
serve.ts
```

## `appearance.ts`

Отвечает только за раннюю установку:

* theme;
* color scheme;
* accent;
* density;
* appearance attributes.

Он должен выполняться до подключения основного CSS.

React в него не импортируется.

---

## `portal.ts`

Отвечает за лёгкое универсальное enhancement-поведение Portal:

* sidebar;
* глобальный search;
* print;
* простые keyboard interactions;
* theme controls;
* другие дешёвые portal-wide behaviours.

React в него не импортируется.

---

## `serve.ts`

Остаётся infrastructure composition root режима `serve`.

Он отвечает за:

```text
bootstrap
runtime
serve runtime
soft navigation
serve navigation
IslandHost
```

`serve.ts` не является React entrypoint.

---

# 5. React entrypoints

Целевое разделение:

```text
FRAMEWORK-FREE

appearance.ts
portal.ts
serve.ts


CANONICAL SERVE ISLANDS

roadmap.tsx
discussions.tsx


WORKSPACE APPLICATIONS

editor.tsx
changes.tsx


SPECIALIZED SURFACES

screen-map.ts
playable-flow.ts
api-docs.ts
```

`screen-map`, `playable-flow` и `api-docs` не требуется переводить на React только ради унификации.

---

# 6. React islands

Добавить общий framework-free runtime:

```text
core/react/island-host.ts
```

Он не должен статически импортировать React.

Его ответственность:

```text
discover
activate
mount
unmount
unmountAll
registry
```

React feature modules загружаются через dynamic `import()`.

Пример registry:

```ts
const islands = {
  roadmap: () => import("../../features/roadmap/island"),
  discussions: () => import("../../features/discussions/island"),
};
```

Go-generated mount point:

```html
<div
  data-td-island="roadmap"
  data-td-island-instance="roadmap-main"
></div>
```

Гарантия:

```text
1 DOM island instance
=
не более 1 активного React root
```

---

# 7. Eager и lazy islands

Поддержать два режима.

## Eager

Island монтируется сразу при обнаружении mount point.

Использовать для feature, который является частью текущей страницы.

Первый пример:

```text
Roadmap
```

---

## Lazy / activated

React bundle не загружается до первого пользовательского действия.

Использовать для глобальных application surfaces.

Первый пример:

```text
Discussions
```

Пример:

```text
canonical serve page
      ↓
framework-free Discuss trigger
      ↓ click
dynamic import discussions
      ↓
mount React island
```

Таким образом обычная canonical serve page не должна загружать React только потому, что capability `review` доступна.

---

# 8. Soft navigation lifecycle

Сохранить текущую canonical `serve` soft navigation.

Она остаётся владельцем:

* navigation interception;
* prefetch;
* cache;
* history;
* scroll restoration;
* `.site-layout` replacement;
* PageBootstrap replacement;
* page-specific styles;
* focus restoration.

React не реализует собственный router.

---

## 8.1. Новый lifecycle

Добавить внутреннее событие:

```text
toudocu:pagebeforechange
```

Целевой flow:

```text
initial load
    ↓
IslandHost.discover()
    ↓
mount eager islands


soft navigation
    ↓
fetch target page
    ↓
validate target
    ↓
toudocu:pagebeforechange
    ↓
IslandHost.unmountAll()
    ↓
replace .site-layout
    ↓
replace PageBootstrap
    ↓
sync page assets
    ↓
toudocu:pagechange
    ↓
IslandHost.discover()
    ↓
mount eager islands
```

---

## 8.2. Commit boundary

`toudocu:pagebeforechange` отправлять только после успешной предварительной проверки target page.

До него должны быть подтверждены:

```text
valid HTTP HTML response
matching serve revision
canonical serve marker
valid .site-layout
valid PageBootstrap
runtime == serve
```

После `pagebeforechange` navigation считается вошедшей в DOM commit phase.

---

## 8.3. React cleanup

Перед удалением текущей `.site-layout`:

* каждый React root должен получить `root.unmount()`;
* feature effect/listeners должны быть освобождены;
* feature AbortController должен быть отменён;
* feature-owned overlay, portal или dialog должен быть удалён.

Запрещено просто удалять DOM subtree с активным React root.

---

# 9. `PageBootstrap schema v1`

Не проектировать новый runtime config для React.

Сохранить существующий контракт:

```text
schemaVersion
runtime
page
portal
ui
capabilities
endpoints
```

React features используют существующий:

```text
PageBootstrap schema v1
```

и не создают:

```text
ReactConfig
ReactBootstrap
global Redux bootstrap
другой window runtime contract
```

---

## 9.1. Capability-driven UI

Feature availability определяется:

```text
bootstrap.capabilities
```

а backend address:

```text
bootstrap.endpoints
```

DOM mount point отвечает только за наличие места отображения.

Не определять permissions через:

* URL;
* pathname;
* CSS classes;
* случайное наличие DOM element.

Пример:

```text
capability.review
→ feature разрешена

endpoint.review
→ backend destination

data-td-island
→ место отображения
```

---

# 10. i18n

Сохранить:

```text
internal/site/i18n/en.json
internal/site/i18n/ru.json
```

как единственный source of truth пользовательских строк Portal/Serve.

Не добавлять:

* `react-i18next`;
* отдельный React catalog;
* отдельные `web/locales`;
* дублированные product strings.

---

## 10.1. Existing browser catalog

React features используют существующий frontend catalog API.

Feature layer может вызывать:

```ts
text("features.roadmap.title")
```

---

## 10.2. Shared UI

Низкоуровневый `ui` не должен знать Toudocu translation keys.

Generic component получает строки через props:

```tsx
<Dialog closeLabel={closeLabel}>
```

а не:

```tsx
<Dialog closeLabel={text("common.close")}>
```

внутри generic implementation.

---

## 10.3. `docs-ui`

Подготовить generic translator interface:

```ts
type Translator = (
  key: string,
  values?: readonly unknown[],
) => string;
```

Чтобы `docs-ui` не зависел непосредственно от Portal runtime и оставался
пригодным для других интерфейсов, он не привязывается к:

```text
window.ToudocuPage
```

---

# 11. Design system

Создать единый semantic design contract.

Все новые shared CSS variables именовать:

```text
--td-*
```

---

# 12. Font roles

Сохранить существующие роли:

```text
body
interface
heading
mono
```

Использовать:

```css
--td-font-body;
--td-font-interface;
--td-font-heading;
--td-font-mono;
```

Не вводить альтернативные `ui/content` font roles.

Remote fonts запрещены.

---

# 13. Semantic tokens

Минимальный обязательный набор:

```css
--td-canvas;

--td-surface;
--td-surface-subtle;
--td-surface-raised;
--td-surface-overlay;

--td-text;
--td-text-secondary;
--td-text-muted;
--td-text-inverse;

--td-border;
--td-border-subtle;
--td-border-strong;

--td-accent;
--td-accent-hover;
--td-accent-active;
--td-accent-soft;

--td-info;
--td-success;
--td-warning;
--td-danger;

--td-selection;
--td-focus;

--td-font-body;
--td-font-interface;
--td-font-heading;
--td-font-mono;

--td-font-xs;
--td-font-sm;
--td-font-md;
--td-font-lg;
--td-font-xl;
--td-font-2xl;

--td-space-1;
--td-space-2;
--td-space-3;
--td-space-4;
--td-space-5;
--td-space-6;
--td-space-8;

--td-radius-xs;
--td-radius-sm;
--td-radius-md;

--td-control-height-sm;
--td-control-height-md;

--td-duration-fast;
--td-duration-normal;
--td-duration-slow;

--td-ease-standard;
--td-ease-emphasized;
```

---

# 14. Базовые visual scales

## Spacing

Использовать компактную шкалу:

```text
2
4
6
8
12
16
24
32 px
```

---

## Radius

Основная шкала:

```text
xs = 4px
sm = 6px
md = 10px
```

Большие радиусы не использовать как основной visual language.

---

## Shadows

Shadows применять преимущественно для:

* dialog;
* menu;
* popover;
* floating overlay.

Обычные панели разделять:

* border;
* surface;
* spacing.

---

# 15. Visual language

Целевой UI должен выглядеть как профессиональный developer/documentation tool.

Избегать:

* SaaS dashboard aesthetic;
* nested cards;
* больших теней;
* excessive rounded containers;
* декоративных gradients;
* большого числа accent colors;
* избыточной анимации.

Предпочитать:

* плотный layout;
* тонкие separators;
* спокойные surfaces;
* чёткую типографическую иерархию;
* один dominant accent;
* сдержанные status colors;
* хорошие selected/hover/focus states;
* keyboard-first usability.

---

# 16. Themes

Сохранить текущие theme identifiers:

```text
classic
paper
terminal
```

Сохранить:

```text
system
light
dark
```

и текущие accent values.

Themes должны переопределять semantic tokens.

Feature CSS не должен содержать theme-specific palette.

Плохо:

```css
[data-site-theme="paper"] .task-card {
  background: #faf7f2;
}
```

Правильно:

```css
.task-card {
  background: var(--td-surface);
}
```

---

# 17. Density

Сохранить:

```text
compact
comfortable
```

Density должна задавать общие variables:

```text
control heights
row heights
toolbar spacing
panel spacing
```

Feature-specific density hacks минимизировать.

---

# 18. Icon system

Перейти на единый Lucide-based SVG icon contract.

Удалить из application chrome Unicode icons типа:

```text
☰
×
⎙
```

где они используются как UI-icons.

Минимальный набор:

```text
menu
search
print
refresh
sun
moon
monitor
chevron-left
chevron-right
chevron-down
x
check
warning
error
info
edit
file
folder
git
diff
plus
minus
more-horizontal
external-link
copy
settings
play
stop
```

Все icons:

* bundled locally;
* inline SVG;
* единый stroke;
* semantic button имеет accessible label;
* decorative icon имеет `aria-hidden`.

---

# 19. Base UI

Использовать Base UI для сложных interaction primitives:

```text
Dialog
Tabs
Tooltip
Popover
Menu
Context Menu
Select
Combobox
```

Не реализовывать вручную, если Base UI уже предоставляет необходимое поведение:

```text
focus trap
roving tabindex
keyboard navigation
nested menu
typeahead
dialog stacking
popover positioning
```

---

# 20. Собственные primitives

Toudocu сохраняет собственный visual/API layer для:

```text
Button
IconButton
Badge
Separator
Spinner
EmptyState
Diagnostic
Status
```

Они используют design tokens.

---

# 21. Frontend structure

Целевая структура:

```text
web/
├── src/
│   ├── design/
│   │   ├── tokens.css
│   │   ├── themes.css
│   │   ├── typography.css
│   │   ├── motion.css
│   │   ├── layout.css
│   │   └── icons.ts
│   │
│   ├── ui/
│   │   ├── button/
│   │   ├── icon-button/
│   │   ├── badge/
│   │   ├── tabs/
│   │   ├── dialog/
│   │   ├── tooltip/
│   │   ├── popover/
│   │   ├── menu/
│   │   ├── select/
│   │   ├── empty-state/
│   │   ├── diagnostic/
│   │   └── index.ts
│   │
│   ├── docs-ui/
│   │   ├── task/
│   │   ├── use-case/
│   │   ├── flow/
│   │   ├── roadmap/
│   │   ├── relations/
│   │   ├── diagnostics/
│   │   └── index.ts
│   │
│   ├── core/
│   │   ├── bootstrap/
│   │   ├── api/
│   │   ├── events/
│   │   ├── runtime/
│   │   ├── navigation/
│   │   └── react/
│   │       └── island-host.ts
│   │
│   ├── features/
│   │   ├── editor/
│   │   ├── changes/
│   │   ├── discussions/
│   │   ├── roadmap/
│   │   ├── screen-map/
│   │   ├── playable-flow/
│   │   ├── use-case/
│   │   └── api-docs/
│   │
│   └── entries/
│       ├── appearance.ts
│       ├── portal.ts
│       ├── serve.ts
│       ├── editor.tsx
│       ├── changes.tsx
│       ├── roadmap.tsx
│       ├── discussions.tsx
│       ├── screen-map.ts
│       ├── playable-flow.ts
│       └── api-docs.ts
│
├── dev/
│   └── ui.html
│
├── scripts/
│   └── package-assets.mjs
│
├── vite.config.ts
└── package.json
```

Не создавать npm workspace/monorepo без самостоятельной необходимости.

---

# 22. Future package boundaries

Структура должна позволять позже выделить:

```text
@toudocu/design
@toudocu/ui
@toudocu/docs-ui
```

без архитектурного переписывания.

---

## `design`

Не зависит от React.

Владеет:

```text
tokens
themes
typography
icons
motion constants
```

---

## `ui`

Может зависеть только от:

```text
React
Base UI
design
```

Не знает:

```text
Toudocu APIs
repository
tasks
serve
PageBootstrap
```

---

## `docs-ui`

Зависит от:

```text
design
ui
```

Владеет предметными representations.

Не выполняет backend requests напрямую.

---

## `features`

Связывают:

```text
PageBootstrap
API
events
feature state
ui/docs-ui
```

---

# 23. Discussions migration

Перевести Discussions на React/Base UI одним из первых.

Сохранить существующую backend semantics Agent Feedback.

React не становится source of truth для discussion/delivery state.

Использовать общие:

```text
Dialog
Button
Textarea
Badge
Status
EmptyState
```

Canonical-page discussion panel должен быть lazy island.

---

# 24. Roadmap migration

Перевести current stateful Roadmap write UI на React.

Сохранить текущий restricted write contract.

Не менять:

```text
GET roadmap state
expectedDigest
stale_digest handling
roadmap-add action
X-Toudocu-Action
editor endpoint boundary
```

React меняет только presentation/state management.

Не расширять Roadmap write permissions.

Roadmap является eager island только на страницах, где присутствует соответствующий mount point.

---

# 25. Editor migration

CodeMirror 6 сохранить.

Не переписывать editor engine.

Перевести на React:

* workspace chrome;
* toolbar;
* file tree shell;
* view mode controls;
* conflict UI;
* dialogs;
* diagnostics presentation;
* status/toast UI;
* create-document workflow.

CodeMirror остаётся отдельным imperative editor surface внутри React component.

CodeMirror theme должен использовать `--td-*` variables.

---

# 26. Changes migration

Backend Changes semantics не менять.

Разделить frontend на:

```text
API client
state/controller
React workspace UI
CodeMirror merge/diff surface
```

Перевести:

* range controls;
* filters;
* file list;
* detail shell;
* notices;
* review dialogs;
* linked-file picker;
* discussion integration;
* toasts.

HTTP logic не помещать внутрь generic `ui` или `docs-ui`.

---

# 27. Go shell

Portal shell продолжает генерироваться Go/template layer.

Не переносить в React:

```text
site header
navigation
main document layout
TOC
footer
```

Одновременно уменьшить длинную HTML string concatenation там, где она мешает развитию UI.

Typed Go view data и templates должны использоваться вместо сложной строковой сборки по мере миграции.

---

# 28. CSS architecture

Использовать CSS Cascade Layers:

```css
@layer reset, tokens, base, components, features, utilities;
```

Разделить:

```text
reset
semantic tokens
themes
typography
base document styles
shared components
feature styles
utilities
```

Не переносить новый UI в один монолитный `portal.css`.

---

# 29. CSS restrictions

В feature CSS без обоснования запрещены:

* raw theme colors;
* новая spacing scale;
* собственные radius scales;
* копии shared controls;
* глобальные element overrides;
* `!important`.

Raw palette values допустимы внутри theme/accent definitions.

---

# 30. Tailwind

Tailwind CSS в рамках этой миграции в Toudocu Portal не добавлять.

Portal использует:

```text
semantic classes
semantic CSS variables
plain CSS
```

---

# 31. Motion

Motion React dependency в Portal не добавлять.

Для Portal использовать CSS transitions.

Примерные tokens:

```text
fast   ≈ 80ms
normal ≈ 140ms
slow   ≈ 220ms
```

Учитывать:

```text
prefers-reduced-motion
```

---

# 32. Переход с esbuild на Vite

Application frontend bundling перевести на Vite 8.

Старый прямой esbuild bundling удалить после завершения миграции.

Использовать Vite multi-entry build.

---

# 33. Vite config

Использовать:

```text
build.rolldownOptions
```

а не deprecated `build.rollupOptions`.

Logical inputs:

```text
appearance
portal
serve
editor
changes
roadmap
discussions
screen-map
playable-flow
api-docs
```

---

# 34. Vite не генерирует Portal HTML

Production flow:

```text
TypeScript / TSX / CSS
          ↓
        Vite
          ↓
    frontend assets
          ↓
 package-assets.mjs
          ↓
 generated Go assets
          ↓
       Go embed
          ↓
    toudocu binary
```

HTML по-прежнему генерирует Go.

---

# 35. Vite manifest

Включить:

```text
build.manifest = true
```

Vite manifest используется для определения:

```text
entry
transitive JS chunks
CSS
assets
```

---

# 36. Toudocu frontend manifest

После Vite build `package-assets.mjs` создаёт собственный Toudocu manifest.

Он отделяет Toudocu runtime contract от внутренних деталей Vite.

Содержит как минимум:

```text
logical asset
generated filename
SHA-256
runtime scope
```

---

# 37. Static/serve runtime closure

Сохранить отдельные logical runtime sets:

```text
static
serve
```

Но больше не поддерживать вручную transitive JS/CSS dependencies.

Runtime closure вычислять через Vite manifest.

Пример:

```text
serve entry
   ↓
Vite dependency graph
   ↓
all required chunks/CSS
```

---

# 38. Static build isolation

`build` не должен включать runtime code, который нужен только для:

```text
Editor
Changes
Discussions
Roadmap write
serve API
React islands
```

если конкретная static surface их не использует.

Обычная documentation page не должна загружать React.

---

# 39. Nested URL deployment

Сохранить работу Portal:

```text
https://example.com/
```

и:

```text
https://example.com/project/docs/
```

без обязательного config `baseURL`.

Vite не должен генерировать зависимости от абсолютного `/assets/...`.

Frontend chunk URLs должны быть переносимыми относительно asset package.

Go продолжает владеть public URL generation через:

```text
rootPrefix
portal.assetBase
portal.dataBase
relative URL helpers
```

Добавить browser test на nested deployment.

---

# 40. Static HTTP hosting

Сохранить обычный static HTTP deployment без Toudocu runtime.

Production assets не должны требовать:

* dev server;
* Node;
* special rewrite router;
* backend asset resolution.

---

# 41. Appearance ordering

Сохранить порядок:

```text
appearance.js
   ↓
portal CSS
   ↓
page render
```

Theme/color scheme должны применяться до основного CSS paint.

React/Vite migration не должна создавать flash неправильной theme.

---

# 42. Vite development mode

Добавить scripts:

```text
npm run dev
npm run build
npm run typecheck
npm run test
npm run test:browser
npm run test:visual
```

`npm run dev` использует Vite HMR.

Добавить internal contributor-only способ использовать Go-generated portal pages с Vite development assets.

Не вводить публичный user CLI-флаг для этого.

---

# 43. Development UI page

Добавить:

```text
web/dev/ui.html
```

Только для development.

Показывать representative states:

```text
buttons
icon buttons
badges
tabs
dialogs
menus
tooltips
forms
empty states
diagnostics
task states
light
dark
compact
comfortable
```

Не включать страницу в released assets.

Storybook не добавлять.

---

# 44. Testing stack

Добавить:

```text
Vitest
React Testing Library
user-event
```

Playwright сохранить.

---

# 45. Component tests

Проверить:

* keyboard behaviour;
* Dialog;
* Tabs;
* Menu;
* Select;
* controlled state;
* island mounting;
* island cleanup;
* lazy island activation;
* capability gating.

---

# 46. Soft navigation tests

Отдельно проверить:

```text
page A
→ island mounted

navigate A → B
→ pagebeforechange
→ root A unmounted
→ layout replaced
→ bootstrap B active
→ pagechange
→ island B mounted once
```

Также:

```text
A → B → A → B
```

не должно создавать:

* duplicate listeners;
* duplicate dialogs;
* duplicate roots;
* stale feature state.

---

# 47. Visual regression

Playwright screenshot tests минимум для:

```text
portal document light
portal document dark
portal compact
serve canonical page
Roadmap
Editor
Changes
Discussions
use-case
diagnostics
empty state
mobile sidebar
```

Не создавать visual snapshot для каждой documentation page.

---

# 48. Accessibility

Новый UI должен соответствовать применимым требованиям WCAG 2.2 AA.

Обязательно:

* keyboard operation;
* visible focus;
* proper accessible names;
* modal focus management;
* Escape;
* semantic state announcements;
* state не определяется только цветом;
* sufficient contrast;
* `prefers-reduced-motion`.

---

# 49. Security

Production frontend:

* не использует CDN;
* не загружает remote JS;
* не загружает remote fonts;
* не использует `eval`;
* не создаёт client-side Markdown parser;
* не создаёт alternative unsafe HTML renderer;
* сохраняет same-origin serve boundaries;
* сохраняет текущие editor/review write restrictions.

---

# 50. Licenses

Использовать Vite 8:

```text
build.license
```

с JSON output:

```text
licenses.json
```

для bundled dependencies.

Vite является источником dependency license metadata для:

```text
React
React DOM
Base UI
Lucide
CodeMirror modules
прочих Vite-bundled dependencies
```

---

## Vendored assets

Для отдельно vendored artifacts:

```text
Mermaid
Swagger UI
```

сохранить их vendored license/notices.

`package-assets.mjs` объединяет:

```text
Vite licenses
+
vendored notices
```

в release frontend notices.

---

# 51. Packaging pipeline

`package-assets.mjs` должен:

1. проверить canonical source/destination;
2. читать Vite manifest;
3. вычислять runtime closure;
4. читать Vite licenses;
5. добавлять vendored assets;
6. добавлять vendored license notices;
7. вычислять SHA-256;
8. формировать Toudocu frontend manifest;
9. копировать только необходимые production assets;
10. не удалять произвольные directories.

---

# 52. Production runtime

Released Toudocu не требует:

```text
Node.js
npm
pnpm
yarn
Vite
React dev server
```

Пользователь получает тот же UX:

```text
install toudocu
toudocu build
toudocu serve
```

---

# 53. Release smoke test без Node

Добавить обязательный test environment, в котором Node/npm отсутствуют.

Проверить:

```text
toudocu check
toudocu build
toudocu serve
```

и browser smoke для embedded assets.

---

# 54. Что разрешено ломать

Совместимость не требуется для:

* старых internal CSS class names;
* старой UI DOM structure;
* `web/src/components` helper API;
* internal frontend imports;
* generated asset filenames;
* old frontend manifest schema;
* old non-public JS helper functions;
* old internal CSS variable names;
* старого esbuild build script.

После миграции старый foundation удалить.

---

# 55. Что нельзя менять без отдельного ТЗ

Не менять:

## CLI и данные

* CLI commands;
* CLI JSON schemas;
* Markdown syntax;
* typed documentation model;
* public static routes.

## Bootstrap/runtime

* `PageBootstrap schema v1` semantics;
* runtime values;
* capability-driven UI semantics;
* endpoint semantics.

## Portal deployment

* static HTTP hosting;
* relative asset addressing;
* relative data addressing;
* nested URL deployment;
* absence of mandatory base URL.

## Serve

* canonical serve soft navigation;
* existing navigation cache semantics;
* navigation history/scroll behaviour;
* translation read-only isolation.

## Localization

* shared Go/browser localization catalog;
* locale selection semantics.

## Appearance

* theme identifiers;
* color scheme values;
* accents;
* density values;
* content width values;
* font roles;
* appearance-before-CSS ordering.

## Existing capabilities

* update-check capability semantics;
* Editor API semantics;
* Changes API semantics;
* Agent Feedback semantics;
* Roadmap restricted write semantics;
* API Docs serve-only semantics.

## Distribution

* static/serve asset isolation;
* released runtime without Node.js.

---

# 56. Migration plan

## Этап 1. Vite foundation

Добавить:

```text
Vite 8
React
ReactDOM
Base UI
Lucide
Vitest
Testing Library
```

Перевести frontend build на Vite.

Добавить:

```text
Vite manifest
Vite licenses
package-assets
Toudocu manifest
```

Внешний UI максимально не менять.

---

## Этап 2. Design foundation

Создать:

```text
--td-* tokens
themes
typography
font roles
spacing
radius
density
icons
CSS layers
```

Перевести существующий CSS на semantic variables.

---

## Этап 3. Shared primitives

Реализовать:

```text
Button
IconButton
Badge
Separator
EmptyState
Diagnostic
Dialog
Tabs
Tooltip
Popover
Menu
Select
```

Сложные interactions строить через Base UI.

---

## Этап 4. Island infrastructure

Добавить:

```text
IslandHost
dynamic registry
eager/lazy islands
pagebeforechange
pagechange integration
cleanup lifecycle
```

Добавить lifecycle tests.

---

## Этап 5. Discussions

Перевести Discussions на lazy React island.

Сохранить Agent Feedback backend semantics.

---

## Этап 6. Roadmap

Перевести Roadmap interactive write controls на eager React island.

Сохранить restricted write contract.

Удалить старый DOM-based Roadmap UI.

---

## Этап 7. Editor

Перевести workspace shell на React.

CodeMirror сохранить.

---

## Этап 8. Changes

Перевести workspace shell на React.

CodeMirror Merge сохранить.

---

## Этап 9. Visual redesign

После архитектурной миграции обновить:

* header;
* sidebar;
* search;
* controls;
* forms;
* dialogs;
* tabs;
* task UI;
* diagnostics;
* Editor;
* Changes;
* Roadmap;
* Discussions;
* empty/loading/error states.

---

## Этап 10. Cleanup

Удалить:

* старый DOM component foundation;
* ручные replacements Base UI primitives;
* obsolete CSS;
* obsolete variables;
* Unicode chrome icons;
* старый esbuild application bundling;
* deprecated asset build paths.

Не оставлять два UI foundation параллельно.

---

# 57. Документация

Обновить архитектурную документацию Toudocu.

Зафиксировать:

* static vs interactive frontend boundary;
* framework-free runtime entries;
* React islands;
* IslandHost lifecycle;
* soft navigation lifecycle;
* `pagebeforechange`;
* PageBootstrap v1;
* capability-driven UI;
* shared i18n;
* design system;
* font roles;
* Vite pipeline;
* asset/runtime closure;
* licenses;
* nested URL deployment;
* Node.js development-only policy;
* правила добавления новых UI features.

---

# 58. Decision tree для нового frontend-кода

Использовать следующий порядок.

```text
Только статический контент?
        │
       да
        ↓
Go-generated semantic HTML

       нет
        ↓
Есть stateful application behaviour?
        │
       да
        ↓
React

        ↓
Стандартный interaction primitive?
        │
       да
        ↓
Base UI + Toudocu styling

        ↓
Generic visual primitive?
        │
       да
        ↓
ui/

        ↓
Documentation-specific representation?
        │
       да
        ↓
docs-ui/

        ↓
Portal/Serve-specific behavior?
        │
       да
        ↓
features/
```

---

# 59. Критерии приёмки

## Architecture

* [ ] Portal не стал SPA.
* [ ] Go остаётся владельцем document HTML и documentation semantics.
* [ ] `appearance.ts` framework-free.
* [ ] `portal.ts` framework-free.
* [ ] `serve.ts` framework-free.
* [ ] `serve.ts` не имеет eager dependency на React.
* [ ] React используется только через workspace roots и islands.
* [ ] `PageBootstrap schema v1` сохранён.
* [ ] Capability semantics сохранены.
* [ ] Generic UI не делает backend requests.
* [ ] `design`, `ui`, `docs-ui`, `features` имеют явные границы.

## Soft navigation

* [ ] `toudocu:pagebeforechange` существует.
* [ ] Событие отправляется перед удалением current layout.
* [ ] Все React roots unmount до DOM replacement.
* [ ] Bootstrap заменяется до `pagechange`.
* [ ] После `pagechange` новые islands обнаруживаются.
* [ ] Каждый island монтируется не более одного раза.
* [ ] Navigation A→B→A→B не создаёт leaks/duplicates.
* [ ] Existing cache/history/scroll semantics сохранены.

## React loading

* [ ] Обычная static page не загружает React.
* [ ] Canonical `serve` page без активного React feature не загружает React.
* [ ] Discussions React bundle загружается лениво.
* [ ] Roadmap bundle загружается только на Roadmap surface.
* [ ] CodeMirror загружается только для Editor/Changes surfaces, где он нужен.

## i18n

* [ ] `internal/site/i18n/*.json` остаются единственным источником Portal/Serve strings.
* [ ] React-specific translation store отсутствует.
* [ ] Generic UI получает labels через props.
* [ ] Existing locale fallback semantics сохранены.

## Design

* [ ] Все shared semantic variables используют `--td-*`.
* [ ] Сохранены `body/interface/heading/mono` font roles.
* [ ] Themes работают через tokens.
* [ ] Density работает через tokens.
* [ ] Нет Unicode icons в application chrome.
* [ ] Используется единая SVG icon system.
* [ ] Light/dark visual regression проходит.

## Features

* [ ] Discussions переведены на React.
* [ ] Roadmap переведён на React.
* [ ] Roadmap write semantics не расширены.
* [ ] Editor shell переведён на React.
* [ ] CodeMirror сохранён.
* [ ] Changes shell переведён на React.
* [ ] Changes backend contract сохранён.
* [ ] Agent Feedback backend contract сохранён.

## Vite/build

* [ ] Production frontend собирается Vite 8.
* [ ] Используется `build.rolldownOptions`.
* [ ] Используется Vite manifest.
* [ ] Используется Vite license generation.
* [ ] Vite не генерирует Portal HTML.
* [ ] `package-assets.mjs` строит Toudocu runtime closure.
* [ ] Все production assets имеют SHA-256.
* [ ] static и serve runtime sets разделены.
* [ ] старый esbuild application pipeline удалён.

## Deployment

* [ ] Static site работает от `/`.
* [ ] Static site работает под nested path.
* [ ] Frontend не предполагает `/assets/...` от origin root.
* [ ] Go остаётся владельцем relative public asset URLs.
* [ ] Static HTTP hosting не требует rewrite server.
* [ ] `appearance.js` выполняется до основного CSS.

## Quality

* [ ] `npm run typecheck` проходит.
* [ ] Vitest проходит.
* [ ] React Testing Library tests проходят.
* [ ] Playwright browser tests проходят.
* [ ] visual regression tests проходят.
* [ ] soft-navigation lifecycle tests проходят.
* [ ] Go tests проходят.
* [ ] собственный `toudocu check` проходит.

## Distribution

* [ ] Production binary включает frontend assets.
* [ ] Пользователю не нужен Node.js.
* [ ] Пользователю не нужен npm/pnpm/yarn.
* [ ] release smoke test без Node проходит.
* [ ] `toudocu build` запускается прежним способом.
* [ ] `toudocu serve` запускается прежним способом.

---

# 60. Целевая архитектура после миграции

```text
                         TOUDOCU CORE
                              Go
                               │
             safe HTML + PageBootstrap v1
                               │
           ┌───────────────────┴───────────────────┐
           │                                       │
           ▼                                       ▼
      STATIC PORTAL                         SERVE RUNTIME
      semantic HTML                         framework-free
      semantic CSS                          soft navigation
           │                                IslandHost
           │                                       │
           │                         ┌─────────────┴─────────────┐
           │                         │                           │
           │                  React islands              React workspaces
           │                  Discussions                Editor
           │                  Roadmap                    Changes
           │
           └──────────────────────┬──────────────────────────────┘
                                  │
                          TOUDOCU DESIGN
                          semantic tokens
                          themes
                          typography
                          icons
```

Следующие слои не зависят от Portal-specific runtime:

```text
design
ui
docs-ui
```

но не:

```text
Portal routing
serve navigation
Portal-specific feature controllers
Go HTML shell
```

---

# 61. Итоговый продуктовый инвариант

После миграции Toudocu должен сохранить основное свойство текущего продукта:

```text
скачать бинарник
        ↓
toudocu build / serve
        ↓
готовый интерфейс
```

React, Base UI, Vite, TypeScript и Node.js являются исключительно development/build-time инструментами.

Новая UI-система не должна превращать Toudocu из автономного Go-инструмента в JavaScript application runtime.

Она должна только дать Toudocu современный и расширяемый UI-фундамент для
дальнейшего развития Portal и `serve`.
