<!-- toudocu
id: FLOW-TASK-WORKFLOW
module: MOD-CLI
useCase: UC-TASK-01, UC-TASK-02, UC-TASK-03
updated: 2026-08-29
-->

# FLOW-TASK-WORKFLOW: Работа с проверяемой задачей

Схема связывает подготовку задачи, безопасное получение контекста и отдельный,
явно разрешённый запуск команд проверки.

## Процесс

```mermaid
flowchart TD
    Search["Найти связанные документы: toudocu search"] --> Init["Создать черновик: toudocu task init"]
    Init --> Split{"Нужна декомпозиция по самостоятельным результатам?"}
    Split -->|Да| Children["Создать дочерние TASK-* с Parent"]
    Split -->|Нет| Fill["Заполнить цель, границы, связи, критерии и проверки"]
    Children --> Fill
    Fill --> Ready["Проверить полноту: toudocu task ready TASK-ID"]
    Ready --> Complete{"Контракт полный?"}
    Complete -->|Нет| Fill
    Complete -->|Да| Status["Вручную сменить статус Draft на Ready"]
    Status --> Selected{"Задача уже выбрана?"}
    Selected -->|Нет| Candidates["Получить фронт работы: toudocu task candidates"]
    Candidates --> Choose["Исполнитель выбирает задачу"]
    Choose --> Context["Получить контекст: toudocu task context TASK-ID"]
    Selected -->|Да| Context
    Status -.->|Нужен обзор декомпозиции| Tree["Посмотреть дерево: toudocu task tree TASK-ID"]
    Tree --> Context["Получить контекст: toudocu task context TASK-ID"]
    Context --> Found{"Найдена ровно одна задача?"}
    Found -->|Нет| ContextError["Вернуть ошибку, команды не запускать"]
    Found -->|Да| Plan["Спланировать и выполнить работу вне Toudocu"]
    Plan --> DryRun["Явно посмотреть план: task verify --dry-run"]
    DryRun --> Run["После отдельного разрешения: task verify --run"]
    Run --> Valid{"Контракт проверки корректен?"}
    Valid -->|Нет| Blocked["Вернуть blocked, команды не запускать"]
    Valid -->|Да| Commands["Последовательно выполнить объявленные команды"]
    Commands --> Passed{"Все команды успешны?"}
    Passed -->|Да| Success["Вернуть passed и код 0"]
    Passed -->|Нет| Failed["Вернуть failed и код 1"]
```

## Что важно

- `task context`, `task ready`, `task candidates` и `task verify --dry-run` не
  выполняют системные команды и не меняют Markdown.
- `task candidates` вычисляет общий `workState` и прогресс `descendants`, но не
  выбирает следующую задачу и не продвигает статус родителя из-за потомков.
- `task verify --run` запускается только после явного разрешения и только для
  доверенных команд, записанных в выбранной задаче; команды детей не запускаются.
- Ошибка одной команды не скрывает результаты остальных; превышение времени
  останавливает всё дерево процессов этой команды.
- Смысл запроса, уместность связей и выполнение критериев оценивает агент или
  разработчик, а не Toudocu.

## Связанные документы

- [UC-TASK-01: Получить контекст рабочей задачи](../use-cases/task-workflow.md)
- [UC-TASK-02: Выполнить проверки рабочей задачи](../use-cases/task-verify.md)
- [UC-TASK-03: Подготовить новую рабочую задачу](../use-cases/UC-TASK-03.md)
- [MOD-CLI: CLI и процессы задач](../modules/cli.md)
- [Руководство по рабочим задачам](../guides/work-items.md)
- [CLI-контракт](../contracts/cli.md)
