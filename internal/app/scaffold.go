package toudocu

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	taskAreaRE = regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*$`)
	taskIDRE   = regexp.MustCompile(`^(?:TASK|BUG)-[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)*-[0-9]{3,}$`)
	entityIDRE = regexp.MustCompile(`^[A-Z0-9]+(?:-[A-Z0-9]+)*$`)
)

func validTaskInitType(value string) bool {
	template, _ := scaffoldTemplate("task-init")
	for _, field := range template.Fields {
		if field.Name == "type" {
			return containsString(field.Options, value)
		}
	}
	return false
}

func validateScaffoldTitle(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("--title must not be empty")
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("--title must be one line")
	}
	return nil
}

func atomicCreateFile(target, content string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("file already exists: %s", target)
		}
		return err
	}
	name := file.Name()
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(name)
		}
	}()
	if _, err := io.WriteString(file, content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func scaffoldDate(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	return now.Format("2006-01-02")
}

func nextTaskNumber(docsDir, prefix, area string) (int, error) {
	workDirectory := filepath.Join(docsDir, "work")
	idPrefix := prefix + "-" + area + "-"
	maximum := 0

	consider := func(value string) {
		if !strings.HasPrefix(value, idPrefix) {
			return
		}
		suffix := strings.TrimPrefix(value, idPrefix)
		end := 0
		for end < len(suffix) && suffix[end] >= '0' && suffix[end] <= '9' {
			end++
		}
		number, err := strconv.Atoi(suffix[:end])
		if err == nil && number > maximum {
			maximum = number
		}
	}
	err := filepath.WalkDir(workDirectory, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			return nil
		}
		consider(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		content, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return readErr
		}
		consider(analyzeMarkdown(string(content)).Metadata["id"])
		return nil
	})
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return maximum + 1, nil
	}
	return 0, err
}

func renderTaskScaffold(id, title, taskType, language, date, parentID string) string {
	parent := ""
	if parentID != "" {
		parent = "parentTask: " + parentID + "\n"
	}
	if taskType == "Bug" {
		if language == "ru" {
			return fmt.Sprintf(`<!-- toudocu
id: %s
status: draft
taskType: bug
updated: %s
%s-->

# %s: %s

<!-- toudocu:section symptom -->
## Симптом

<!-- toudocu:section expected-behavior -->
## Ожидаемое поведение

<!-- toudocu:section actual-behavior -->
## Фактическое поведение

<!-- toudocu:section steps-to-reproduce -->
## Шаги воспроизведения

<!-- toudocu:section evidence -->
## Доказательства

<!-- toudocu:section cause -->
## Причина

Не установлена.

<!-- toudocu:section scope -->
## Область изменения

<!-- toudocu:section out-of-scope -->
## Не входит в исправление

<!-- toudocu:section plan -->
## План

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

<!-- toudocu:section verification -->
## Проверка

<!-- toudocu:section regression-test -->
## Регрессионный тест

<!-- toudocu:section documentation-impact -->
## Влияние на документацию
`, id, date, parent, id, title)
		}
		return fmt.Sprintf(`<!-- toudocu
id: %s
status: draft
taskType: bug
updated: %s
%s-->

# %s: %s

<!-- toudocu:section symptom -->
## Symptom

<!-- toudocu:section expected-behavior -->
## Expected behavior

<!-- toudocu:section actual-behavior -->
## Actual behavior

<!-- toudocu:section steps-to-reproduce -->
## Steps to reproduce

<!-- toudocu:section evidence -->
## Evidence

<!-- toudocu:section cause -->
## Cause

Not established.

<!-- toudocu:section scope -->
## Scope

<!-- toudocu:section out-of-scope -->
## Out of scope

<!-- toudocu:section plan -->
## Plan

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

<!-- toudocu:section verification -->
## Verification

<!-- toudocu:section regression-test -->
## Regression test

<!-- toudocu:section documentation-impact -->
## Documentation impact
`, id, date, parent, id, title)
	}
	canonicalType := strings.ToLower(taskType)
	if language == "ru" {
		return fmt.Sprintf(`<!-- toudocu
id: %s
status: draft
taskType: %s
updated: %s
%s-->

# %s: %s

<!-- toudocu:section result -->
## Результат

<!-- toudocu:section behavior-change -->
## Изменение поведения

<!-- toudocu:section before -->
### Было

<!-- toudocu:section after -->
### Станет

<!-- toudocu:section scope -->
## Область изменения

<!-- toudocu:section out-of-scope -->
## Не входит в задачу

<!-- toudocu:section acceptance-criteria -->
## Критерии приёмки

<!-- toudocu:section plan -->
## План

<!-- toudocu:section verification -->
## Проверка

<!-- toudocu:section documentation-impact -->
## Влияние на документацию
`, id, canonicalType, date, parent, id, title)
	}
	return fmt.Sprintf(`<!-- toudocu
id: %s
status: draft
taskType: %s
updated: %s
%s-->

# %s: %s

<!-- toudocu:section result -->
## Result

<!-- toudocu:section behavior-change -->
## Behavior change

<!-- toudocu:section before -->
### Before

<!-- toudocu:section after -->
### After

<!-- toudocu:section scope -->
## Scope

<!-- toudocu:section out-of-scope -->
## Out of scope

<!-- toudocu:section acceptance-criteria -->
## Acceptance criteria

<!-- toudocu:section plan -->
## Plan

<!-- toudocu:section verification -->
## Verification

<!-- toudocu:section documentation-impact -->
## Documentation impact
`, id, canonicalType, date, parent, id, title)
}

func InitTask(options Options) (TaskInitReport, error) {
	if err := requireCurrentDocumentationVersion(options); err != nil {
		return TaskInitReport{}, err
	}
	if info, err := os.Stat(options.InputDirectory); err != nil || !info.IsDir() {
		return TaskInitReport{}, fmt.Errorf("documentation directory not found: %s", options.InputDirectory)
	}
	if !taskAreaRE.MatchString(options.Area) {
		return TaskInitReport{}, fmt.Errorf("--area must contain A-Z, 0-9, and hyphens and start with a letter")
	}
	if err := validateScaffoldTitle(options.Title); err != nil {
		return TaskInitReport{}, err
	}
	if options.Language == "" {
		options.Language = "en"
	}
	if options.Language != "en" && options.Language != "ru" {
		return TaskInitReport{}, fmt.Errorf("--lang must be en or ru")
	}
	if !validTaskInitType(options.TaskType) {
		return TaskInitReport{}, fmt.Errorf("--type must be Feature, Bug, Maintenance, Documentation, or Research")
	}
	if options.ParentTaskID != "" {
		if options.TaskType == "Bug" {
			return TaskInitReport{}, fmt.Errorf("--parent is supported only for TASK-* work items")
		}
		if !taskIDRE.MatchString(options.ParentTaskID) || !strings.HasPrefix(options.ParentTaskID, "TASK-") {
			return TaskInitReport{}, fmt.Errorf("--parent must contain exactly one TASK-* identifier")
		}
		model, err := BuildDocumentationModel(options)
		if err != nil {
			return TaskInitReport{}, err
		}
		parent, err := findWorkItem(model, options.ParentTaskID)
		if err != nil {
			return TaskInitReport{}, fmt.Errorf("parent task %s not found", options.ParentTaskID)
		}
		if !strings.HasPrefix(parent.ID, "TASK-") {
			return TaskInitReport{}, fmt.Errorf("parent must be a TASK-* work item")
		}
	}
	taskType := options.TaskType
	prefix := "TASK"
	if taskType == "Bug" {
		prefix = "BUG"
	}
	for attempts := 0; attempts < 100; attempts++ {
		number, err := nextTaskNumber(options.InputDirectory, prefix, options.Area)
		if err != nil {
			return TaskInitReport{}, err
		}
		id := fmt.Sprintf("%s-%s-%03d", prefix, options.Area, number)
		if id == options.ParentTaskID {
			return TaskInitReport{}, fmt.Errorf("task cannot be its own parent")
		}
		relative := filepath.ToSlash(filepath.Join("work", id+".md"))
		target := filepath.Join(options.InputDirectory, filepath.FromSlash(relative))
		err = atomicCreateFile(target, renderTaskScaffold(id, options.Title, taskType, options.Language, scaffoldDate(options.Now), options.ParentTaskID))
		if err != nil && strings.Contains(err.Error(), "already exists") {
			continue
		}
		if err != nil {
			return TaskInitReport{}, err
		}
		return TaskInitReport{
			SchemaVersion: 1, Kind: "task-init", Generator: GeneratorInfo{Name: "Toudocu", Version: Version},
			ID: id, Title: options.Title, Type: taskType, Language: options.Language, Path: relative, ParentID: optionalString(options.ParentTaskID),
		}, nil
	}
	return TaskInitReport{}, fmt.Errorf("could not allocate a free work-item identifier after concurrent changes")
}

type scaffoldSpec struct {
	prefix    string
	directory string
}

type editorTemplateField struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

type editorTemplate struct {
	Key       string                `json:"key"`
	Label     string                `json:"label"`
	Fields    []editorTemplateField `json:"fields"`
	Languages []string              `json:"languages"`
	spec      scaffoldSpec
}

var scaffoldRegistry = []editorTemplate{
	{Key: "task-init", Label: "Рабочая задача", Languages: []string{"ru", "en"}, Fields: []editorTemplateField{
		{Name: "area", Label: "Область", Type: "text", Required: true},
		{Name: "title", Label: "Название", Type: "text", Required: true},
		{Name: "type", Label: "Тип", Type: "select", Required: true, Options: []string{"Feature", "Bug", "Maintenance", "Documentation", "Research"}},
	}},
	{Key: "module", Label: "Модуль", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "MOD-", directory: "modules"}},
	{Key: "use-case", Label: "Пользовательский сценарий", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "UC-", directory: "use-cases"}},
	{Key: "flow", Label: "Процесс", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "FLOW-", directory: "flows"}},
	{Key: "screen", Label: "Экран", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "SC-", directory: "screens"}},
	{Key: "decision", Label: "Решение", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "ADR-", directory: "decisions"}},
	{Key: "standard", Label: "Стандарт", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "STD-", directory: "quality"}},
	{Key: "runbook", Label: "Runbook", Languages: []string{"ru", "en"}, Fields: entityTemplateFields(), spec: scaffoldSpec{prefix: "RB-", directory: "runbooks"}},
	{Key: "draft", Label: "Черновик", Languages: []string{"ru", "en"}, Fields: []editorTemplateField{{Name: "title", Label: "Название", Type: "text", Required: true}}, spec: scaffoldSpec{directory: "drafts"}},
}

func entityTemplateFields() []editorTemplateField {
	return []editorTemplateField{
		{Name: "id", Label: "Идентификатор", Type: "text", Required: true},
		{Name: "title", Label: "Название", Type: "text", Required: true},
	}
}

func scaffoldTemplate(key string) (editorTemplate, bool) {
	for _, template := range scaffoldRegistry {
		if template.Key == key {
			return template, true
		}
	}
	return editorTemplate{}, false
}

func editorTemplates(model *Model) []editorTemplate {
	ui := portalUI(model)
	out := make([]editorTemplate, len(scaffoldRegistry))
	for index, item := range scaffoldRegistry {
		out[index] = item
		out[index].Label = ui.Text("template." + item.Key)
		out[index].Fields = append([]editorTemplateField{}, item.Fields...)
		for fieldIndex := range out[index].Fields {
			out[index].Fields[fieldIndex].Label = ui.Text("template.field." + out[index].Fields[fieldIndex].Name)
		}
	}
	return out
}

func validScaffoldID(kind, id string) bool {
	template, ok := scaffoldTemplate(kind)
	if !ok || template.spec.prefix == "" || !strings.HasPrefix(id, template.spec.prefix) || !safeStableID(id) {
		return false
	}
	return entityIDRE.MatchString(strings.TrimPrefix(id, template.spec.prefix))
}

func renderEntityScaffold(kind, id, title, language, date string) string {
	if language == "ru" {
		switch kind {
		case "module":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nupdated: %s\n-->\n\n# %s: %s\n\n## Назначение\n\n<!-- toudocu:section business-rules -->\n## Бизнес-правила\n\n<!-- toudocu:section invariants -->\n## Инварианты\n\n<!-- toudocu:section stable-interfaces -->\n## Интерфейсы\n", id, date, id, title)
		case "use-case":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nupdated: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section main-scenario -->\n## Основной сценарий\n\n## Альтернативные сценарии\n\n<!-- toudocu:section postconditions -->\n## Постусловия\n\n<!-- toudocu:section acceptance-criteria -->\n## Критерии приёмки\n\n<!-- toudocu:section business-rules -->\n## Бизнес-правила\n", id, date, id, title)
		case "flow":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nupdated: %s\n-->\n\n# %s: %s\n\n## Процесс\n", id, date, id, title)
		case "screen":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nscreenKind: screen\nstatus: planned\nupdated: %s\n-->\n\n# %s: %s\n\n## Назначение\n\n## Состояния\n\n## Переходы\n", id, date, id, title)
		case "standard":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nscope: TODO\nupdated: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section rules -->\n## Правила\n\n<!-- toudocu:section automated-checks -->\n## Автоматические проверки\n", id, date, id, title)
		case "runbook":
			return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nrisk: low\nlastVerified: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section prerequisites -->\n## Предварительные условия\n\n<!-- toudocu:section procedure -->\n## Процедура\n\n<!-- toudocu:section verification -->\n## Проверка\n\n<!-- toudocu:section rollback -->\n## Откат\n", id, date, id, title)
		default:
			return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: proposed\ndate: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section context -->\n## Контекст\n\n<!-- toudocu:section decision -->\n## Решение\n\n<!-- toudocu:section consequences -->\n## Последствия\n", id, date, id, title)
		}
	}
	switch kind {
	case "module":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nupdated: %s\n-->\n\n# %s: %s\n\n## Purpose\n\n<!-- toudocu:section business-rules -->\n## Business rules\n\n<!-- toudocu:section invariants -->\n## Invariants\n\n<!-- toudocu:section stable-interfaces -->\n## Interfaces\n", id, date, id, title)
	case "use-case":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nupdated: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section main-scenario -->\n## Main scenario\n\n## Alternative scenarios\n\n<!-- toudocu:section postconditions -->\n## Postconditions\n\n<!-- toudocu:section acceptance-criteria -->\n## Acceptance criteria\n\n<!-- toudocu:section business-rules -->\n## Business rules\n", id, date, id, title)
	case "flow":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nupdated: %s\n-->\n\n# %s: %s\n\n## Process\n", id, date, id, title)
	case "screen":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nscreenKind: screen\nstatus: planned\nupdated: %s\n-->\n\n# %s: %s\n\n## Purpose\n\n## States\n\n## Transitions\n", id, date, id, title)
	case "standard":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nscope: TODO\nupdated: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section rules -->\n## Rules\n\n<!-- toudocu:section automated-checks -->\n## Automated checks\n", id, date, id, title)
	case "runbook":
		return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: draft\nrisk: low\nlastVerified: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section prerequisites -->\n## Prerequisites\n\n<!-- toudocu:section procedure -->\n## Procedure\n\n<!-- toudocu:section verification -->\n## Verification\n\n<!-- toudocu:section rollback -->\n## Rollback\n", id, date, id, title)
	default:
		return fmt.Sprintf("<!-- toudocu\nid: %s\nstatus: proposed\ndate: %s\n-->\n\n# %s: %s\n\n<!-- toudocu:section context -->\n## Context\n\n<!-- toudocu:section decision -->\n## Decision\n\n<!-- toudocu:section consequences -->\n## Consequences\n", id, date, id, title)
	}
}

func Scaffold(options Options) (ScaffoldReport, error) {
	if err := requireCurrentDocumentationVersion(options); err != nil {
		return ScaffoldReport{}, err
	}
	if info, err := os.Stat(options.InputDirectory); err != nil || !info.IsDir() {
		return ScaffoldReport{}, fmt.Errorf("documentation directory not found: %s", options.InputDirectory)
	}
	template, ok := scaffoldTemplate(options.EntityKind)
	if !ok || template.spec.prefix == "" || !validScaffoldID(options.EntityKind, options.EntityID) {
		return ScaffoldReport{}, fmt.Errorf("invalid %s ID: %s", options.EntityKind, options.EntityID)
	}
	if err := validateScaffoldTitle(options.Title); err != nil {
		return ScaffoldReport{}, err
	}
	if options.Language == "" {
		options.Language = "en"
	}
	if options.Language != "en" && options.Language != "ru" {
		return ScaffoldReport{}, fmt.Errorf("--lang must be en or ru")
	}
	relative := filepath.ToSlash(filepath.Join(template.spec.directory, options.EntityID+".md"))
	target := filepath.Join(options.InputDirectory, filepath.FromSlash(relative))
	if err := atomicCreateFile(target, renderEntityScaffold(options.EntityKind, options.EntityID, options.Title, options.Language, scaffoldDate(options.Now))); err != nil {
		return ScaffoldReport{}, err
	}
	return ScaffoldReport{
		SchemaVersion: 1, Kind: "scaffold", Generator: GeneratorInfo{Name: "Toudocu", Version: Version},
		EntityType: options.EntityKind, ID: options.EntityID, Title: options.Title, Language: options.Language, Path: relative,
	}, nil
}

func createFromEditorTemplate(options Options, key, language string, fields map[string]string) (string, error) {
	template, ok := scaffoldTemplate(key)
	if !ok {
		return "", fmt.Errorf("unknown template: %s", key)
	}
	if language == "" {
		language = "ru"
	}
	if !containsString(template.Languages, language) {
		return "", fmt.Errorf("template language must be ru or en")
	}
	allowedFields := map[string]struct{}{}
	for _, field := range template.Fields {
		allowedFields[field.Name] = struct{}{}
	}
	for field := range fields {
		if _, ok := allowedFields[field]; !ok {
			return "", fmt.Errorf("unknown template field: %s", field)
		}
	}
	for _, field := range template.Fields {
		value := strings.TrimSpace(fields[field.Name])
		if field.Required && value == "" {
			return "", fmt.Errorf("field %s is required", field.Name)
		}
		if field.Type == "select" && !containsString(field.Options, value) {
			return "", fmt.Errorf("invalid value for field %s", field.Name)
		}
	}
	options.Language = language
	if key == "task-init" {
		options.Area = strings.TrimSpace(fields["area"])
		options.Title = strings.TrimSpace(fields["title"])
		options.TaskType = strings.TrimSpace(fields["type"])
		report, err := InitTask(options)
		if err != nil {
			return "", err
		}
		return report.Path, nil
	}
	if key == "draft" {
		title := strings.TrimSpace(fields["title"])
		if err := validateScaffoldTitle(title); err != nil {
			return "", err
		}
		for suffix := 1; suffix <= 100; suffix++ {
			name := slugify(title)
			if suffix > 1 {
				name += fmt.Sprintf("-%d", suffix)
			}
			relative := filepath.ToSlash(filepath.Join(template.spec.directory, name+".md"))
			err := atomicCreateFile(filepath.Join(options.InputDirectory, filepath.FromSlash(relative)), "# "+title+"\n")
			if err != nil && strings.Contains(err.Error(), "already exists") {
				continue
			}
			if err != nil {
				return "", err
			}
			return relative, nil
		}
		return "", fmt.Errorf("could not allocate a free draft filename after concurrent changes")
	}
	options.EntityKind = key
	options.EntityID = strings.TrimSpace(fields["id"])
	options.Title = strings.TrimSpace(fields["title"])
	report, err := Scaffold(options)
	if err != nil {
		return "", err
	}
	return report.Path, nil
}
