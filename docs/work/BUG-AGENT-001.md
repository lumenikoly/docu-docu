<!-- toudocu
id: BUG-AGENT-001
status: done
taskType: bug
severity: high
priority: high
reproducibility: often
regression: false
module: MOD-AGENT-CONSOLE
useCase: UC-AGENT-CONSOLE-01
standards: STD-GO-001, STD-DOCS-001
updated: 2026-08-29
-->

# BUG-AGENT-001: Устранить зависание Agent Console при длинном потоке событий

<!-- toudocu:section symptom -->
## Симптом

После длинного ответа с большим числом событий открытая Agent Console может
навсегда остаться в состоянии «Агент работает». Финальное сообщение не
появляется, а переходы по порталу и повторная загрузка страницы надолго
перестают отвечать.

<!-- toudocu:section expected-behavior -->
## Ожидаемое поведение

При отставании вкладки соединение восстанавливается, актуальный снимок сессии
снимает устаревший индикатор работы, а доступные события повторяются из
ограниченного 3 МиБ хвоста. Фоновое обнаружение изменений не создаёт постоянную
нагрузку чтением неизменных файлов.

<!-- toudocu:section actual-behavior -->
## Фактическое поведение

Медленная вкладка теряет события без разрыва WebSocket и не знает, что должна
запросить replay. Reload начинает обрабатывать весь доступный хвост мелкими
React-обновлениями. Одновременно `serve` повторно читает все поддерживаемые
файлы при каждом цикле watcher и HTTP polling даже без изменений.

<!-- toudocu:section steps-to-reproduce -->
## Шаги воспроизведения

1. Запустить loopback `serve` и открыть Agent Console.
2. Выполнить длинный turn с несколькими командами и объёмным выводом.
3. Замедлить серверный WebSocket writer либо создать burst событий быстрее,
   чем writer успевает их отправлять.
4. Дождаться завершения turn и сравнить открытую вкладку с новым подключением и
   серверным снимком.

<!-- toudocu:section evidence -->
## Доказательства

- открытая вкладка показывала «Агент работает» без финального ответа, когда
  `GET /_toudocu/api/agent-console/` уже возвращал `status: idle`;
- новый browser-сеанс восстановил финальный ответ из server-side replay;
- `publish` молча пропускает сообщение при заполненном канале подписчика
  ёмкостью 64;
- живой `toudocu serve` потреблял 50–79% CPU, а счётчик чтения стабильно рос на
  мегабайты даже без изменений workspace.

<!-- toudocu:section cause -->
## Причина

Live transport не разрывает отставшего подписчика при переполнении его канала,
поэтому frontend не выполняет reconnect и не получает пропущенный хвост.
Frontend применяет каждую replay-дельту отдельной серией обновлений состояния.
Watcher сначала запускает некэшированный `rootInputRevision`, даже когда затем
использует модельный revision, и не переиспользует scanner для разных portal
roots. Endpoint `/editor/files` проверяет условный ETag только после полного
сканирования.

<!-- toudocu:section scope -->
## Область изменения

- `internal/app/agent_console_http.go` и его unit-тесты;
- `web/src/features/agent-console/island.tsx` и существующий frontend unit-тест;
- `internal/app/editor_workspace.go`, `internal/app/editor_http.go` и их
  unit-тесты;
- `internal/app/server.go` и компактный unit-тест повторного использования
  portal workspace;
- `docs/contracts/agent-console.md`;
- `docs/work/BUG-AGENT-001.md`.

<!-- toudocu:section out-of-scope -->
## Не входит в исправление

- изменение provider protocol или жизненного цикла Agent Session;
- увеличение server-side replay сверх 3 МиБ;
- новый filesystem watcher или внешняя зависимость;
- тяжёлый browser/load test.

<!-- toudocu:section plan -->
## План

1. Разрывать соединение отставшего WebSocket-подписчика, чтобы он восстановил
   пропущенные события через существующий replay.
2. Пакетно применять входящие WebSocket-сообщения во frontend и объединять
   соседние текстовые дельты одного элемента.
3. Переиспользовать digest неизменных файлов и отвечать `304` до сканирования,
   когда ETag уже совпадает.
4. Обновить контракт и выполнить точечные, затем проектные проверки.

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

- [x] `AC-01` При переполнении очереди сервер закрывает отстающее
  WebSocket-соединение, а сохранённые события остаются доступны для replay с
  последней подтверждённой sequence.
- [x] `AC-02` Соседние replay-события `message_delta` и `command_output` одного
  элемента объединяются без дублирования и сохраняют порядок завершающих
  событий.
- [x] `AC-03` Watcher не запускает лишний некэшированный revision scan,
  переиспользует scanner каждого portal root и не читает содержимое неизменных
  файлов; совпавший ETag получает `304` без полного scan.

<!-- toudocu:section verification -->
## Проверка

- `AC-01` → `go test ./internal/app -run TestAgentConsoleSlowClientReconnects`
- `AC-02` → `cd web && npm exec vitest run tests/ui.test.tsx`
- `AC-03` → `go test ./internal/app -run 'TestEditorWorkspaceReusesUnchangedDigest|TestRootRevisionReusesPortalWorkspace|TestEditorFilesNotModifiedSkipsScan'`
- `ALL` → `go test ./... && npm --prefix web test`
- `DOCS` → `go run ./cmd/toudocu check ./docs --repository-root . --strict --stale-days 0`
- `QUALITY` → `make fmt-check && go vet ./... && go test -race ./... && go mod verify && for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do GOOS=${target%/*} GOARCH=${target#*/} CGO_ENABLED=0 go build -trimpath -o /dev/null ./cmd/toudocu || exit 1; done`

<!-- toudocu:section regression-test -->
## Регрессионный тест

Пять небольших unit-сценариев проверяют переподключение отставшего подписчика,
объединение соседних replay-дельт, повторное использование digest и portal
workspace без чтения неизменного файла, а также ранний ответ `304`.

<!-- toudocu:section documentation-impact -->
## Влияние на документацию

[Контракт Agent Console](../contracts/agent-console.md) уточняет закрытие
отставшего live-подключения и восстановление событий, сохранённых в ограниченном
3 МиБ replay. Остальные документы уже описывают требуемое поведение reload и
переходов и не меняются.
