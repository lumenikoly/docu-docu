package toudocu

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode"
)

func TestEmbeddedMermaidVersionIsPinned(t *testing.T) {
	content, err := EmbeddedFiles.ReadFile("assets/generated/mermaid.tiny.js")
	if err != nil {
		t.Fatal(err)
	}
	const expected = "a1bc2282b3d935693780f77931382c517e72eb72ff3427752cbb29941de11bee"
	if actual := fmt.Sprintf("%x", sha256.Sum256(content)); actual != expected {
		t.Fatalf("unexpected Mermaid bundle checksum: got %s want %s", actual, expected)
	}
	if _, err := EmbeddedFiles.ReadFile("assets/generated/mermaid.LICENSE.txt"); err != nil {
		t.Fatal("embedded Mermaid license is missing")
	}
}

func TestArchivedTasksAreFilteredFromDefaultPortalSurfaces(t *testing.T) {
	root, docs, _ := createFixture(t)
	active := strings.Replace(completeTaskFixture("Ready"), "Add verification workflow", "Active portal task", 1)
	archived := strings.Replace(terminalTaskFixture("Done"), "Add verification workflow", "Archived portal task", 1)
	archived = strings.Replace(archived, "TASK-AUTH-021", "TASK-AUTH-022", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", active)
	writeTestFile(t, docs, "work/archive/2031/TASK-AUTH-022.md", archived)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	catalog := renderDirectoryPage(model, "work")
	for _, expected := range []string{"task-workspace-board", `data-workspace-view-panel="list"`, `data-workspace-view-panel="tree"`, "Archived portal task", `"workspaceState":"archive"`} {
		if !strings.Contains(catalog, expected) {
			t.Fatalf("work catalog missing %q", expected)
		}
	}
	navigation := renderNavigation(model, "work/index.html")
	if !strings.Contains(navigation, "Active portal task") || strings.Contains(navigation, "Archived portal task") {
		t.Fatalf("unexpected work navigation: %s", navigation)
	}
	dashboard := renderDashboard(model)
	focusStart := strings.Index(dashboard, `class="dashboard-section dashboard-focus"`)
	focusEnd := strings.Index(dashboard, `class="dashboard-section recommended-entries"`)
	if focusStart < 0 || focusEnd <= focusStart {
		t.Fatal("dashboard focus bounds are missing")
	}
	focus := dashboard[focusStart:focusEnd]
	if strings.Contains(focus, "Active portal task") || strings.Contains(focus, "Archived portal task") || !strings.Contains(focus, "Nearest outcome") {
		t.Fatalf("dashboard must keep focus compact without listing task names")
	}
}

func TestNavigationIconsReflectDocumentStatus(t *testing.T) {
	_, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	document := func(sourcePath, documentType, title, status string) *Document {
		metadata := Metadata{}
		if status != "" {
			metadata["status"] = status
		}
		return &Document{
			SourcePath: sourcePath,
			OutputPath: strings.TrimSuffix(sourcePath, ".md") + ".html",
			Directory:  path.Dir(sourcePath),
			FileName:   path.Base(sourcePath),
			Type:       documentType,
			Title:      title,
			Metadata:   metadata,
			Status:     StatusFor(status),
		}
	}
	model.Documents = append(model.Documents,
		document("work/TASK-DONE.md", "work", "Done task", "done"),
		document("work/TASK-DRAFT.md", "work", "Draft task", "draft"),
		document("work/BUG-BLOCKED.md", "work", "Blocked bug", "blocked"),
		document("work/TASK-CANCELLED.md", "work", "Cancelled task", "cancelled"),
		document("modules/done.md", "module", "Done module", "done"),
		document("decisions/accepted.md", "decision", "Accepted decision", "accepted"),
		document("reference/unknown.md", "reference", "Unknown status", "Новый статус"),
		document("architecture/no-status.md", "architecture", "No status", ""),
	)

	navigation := renderNavigation(model, "index.html")
	for _, expected := range []string{
		`class="nav-icon status-done" aria-hidden="true" title="Status: Done"><svg`,
		`class="nav-icon status-not-started" aria-hidden="true" title="Status: Not started"><svg`,
		`class="nav-icon status-blocked" aria-hidden="true" title="Status: Blocked"><svg`,
		`class="nav-icon status-cancelled" aria-hidden="true" title="Status: Cancelled"><svg`,
		`class="nav-icon status-accepted" aria-hidden="true" title="Status: Accepted"><svg`,
		`class="nav-icon" aria-hidden="true" title="Status: Not specified"><svg`,
		`<span class="visually-hidden"> · Status: Done</span>`,
		`<span class="nav-icon" aria-hidden="true"><svg`,
	} {
		if !strings.Contains(navigation, expected) {
			t.Fatalf("status navigation missing %q: %s", expected, navigation)
		}
	}
	style, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "styles", "portal.css"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`.nav-icon.status-done, .nav-icon.status-accepted`,
		`.nav-icon.status-in-progress`,
		`.nav-icon.status-blocked, .nav-icon.status-rejected, .nav-icon.status-review-required`,
		`.nav-icon.status-planned, .nav-icon.status-not-started, .nav-icon.status-open, .nav-icon.status-paused`,
		`.nav-icon.status-cancelled, .nav-icon.status-superseded, .nav-icon.status-obsolete, .nav-icon.status-risk-accepted`,
	} {
		if !strings.Contains(string(style), expected) {
			t.Fatalf("status icon styles missing %q", expected)
		}
	}
	if strings.Contains(string(style), `.nav-status-dot`) {
		t.Fatal("navigation must not render a separate status dot")
	}
	representativeTypes := []string{"overview", "status", "roadmap", "risks", "ideas", "notes", "changelog", "use-case", "module", "architecture", "contract", "decision", "flow", "screen", "guide", "work", "draft", "reference", "standard", "runbook", "document"}
	seen := make(map[string]bool, len(representativeTypes))
	for _, documentType := range representativeTypes {
		icon, _, _ := navigationDocumentIcon(document(documentType+".md", documentType, documentType, ""))
		seen[icon] = true
	}
	if len(seen) != len(representativeTypes) {
		t.Fatalf("representative document types must keep distinct icons: got %d icons for %d types", len(seen), len(representativeTypes))
	}
}

func writeTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	ensureTestDocumentationVersion(t, root)
	if relative == ".toudocu/config.yml" && !strings.Contains(content, "documentationVersion:") {
		content = "documentationVersion: 2\n" + content
	}
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(canonicalizeTestMarkdown(relative, content)), 0644); err != nil {
		t.Fatal(err)
	}
}

func fileLineContaining(t *testing.T, path, needle string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, needle) {
			return index + 1
		}
	}
	t.Fatalf("%q is missing from %s", needle, path)
	return 0
}

func createFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, docs, "index.md", "# Test Project\n\nОписание тестового проекта.\n\n- [x] Общая задача\n\n[Статус](status.md)\n")
	writeArchitectureOverview(t, docs, "")
	writeTestFile(t, docs, "status.md", "# Состояние\n\n- Статус: В работе\n- Этап: MVP\n- Последнее обновление: 2026-07-24\n\nТекущее состояние.\n\n## Краткое состояние\n\nОсновной поток работает.\n")
	writeTestFile(t, docs, "roadmap.md", "# Roadmap\n\nПлан.\n\n## MVP\n\n- Статус: В работе\n- Плановая дата: 2026-09-01\n\n- [x] `DLV-DOCS-01` Документация готова.\n- [ ] `UC-AUTH-01` Пользователь входит.\n")
	writeTestFile(t, docs, "risks.md", "# Риски\n\nОписание рисков.\n\n## RISK-01: Тестовый риск\n\n- Статус: Открыт\n- Вероятность: Высокая\n- Влияние: Среднее\n\n- [ ] Снизить риск\n")
	writeTestFile(t, docs, "modules/auth.md", `# Авторизация

- Идентификатор: MOD-AUTH
- Статус: В работе

Модуль входа.

## Назначение

Вход пользователей.

## Расположение в коде

Определяется приложением.

## Границы модуля

Отвечает за вход, но не за профиль.

## Бизнес-правила

### BR-AUTH-001: Сессия создаётся только после проверки

Неверные данные не создают сессию.

## Инварианты

Сессия принадлежит пользователю.

## Стабильные интерфейсы

Сценарий входа.

## Связанные сценарии

[UC-01](../use-cases/login.md)

## Проверка

Запустить тесты.

## Задачи

- [x] Вход
- [ ] Восстановление
`)
	writeTestFile(t, docs, "use-cases/login.md", `# UC-01: Вход

- Идентификатор: UC-AUTH-01
- Статус: В работе
- Актор: Пользователь
- Модуль: MOD-AUTH

Пользователь входит.

## Цель

Открыть аккаунт.

## Входные данные

Учётные данные.

## Предусловия

Пользователь существует.

## Основной сценарий

1. Ввести пароль.
2. Открыть проект.

## Альтернативные сценарии

Неверный пароль отклоняется.

## Постусловия

Создана сессия или состояние не изменилось.

## Бизнес-правила

[BR-AUTH-001](../modules/auth.md#br-auth-001-сессия-создаётся-только-после-проверки)

## Критерии приёмки

- [x] Форма
- [ ] Защита

## Реализация

[Модуль](../modules/auth.md)

## Проверка

Запустить тесты.
`)
	writeTestFile(t, docs, "decisions/001-db.md", `# ADR-001: База данных

- Идентификатор: ADR-001
- Статус: Принято
- Дата: 2026-07-01

Выбор базы.

## Контекст

Нужно хранение.

## Решение

Использовать SQL.

## Последствия

Нужны миграции.
`)
	return root, docs, filepath.Join(root, "site")
}

func buildFixture(t *testing.T, docs string) *Model {
	t.Helper()
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: filepath.Dir(docs), StaleDays: 90, Now: time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func writeArchitectureOverview(t *testing.T, docs, links string) {
	t.Helper()
	writeTestFile(t, docs, "architecture/overview.md", `# Architecture

- Document type: Architecture Overview

System architecture map.

## Questions

`+links+`
`)
}

func architectureIssueCodes(model *Model) map[string][]Issue {
	result := map[string][]Issue{}
	for _, issue := range model.Issues {
		result[issue.Code] = append(result[issue.Code], issue)
	}
	return result
}

func TestArchitectureContract(t *testing.T) {
	t.Run("overview is mandatory", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		if len(architectureIssueCodes(model)["missing-architecture-overview"]) != 1 {
			t.Fatalf("missing overview diagnostic not found: %#v", model.Issues)
		}
	})

	t.Run("details remain validated when overview is absent", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeTestFile(t, docs, "architecture/legacy.md", "# Legacy\n\nAnswer.\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		codes := architectureIssueCodes(model)
		for _, code := range []string{"missing-architecture-overview", "missing-architecture-question", "unlisted-architecture-document"} {
			if len(codes[code]) != 1 {
				t.Fatalf("legacy detail must produce %s: %#v", code, model.Issues)
			}
		}
	})

	t.Run("overview type comes from path", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeTestFile(t, docs, "architecture/overview.md", "# Architecture\n\n- Document type: Architecture\n\nMap.\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		if len(architectureIssueCodes(model)["invalid-architecture-overview-type"]) != 0 {
			t.Fatalf("reader-facing type text must not affect the model: %#v", model.Issues)
		}
	})

	t.Run("detail requires a non-empty question", func(t *testing.T) {
		for _, metadata := range []string{"", "- Architecture question:\n"} {
			root := t.TempDir()
			docs := filepath.Join(root, "docs")
			writeTestFile(t, docs, "index.md", "# Project\n")
			writeArchitectureOverview(t, docs, "[Detail](detail.md)")
			writeTestFile(t, docs, "architecture/detail.md", "# Detail\n\n"+metadata+"\nAnswer.\n")
			model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
			if err != nil {
				t.Fatal(err)
			}
			if len(architectureIssueCodes(model)["missing-architecture-question"]) != 1 {
				t.Fatalf("missing question diagnostic not found for %q: %#v", metadata, model.Issues)
			}
		}
	})

	t.Run("question does not require punctuation", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeArchitectureOverview(t, docs, "[Detail](nested/detail.md)")
		writeTestFile(t, docs, "architecture/nested/detail.md", "# Detail\n\n- Architecture question: Runtime responsibility split\n\nAnswer.\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		codes := architectureIssueCodes(model)
		for _, code := range []string{"missing-architecture-question", "unlisted-architecture-document"} {
			if len(codes[code]) != 0 {
				t.Fatalf("valid recursive detail produced %s: %#v", code, codes[code])
			}
		}
	})

	t.Run("overview map must link every detail directly", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeArchitectureOverview(t, docs, "[First](first.md)")
		writeTestFile(t, docs, "architecture/first.md", "# First\n\n- Architecture question: First boundary\n\n[Second](nested/second.md)\n")
		writeTestFile(t, docs, "architecture/nested/second.md", "# Second\n\n- Architecture question: Second boundary\n\nAnswer.\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		issues := architectureIssueCodes(model)["unlisted-architecture-document"]
		if len(issues) != 1 || issues[0].DocumentPath != "architecture/nested/second.md" {
			t.Fatalf("transitively linked detail must be unlisted: %#v", issues)
		}
	})

	t.Run("architecture link diagnostics are errors", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeArchitectureOverview(t, docs, "[Detail](detail.md)\n\n[Missing](missing.md)")
		writeTestFile(t, docs, "architecture/detail.md", "# Detail\n\n- Architecture question: Trust boundary\n\n[Blocked](/etc/passwd)\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		codes := architectureIssueCodes(model)
		for _, code := range []string{"broken-link", "blocked-link"} {
			if len(codes[code]) != 1 || codes[code][0].Severity != "error" {
				t.Fatalf("%s must be one architecture error: %#v", code, codes[code])
			}
		}
	})

	t.Run("optional architecture IDs keep global uniqueness", func(t *testing.T) {
		root := t.TempDir()
		docs := filepath.Join(root, "docs")
		writeTestFile(t, docs, "index.md", "# Project\n")
		writeTestFile(t, docs, "architecture/overview.md", `# Architecture

- Document type: Architecture Overview
- Identifier: ARCH-SHARED

System map.

[Detail](detail.md)
`)
		writeTestFile(t, docs, "architecture/detail.md", `# Detail

- Document type: Architecture
- Architecture question: Runtime boundary
- Identifier: ARCH-SHARED

Answer.
`)
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		issues := architectureIssueCodes(model)["duplicate-id"]
		if len(issues) != 1 || issues[0].Severity != "error" {
			t.Fatalf("optional architecture ID must retain duplicate-id behavior: %#v", issues)
		}
	})
}

func TestArchitectureSchemaContract(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Project\n")
	writeArchitectureOverview(t, docs, "")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	report := BuildReport(model)
	if report.SchemaVersion != 1 {
		t.Fatalf("schema version=%d", report.SchemaVersion)
	}
	for _, document := range report.Documents {
		if document.SourcePath == "architecture/overview.md" {
			if document.Type != "architecture" {
				t.Fatalf("overview type=%q", document.Type)
			}
			if document.Metadata["documentType"] != "" {
				t.Fatalf("overview metadata=%#v", document.Metadata)
			}
			return
		}
	}
	t.Fatal("architecture overview missing from report")
}

func TestModelAndStatistics(t *testing.T) {
	_, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	if model.Project.Title != "Test Project" || model.Stats.Documents != 8 || model.Stats.TotalTasks != 2 || model.Stats.CompletedTasks != 1 || len(model.Risks) != 1 || len(model.RoadmapStages) != 1 || model.Stats.BrokenLinks != 0 {
		t.Fatalf("unexpected model: %#v %#v", model.Project, model.Stats)
	}
	module := model.DocByPath["modules/auth.md"]
	useCase := model.DocByPath["use-cases/login.md"]
	if len(module.LinkedUseCases) != 1 || module.LinkedUseCases[0] != useCase || len(useCase.LinkedModules) != 1 || useCase.LinkedModules[0] != module {
		t.Fatalf("links failed")
	}
}

func TestNotesAndIdeasAreUnvalidatedFreeFormDocuments(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Проект\n\nОписание.\n")
	writeTestFile(t, docs, "notes.md", `# Заметки

## Наблюдения

- Первый пункт с **выделением**
  - Вложенный пункт
`)
	writeTestFile(t, docs, "ideas.md", `- Статус: неизвестная идея
- Обновлено: 2000-01-01

## Возможности

1. Первый план.
2. Второй план.
`)

	model, err := BuildDocumentationModel(Options{
		InputDirectory: docs,
		RepositoryRoot: root,
		StaleDays:      1,
		Now:            time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if model.DocByPath["notes.md"].Type != "notes" || model.DocByPath["ideas.md"].Type != "ideas" {
		t.Fatalf("unexpected free-form types: notes=%q ideas=%q", model.DocByPath["notes.md"].Type, model.DocByPath["ideas.md"].Type)
	}
	for _, issue := range model.Issues {
		if issue.DocumentPath == "notes.md" || issue.DocumentPath == "ideas.md" {
			t.Fatalf("free-form documents must not produce validation issues: %#v", issue)
		}
	}

	output := filepath.Join(root, "site")
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	notesHTML, err := os.ReadFile(filepath.Join(output, "notes.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"Заметки", "Наблюдения", "<ul>", "<strong>выделением</strong>", "Вложенный пункт"} {
		if !strings.Contains(string(notesHTML), fragment) {
			t.Fatalf("notes page does not contain %q", fragment)
		}
	}
	ideasHTML, err := os.ReadFile(filepath.Join(output, "ideas.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ideasHTML), "<ol>") || !strings.Contains(string(ideasHTML), "Ideas") {
		t.Fatal("ideas page must render ordered plans and its document type")
	}
}

func TestMermaidFlowModelAndRelationships(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Проект\n\nОписание.\n")
	writeTestFile(t, docs, "modules/auth.md", `# Авторизация

- Идентификатор: MOD-AUTH
- Статус: В работе

Модуль авторизации.
`)
	writeTestFile(t, docs, "use-cases/login.md", `# UC-AUTH-01: Вход

- Идентификатор: UC-AUTH-01
- Статус: В работе
- Модуль: MOD-AUTH

Пользователь входит.
`)
	writeTestFile(t, docs, "flows/login.md", `# FLOW-AUTH-LOGIN: Вход

- Идентификатор: FLOW-AUTH-LOGIN
- Сценарий: UC-AUTH-01
- Модуль: MOD-AUTH

Процесс входа.

## Процесс

`+"```mermaid"+`
sequenceDiagram
    User->>Web: Login
    Web->>API: Authenticate
    API-->>Web: Session
    Web-->>User: Dashboard
`+"```"+`
`)
	writeTestFile(t, docs, "architecture/services.md", `# Взаимодействие сервисов

- Идентификатор: ADR-SERVICES

Архитектурное взаимодействие.

`+"```mermaid"+`
sequenceDiagram
    Web->>API: Login
`+"```"+`
`)
	writeTestFile(t, docs, "flows/services.md", `# FLOW-SERVICES: Вызов API

- Идентификатор: FLOW-SERVICES

Архитектурный процесс.

`+"```mermaid"+`
sequenceDiagram
    Web->>API: Login
`+"```"+`

[Архитектура](../architecture/services.md)
`)

	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	flow := model.DocByPath["flows/login.md"]
	if flow == nil || flow.Type != "flow" {
		t.Fatalf("flow document was not classified: %#v", flow)
	}
	for _, issue := range model.Issues {
		if strings.Contains(issue.Code, "mermaid") || strings.Contains(issue.Code, "flow") || issue.Code == "dangling-use-case-reference" || issue.Code == "dangling-module-reference" {
			t.Fatalf("valid Mermaid model produced issue: %#v", issue)
		}
	}
	output := filepath.Join(root, "site")
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	flowHTML, err := os.ReadFile(filepath.Join(output, "flows", "FLOW-AUTH-LOGIN.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`data-mermaid-stage`, `data-mermaid-diagram`, `data-mermaid-zoom-out`,
		`data-mermaid-fit`, `data-mermaid-zoom-in`, `data-mermaid-fullscreen`,
		`UC-AUTH-01`, `../use-cases/UC-AUTH-01.html`,
	} {
		if !strings.Contains(string(flowHTML), part) {
			t.Fatalf("flow Mermaid page missing %q: %s", part, flowHTML)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "flows", "login.html")); !os.IsNotExist(err) {
		t.Fatalf("source-filename URL must not be generated when a stable FLOW ID exists: %v", err)
	}
	processHTML, err := os.ReadFile(filepath.Join(output, "processes", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"UC-AUTH-01", "FLOW-AUTH-LOGIN", "FLOW-SERVICES"} {
		if !strings.Contains(string(processHTML), expected) {
			t.Fatalf("aggregate processes catalog missing %q", expected)
		}
	}
}

func TestKnowledgeFlowsLinkMultipleUseCasesInBothDirections(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Проект\n")
	for _, id := range []string{"UC-AUTH-01", "UC-AUTH-02"} {
		writeTestFile(t, docs, "use-cases/"+id+".md", "# "+id+": Сценарий\n\n- Идентификатор: "+id+"\n- Статус: В работе\n")
	}
	writeTestFile(t, docs, "flows/FLOW-AUTH-LOGIN.md", `# FLOW-AUTH-LOGIN: Общий процесс

- Идентификатор: FLOW-AUTH-LOGIN
- Сценарий: UC-AUTH-01, UC-AUTH-02

`+"```mermaid"+`
flowchart TD
    Start --> Finish
`+"```"+`
`)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Knowledge.Flows) != 1 {
		t.Fatalf("flows=%#v", model.Knowledge.Flows)
	}
	if got := strings.Join(model.Knowledge.Flows[0].UseCaseIDs, ","); got != "UC-AUTH-01,UC-AUTH-02" {
		t.Fatalf("flow use cases=%q", got)
	}
	for _, useCase := range model.Knowledge.UseCases {
		if got := strings.Join(useCase.FlowIDs, ","); got != "FLOW-AUTH-LOGIN" {
			t.Fatalf("%s flow IDs=%q", useCase.ID, got)
		}
	}
}

func TestMermaidContractErrors(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Проект\n\nОписание.\n")
	writeTestFile(t, docs, "flows/missing.md", `# Процесс без схемы

- Идентификатор: FLOW-MISSING
`)
	writeTestFile(t, docs, "guides/unlinked.md", `# Непривязанная схема

Описание.

`+"```mermaid"+`
sequenceDiagram
    A->>B: Test
`+"```"+`
`)
	writeTestFile(t, docs, "work/TASK-FLOW-001.md", `# TASK-FLOW-001: Проверить процесс

- Статус: Черновик
- Тип: Research
- Процесс: FLOW-UNKNOWN

## Результат

Проверка выполнена.
`)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, issue := range model.Issues {
		codes[issue.Code] = true
	}
	for _, code := range []string{"missing-flow-diagram", "unlinked-mermaid-diagram", "dangling-flow-reference"} {
		if !codes[code] {
			t.Fatalf("missing %s in %#v", code, model.Issues)
		}
	}
	if codes["sequence-diagram-outside-architecture"] {
		t.Fatalf("removed sequence diagram diagnostic returned in %#v", model.Issues)
	}
}

func TestCanonicalOutputPathsRequireSafeStableIDs(t *testing.T) {
	model := &Model{Documents: []*Document{
		{Type: "use-case", Metadata: map[string]string{"id": "UC-AUTH-01"}, OutputPath: "use-cases/login.html"},
		{Type: "flow", Metadata: map[string]string{"id": "FLOW-../ESCAPE"}, OutputPath: "flows/unsafe.html"},
		{Type: "screen", Metadata: map[string]string{"id": "SC-AUTH/LOGIN"}, OutputPath: "screens/unsafe.html"},
	}}
	assignUniqueOutputPaths(model)
	if got := model.Documents[0].OutputPath; got != "use-cases/UC-AUTH-01.html" {
		t.Fatalf("canonical output=%q", got)
	}
	if got := model.Documents[1].OutputPath; got != "flows/unsafe.html" {
		t.Fatalf("unsafe FLOW ID changed output path to %q", got)
	}
	if got := model.Documents[2].OutputPath; got != "screens/unsafe.html" {
		t.Fatalf("unsafe screen ID changed output path to %q", got)
	}
}

func TestTaskContextIncludesReferencedFlow(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "flows/login.md", `# FLOW-AUTH-LOGIN: Вход

- Идентификатор: FLOW-AUTH-LOGIN
- Сценарий: UC-AUTH-01

Процесс входа.

`+"```mermaid"+`
flowchart TD
    Login --> Dashboard
`+"```"+`
`)
	commands := map[string]string{"AC-01": "pass", "AC-02": "pass", "ALL": "pass", "DOCS": "pass"}
	task := strings.Replace(taskVerifyFixture("Готово к работе", false, commands, ""), "- Сценарий: UC-AUTH-01", "- Сценарий: UC-AUTH-01\n- Процесс: FLOW-AUTH-LOGIN", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-020-flow.md", task)
	model := buildFixture(t, docs)
	report, err := BuildTaskContext(model, "TASK-AUTH-020")
	if err != nil {
		t.Fatal(err)
	}
	if report.Task.FlowID != "FLOW-AUTH-LOGIN" {
		t.Fatalf("flow ID missing from task: %#v", report.Task)
	}
	found := false
	for _, document := range report.Documents {
		if document.Path == "flows/login.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("flow missing from task context: %#v", report.Documents)
	}
}

func TestGenerateSite(t *testing.T) {
	_, docs, output := createFixture(t)
	indexFile, err := os.OpenFile(filepath.Join(docs, "index.md"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := indexFile.WriteString("\n[Сценарии](use-cases/)\n"); err != nil {
		_ = indexFile.Close()
		t.Fatal(err)
	}
	if err := indexFile.Close(); err != nil {
		t.Fatal(err)
	}
	model := buildFixture(t, docs)
	result, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Pages < 10 {
		t.Fatalf("pages=%d", result.Pages)
	}
	for _, name := range []string{"index.html", "404.html", "health.html", "report.json", "assets/manifest.json", "assets/portal.css", "assets/portal.js", "data/search-index.json", "data/navigation.json", "data/relations.json", "data/screens.json", "data/use-cases/index.json", "modules/auth.html", "modules/index.html", "processes/index.html", "use-cases/UC-AUTH-01.html", "use-cases/index.html"} {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "flows", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("legacy flows catalog must not be generated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "use-cases", "login.html")); !os.IsNotExist(err) {
		t.Fatalf("source-filename URL must not be generated when a stable UC ID exists: %v", err)
	}
	htmlBytes, _ := os.ReadFile(filepath.Join(output, "modules/auth.html"))
	html := string(htmlBytes)
	for _, part := range []string{"Document readiness", "../use-cases/UC-AUTH-01.html", `class="metadata-grid"`, "<dt>Status</dt>", `class="collapse-all-button"`, `data-collapse-label`} {
		if !strings.Contains(html, part) {
			t.Fatalf("missing %s", part)
		}
	}
	if strings.Contains(html, `data-task-filter`) {
		t.Fatal("document page must not include a task checklist filter")
	}
	collapseAllMarkup := `<span class="collapse-all-icon" aria-hidden="true"><span class="collapse-icon collapse-icon-up"></span><span class="collapse-icon collapse-icon-down"></span></span><span data-collapse-label>Collapse sections</span>`
	if !strings.Contains(html, collapseAllMarkup) {
		t.Fatal("collapse-all icons must remain inside their positioning container")
	}
	if strings.Contains(html, "<script>alert") {
		t.Fatal("unsafe")
	}
	dashboardBytes, _ := os.ReadFile(filepath.Join(output, "index.html"))
	if strings.Contains(string(dashboardBytes), `data-filter-control="type"`) {
		t.Fatal("dashboard must not duplicate the document catalog filters")
	}
	directoryBytes, _ := os.ReadFile(filepath.Join(output, "modules/index.html"))
	directoryHTML := string(directoryBytes)
	if strings.Contains(directoryHTML, `data-filter-control="type"`) {
		t.Fatal("typed directory must not offer a redundant type filter")
	}
	if strings.Contains(directoryHTML, `data-filter-control="search"`) {
		t.Fatal("single-document directory must not offer redundant collection filters")
	}
	reportBytes, _ := os.ReadFile(filepath.Join(output, "report.json"))
	var report ProjectReport
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || report.Documents == nil || report.Issues == nil || report.Screens == nil || report.Transitions == nil || report.PlayableFlows == nil || report.Hotspots == nil || report.Traceability == nil || report.CurrentStatus.ActiveWork == nil || report.CurrentStatus.Blockers == nil {
		t.Fatalf("unstable report collections: %#v", report)
	}
	foundDirectoryTarget := false
	for _, document := range report.Documents {
		if document.SourcePath != "index.md" {
			continue
		}
		for _, link := range document.Links {
			if link.Destination == "use-cases/" && link.TargetKind == "directory" && link.Target == "use-cases/index.html" {
				foundDirectoryTarget = true
			}
		}
	}
	if !foundDirectoryTarget {
		t.Fatal("generated directory link is missing a typed report target")
	}
}

func TestDraftsSectionBuildsCatalogAndReport(t *testing.T) {
	root, docs, output := createFixture(t)
	writeTestFile(t, docs, "drafts/index.md", "# Наброски\n")
	writeTestFile(t, docs, "drafts/entry.md", "# Первый черновик\n\nТекст для поиска.\n")
	writeSiteConfig(t, root, "project:\n  locale: ru\n  sections:\n    architecture: Архитектура\n    modules: Модули\n    use-cases: Пользовательские сценарии\n    flows: Процессы\n    screens: Экраны\n    decisions: Архитектурные решения\n    contracts: Контракты\n    quality: Стандарты качества\n    runbooks: Runbooks\n    reference: Справочник\n    work: Рабочие задачи\n    drafts: Наброски\n    guides: Руководства\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	draft := model.DocByPath["drafts/entry.md"]
	if draft == nil || draft.Type != "draft" || draft.SectionType != SectionDrafts || localizedTypeLabel(model, draft.Type) != "Черновик" {
		t.Fatalf("draft model: %#v", draft)
	}
	if modelDirectoryLabel(model, "drafts") != "Наброски" || !strings.Contains(renderNavigation(model, "drafts/index.html"), "Наброски") {
		t.Fatalf("draft navigation: %s", renderNavigation(model, "drafts/index.html"))
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	catalog, err := os.ReadFile(filepath.Join(output, "drafts", "index.html"))
	if err != nil || !strings.Contains(string(catalog), "Первый черновик") || !strings.Contains(string(catalog), "Черновик") {
		t.Fatalf("draft catalog: %v %s", err, catalog)
	}
	page, err := os.ReadFile(filepath.Join(output, "drafts", "entry.html"))
	if err != nil || !strings.Contains(string(page), "Черновик") || !strings.Contains(string(page), "Текст для поиска") {
		t.Fatalf("draft page: %v %s", err, page)
	}
	foundSearch := false
	for _, item := range model.SearchIndex {
		if item.Path == "drafts/entry.md" && item.Type == "draft" && item.TypeLabel == "Черновик" {
			foundSearch = true
		}
	}
	if !foundSearch {
		t.Fatalf("draft missing from search index: %#v", model.SearchIndex)
	}
	reportBytes, err := os.ReadFile(filepath.Join(output, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report ProjectReport
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatal(err)
	}
	for _, document := range report.Documents {
		if document.SourcePath == "drafts/entry.md" && document.Type == "draft" && document.SectionType == SectionDrafts {
			return
		}
	}
	t.Fatalf("draft missing from report: %#v", report.Documents)
}

func TestDocumentContextCopyMarkupAndAssets(t *testing.T) {
	root, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	module := model.DocByPath["modules/auth.md"]
	useCase := model.DocByPath["use-cases/login.md"]

	moduleHTML := renderDocumentPage(model, module)
	for _, part := range []string{
		`data-copy-document-context`,
		`data-document-context-title="Авторизация"`,
		`data-document-context-path="docs/modules/auth.md"`,
		`data-copy-document-context-label`,
		`Copy context`,
	} {
		if !strings.Contains(moduleHTML, part) {
			t.Fatalf("document context markup missing %q", part)
		}
	}
	if !strings.Contains(renderUseCasePage(model, useCase), `data-document-context-path="docs/use-cases/login.md"`) {
		t.Fatal("canonical use-case page must expose its source document context")
	}
	if !strings.Contains(renderDashboard(model), `data-document-context-path="docs/index.md"`) {
		t.Fatal("dashboard must expose the root index document context")
	}

	for name, html := range map[string]string{
		"directory":    renderDirectoryPage(model, "modules"),
		"processes":    renderProcessCatalogPage(model, "processes/index.html", "flow"),
		"health":       renderHealthPage(model),
		"traceability": renderTraceabilityPage(model, "traceability.html"),
	} {
		if strings.Contains(html, `data-copy-document-context`) {
			t.Fatalf("%s page must not expose synthetic document context", name)
		}
	}

	model.RepositoryRoot = filepath.Join(root, "unrelated")
	fallbackHTML := renderDocumentPage(model, module)
	if !strings.Contains(fallbackHTML, `data-document-context-path="modules/auth.md"`) ||
		strings.Contains(fallbackHTML, module.AbsolutePath) {
		t.Fatal("context outside repository root must use SourcePath without exposing an absolute path")
	}

	appScript, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "core", "portal.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		"initializeDocumentContextCopy",
		"navigator.clipboard?.writeText",
		"document.execCommand('copy')",
		"core.portal.015",
		"core.portal.017",
	} {
		if !strings.Contains(string(appScript), part) {
			t.Fatalf("document context browser behavior missing %q", part)
		}
	}
	style, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "styles", "portal.css"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(style), `.document-context-button`) {
		t.Fatal("document context button styles are missing")
	}
}

func TestRiskPageExplainsRiskStatusesAndMitigationProgress(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "risks.md", `# Риски

Описание рисков.

## RISK-01: Требует решения

- Статус: Открыт

- [ ] Назначить решение

## RISK-02: Меры выполняются

- Статус: Снижается

- [x] Выполнить первую меру

## RISK-03: Решение принято

- Статус: Риск принят

- [x] Зафиксировать принятие
`)
	model := buildFixture(t, docs)
	if model.Stats.OpenRisks != 2 {
		t.Fatalf("open risks = %d, want 2", model.Stats.OpenRisks)
	}
	html := renderDocumentPage(model, model.DocByPath["risks.md"])
	for _, expected := range []string{
		"Risk status",
		"Open risks: 2 of 3",
		"1 open",
		"1 mitigating",
		"1 accepted risk",
		"Open</strong> — requires a decision.",
		"Mitigating</strong> — measures are in progress; the risk is not closed yet.",
		"Accepted risk</strong> — the team consciously accepts the risk; it is not counted as open.",
		"Mitigation progress",
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("risk page missing %q: %s", expected, html)
		}
	}
	if strings.Contains(html, "Document readiness") || strings.Contains(html, ">Checklist</span>") {
		t.Fatal("risk page must not present mitigation tasks as document readiness")
	}
	dashboard := renderDashboard(model)
	if strings.Contains(dashboard, "Незакрытые риски") || strings.Contains(dashboard, ">2</strong>") {
		t.Fatal("dashboard must not repeat detailed risk counters")
	}
}

func TestPortalSimplifiedNavigationAndAccessibleHeadings(t *testing.T) {
	_, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	dashboard := renderDashboard(model)
	if count := strings.Count(dashboard, `class="recommended-entry"`); count < 3 || count > 5 {
		t.Fatalf("recommended entry count = %d", count)
	}
	for _, expected := range []string{"Current focus", "Nearest outcome", "Where to start", "Project overview", "Traceability matrix"} {
		if !strings.Contains(dashboard+renderTraceabilityPage(model, "traceability.html"), expected) {
			t.Fatalf("portal missing %q", expected)
		}
	}
	for _, forbidden := range []string{"Каталог документации", `data-filter-control="type"`, "Дорожная карта", "Вычисляемое состояние", "RISK-01: Тестовый риск"} {
		if strings.Contains(dashboard, forbidden) {
			t.Fatalf("dashboard still contains detailed surface %q", forbidden)
		}
	}
	ordered := []string{"dashboard-about", "dashboard-focus", "recommended-entries", "dashboard-overview"}
	position := -1
	for _, marker := range ordered {
		next := strings.Index(dashboard, marker)
		if next <= position {
			t.Fatalf("dashboard block %q is out of order", marker)
		}
		position = next
	}
	for _, forbidden := range []string{">⚑<", ">±<", ">⇥<", ">Traceability<"} {
		if strings.Contains(renderNavigation(model, "index.html"), forbidden) {
			t.Fatalf("navigation still contains noisy or untranslated label %q", forbidden)
		}
	}
	if !strings.Contains(renderNavigation(model, "index.html"), "Architecture overview") {
		t.Fatal("architecture overview must have a distinct navigation label")
	}

	documentHTML := renderDocumentPage(model, model.DocByPath["modules/auth.md"])
	if !strings.Contains(documentHTML, `class="heading-anchor"`) || !strings.Contains(documentHTML, `aria-hidden="true" tabindex="-1"`) {
		t.Fatal("permalink must be excluded from the heading accessible name")
	}
	if strings.Contains(documentHTML, `class="heading-anchor" href=`) && strings.Contains(documentHTML, `aria-label="Ссылка на раздел"`) {
		t.Fatal("heading permalink aria-label leaked into accessible heading name")
	}

	app, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "core", "portal.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"containsActivePage",
		"hasSavedState ? folderState[key] === true : true",
		"section.insertBefore(toggle, body)",
	} {
		if !strings.Contains(string(app), expected) {
			t.Fatalf("browser behavior missing %q", expected)
		}
	}
}

func TestDashboardFocusFallbacksAndAlwaysVisibleOverview(t *testing.T) {
	_, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	html := renderDashboard(model)
	for _, expected := range []string{
		`class="dashboard-section dashboard-focus" aria-label="Current focus: UC-AUTH-01 · Пользователь входит." href="status.html"`,
		`class="recommended-entry" href="architecture/overview.html"`,
		"Nearest outcome",
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("dashboard focus missing %q", expected)
		}
	}
	if strings.Contains(html, "Add verification workflow") {
		t.Fatal("dashboard focus must not list active task names")
	}

	model.CurrentStatus.NextResult = nil
	model.Project.StatusDocument = nil
	model.Project.Status = StatusFor("")
	model.DocByPath["status.md"] = nil
	model.DocByPath["risks.md"] = nil
	model.Stats.OpenRisks = 0
	model.Knowledge.WorkItems = append(model.Knowledge.WorkItems, WorkItem{})
	html = renderDashboard(model)
	for _, expected := range []string{
		"The next outcome is not defined.",
		`href="work/index.html"`,
	} {
		if !strings.Contains(html, expected) {
			t.Fatalf("dashboard fallback missing %q", expected)
		}
	}
	focusStart := strings.Index(html, `class="dashboard-section dashboard-focus"`)
	if focusStart >= 0 {
		focusEnd := strings.Index(html[focusStart:], `</a>`)
		if focusEnd > 0 && strings.Contains(html[focusStart:focusStart+focusEnd], `href="risks.html"`) {
			t.Fatal("dashboard must not link to a missing risks page")
		}
	}

	if !strings.Contains(html, `<section class="dashboard-section dashboard-overview" data-dashboard-overview`) ||
		strings.Contains(html, `<details class="dashboard-section dashboard-overview"`) {
		t.Fatal("dashboard overview must remain permanently visible without a disclosure control")
	}

	model.SiteConfig.Project.Locale = "ru"
	html = renderDashboard(model)
	if !strings.Contains(html, `<span class="focus-status">Не указан</span>`) {
		t.Fatalf("Russian portal must localize an unspecified project status: %s", html)
	}
	headingEnd := strings.Index(html, `</div><div class="dashboard-overview-body">`)
	if headingEnd < 0 || !strings.Contains(html[:headingEnd], `data-copy-document-context`) {
		t.Fatal("dashboard context actions must stay in the overview heading")
	}
	mainStart := strings.Index(html, `<main id="main-content"`)
	if mainStart < 0 || strings.Count(html[mainStart:], model.Project.Description) != 1 {
		t.Fatal("project description must appear once in the cover and not repeat in the overview")
	}
}

func TestPortalDoesNotRepeatDocumentDescription(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "index.md", "# Test Project\n\nОписание тестового\nпроекта.\n\n## Раздел\n\nСодержимое.\n")
	model := buildFixture(t, docs)

	dashboard := renderDashboard(model)
	dashboard = dashboard[strings.Index(dashboard, `<main id="main-content"`):]
	if count := strings.Count(dashboard, "Описание тестового"); count != 1 {
		t.Fatalf("dashboard description appears %d times", count)
	}
	document := renderDocumentPage(model, model.DocByPath["modules/auth.md"])
	document = document[strings.Index(document, `<main id="main-content"`):]
	if count := strings.Count(document, "Модуль входа."); count != 1 {
		t.Fatalf("document description appears %d times", count)
	}
}

func TestGenerateMermaidSiteAssetsAndMarkup(t *testing.T) {
	_, docs, output := createFixture(t)
	useCasePath := filepath.Join(docs, "use-cases", "login.md")
	file, err := os.OpenFile(useCasePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteString("\n## Схема\n\n```mermaid\nflowchart TD\n    Login --> Dashboard\n```\n")
	_ = file.Close()

	model := buildFixture(t, docs)
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"assets/mermaid.tiny.js", "assets/mermaid.LICENSE.txt"} {
		if _, err := os.Stat(filepath.Join(output, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing embedded Mermaid asset %s: %v", name, err)
		}
	}
	htmlBytes, err := os.ReadFile(filepath.Join(output, "use-cases", "UC-AUTH-01.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	for _, part := range []string{`data-mermaid-diagram`, `class="mermaid-source"`, `../assets/portal.js`, `id="toudocu-page"`, "Show source code"} {
		if !strings.Contains(html, part) {
			t.Fatalf("Mermaid page missing %q: %s", part, html)
		}
	}
	if strings.Contains(html, `data-mermaid-stage`) {
		t.Fatalf("non-flow Mermaid page must keep the static diagram layout: %s", html)
	}
	if !strings.Contains(html, `type="module"`) || strings.Contains(html, "cdn.") {
		t.Fatalf("Mermaid page must use local ES modules without CDN: %s", html)
	}
	if strings.Contains(html, "assets/mermaid.tiny.js") {
		t.Fatal("Mermaid bundle must be loaded lazily by portal.js")
	}
	indexBytes, _ := os.ReadFile(filepath.Join(output, "index.html"))
	if strings.Contains(string(indexBytes), "assets/mermaid.tiny.js") {
		t.Fatal("pages without diagrams must not load Mermaid")
	}
}

func TestPortalHeavyAssetsAreLazy(t *testing.T) {
	_, docs, output := createFixture(t)
	model := buildFixture(t, docs)
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, eager := range []string{"search-index.json", "assets/mermaid.tiny.js", "assets/screen-map.js", "assets/playable-flow.js"} {
		if strings.Contains(string(page), eager) {
			t.Fatalf("ordinary page eagerly loads %s", eager)
		}
	}
	app, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "core", "portal.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, lazy := range []string{"search-index.json", "loadScript('mermaid.tiny.js')", "loadScript('screen-map.js')", "loadScript('playable-flow.js')"} {
		if !strings.Contains(string(app), lazy) {
			t.Fatalf("portal source missing lazy loader %q", lazy)
		}
	}
}

func TestDocumentationAssetCannotReplaceMermaidLicense(t *testing.T) {
	root, docs, output := createFixture(t)
	writeTestFile(t, docs, "assets/mermaid.LICENSE.txt", "untrusted")
	indexPath := filepath.Join(docs, "index.md")
	index, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = index.WriteString("\n[Лицензия](assets/mermaid.LICENSE.txt)\n")
	_ = index.Close()
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	link := model.DocByPath["index.md"].ResolvedLinks[len(model.DocByPath["index.md"].ResolvedLinks)-1]
	if link.AssetPath != "_files/assets/mermaid.LICENSE.txt" {
		t.Fatalf("reserved Mermaid license path was not isolated: %#v", link)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	embedded, _ := os.ReadFile(filepath.Join(output, "assets", "mermaid.LICENSE.txt"))
	copied, _ := os.ReadFile(filepath.Join(output, "_files", "assets", "mermaid.LICENSE.txt"))
	if strings.TrimSpace(string(copied)) != "untrusted" || strings.TrimSpace(string(embedded)) == "untrusted" {
		t.Fatal("documentation asset replaced embedded Mermaid license")
	}
}

func TestBrokenLinksDoNotStopGeneration(t *testing.T) {
	_, docs, output := createFixture(t)
	f, _ := os.OpenFile(filepath.Join(docs, "status.md"), os.O_APPEND|os.O_WRONLY, 0644)
	_, _ = f.WriteString("\n[Нет файла](missing.md)\n")
	_ = f.Close()
	model := buildFixture(t, docs)
	if model.Stats.BrokenLinks != 1 {
		t.Fatalf("broken=%d", model.Stats.BrokenLinks)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
}

func TestCLIArguments(t *testing.T) {
	options, _, _, err := ParseArguments([]string{"build", "./docs", "--output", "./out", "--title=Проект", "--exclude", "tmp,cache", "--stale-days", "30", "--repository-root", ".", "--repository-url=https://github.com/example/project", "--repository-ref", "abc123", "--clean", "--strict"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Title != "Проект" || options.StaleDays != 30 || len(options.Excludes) != 2 || !options.Clean || !options.Strict || options.RepositoryRef != "abc123" || !filepath.IsAbs(options.InputDirectory) {
		t.Fatalf("options: %#v", options)
	}
	if _, _, _, err := ParseArguments([]string{"./docs"}); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("implicit build must be rejected, got %v", err)
	}
	if _, _, _, err := ParseArguments([]string{"init"}); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("init must be rejected as an unknown command, got %v", err)
	}
	if _, _, _, err := ParseArguments([]string{"refresh"}); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("refresh must be rejected as an unknown command, got %v", err)
	}
	if _, _, _, err := ParseArguments([]string{"build", "./docs", "--force"}); err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("--force must be rejected as an unknown option, got %v", err)
	}
	noMap, _, _, err := ParseArguments([]string{"build", "./docs", "--no-screen-map"})
	if err != nil || !noMap.NoScreenMap {
		t.Fatalf("--no-screen-map not parsed: %#v, %v", noMap, err)
	}
	if _, _, _, err := ParseArguments([]string{"build", "./docs", "--screen-map", "--no-screen-map"}); err == nil {
		t.Fatal("conflicting screen map options must be rejected")
	}
	if _, _, _, err := ParseArguments([]string{"check", "./docs", "--no-screen-map"}); err == nil {
		t.Fatal("screen map build option must be rejected by check")
	}
}

func TestChangeFlagsRejectedOutsideChanges(t *testing.T) {
	options := [][]string{
		{"--base=HEAD"}, {"--base", "HEAD"},
		{"--branch-base=main"}, {"--branch-base", "main"},
		{"--status=modified"}, {"--status", "modified"},
		{"--module=MOD-CLI"}, {"--module", "MOD-CLI"},
		{"--permanent-only"},
		{"--include-assets"},
		{"--translation-input"},
	}
	nonChangesCommands := [][]string{
		{"build", "./docs"},
		{"check", "./docs"},
		{"serve", "./docs"},
		{"search", "query", "./docs"},
		{"scaffold", "module", "MOD-TEST", "./docs", "--title", "Test"},
		{"task", "init", "./docs", "--area", "TEST", "--title", "Test", "--type", "Bug"},
		{"task", "ready", "BUG-CLI-001", "./docs"},
		{"task", "context", "BUG-CLI-001", "./docs"},
		{"task", "verify", "BUG-CLI-001", "./docs", "--dry-run"},
		{"task", "archive", "BUG-CLI-001", "./docs"},
		{"task", "restore", "BUG-CLI-001", "./docs"},
	}
	changesCommands := [][]string{
		{"changes", "./docs"},
		{"changes", "file", "docs/index.md", "./docs"},
		{"task", "changes", "BUG-CLI-001", "./docs"},
	}
	for _, option := range options {
		for _, command := range nonChangesCommands {
			args := append(append([]string{}, command...), option...)
			if _, _, _, err := ParseArguments(args); err == nil || !strings.Contains(err.Error(), "only for changes") {
				t.Fatalf("changes-only option %v must be rejected by %v, got %v", option, command, err)
			}
		}
		for _, command := range changesCommands {
			args := append(append([]string{}, command...), option...)
			if _, _, _, err := ParseArguments(args); err != nil {
				t.Fatalf("changes-only option %v must be accepted by %v: %v", option, command, err)
			}
		}
	}
}

func TestContextualHelp(t *testing.T) {
	tests := []struct {
		args      []string
		contains  []string
		forbidden []string
	}{
		{[]string{"check", "--help"}, []string{"Side effects: none", "--strict", "--format text|json"}, []string{"--host", "--clean"}},
		{[]string{"serve", "--help"}, []string{"HTTP/editor workspace", "--host ADDRESS", "roadmap add"}, []string{"--base REV"}},
		{[]string{"changes", "--help"}, []string{"--include-assets", "--translation-input", "-o writes"}, []string{"--report"}},
		{[]string{"task", "--help"}, []string{"init|ready|candidates|context|verify|archive|restore|changes"}, []string{"requires TASK-ID"}},
		{[]string{"task", "candidates", "--help"}, []string{"all active work items", "--parent TASK-ID"}, []string{"requires TASK-ID"}},
		{[]string{"task", "verify", "--help"}, []string{"--dry-run does not run commands", "--report writes a JSON file"}, []string{"--include-assets", "--translation-input"}},
		{[]string{"task", "changes", "--help"}, []string{"task-scoped read-only", "TASK-ID", "--translation-input"}, []string{"--task"}},
		{[]string{"scaffold", "--help"}, []string{".toudocu/config.yml", "fallback is en"}, []string{"--host"}},
		{[]string{"changes", "file", "--help"}, []string{"one changed path", "PATH", "--translation-input"}, []string{"--task"}},
	}
	for _, test := range tests {
		var stdout, stderr strings.Builder
		if code := RunCLI(test.args, &stdout, &stderr); code != 0 {
			t.Fatalf("%v: exit=%d stderr=%s", test.args, code, stderr.String())
		}
		for _, expected := range test.contains {
			if !strings.Contains(stdout.String(), expected) {
				t.Errorf("%v help missing %q:\n%s", test.args, expected, stdout.String())
			}
		}
		for _, forbidden := range test.forbidden {
			if strings.Contains(stdout.String(), forbidden) {
				t.Errorf("%v help contains inapplicable %q:\n%s", test.args, forbidden, stdout.String())
			}
		}
	}
}

func TestCLIHelpUsesEnglish(t *testing.T) {
	var topLevel strings.Builder
	PrintHelp(&topLevel)
	if strings.IndexFunc(topLevel.String(), func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0 {
		t.Errorf("top-level help contains Cyrillic interface text:\n%s", topLevel.String())
	}

	topics := []string{"build", "check", "serve", "changes", "agent", "changes-file", "search", "scaffold", "task", "task-init", "task-ready", "task-candidates", "task-context", "task-tree", "task-verify", "task-archive", "task-restore", "task-changes", "skill", "version"}
	for _, topic := range topics {
		var output strings.Builder
		PrintCommandHelp(&output, topic)
		if strings.IndexFunc(output.String(), func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0 {
			t.Errorf("%q help contains Cyrillic interface text:\n%s", topic, output.String())
		}
	}
}

func TestTextReportsUseEnglishInterfaceLabels(t *testing.T) {
	var output strings.Builder
	printCheckText(&output, &Model{})
	printChangesText(&output, &ChangeSetReport{})
	printSearchText(&output, SearchReport{})
	printTaskReadyText(&output, TaskReadyReport{Task: TaskVerifyTask{ID: "TASK-CLI-001"}})
	printTaskCandidatesText(&output, TaskCandidatesReport{Candidates: []TaskCandidate{{ID: "TASK-CLI-002", ReadyForWork: true}}})
	printTaskContextText(&output, TaskContextReport{Task: WorkItem{ID: "TASK-CLI-001"}})
	printTaskVerifyText(&output, TaskVerifyReport{Task: TaskVerifyTask{ID: "TASK-CLI-001"}})
	printTaskMoveText(&output, TaskMoveReport{Task: TaskMoveTask{ID: "TASK-CLI-001"}, Status: "archived", DestinationPath: "work/archive/2026/TASK-CLI-001.md"})
	printAgentHelp(&output)

	for _, expected := range []string{"Documents:", "Documentation changes", "Found:", "Readiness status:", "Required documents:", "Verification status:", "archived:", "Usage:"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("text reports missing %q:\n%s", expected, output.String())
		}
	}
	if strings.IndexFunc(output.String(), func(r rune) bool { return unicode.Is(unicode.Cyrillic, r) }) >= 0 {
		t.Errorf("text reports contain Cyrillic interface text:\n%s", output.String())
	}
}

func TestTaskVerifyCLIArguments(t *testing.T) {
	options, _, _, err := ParseArguments([]string{
		"task", "verify", "TASK-AUTH-014", "./docs", "--run",
		"--repository-root", ".", "--format", "json", "--report", "./task-report.json", "--timeout", "45s",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.Command != "task-verify" || options.TaskID != "TASK-AUTH-014" || options.Format != "json" || options.Timeout != 45*time.Second || !filepath.IsAbs(options.ReportPath) {
		t.Fatalf("options: %#v", options)
	}
	if _, _, _, err := ParseArguments([]string{"check", "./docs", "--timeout", "1s"}); err == nil {
		t.Fatal("--timeout must be rejected outside task verify")
	}
	if _, _, _, err := ParseArguments([]string{"task", "verify", "TASK-AUTH-014", "./docs", "--run", "--report", "./docs/report.json"}); err == nil {
		t.Fatal("--report must not overwrite source documentation")
	}
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "docs-alias")
	if err := os.Symlink(docs, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, _, err := ParseArguments([]string{
		"task", "verify", "TASK-AUTH-014", docs, "--run", "--report", filepath.Join(alias, "report.json"),
	}); err == nil {
		t.Fatal("--report must not overwrite source documentation through a symlink")
	}
}

func TestKnowledgeModel(t *testing.T) {
	_, docs, _ := createFixture(t)
	content := "# TASK-AUTH-001: Защитить вход\n\n" +
		"- Статус: В работе\n- Тип: Feature\n- Приоритет: Высокий\n- Модуль: MOD-AUTH\n- Сценарий: UC-AUTH-01\n\n" +
		"## Результат\n\nВход проверяет ограничения.\n\n" +
		"## Изменение поведения\n\n### Было\n\nВход не учитывал ограничение.\n\n### Станет\n\nВход учитывает ограничение.\n\n" +
		"## Область изменения\n\n- `docs/`\n\n" +
		"## Не входит в задачу\n\nПрофиль.\n\n" +
		"## Критерии приёмки\n\n- [x] `AC-01` Неверный пароль отклоняется.\n\n" +
		"## План\n\n1. Проверить поток.\n2. Изменить реализацию.\n3. Запустить тесты и обновить документацию.\n\n" +
		"## Проверка\n\n- `AC-01` → `go test ./...`\n- `ALL` → `go test ./...`\n- `DOCS` → `go test ./...`\n\n" +
		"## Влияние на документацию\n\nОбновить сценарий входа.\n"
	writeTestFile(t, docs, "work/TASK-AUTH-001-login.md", content)
	model := buildFixture(t, docs)
	if len(model.Knowledge.Modules) != 1 || len(model.Knowledge.UseCases) != 1 || len(model.Knowledge.BusinessRules) != 1 || len(model.Knowledge.WorkItems) != 1 || model.Knowledge.WorkItems[0].ID != "TASK-AUTH-001" || len(model.Knowledge.WorkItems[0].Verification) != 1 || model.Stats.Errors != 0 {
		t.Fatalf("knowledge %#v issues %#v", model.Knowledge, model.Issues)
	}
}

func issueCodes(model *Model) map[string]bool {
	result := map[string]bool{}
	for _, issue := range model.Issues {
		result[issue.Code] = true
	}
	return result
}

func TestWorkItemValidationRules(t *testing.T) {
	_, docs, _ := createFixture(t)
	content := `# TASK-AUTH-002: Некорректная завершённая задача

- Статус: Выполнено
- Тип: Feature
- Модуль: MOD-AUTH
- Сценарий: UC-AUTH-01

## Результат

Результат.
- [ ] Чекбокс вне критериев и плана.

## Область изменения

- ` + "`missing/path.go`" + `
- ` + "`../outside.go`" + `

## Не входит в задачу

Прочее.

## Критерии приёмки

- [x] ` + "`AC-01`" + ` Первый критерий.
- [ ] ` + "`AC-01`" + ` Повтор критерия.

## План

- [ ] Чекбокс в плане.
- [x] Завершённый шаг в плане.

## Проверка

- ` + "`AC-02`" + ` → ` + "`go test ./...`" + `

## Влияние на документацию
`
	writeTestFile(t, docs, "work/TASK-AUTH-002-invalid.md", content)
	model := buildFixture(t, docs)
	codes := issueCodes(model)
	for _, code := range []string{
		"duplicate-acceptance-criterion-id",
		"unknown-criterion-verification",
		"missing-criterion-verification",
		"task-checkbox-outside-criteria",
		"incomplete-completed-task",
		"missing-completed-task-check",
		"missing-scope-path",
		"unsafe-scope-path",
		"empty-work-section",
	} {
		if !codes[code] {
			t.Fatalf("missing issue %s in %#v", code, model.Issues)
		}
	}
}

func TestWorkItemPlanChecklistAllowed(t *testing.T) {
	_, docs, _ := createFixture(t)
	content := strings.Replace(
		taskVerifyFixture("Готово к работе", false, map[string]string{
			"AC-01": "go test ./...",
			"AC-02": "go test ./...",
			"ALL":   "go test ./...",
			"DOCS":  "go run ./cmd/toudocu check ./docs --strict",
		}, ""),
		"## План\n\n1. Подготовить команды.\n2. Выполнить проверки.\n3. Сформировать отчёт и обновить документацию.",
		"## План\n\n- [x] Выполненный шаг.\n- [ ] Следующий шаг.",
		1,
	)
	writeTestFile(t, docs, "work/TASK-AUTH-020-plan-checklist.md", content)
	model := buildFixture(t, docs)
	for _, issue := range model.Issues {
		if issue.DocumentPath == "work/TASK-AUTH-020-plan-checklist.md" && issue.Code == "task-checkbox-outside-criteria" {
			t.Fatalf("plan checklist must be allowed: %#v", issue)
		}
	}
}

func TestDraftWorkItemCanBeMinimal(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/TASK-AUTH-030-draft.md", `# TASK-AUTH-030: Исследовать вариант

- Статус: Черновик
- Тип: Research

## Результат

Понятно, стоит ли продолжать реализацию.
`)
	model := buildFixture(t, docs)
	for _, issue := range model.Issues {
		if issue.DocumentPath == "work/TASK-AUTH-030-draft.md" {
			t.Fatalf("minimal draft must be valid: %#v", issue)
		}
	}
	item := model.Knowledge.WorkItems[0]
	if item.ID != "TASK-AUTH-030" || item.Criteria == nil || item.Verification == nil || item.Checks == nil || item.RepositoryPaths == nil {
		t.Fatalf("minimal draft has unstable collections: %#v", item)
	}

	writeTestFile(t, docs, "work/TASK-AUTH-031-ready.md", `# TASK-AUTH-031: Начать реализацию

- Статус: Готово к работе
- Тип: Feature

## Результат

Результат реализован.
`)
	model = buildFixture(t, docs)
	found := false
	for _, issue := range model.Issues {
		if issue.DocumentPath == "work/TASK-AUTH-031-ready.md" && issue.Code == "missing-work-section" {
			found = true
		}
	}
	if !found {
		t.Fatal("ready task must require the complete workflow contract")
	}
}

func TestSpecialTaskStatusesAndSingleTaskFile(t *testing.T) {
	_, docs, _ := createFixture(t)
	task := func(id, status string) string {
		return "# " + id + ": Статусная задача\n\n" +
			"- Статус: " + status + "\n- Тип: Feature\n- Модуль: MOD-AUTH\n- Сценарий: UC-AUTH-01\n\n" +
			"## Результат\n\nРезультат.\n\n## Область изменения\n\n- `docs/`\n\n" +
			"## Не входит в задачу\n\nПрочее.\n\n## Критерии приёмки\n\n- [ ] `AC-01` Результат наблюдаем.\n\n" +
			"## План\n\n1. Проверить.\n2. Реализовать.\n3. Протестировать и обновить документацию.\n\n" +
			"## Проверка\n\n- `AC-01` → `go test ./...`\n\n## Влияние на документацию\n\nНе требуется: поведение не меняется.\n"
	}
	writeTestFile(t, docs, "work/TASK-AUTH-003-blocked.md", task("TASK-AUTH-003", "Заблокировано"))
	writeTestFile(t, docs, "work/TASK-AUTH-004-cancelled.md", task("TASK-AUTH-004", "Отменено"))
	writeTestFile(t, docs, "work/two-items.md", "# Список\n\n## TASK-AUTH-005: Первая\n\n## TASK-AUTH-006: Вторая\n")
	model := buildFixture(t, docs)
	codes := issueCodes(model)
	for _, code := range []string{"missing-work-section"} {
		if !codes[code] {
			t.Fatalf("missing issue %s in %#v", code, model.Issues)
		}
	}
	messages := strings.Builder{}
	for _, issue := range model.Issues {
		messages.WriteString(issue.Message)
		messages.WriteByte('\n')
	}
	if !strings.Contains(messages.String(), "Blocker") || !strings.Contains(messages.String(), "Cancellation reason") {
		t.Fatalf("special status sections not validated: %s", messages.String())
	}
}

func TestTaskTypeAndUseCaseRules(t *testing.T) {
	_, docs, _ := createFixture(t)
	task := func(id, status, taskType string) string {
		return "# " + id + ": Техническая задача\n\n" +
			"- Статус: " + status + "\n- Тип: " + taskType + "\n- Модуль: MOD-AUTH\n\n" +
			"## Результат\n\nРезультат.\n\n## Область изменения\n\n- `docs/`\n\n" +
			"## Не входит в задачу\n\nПрочее.\n\n## Критерии приёмки\n\n- [ ] `AC-01` Результат наблюдаем.\n\n" +
			"## План\n\n1. Проверить.\n2. Реализовать.\n3. Протестировать и обновить документацию.\n\n" +
			"## Проверка\n\n- `AC-01` → `go test ./...`\n\n## Влияние на документацию\n\nНе требуется: поведение не меняется.\n"
	}
	writeTestFile(t, docs, "work/TASK-AUTH-008-invalid-enums.md", task("TASK-AUTH-008", "Запланировано", "Chore"))
	writeTestFile(t, docs, "work/TASK-AUTH-009-maintenance.md", task("TASK-AUTH-009", "Готово к работе", "Maintenance"))
	model := buildFixture(t, docs)
	codes := issueCodes(model)
	for _, code := range []string{"invalid-task-status", "invalid-task-type", "missing-use-case-omission-reason"} {
		if !codes[code] {
			t.Fatalf("missing issue %s in %#v", code, model.Issues)
		}
	}
}

func TestStatusAndRoadmapConsistency(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "status.md", "# Состояние\n\n- Статус: В работе\n\n## В работе\n\n- [ ] Сделать авторизацию.\n")
	writeTestFile(t, docs, "roadmap.md", "# Roadmap\n\n## MVP\n\n- Status: In progress\n\n- [x] `UC-AUTH-01` Пользователь входит.\n- [ ] Произвольный результат.\n- [ ] `UC-UNKNOWN-01` Неизвестный сценарий.\n")
	model := buildFixture(t, docs)
	codes := issueCodes(model)
	for _, code := range []string{"status-requirement-checklist", "invalid-roadmap-item-id", "dangling-roadmap-reference"} {
		if !codes[code] {
			t.Fatalf("missing issue %s in %#v", code, model.Issues)
		}
	}
	if model.RoadmapStages[0].Items[0].EffectiveCompleted {
		t.Fatal("roadmap completion must be derived from the unfinished use case")
	}
}

func TestRoadmapCompletionIsDerivedAndRendered(t *testing.T) {
	root, docs, output := createFixture(t)
	useCasePath := filepath.Join(docs, "use-cases", "login.md")
	content, err := os.ReadFile(useCasePath)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, docs, "use-cases/login.md", strings.Replace(string(content), "status: in-progress", "status: done", 1))
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	stage := model.RoadmapStages[0]
	if stage.TaskStats.Completed != 1 || model.Stats.CompletedTasks != 1 || stage.TaskStats.Remaining != 1 || stage.Items[1].DeclaredCompleted || stage.Items[1].EffectiveCompleted || stage.Items[1].CompletionSource != "use-case-status" {
		t.Fatalf("derived roadmap: %#v %#v", stage, model.Stats)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	roadmapHTML, err := os.ReadFile(filepath.Join(output, "roadmap.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(roadmapHTML), `data-task-state="open"`) {
		t.Fatalf("roadmap HTML did not apply use-case readiness: %s", roadmapHTML)
	}
	report, err := json.Marshal(BuildReport(model))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`"schemaVersion":1`, `"declaredCompleted":false`, `"effectiveCompleted":false`, `"completionSource":"use-case-status"`} {
		if !strings.Contains(string(report), part) {
			t.Fatalf("missing %s in %s", part, report)
		}
	}
}

func TestRoadmapContractAndDeliverableRemainManual(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "contracts/auth.md", "# Auth contract\n\n- Идентификатор: CON-AUTH-API\n- Статус: В работе\n\nContract.\n")
	writeTestFile(t, docs, "contracts/session.md", "# Session contract\n\n- Идентификатор: CONTRACT-SESSION-API\n- Статус: В работе\n\nContract.\n")
	writeTestFile(t, docs, "roadmap.md", "# Roadmap\n\n## MVP\n\n- Status: In progress\n\n- [x] `CON-AUTH-API` Контракт опубликован.\n- [ ] `CONTRACT-SESSION-API` Контракт сессии опубликован.\n- [x] `DLV-RELEASE-01` Релиз подготовлен.\n- [ ] `DELIVERABLE-NOTES-01` Примечания подготовлены.\n")
	model := buildFixture(t, docs)
	items := model.RoadmapStages[0].Items
	if len(items) != 4 || !items[0].EffectiveCompleted || items[0].CompletionSource != "roadmap-checkbox" || items[1].EffectiveCompleted || !items[2].EffectiveCompleted || items[3].EffectiveCompleted {
		t.Fatalf("manual roadmap items: %#v", items)
	}
}

func TestComputedStatusIsSummarizedOnDashboardAndDetailedOnStatusPage(t *testing.T) {
	root, docs, output := createFixture(t)
	commands := map[string]string{"AC-01": "pass", "AC-02": "pass", "ALL": "pass", "DOCS": "pass"}
	task := taskVerifyFixture("Заблокировано", false, commands, "\n## Блокер\n\nОжидается решение ADR-014.\n")
	writeTestFile(t, docs, "work/TASK-AUTH-020-check.md", task)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.CurrentStatus.ActiveWork) != 1 || len(model.CurrentStatus.Blockers) != 1 || model.CurrentStatus.NextResult == nil || model.CurrentStatus.NextResult.ID != "UC-AUTH-01" {
		t.Fatalf("current status: %#v", model.CurrentStatus)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	dashboardData, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	dashboard := string(dashboardData)
	for _, part := range []string{"Current focus", "Nearest outcome", "UC-AUTH-01"} {
		if !strings.Contains(dashboard, part) {
			t.Fatalf("index.html missing %q", part)
		}
	}
	focusStart := strings.Index(dashboard, `class="dashboard-section dashboard-focus"`)
	focusEnd := strings.Index(dashboard, `class="dashboard-section recommended-entries"`)
	if focusStart < 0 || focusEnd <= focusStart {
		t.Fatal("dashboard focus bounds are missing")
	}
	focus := dashboard[focusStart:focusEnd]
	for _, detail := range []string{"TASK-AUTH-020", "Ожидается решение ADR-014", "Computed status"} {
		if strings.Contains(focus, detail) {
			t.Fatalf("dashboard focus must not list detail %q", detail)
		}
	}
	statusData, err := os.ReadFile(filepath.Join(output, "status.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"Computed status", "TASK-AUTH-020", "Ожидается решение ADR-014", "UC-AUTH-01"} {
		if !strings.Contains(string(statusData), part) {
			t.Fatalf("status.html missing %q", part)
		}
	}
	if !strings.Contains(string(statusData), `class="dashboard-section dashboard-support-panel"`) {
		t.Fatal("status computed state must render as a support panel")
	}
}

func TestVerificationMatrixIsReported(t *testing.T) {
	_, docs, _ := createFixture(t)
	content := `# TASK-AUTH-007: Матрица

- Статус: Черновик
- Тип: Feature
- Модуль: MOD-AUTH
- Сценарий: UC-AUTH-01

## Результат

Результат.

## Область изменения

Будет уточнена.

## Не входит в задачу

Прочее.

## Критерии приёмки

- [ ] ` + "`AC-01`" + ` Результат наблюдаем.

## План

Будет уточнён.

## Проверка

- ` + "`AC-01`" + ` → ` + "`go test ./...`" + `

## Влияние на документацию

Не требуется: это тест.
`
	writeTestFile(t, docs, "work/TASK-AUTH-007-matrix.md", content)
	stored, err := os.ReadFile(filepath.Join(docs, "work", "TASK-AUTH-007-matrix.md"))
	if err != nil {
		t.Fatal(err)
	}
	model := buildFixture(t, docs)
	expectedLine := 0
	for index, line := range strings.Split(string(stored), "\n") {
		if strings.Contains(line, "AC-01") && strings.Contains(line, "[ ]") {
			expectedLine = index + 1
			break
		}
	}
	if line := model.Knowledge.WorkItems[0].Criteria[0].Line; line != expectedLine {
		t.Fatalf("criterion line must be 1-based: got %d want %d", line, expectedLine)
	}
	data, err := json.Marshal(BuildReport(model))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`"verificationMatrix"`, `"criterionId":"AC-01"`, `"commands":["go test ./..."]`} {
		if !strings.Contains(string(data), part) {
			t.Fatalf("report does not contain %s: %s", part, data)
		}
	}
}

func TestRepositoryLinksAndTraversal(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, root, "src/auth.go", "package auth\n")
	f, _ := os.OpenFile(filepath.Join(docs, "modules/auth.md"), os.O_APPEND|os.O_WRONLY, 0644)
	_, _ = f.WriteString("\n[Код](../../src/auth.go)\n[Снаружи](../../../outside.txt)\n")
	_ = f.Close()
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, RepositoryURL: "https://github.com/example/project", RepositoryRef: "abc123", StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	links := model.DocByPath["modules/auth.md"].ResolvedLinks
	if links[len(links)-2].RepositoryPath != "src/auth.go" || links[len(links)-2].Href != "https://github.com/example/project/blob/abc123/src/auth.go" || !links[len(links)-1].Blocked {
		t.Fatalf("links %#v", links[len(links)-2:])
	}
}

func TestTaskDependencyCycle(t *testing.T) {
	_, docs, _ := createFixture(t)
	task := func(id, dep string) string {
		return "# " + id + ": Цикл\n\n" +
			"- Статус: В работе\n- Тип: Feature\n- Модуль: MOD-AUTH\n- Сценарий: UC-AUTH-01\n- Зависит от: " + dep + "\n\n" +
			"## Результат\n\nРезультат.\n\n" +
			"## Область изменения\n\n- `docs/`\n\n" +
			"## Не входит в задачу\n\nПрочее.\n\n" +
			"## Критерии приёмки\n\n- [ ] `AC-01` Проверено.\n\n" +
			"## План\n\n1. Исследовать.\n2. Реализовать.\n3. Проверить и обновить документацию.\n\n" +
			"## Проверка\n\n- `AC-01` → `go test ./...`\n\n" +
			"## Влияние на документацию\n\nНе требуется: тестовая задача.\n"
	}
	writeTestFile(t, docs, "work/TASK-AUTH-001.md", task("TASK-AUTH-001", "TASK-AUTH-002"))
	writeTestFile(t, docs, "work/TASK-AUTH-002.md", task("TASK-AUTH-002", "TASK-AUTH-001"))
	model := buildFixture(t, docs)
	found := false
	for _, issue := range model.Issues {
		if issue.Code == "task-dependency-cycle" {
			found = true
		}
	}
	if !found || model.Stats.Errors == 0 {
		t.Fatal("cycle not reported")
	}
}

func TestDuplicateAndDanglingRule(t *testing.T) {
	_, docs, _ := createFixture(t)
	writeTestFile(t, docs, "modules/duplicate.md", "# Duplicate\n\n- Идентификатор: MOD-AUTH\n- Статус: В работе\n\nDuplicate module.\n")
	f, _ := os.OpenFile(filepath.Join(docs, "use-cases/login.md"), os.O_APPEND|os.O_WRONLY, 0644)
	_, _ = f.WriteString("\nНеизвестное правило BR-AUTH-999.\n")
	_ = f.Close()
	model := buildFixture(t, docs)
	codes := map[string]bool{}
	for _, issue := range model.Issues {
		codes[issue.Code] = true
	}
	if !codes["duplicate-id"] || !codes["dangling-rule-reference"] {
		t.Fatalf("codes %#v", codes)
	}
}

func TestHealthCollision(t *testing.T) {
	_, docs, output := createFixture(t)
	writeTestFile(t, docs, "health.md", "# Пользовательский health\n\nОписание пользовательского документа.\n")
	model := buildFixture(t, docs)
	if model.DocByPath["health.md"].OutputPath != "health.html" || model.HealthOutputPath != "documentation-health.html" {
		t.Fatalf("paths")
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
}

func TestOutputSafety(t *testing.T) {
	root, docs, _ := createFixture(t)
	model := buildFixture(t, docs)
	if _, err := GenerateSite(model, Options{OutputDirectory: root, Clean: true}); err == nil || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("expected safety error, got %v", err)
	}

	alias := filepath.Join(t.TempDir(), "output-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: alias, Clean: true}); err == nil {
		t.Fatal("output symlink to documentation parent must be rejected")
	}
	if _, err := os.Stat(filepath.Join(docs, "index.md")); err != nil {
		t.Fatalf("source documentation was changed through output symlink: %v", err)
	}

	unrelated := filepath.Join(t.TempDir(), "existing-output")
	if err := os.MkdirAll(unrelated, 0755); err != nil {
		t.Fatal(err)
	}
	directAlias := filepath.Join(t.TempDir(), "direct-output-alias")
	if err := os.Symlink(unrelated, directAlias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: directAlias, Clean: true}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("direct output symlink must be rejected, got %v", err)
	}
}

func TestBuildRejectsExistingOutputSymlinkWithoutClean(t *testing.T) {
	_, docs, output := createFixture(t)
	model := buildFixture(t, docs)
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(output, "index.html")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err == nil {
		t.Fatal("build without clean must reject an existing output symlink")
	}
	content, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "keep" {
		t.Fatalf("file outside output was overwritten: %q", content)
	}
}

func TestMinimalDocumentationCheckAndBuild(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Минимальный проект\n\nКраткое описание проекта.\n\n## Подробности\n\nУникальное содержимое главной страницы.\n")
	writeArchitectureOverview(t, docs, "")
	writeSiteConfig(t, root, "project:\n  locale: ru\n  sections:\n    architecture: Architecture\n    modules: Модули\n    use-cases: Пользовательские сценарии\n    flows: Процессы\n    screens: Экраны\n    decisions: Архитектурные решения\n    contracts: Контракты\n    quality: Стандарты качества\n    runbooks: Runbooks\n    reference: Справочник\n    work: Рабочие задачи\n    guides: Руководства\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if model.Stats.Warnings != 0 || model.Stats.Errors != 0 {
		t.Fatalf("minimal documentation must be valid: %#v", model.Issues)
	}
	var stdout, stderr strings.Builder
	if code := RunCLI([]string{"check", docs, "--repository-root", root, "--strict", "--stale-days", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("strict check failed: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	output := filepath.Join(root, "site")
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", "report.json"} {
		if _, err := os.Stat(filepath.Join(output, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(data)
	if !strings.Contains(html, "Уникальное содержимое главной страницы.") {
		t.Fatalf("index.md body is missing from dashboard: %s", html)
	}
	if !strings.Contains(html, `<section class="dashboard-section dashboard-overview" data-dashboard-overview`) || strings.Contains(html, `<details class="dashboard-section dashboard-overview"`) {
		t.Fatal("minimal dashboard overview must remain permanently visible")
	}
	for _, absent := range []string{"Текущий фокус", "Дорожная карта", "Вычисляемое состояние", "Незакрытые риски", "metric-label\">Модули", "Каталог документации"} {
		if strings.Contains(html, absent) {
			t.Fatalf("minimal dashboard contains optional block %q", absent)
		}
	}
}

func TestMissingIndexRemainsStrictFailure(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "notes.md", "# Заметки\n\nОписание заметок.\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, issue := range model.Issues {
		if issue.Code == "missing-index" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing-index diagnostic not found: %#v", model.Issues)
	}
	var stdout, stderr strings.Builder
	if code := RunCLI([]string{"check", docs, "--repository-root", root, "--strict", "--stale-days", "0"}, &stdout, &stderr); code == 0 {
		t.Fatalf("strict check must fail without index.md: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}
