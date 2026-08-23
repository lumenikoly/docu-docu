# Toudocu

[English](README.md) | **Русский**

[![CI](https://github.com/lumenikoly/toudocu/actions/workflows/test.yml/badge.svg)](https://github.com/lumenikoly/toudocu/actions/workflows/test.yml)
[![Docs contract](https://github.com/lumenikoly/toudocu/actions/workflows/docs.yml/badge.svg)](https://github.com/lumenikoly/toudocu/actions/workflows/docs.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lumenikoly/toudocu)](https://go.dev/)
[![golangci-lint](https://img.shields.io/badge/linted%20by-golangci--lint-brightgreen)](https://golangci-lint.run/)
[![License](https://img.shields.io/github/license/lumenikoly/toudocu)](LICENSE)

**Документация, которая живёт рядом с кодом — и остаётся полезной людям и AI-агентам.**

Toudocu — локальный Go CLI для проектной Markdown-документации. Он помогает создать документацию по существующему проекту, проверять её структуру и связи, обновлять вместе с кодом, обсуждать изменения и публиковать готовый портал.

Вместо отдельной CMS или тяжёлого documentation stack исходником остаётся обычный Markdown в репозитории. Toudocu добавляет к нему проверяемую структуру, локальный интерфейс и workflow для AI-агентов.

**Один бинарник. Markdown в Git. Проверяемая документация рядом с кодом.**

[Посмотреть документацию Toudocu →](https://lumenikoly.github.io/toudocu/project-docs/)

[Установить Toudocu →](#1-установите-toudocu)

## Что даёт Toudocu

### Создание и обновление документации с AI-агентом

Toudocu поставляется со skill для поддерживаемых coding agents. Агент может изучить существующий репозиторий и создать минимальную документацию, а затем поддерживать её по мере изменения проекта.

```text
$toudocu init
$toudocu refresh
$toudocu refresh diff
```

Skill не заставляет проект следовать заранее придуманной архитектуре: агент должен опираться на то, что действительно существует в коде и репозитории.

Toudocu — гибкий инструмент, не привязанный к конкретному агенту: CLI и процесс работы с Markdown можно использовать с любым AI-агентом и средой его запуска (agent harness).

Автоматическая установка встроенного skill предусмотрена для **Codex**, **Claude Code** и **GitHub Copilot**.

### Проверяемая документация, а не просто набор Markdown-файлов

```bash
toudocu check ./docs
```

Toudocu проверяет документацию как связанную модель проекта:

- Markdown-документы и ссылки;
- стабильные ID и связи между сущностями;
- архитектурные документы;
- пользовательские сценарии и процессы;
- Mermaid-диаграммы;
- стандарты и runbooks;
- roadmap и рабочие задачи.

Проверку можно запускать локально или использовать как gate в CI.

`check` намеренно не делает вид, что способен автоматически оценить качество любого текста. Он проверяет формальные контракты и связи; смысловая корректность остаётся задачей автора и ревью.

### Локальное рабочее пространство для документации

```bash
toudocu serve ./docs
```

Toudocu запускает локальный портал, в котором документацию можно не только читать.

В нём доступны:

- навигация и поиск;
- редактор исходного Markdown;
- предварительный просмотр;
- diagnostics после изменений;
- просмотр текущих Git-изменений;
- локальный Swagger UI;
- обсуждения непосредственно по выбранному фрагменту документации.

По умолчанию портал доступен на `http://127.0.0.1:8080`.

### Вопросы и изменения прямо из документации

Во время чтения документа можно выделить текст или заголовок и оставить вопрос или запрос на изменение.

Например:

> Здесь всё ещё используется старый механизм авторизации?

или:

> Обнови этот раздел с учётом нового API.

Toudocu сохраняет такой запрос в локальной очереди. Установленный skill может получить его через CLI, обработать в контексте проекта и вернуть ответ обратно в исходное обсуждение.

Так документация становится частью обычной работы с агентом, а не отдельным набором файлов, которые нужно сначала найти и вручную объяснить.

Сам `serve` агента не запускает: момент передачи запроса агенту остаётся под контролем пользователя.

### Понятные Git-изменения документации

```bash
toudocu changes ./docs
```

Toudocu использует локальное состояние Git и показывает, что именно изменилось в документации.

Changes помогает:

- посмотреть текущий documentation diff;
- открыть изменение конкретного файла;
- сопоставить изменения с рабочей задачей;
- обсуждать изменённую документацию до коммита.

Анализ не изменяет working tree, index или историю Git.

### Статический портал для команды и пользователей

Когда документация готова к публикации:

```bash
toudocu build ./docs --output ./site --clean
```

Toudocu создаёт обычный статический HTTP-портал.

Каталог `site/` можно опубликовать, например, через GitHub Pages или другой статический хостинг. На сервере после этого не нужен ни Toudocu, ни отдельный backend.

Статическая версия предназначена для чтения и не включает локальный Editor, Changes и API режима `serve`.

### Документация как часть рабочего процесса

Toudocu умеет хранить рядом с постоянной документацией рабочие задачи и предоставлять агенту только нужный контекст.

```bash
toudocu task context TASK-ID ./docs
toudocu task changes TASK-ID ./docs
toudocu task verify TASK-ID ./docs --dry-run
```

Команды проверки из задачи не выполняются автоматически. Для их запуска требуется явный `--run`:

```bash
toudocu task verify TASK-ID ./docs --run
```

Это позволяет использовать документацию не только как справочник, но и как проверяемый контекст для выполнения работы.

### Несколько языков без второго источника правды

Skill умеет поддерживать переводы документации:

```text
$toudocu translate en --all-stale
$toudocu translate diff
```

Канонический каталог остаётся источником правды, а переводы создаются как read-only зеркала.

Это позволяет обновлять основной текст вместе с проектом и отдельно отслеживать устаревшие переводы.

---

## Быстрый старт

### 1. Установите Toudocu

**Linux и macOS**

```sh
curl -fsSL https://github.com/lumenikoly/toudocu/releases/latest/download/install.sh | sh
```

**Windows PowerShell**

```powershell
irm https://github.com/lumenikoly/toudocu/releases/latest/download/install.ps1 | iex
```

Проверьте установку:

```bash
toudocu version
```

Установщик выбирает подходящий бинарник для ОС и архитектуры, загружает `checksums.txt` и проверяет SHA-256 перед заменой бинарника.

Подробнее: [руководство по установке](docs/guides/installation.md).

### 2. Подключите Toudocu к своему AI-агенту

Перейдите в корень проекта и установите встроенный skill.

Для Codex:

```bash
toudocu skill install --agent codex
```

Для Claude Code:

```bash
toudocu skill install --agent claude-code
```

Для GitHub Copilot:

```bash
toudocu skill install --agent copilot
```

По умолчанию skill устанавливается для текущего проекта. Если хотите использовать его во всех проектах, добавьте `--scope user`.

### 3. Попросите агента создать документацию

Из корня репозитория:

```text
$toudocu init
```

Агент изучит проект и создаст минимальную структуру документации на основании того, что действительно найдёт в репозитории.

### 4. Проверьте результат

```bash
toudocu check ./docs
```

### 5. Откройте документацию

```bash
toudocu serve ./docs
```

После этого можно читать и редактировать документы, смотреть изменения и оставлять запросы агенту прямо из локального портала.

**На этом базовая настройка закончена.**

---

## Обновление

Чтобы обновить Toudocu, повторно запустите установщик из шага 1, затем из корня
проекта обновите skill:

```bash
toudocu skill update --agent codex
```

Используйте те же значения `--agent` и `--scope`, что и при установке. Если
`toudocu check` сообщает `DOCS_MIGRATION_REQUIRED`, попросите агента перенести
каноническую документацию на новый формат:

```text
$toudocu migrate
```

---

## Если AI-агент не нужен

Toudocu можно использовать как обычный CLI для Markdown-документации.

Минимальная структура проекта:

```text
docs/
├── index.md
└── architecture/
    └── overview.md
```

`index.md` знакомит читателя с проектом.

`architecture/overview.md` описывает границы системы и связывает остальные архитектурные документы.

Дальше достаточно:

```bash
toudocu check ./docs
toudocu serve ./docs
```

Skill — дополнительный workflow, а не обязательная зависимость Toudocu.

---

## Ежедневная работа с агентом

### Проверить всю документацию

```text
$toudocu refresh
```

Агент повторно сверит документацию с проектом и обновит то, что действительно устарело.

### Проверить только текущие изменения

```text
$toudocu refresh diff
```

Агент анализирует изменения относительно `HEAD`, включая index, working tree и новые неотслеживаемые файлы, а затем проверяет связанные документы.

Это удобно перед коммитом или pull request.

### Обновить переводы

```text
$toudocu translate en --all-stale
```

или только для текущих изменений:

```text
$toudocu translate diff
```

`$toudocu init`, `$toudocu refresh` и `$toudocu translate` — команды skill, которые выполняет AI-агент.

Это не верхнеуровневые команды Go CLI.

---

## Обсуждения с агентом

В режиме `serve` можно выделить фрагмент документации и создать вопрос или запрос на изменение.

После сохранения Toudocu создаёт запись локальной очереди.

Пока запрос не забрал агент, его можно изменить или удалить.

Агент получает следующий запрос через:

```bash
toudocu agent next --json
```

и возвращает результат:

```bash
toudocu agent respond --input response.json
```

В обычной работе эти низкоуровневые команды вызывает установленный skill, поэтому пользователю не нужно вручную переносить содержание обсуждения между порталом и агентом.

---

## Основные команды

| Что нужно сделать | Команда |
|---|---|
| Проверить документацию | `toudocu check ./docs` |
| Открыть локальный портал | `toudocu serve ./docs` |
| Собрать портал для публикации | `toudocu build ./docs --output ./site --clean` |
| Найти текст или ID | `toudocu search "запрос" ./docs` |
| Посмотреть Git-изменения | `toudocu changes ./docs` |
| Посмотреть изменение одного файла | `toudocu changes file PATH ./docs` |
| Создать нейтральный каркас | `toudocu scaffold module MOD-PAYMENTS ./docs --title "Платежи"` |
| Установить или обновить skill | `toudocu skill install\|update` |
| Получить контекст рабочей задачи | `toudocu task context TASK-ID ./docs` |
| Сопоставить задачу с Git diff | `toudocu task changes TASK-ID ./docs` |
| Проверить задачу без запуска команд | `toudocu task verify TASK-ID ./docs --dry-run` |
| Запустить разрешённые проверки задачи | `toudocu task verify TASK-ID ./docs --run` |

Полный CLI-контракт: [docs/contracts/cli.md](docs/contracts/cli.md).

---

## Публикация документации

Создайте статический портал:

```bash
toudocu build ./docs --output ./site --clean
```

После сборки содержимое `site/` можно разместить на обычном HTTP(S)-хостинге.

Например:

- GitHub Pages;
- Cloudflare Pages;
- Netlify;
- собственный web server;
- любой другой static hosting.

Toudocu не требуется на production-сервере.

Прямое открытие сгенерированного `index.html` через `file://` не поддерживается. Для локального просмотра используйте:

```bash
toudocu serve ./docs
```

---

## Поддерживаемый Markdown

Toudocu использует Goldmark 1.8.5 и единый CommonMark/GFM parser во всех командах.

Поддерживаются:

- заголовки и абзацы;
- ссылки;
- выделение и зачёркивание;
- цитаты;
- обычные списки и task lists;
- таблицы;
- автоматические ссылки;
- inline code и code blocks;
- безопасные растровые изображения.

Для Mermaid поддерживаются:

- `flowchart`;
- `stateDiagram-v2`;
- `sequenceDiagram`.

Toudocu намеренно не поддерживает часть расширений, которые усложняют безопасную и предсказуемую обработку Markdown, включая raw HTML, front matter, Markdown attributes, footnotes и активные SVG/XML/HTML resources.

Подробнее: [модуль Markdown](docs/modules/markdown.md).

---

## Конфигурация

Toudocu работает и без конфигурационного файла.

При необходимости `.toudocu/config.yml` позволяет настроить язык проекта, внешний вид портала, переводы и Changes.

Например:

```yaml
project:
  locale: ru

site:
  title: Мой проект
  theme: classic
  colorScheme: system
  accent: indigo
```

Все параметры описаны в [справочнике конфигурации](docs/reference/configuration.md).

---

## Безопасность локального сервера

`toudocu serve` предназначен для локальной работы.

По умолчанию сервер слушает только:

```text
127.0.0.1:8080
```

У него нет встроенной TLS-терминации и пользовательской аутентификации, поэтому не следует открывать его во внешнюю сеть без отдельного защитного слоя.

Автоматическую проверку новой стабильной версии можно отключить:

```bash
toudocu serve --no-update-check ./docs
```

---

## Публичный Go API

Корневой Go-пакет предоставляет типизированные операции модели, генератора и отчётов.

Module path проекта — `toudocu`, поэтому API предназначен прежде всего для программ, которые собираются внутри этого исходного дерева или используют явный локальный `replace`.

Для обычного использования поддерживаемым интерфейсом поставки остаётся CLI.

---

## Разработка Toudocu

Пользователю готового Toudocu не нужно собирать проект из исходников.

Если вы хотите разрабатывать сам Toudocu, потребуется Go 1.22 или новее.

```bash
git clone https://github.com/lumenikoly/toudocu.git
cd toudocu
go build -o toudocu ./cmd/toudocu
```

Основные команды разработки:

```bash
make fmt
make fmt-check
make lint
make test
make web
make web-check
make browser-test
make check
make build
make docs
make docs-serve
make release
```

Node.js нужен только для разработки браузерной части самого Toudocu и не требуется пользователю готового бинарника.

---

## Документация

- [Возможности](docs/reference/features.md)
- [Установка](docs/guides/installation.md)
- [Команды CLI](docs/contracts/cli.md)
- [Конфигурация](docs/reference/configuration.md)
- [Локальная работа](docs/guides/local-workflow.md)
- [Просмотр изменений](docs/guides/documentation-changes.md)
- [Работа с AI-агентом](docs/guides/agent-workflows.md)
- [Установка skill](docs/guides/skill-installation.md)
- [Рабочие задачи](docs/guides/work-items.md)
- [Виды документов](docs/reference/document-types.md)
- [Исходная документация Toudocu](docs/index.md)
- [Участие в разработке](CONTRIBUTING.ru.md)

## Лицензия

Условия распространения находятся в [LICENSE](LICENSE).

Лицензии встроенных сторонних компонентов перечислены в [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
