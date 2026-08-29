package toudocu

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var testMetadataKeys = map[string]string{
	"идентификатор": "id", "identifier": "id", "статус": "status", "status": "status",
	"тип": "type", "type": "type", "приоритет": "priority", "priority": "priority",
	"тип документа": "type", "document type": "type",
	"серьёзность": "severity", "severity": "severity", "воспроизводимость": "reproducibility", "reproducibility": "reproducibility",
	"регрессия": "regression", "regression": "regression", "модуль": "module", "module": "module",
	"наблюдалась в": "observedIn", "observed in": "observedIn",
	"сценарий": "useCase", "scenario": "useCase", "use case": "useCase", "процесс": "flow", "flow": "flow",
	"экраны": "screens", "screens": "screens", "переходы": "transitions", "transitions": "transitions",
	"стандарты": "standards", "standards": "standards", "затронутые runbooks": "runbooks", "affected runbooks": "runbooks",
	"зависит от": "dependsOn", "depends on": "dependsOn", "родительская задача": "parentTask", "parent": "parentTask",
	"dependencies":    "dependsOn",
	"начальный экран": "startScreen", "start screen": "startScreen", "конечные экраны": "terminalScreens", "terminal screens": "terminalScreens",
	"разрешить цикл": "allowCycle", "allow cycle": "allowCycle", "маршрут": "route", "route": "route",
	"превью": "preview", "preview": "preview", "родительский экран": "parentScreen", "parent screen": "parentScreen",
	"компонент": "component", "component": "component", "последнее обновление": "updated", "last updated": "updated",
	"область": "scope", "scope": "scope", "среда": "environment", "environment": "environment",
	"риск": "risk", "risk": "risk", "последняя проверка": "lastVerified", "last verified": "lastVerified",
	"заменён": "supersededBy", "superseded by": "supersededBy", "дата": "date", "date": "date",
	"архитектурный вопрос": "architectureQuestion", "architecture question": "architectureQuestion",
	"вероятность": "probability", "probability": "probability", "влияние": "impact", "impact": "impact",
	"плановая дата": "plannedDate", "planned date": "plannedDate", "этап": "stage", "stage": "stage",
}

var testSectionKinds = map[string]string{
	"краткое состояние": "summary", "summary": "summary", "критерии приёмки": "acceptance-criteria", "критерии приемки": "acceptance-criteria", "acceptance criteria": "acceptance-criteria",
	"проверка": "verification", "verification": "verification", "правила": "rules", "rules": "rules", "автоматические проверки": "automated-checks", "automated checks": "automated-checks",
	"предварительные условия": "prerequisites", "предусловия": "prerequisites", "предпосылки": "prerequisites", "prerequisites": "prerequisites", "preconditions": "prerequisites",
	"процедура": "procedure", "procedure": "procedure", "откат": "rollback", "rollback": "rollback", "условия остановки": "stop-conditions", "stop conditions": "stop-conditions",
	"основной сценарий": "main-scenario", "main scenario": "main-scenario", "постусловия": "postconditions", "postconditions": "postconditions",
	"бизнес-правила": "business-rules", "business rules": "business-rules", "реализация": "implementation", "implementation": "implementation",
	"расположение в коде": "code-location", "code location": "code-location", "границы": "boundaries", "границы модуля": "boundaries", "module boundaries": "boundaries",
	"инварианты": "invariants", "invariants": "invariants", "стабильные интерфейсы": "stable-interfaces", "stable interfaces": "stable-interfaces",
	"связанные сценарии": "related-use-cases", "related use cases": "related-use-cases", "контекст": "context", "context": "context",
	"решение": "decision", "decision": "decision", "последствия": "consequences", "consequences": "consequences",
	"результат": "result", "result": "result", "изменение поведения": "behavior-change", "behavior change": "behavior-change",
	"было": "before", "before": "before", "станет": "after", "after": "after", "область изменения": "scope", "scope": "scope",
	"не входит в задачу": "out-of-scope", "не входит в исправление": "out-of-scope", "out of scope": "out-of-scope",
	"план": "plan", "plan": "plan", "влияние на документацию": "documentation-impact", "documentation impact": "documentation-impact",
	"блокер": "blocker", "blocker": "blocker", "причина отмены": "cancellation-reason", "cancellation reason": "cancellation-reason",
	"обоснование отсутствия сценария": "use-case-omission-reason", "use-case omission reason": "use-case-omission-reason",
	"симптом": "symptom", "symptom": "symptom", "ожидаемое поведение": "expected-behavior", "expected behavior": "expected-behavior",
	"фактическое поведение": "actual-behavior", "actual behavior": "actual-behavior", "шаги воспроизведения": "steps-to-reproduce", "steps to reproduce": "steps-to-reproduce",
	"доказательства": "evidence", "evidence": "evidence", "причина": "cause", "cause": "cause", "регрессионный тест": "regression-test", "regression test": "regression-test",
	"связь с пользовательским поведением": "relationship-to-user-behavior", "relationship to user behavior": "relationship-to-user-behavior",
}

func canonicalTestValue(key, value string) string {
	value = strings.Trim(strings.TrimSpace(value), "`")
	lower := strings.ToLower(value)
	values := map[string]string{
		"черновик": "draft", "draft": "draft", "готово к работе": "ready", "ready": "ready", "запланировано": "planned", "planned": "planned",
		"в работе": "in-progress", "in progress": "in-progress", "заблокировано": "blocked", "blocked": "blocked", "выполнено": "done", "готово": "done", "done": "done", "completed": "done",
		"отменено": "cancelled", "cancelled": "cancelled", "canceled": "cancelled", "действует": "active", "active": "active", "устарел": "obsolete", "obsolete": "obsolete", "deprecated": "obsolete",
		"реализован": "done", "implemented": "done",
		"заменён": "superseded", "superseded": "superseded", "требует проверки": "review-required", "requires review": "review-required", "принято": "accepted", "accepted": "accepted",
		"открыт": "open", "open": "open", "снижается": "in-progress", "риск принят": "risk-accepted",
		"feature": "feature", "функциональность": "feature", "bug": "bug", "ошибка": "bug", "maintenance": "maintenance", "обслуживание": "maintenance",
		"documentation": "documentation", "документация": "documentation", "research": "research", "исследование": "research",
		"экран": "screen", "screen": "screen", "страница": "page", "page": "page", "модальное окно": "modal", "modal window": "modal", "панель": "panel", "panel": "panel",
		"внешняя страница": "external", "external page": "external", "системное состояние": "system", "system state": "system",
		"высокий": "high", "высокая": "high", "высокое": "high", "high": "high", "средний": "medium", "средняя": "medium", "среднее": "medium", "medium": "medium", "низкий": "low", "низкая": "low", "низкое": "low", "low": "low",
		"критический": "critical", "критическая": "critical", "критическое": "critical", "critical": "critical", "обычный": "normal", "normal": "normal", "срочный": "urgent", "urgent": "urgent",
		"всегда": "always", "always": "always", "часто": "often", "often": "often", "иногда": "sometimes", "sometimes": "sometimes", "редко": "rarely", "rarely": "rarely",
		"не воспроизводится": "not-reproduced", "not reproduced": "not-reproduced", "неизвестно": "unknown", "unknown": "unknown", "да": "true", "yes": "true", "нет": "false", "no": "false",
		"не применяется": "not-applicable", "not applicable": "not-applicable",
	}
	if key == "status" || key == "taskType" || key == "screenKind" || key == "risk" || key == "probability" || key == "impact" || key == "priority" || key == "severity" || key == "reproducibility" || key == "regression" || key == "allowCycle" || key == "useCase" {
		if canonical := values[lower]; canonical != "" {
			return canonical
		}
	}
	return value
}

func testTypedKind(relative string) string {
	relative = filepath.ToSlash(relative)
	base := filepath.Base(relative)
	switch relative {
	case "status.md", "roadmap.md", "risks.md", "architecture/overview.md":
		return strings.TrimSuffix(base, ".md")
	}
	directory := strings.Split(relative, "/")[0]
	switch directory {
	case "modules":
		return "module"
	case "use-cases":
		return "use-case"
	case "flows":
		return "flow"
	case "decisions":
		return "decision"
	case "contracts":
		return "contract"
	case "architecture":
		return "architecture"
	case "quality":
		if strings.HasPrefix(base, "STD-") {
			return "standard"
		}
	case "runbooks":
		if strings.HasPrefix(base, "RB-") {
			return "runbook"
		}
	case "screens":
		if strings.HasPrefix(base, "SC-") {
			return "screen"
		}
	case "work":
		if strings.HasPrefix(base, "TASK-") || strings.HasPrefix(base, "BUG-") {
			return "work"
		}
	}
	return ""
}

func canonicalizeTestMarkdown(relative, content string) string {
	kind := testTypedKind(relative)
	if kind == "" || strings.Contains(content, "<!-- toudocu") {
		return content
	}
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	h1 := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			h1 = i
			break
		}
	}
	if h1 < 0 {
		return content
	}
	metadata := map[string]string{}
	id := stableEntityIDRE.FindString(lines[h1])
	if id != "" {
		metadata["id"] = id
	}
	start := h1 + 1
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := start
	preserved := []string{}
	for end < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[end]), "- ") {
		entry := strings.TrimPrefix(strings.TrimSpace(lines[end]), "- ")
		name, value, ok := strings.Cut(entry, ":")
		key := testMetadataKeys[strings.ToLower(strings.TrimSpace(name))]
		if !ok || key == "" {
			preserved = append(preserved, lines[end])
			end++
			continue
		}
		if key == "type" {
			switch kind {
			case "work":
				key = "taskType"
			case "screen":
				key = "screenKind"
			default:
				end++
				continue
			}
		}
		metadata[key] = canonicalTestValue(key, value)
		end++
	}
	if end > start {
		lines = append(lines[:start], append(preserved, lines[end:]...)...)
		h1 = 0
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				h1 = i
				break
			}
		}
	}
	if kind == "roadmap" || kind == "risks" {
		lines = canonicalizeTestRepeatedSections(lines, kind)
	}
	lines = canonicalizeTestSectionsAndTables(lines)
	order := []string{"id", "status", "taskType", "screenKind", "severity", "priority", "reproducibility", "regression", "observedIn", "module", "useCase", "flow", "screens", "transitions", "standards", "runbooks", "dependsOn", "parentTask", "startScreen", "terminalScreens", "allowCycle", "route", "preview", "parentScreen", "component", "stage", "date", "scope", "updated", "environment", "risk", "lastVerified", "supersededBy", "architectureQuestion"}
	block := []string{"<!-- toudocu"}
	seen := map[string]bool{}
	for _, key := range order {
		if metadata[key] != "" {
			block = append(block, key+": "+metadata[key])
			seen[key] = true
		}
	}
	extra := []string{}
	for key := range metadata {
		if !seen[key] && metadata[key] != "" {
			extra = append(extra, key)
		}
	}
	sort.Strings(extra)
	for _, key := range extra {
		block = append(block, key+": "+metadata[key])
	}
	if len(block) == 1 {
		return strings.Join(lines, "\n") + "\n"
	}
	block = append(block, "-->", "")
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			h1 = i
			break
		}
	}
	lines = append(lines[:h1], append(block, lines[h1:]...)...)
	return strings.Join(lines, "\n") + "\n"
}

func ensureTestDocumentationVersion(t testing.TB, root string) {
	t.Helper()
	repositoryRoot := root
	base := filepath.Base(root)
	if base == "docs" || strings.HasPrefix(base, "docs-") {
		repositoryRoot = filepath.Dir(root)
	}
	configPath := filepath.Join(repositoryRoot, ".toudocu", "config.yml")
	if _, err := os.Stat(configPath); err == nil {
		return
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("documentationVersion: 3\nproject:\n  defaultLocale: en\nlocales:\n  en:\n    root: docs\n    sections:\n      architecture: Architecture\n      modules: Modules\n      use-cases: Use Cases\n      flows: Processes\n      screens: Screens\n      decisions: Decisions\n      contracts: Contracts\n      quality: Quality\n      runbooks: Runbooks\n      reference: Reference\n      work: Work\n      drafts: Drafts\n      guides: Guides\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func canonicalizeTestRepeatedSections(lines []string, kind string) []string {
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "## ") {
			result = append(result, lines[i])
			continue
		}
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		metadata := []string{}
		for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "- ") {
			entry := strings.TrimPrefix(strings.TrimSpace(lines[j]), "- ")
			name, value, ok := strings.Cut(entry, ":")
			key := testMetadataKeys[strings.ToLower(strings.TrimSpace(name))]
			if !ok || key == "" {
				break
			}
			metadata = append(metadata, key+": "+canonicalTestValue(key, value))
			j++
		}
		if len(metadata) == 0 {
			result = append(result, lines[i])
			continue
		}
		sectionKind := "roadmap-stage"
		if kind == "risks" {
			sectionKind = "risk"
		}
		result = append(result, "<!-- toudocu:section "+sectionKind+" -->", "<!-- toudocu")
		result = append(result, metadata...)
		result = append(result, "-->", "", lines[i])
		i = j - 1
	}
	return result
}

func canonicalizeTestSectionsAndTables(lines []string) []string {
	result := []string{}
	inFence := false
	tableKind := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			result = append(result, line)
			continue
		}
		if !inFence && (strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ")) {
			title := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "#")))
			if sectionKind := testSectionKinds[title]; sectionKind != "" {
				result = append(result, "<!-- toudocu:section "+sectionKind+" -->")
			}
			tableKind = map[string]string{"состояния": "states", "states": "states", "переходы": "transitions", "transitions": "transitions", "ошибки": "errors", "errors": "errors"}[title]
		}
		if !inFence && tableKind != "" && strings.HasPrefix(trimmed, "|") {
			columns := []string{}
			mapping := map[string]string{"id": "id", "название": "title", "name": "title", "превью": "preview", "preview": "preview", "сценарий": "useCase", "use case": "useCase", "действие": "action", "action": "action", "условие": "condition", "condition": "condition", "результат": "target", "result": "target", "состояние": "state", "state": "state", "ошибка": "error", "error": "error", "сообщение": "message", "message": "message", "контракт": "contract", "contract": "contract", "тип": "kind", "type": "kind"}
			for _, cell := range strings.Split(strings.Trim(trimmed, "|"), "|") {
				columns = append(columns, mapping[strings.ToLower(strings.TrimSpace(cell))])
			}
			valid := len(columns) > 0
			for _, column := range columns {
				valid = valid && column != ""
			}
			if valid {
				result = append(result, "<!-- toudocu:table "+tableKind+" columns="+strings.Join(columns, ",")+" -->")
			}
			tableKind = ""
		}
		result = append(result, line)
	}
	return result
}
