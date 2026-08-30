<!-- toudocu
id: TASK-AGENT-017
status: done
taskType: feature
priority: high
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-30
-->

# TASK-AGENT-017: Выполнять дерево задач одной целью

<!-- toudocu:section result -->
## Результат

Разработчик запускает на странице родительской задачи одно действие, после
чего Toudocu последовательно доводит до `Done` её готовых потомков и саму
родительскую задачу через одну Agent Session.

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

Подготовленное действие запускает или продолжает только одну задачу. После
каждого завершённого ответа разработчик сам выбирает следующую подзадачу.

<!-- toudocu:section after -->
### Станет

Действие `complete-tree` создаёт временную цель для дерева задач. Toudocu
выбирает следующую задачу по зависимостям и идентификатору, продолжает работу
после завершения ответа и показывает текущую задачу и общий прогресс.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_*.go` и связанные тесты;
- `internal/app/task_*.go`;
- `web/src/features/task-actions/` и frontend-тесты;
- `internal/site/i18n/` и сгенерированные frontend assets;
- `docs/contracts/`, `docs/modules/MOD-AGENT-CONSOLE.md`,
  `docs/use-cases/UC-AGENT-CONSOLE-01.md`, `docs/guides/work-items.md`;
- `CHANGELOG.md`.

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

- параллельный запуск нескольких задач или Agent Session;
- автоматическая подготовка `Draft`, `Needs attention` или `Blocked`;
- сохранение активной цели после перезапуска `serve`;
- provider-specific goal API и автоматическое архивирование задач;
- изменение переводных documentation roots.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` Родительская задача в `Ready` или `In progress` предлагает
  действие полного последовательного выполнения дерева.
- [x] `AC-02` Toudocu выбирает готовых потомков по зависимостям и `TASK-ID`,
  завершает родителей после детей и не запускает параллельные turns.
- [x] `AC-03` После завершения turn цель продолжает текущую или следующую
  задачу, становится `complete` для полностью выполненного дерева и `blocked`
  после трёх turns без изменения статуса или критериев.
- [x] `AC-04` Цель одинаково работает через общий provider contract, ожидает
  подтверждение и корректно реагирует на остановку или ошибку сессии.
- [x] `AC-05` Страница родителя показывает статус цели, текущую задачу и
  прогресс, а handoff содержит переносимую инструкцию без обещания внешнего
  автоматического продолжения.
- [x] `AC-06` Публичные контракты и каноническая документация описывают новое
  действие, состояние цели, ограничения и восстановление после перезапуска.

<!-- toudocu:section plan -->
## План

- [x] Добавить проверку и детерминированный выбор задач дерева.
- [x] Связать временную цель с жизненным циклом Agent Session.
- [x] Расширить Task Actions API и интерфейс страницы задачи.
- [x] Добавить поведенческие тесты и обновить документацию.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run 'TestTaskTreeGoal|TestAgentTask'`
- `AC-02` → `go test ./internal/app -run TestTaskTreeGoalSelection`
- `AC-03` → `go test ./internal/app -run TestTaskTreeGoalLifecycle`
- `AC-04` → `go test ./internal/app -run 'TestTaskTreeGoalSession|TestAgentSession'`
- `AC-05` → `npm --prefix web test -- task-actions`
- `AC-06` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `ALL` → `make check && make browser-test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make fmt-check && golangci-lint run --new-from-rev=HEAD ./... && go test ./... && go test -race ./... && go mod verify`

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

- `docs/contracts/task-actions.md` и
  `docs/contracts/task-actions.openapi.yaml`;
- `docs/contracts/agent-console.md`;
- `docs/modules/MOD-AGENT-CONSOLE.md`;
- `docs/use-cases/UC-AGENT-CONSOLE-01.md`;
- `docs/guides/work-items.md`;
- `CHANGELOG.md`.
