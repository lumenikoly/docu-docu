package toudocu

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func completeTaskFixture(status string) string {
	return `# TASK-AUTH-021: Add verification workflow

- Status: ` + status + `
- Type: Feature
- Module: MOD-AUTH
- Use case: UC-AUTH-01

## Result

The requested behavior is implemented.

## Behavior change

### Before

The workflow is unavailable.

### After

The workflow is available and verified.

## Scope

- ` + "`new.go`" + `

## Out of scope

Unrelated commands.

## Acceptance criteria

- [ ] ` + "`AC-01`" + ` The workflow succeeds.

## Plan

1. Implement the workflow.
2. Verify the result.

## Verification

- ` + "`AC-01`" + ` -> ` + "`go version`" + `
- ` + "`ALL`" + ` -> ` + "`go version`" + `
- ` + "`DOCS`" + ` -> ` + "`go version`" + `

## Documentation impact

Update ` + "`docs/index.md`" + `.
`
}

func TestNewCLIFormsAndRemovedTaskCheck(t *testing.T) {
	cases := [][]string{
		{"search", "task workflow", "./docs", "--limit", "5", "--format", "json"},
		{"task", "init", "./docs", "--area", "CLI", "--title", "Title", "--type", "Feature"},
		{"scaffold", "module", "MOD-CLI", "./docs", "--title", "CLI"},
		{"task", "ready", "TASK-CLI-001", "./docs", "--strict", "--format", "json"},
		{"task", "ready", "BUG-CLI-001", "./docs", "--format", "json"},
		{"task", "candidates", "./docs", "--parent", "TASK-CLI-001", "--strict", "--format", "json"},
		{"task", "context", "TASK-CLI-001", "./docs", "--format", "json"},
		{"task", "tree", "TASK-CLI-001", "./docs", "--format", "json"},
		{"task", "verify", "TASK-CLI-001", "./docs", "--dry-run", "--target", "AC-01"},
		{"task", "archive", "TASK-CLI-001", "./docs", "--format", "json"},
		{"task", "restore", "TASK-CLI-001", "./docs", "--format", "json"},
		{"task", "changes", "TASK-CLI-001", "./docs", "--translation-input", "--format", "json"},
		{"task", "changes", "TASK-CLI-001", "./docs", "--tree", "--format", "json"},
		{"task", "init", "./docs", "--area", "CLI", "--title", "Child", "--type", "Feature", "--parent", "TASK-CLI-001"},
	}
	for _, args := range cases {
		if _, _, _, err := ParseArguments(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if _, _, _, err := ParseArguments([]string{"task", "check", "TASK-CLI-001"}); err == nil {
		t.Fatal("task check must be removed")
	}
	if _, _, _, err := ParseArguments([]string{"task", "verify", "TASK-CLI-001", "--dry-run", "--run"}); err == nil {
		t.Fatal("verify modes must be mutually exclusive")
	}
	if _, _, _, err := ParseArguments([]string{"changes", "./docs", "--task", "TASK-CLI-001"}); err == nil {
		t.Fatal("changes --task must be removed in favor of task changes")
	}
	if options, _, _, err := ParseArguments([]string{"changes", "./docs", "--include-assets"}); err != nil || !options.ChangeForceIncludeAssets {
		t.Fatalf("changes --include-assets was not parsed: options=%#v err=%v", options, err)
	}
	if options, _, _, err := ParseArguments([]string{"changes", "./docs", "--translation-input"}); err != nil || !options.ChangeTranslationInput {
		t.Fatalf("changes --translation-input was not parsed: options=%#v err=%v", options, err)
	}
	if _, _, _, err := ParseArguments([]string{"changes", "./docs", "--translation-input", "--permanent-only"}); err == nil {
		t.Fatal("translation input and permanent-only must be rejected together")
	}
	if _, _, _, err := ParseArguments([]string{"task", "changes", "BUG-CLI-001", "./docs", "--tree"}); err == nil {
		t.Fatal("task changes --tree must be limited to TASK-* work items")
	}
	if _, _, _, err := ParseArguments([]string{"check", "./docs", "--include-assets"}); err == nil {
		t.Fatal("--include-assets must be rejected outside changes commands")
	}
}

func TestScaffoldLanguageDefaultsToProjectLocale(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(root, ".toudocu"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(docs, 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, ".toudocu/config.yml", localeConfigForTest("ru-RU"))
	options, _, _, err := ParseArguments([]string{"task", "init", docs, "--area", "CLI", "--title", "Задача", "--type", "Feature"})
	if err != nil || options.Language != "ru" {
		t.Fatalf("configured language = %q, err=%v", options.Language, err)
	}
	override, _, _, err := ParseArguments([]string{"scaffold", "module", "MOD-CLI", docs, "--title", "CLI", "--lang", "en"})
	if err != nil || override.Language != "en" {
		t.Fatalf("explicit language = %q, err=%v", override.Language, err)
	}
	writeTestFile(t, root, ".toudocu/config.yml", localeConfigForTest("de"))
	fallback, _, _, err := ParseArguments([]string{"scaffold", "module", "MOD-CORE", docs, "--title", "Core"})
	if err != nil || fallback.Language != "en" {
		t.Fatalf("fallback language = %q, err=%v", fallback.Language, err)
	}
}

func TestSearchDocumentationRankingAndSections(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Обзор\n\nПоиск документации.\n")
	writeTestFile(t, docs, "guides/plain.md", "# Обычный документ\n\n## Поиск\n\nЁж и Search CLI workflow.\n")
	writeTestFile(t, docs, "modules/search.md", "# Search workflow\n\n- Identifier: MOD-SEARCH\n- Status: Planned\n\nSearch module.\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	report, err := SearchDocumentation(model, "search workflow", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) < 2 || report.Results[0].ID != "MOD-SEARCH" {
		t.Fatalf("ranking: %#v", report.Results)
	}
	ru, err := SearchDocumentation(model, "еж cli", 20)
	if err != nil || len(ru.Results) != 1 || ru.Results[0].ID != "" || strings.Join(ru.Results[0].MatchedSections, ",") != "Поиск" {
		t.Fatalf("normalized search: %#v %v words=%#v text=%q words=%#v", ru, err, searchWords("Ёж и Search CLI workflow."), model.DocByPath["guides/plain.md"].PlainText, searchWords(model.DocByPath["guides/plain.md"].PlainText))
	}
}

func TestSearchIndexExcludesSemanticAnnotations(t *testing.T) {
	document := &Document{
		Title:      "Release",
		SourcePath: "status.md",
		OutputPath: "status.html",
		Type:       "status",
		Metadata: Metadata{
			"status":  "Ready",
			"version": "0.0.1",
		},
		MetadataExtras: []MetadataExtra{{Key: "Channel", Value: "stable"}},
	}
	model := &Model{Documents: []*Document{document}}

	first := buildSearchIndex(model)
	second := buildSearchIndex(model)
	if len(first) != 1 || len(second) != 1 || first[0].Text != second[0].Text {
		t.Fatalf("search index changed between builds: %#v %#v", first, second)
	}
	want := "release status md"
	if first[0].Text != want {
		t.Fatalf("search index metadata order = %q, want %q", first[0].Text, want)
	}
	model = &Model{Documents: []*Document{document}}
	if report, err := SearchDocumentation(model, "stable", 20); err != nil || report.Total != 0 {
		t.Fatalf("semantic annotation leaked into CLI search: report=%#v err=%v", report, err)
	}
}

func TestTaskInitAndScaffoldAtomicCreate(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "work"), 0755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, docs, "work/TASK-CLI-999-old.md", "# Existing\n")
	writeTestFile(t, docs, "work/archive/2025/legacy-name.md", "# TASK-CLI-1200: Archived\n")
	report, err := InitTask(Options{InputDirectory: docs, Area: "CLI", Title: "Next", TaskType: "Bug", Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if report.ID != "BUG-CLI-001" || report.Path != "work/BUG-CLI-001.md" {
		t.Fatalf("allocation: %#v", report)
	}
	data, _ := os.ReadFile(filepath.Join(docs, filepath.FromSlash(report.Path)))
	if !strings.Contains(string(data), "status: draft") || !strings.Contains(string(data), "<!-- toudocu:section symptom -->\n## Symptom") {
		t.Fatalf("english scaffold: %s", data)
	}
	entity, err := Scaffold(Options{InputDirectory: docs, EntityKind: "decision", EntityID: "ADR-002", Title: "Boundary", Language: "ru"})
	if err != nil || entity.Path != "decisions/ADR-002.md" {
		t.Fatalf("scaffold: %#v %v", entity, err)
	}
	if _, err := Scaffold(Options{InputDirectory: docs, EntityKind: "decision", EntityID: "ADR-002", Title: "Overwrite", Language: "ru"}); err == nil {
		t.Fatal("scaffold must not overwrite")
	}
	for _, title := range []string{"Injected\n- Module: MOD-OTHER", "Injected\r\n## Rules"} {
		if _, err := InitTask(Options{InputDirectory: docs, Area: "CLI", Title: title, TaskType: "Feature", Language: "en"}); err == nil {
			t.Fatalf("task init accepted multiline title %q", title)
		}
		if _, err := Scaffold(Options{InputDirectory: docs, EntityKind: "module", EntityID: "MOD-INJECTED", Title: title, Language: "en"}); err == nil {
			t.Fatalf("scaffold accepted multiline title %q", title)
		}
	}
	if _, _, _, err := ParseArguments([]string{"task", "init", docs, "--area", "CLI", "--title", "Injected\nmetadata", "--type", "Feature"}); err == nil {
		t.Fatal("CLI accepted multiline task title")
	}
}

func terminalTaskFixture(status string) string {
	content := completeTaskFixture(status)
	if status == "Done" {
		content = strings.Replace(content, "- [ ] `AC-01`", "- [x] `AC-01`", 1)
	}
	if status == "Cancelled" {
		content += "\n## Cancellation reason\n\nThe request is no longer needed.\n"
	}
	return content
}

func TestTaskArchiveAndRestoreRoundTrip(t *testing.T) {
	root, docs, _ := createFixture(t)
	original := terminalTaskFixture("Done")
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", original)
	original = canonicalizeTestMarkdown("work/TASK-AUTH-021.md", original)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	archived, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021", Now: time.Date(2031, 4, 5, 0, 0, 0, 0, time.Local)}, "archive")
	if err != nil || archived.Status != "archived" || archived.DestinationPath != "work/archive/2031/TASK-AUTH-021.md" {
		t.Fatalf("archive: %#v %v", archived, err)
	}
	archivedPath := filepath.Join(docs, filepath.FromSlash(archived.DestinationPath))
	data, err := os.ReadFile(archivedPath)
	if err != nil || string(data) != original {
		t.Fatalf("archived content changed: %v\n%s", err, data)
	}
	if _, err := os.Stat(filepath.Join(docs, "work", "TASK-AUTH-021.md")); !os.IsNotExist(err) {
		t.Fatalf("source still exists after archive: %v", err)
	}

	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	item, err := findWorkItem(model, "TASK-AUTH-021")
	if err != nil || !item.Archived || item.ArchiveYear != "2031" {
		t.Fatalf("archive metadata: %#v %v", item, err)
	}
	search, err := SearchDocumentation(model, "TASK AUTH 021", 20)
	if err != nil || len(search.Results) != 1 || !search.Results[0].Archived || search.Results[0].ArchiveYear != "2031" {
		t.Fatalf("archive search: %#v %v", search, err)
	}
	restored, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "restore")
	if err != nil || restored.Status != "restored" || restored.DestinationPath != "work/TASK-AUTH-021.md" {
		t.Fatalf("restore: %#v %v", restored, err)
	}
	data, err = os.ReadFile(filepath.Join(docs, "work", "TASK-AUTH-021.md"))
	if err != nil || string(data) != original {
		t.Fatalf("restored content changed: %v\n%s", err, data)
	}
}

func TestTaskArchiveEligibilityAndArchiveValidation(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", completeTaskFixture("Ready"))
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	report, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "archive")
	if err != nil || report.Status != "blocked" || !hasIssueCode(report.Issues, "task-not-terminal") {
		t.Fatalf("nonterminal archive: %#v %v", report, err)
	}

	if err := os.Rename(
		filepath.Join(docs, "work", "TASK-AUTH-021.md"),
		func() string {
			target := filepath.Join(docs, "work", "archive", "2031", "TASK-AUTH-021.md")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatal(err)
			}
			return target
		}(),
	); err != nil {
		t.Fatal(err)
	}
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssueCode(model.Issues, "nonterminal-archived-task") {
		t.Fatalf("manual nonterminal archive was accepted: %#v", model.Issues)
	}
	restored, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "restore")
	if err != nil || restored.Status != "restored" {
		t.Fatalf("restore must repair a manually archived active task: %#v %v", restored, err)
	}
}

func TestTaskArchiveRejectsInvalidLayoutAndUnsafeDestination(t *testing.T) {
	t.Run("invalid layout", func(t *testing.T) {
		root, docs, _ := createFixture(t)
		writeTestFile(t, docs, "work/archive/not-a-year/TASK-AUTH-021.md", terminalTaskFixture("Done"))
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		if !hasIssueCode(model.Issues, "invalid-task-archive-path") {
			t.Fatalf("invalid archive layout was accepted: %#v", model.Issues)
		}
	})

	t.Run("collision", func(t *testing.T) {
		root, docs, _ := createFixture(t)
		writeTestFile(t, docs, "work/TASK-AUTH-021.md", terminalTaskFixture("Done"))
		other := strings.Replace(terminalTaskFixture("Done"), "TASK-AUTH-021", "TASK-AUTH-099", 1)
		writeTestFile(t, docs, "work/archive/2031/TASK-AUTH-021.md", other)
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		report, err := MoveTask(model, Options{
			TaskID: "TASK-AUTH-021",
			Now:    time.Date(2031, 1, 1, 0, 0, 0, 0, time.Local),
		}, "archive")
		if err != nil || !hasIssueCode(report.Issues, "unsafe-task-move") {
			t.Fatalf("archive collision was accepted: %#v %v", report, err)
		}
	})

	t.Run("symlink archive directory", func(t *testing.T) {
		root, docs, _ := createFixture(t)
		writeTestFile(t, docs, "work/TASK-AUTH-021.md", terminalTaskFixture("Done"))
		outside := t.TempDir()
		if err := os.MkdirAll(filepath.Join(docs, "work"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(docs, "work", "archive")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		report, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "archive")
		if err != nil || !hasIssueCode(report.Issues, "unsafe-task-move") {
			t.Fatalf("symlink destination was accepted: %#v %v", report, err)
		}
	})
}

func TestTaskArchiveBlocksIncomingAndChangingOutgoingLinks(t *testing.T) {
	t.Run("incoming", func(t *testing.T) {
		root, docs, _ := createFixture(t)
		writeTestFile(t, docs, "work/TASK-AUTH-021.md", terminalTaskFixture("Done"))
		writeTestFile(t, docs, "guides/task.md", "# Task link\n\n[Task](../work/TASK-AUTH-021.md)\n")
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		report, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "archive")
		if err != nil || !hasIssueCode(report.Issues, "task-move-incoming-link") {
			t.Fatalf("incoming link did not block: %#v %v", report, err)
		}
	})

	t.Run("outgoing", func(t *testing.T) {
		root, docs, _ := createFixture(t)
		content := strings.Replace(terminalTaskFixture("Done"), "Update `docs/index.md`.", "Update [module](../modules/auth.md).", 1)
		writeTestFile(t, docs, "work/TASK-AUTH-021.md", content)
		model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
		if err != nil {
			t.Fatal(err)
		}
		report, err := MoveTask(model, Options{TaskID: "TASK-AUTH-021"}, "archive")
		if err != nil || !hasIssueCode(report.Issues, "task-move-outgoing-link") {
			t.Fatalf("changing outgoing link did not block: %#v %v", report, err)
		}
	})
}

func TestTaskArchiveCLIJSON(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", terminalTaskFixture("Cancelled"))
	var stdout, stderr bytes.Buffer
	code := RunCLI([]string{
		"task", "archive", "TASK-AUTH-021", docs,
		"--repository-root", root, "--format", "json", "--stale-days", "0",
	}, &stdout, &stderr)
	var report TaskMoveReport
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Status != "archived" || report.Kind != "task-archive" {
		t.Fatalf("archive JSON: %#v %v\n%s", report, err, stdout.String())
	}
}

func TestTaskReadyContextAndVerifyDryRun(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", completeTaskFixture("Draft"))
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	ready := BuildTaskReady(model, "TASK-AUTH-021", false)
	if ready.Status != "contract_ready" || !ready.ContractComplete || ready.ReadyForWork {
		t.Fatalf("draft readiness: %#v", ready)
	}
	if _, err := BuildTaskContext(model, "TASK-AUTH-021"); err == nil {
		t.Fatal("draft context must be rejected")
	}

	writeTestFile(t, docs, "work/TASK-AUTH-021.md", strings.Replace(completeTaskFixture("Draft"), "- Module: MOD-AUTH\n", "", 1))
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	missingModule := BuildTaskReady(model, "TASK-AUTH-021", false)
	if missingModule.Status != "contract_incomplete" || !hasIssueCode(missingModule.Issues, "missing-task-module") {
		t.Fatalf("draft without module passed readiness: %#v", missingModule)
	}

	writeTestFile(t, docs, "work/TASK-AUTH-021.md", strings.Replace(completeTaskFixture("Draft"), "`new.go`", "`new-directory/`", 1))
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	missingDirectory := BuildTaskReady(model, "TASK-AUTH-021", false)
	if missingDirectory.Status != "contract_incomplete" || !hasIssueCode(missingDirectory.Issues, "missing-scope-path") {
		t.Fatalf("missing scope directory passed readiness: %#v", missingDirectory)
	}

	writeTestFile(t, docs, "work/TASK-AUTH-021.md", strings.Replace(terminalTaskFixture("Done"), "`new.go`", "`removed/legacy/asset.js`", 1))
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	completed := BuildTaskReady(model, "TASK-AUTH-021", false)
	if hasIssueCode(completed.Issues, "missing-scope-path") {
		t.Fatalf("completed task history must preserve removed scope paths: %#v", completed)
	}

	writeTestFile(t, docs, "work/TASK-AUTH-021.md", completeTaskFixture("Ready"))
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	context, err := BuildTaskContext(model, "TASK-AUTH-021")
	if err != nil || context.SchemaVersion != 1 || len(context.RequiredReads) == 0 || context.Task.Before == "" {
		t.Fatalf("context: %#v %v", context, err)
	}
	var stdout, stderr bytes.Buffer
	code := RunCLI([]string{"task", "verify", "TASK-AUTH-021", docs, "--repository-root", root, "--dry-run", "--format", "json", "--stale-days", "0"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	var report TaskVerifyReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Status != "planned" || len(report.Commands) != 1 || report.Commands[0].Status != "planned" {
		t.Fatalf("dry-run: %#v %v\n%s", report, err, stdout.String())
	}
}

func TestTaskCandidatesReportReadySubtasksAndBlockers(t *testing.T) {
	root, docs, _ := createFixture(t)
	withMetadata := func(status, id, metadata string) string {
		content := strings.Replace(completeTaskFixture(status), "TASK-AUTH-021", id, 1)
		return strings.Replace(content, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n"+metadata, 1)
	}
	writeTestFile(t, docs, "work/TASK-AUTH-100.md", withMetadata("Ready", "TASK-AUTH-100", ""))
	writeTestFile(t, docs, "work/TASK-AUTH-101.md", withMetadata("Ready", "TASK-AUTH-101", "- Parent: TASK-AUTH-100\n- Priority: High\n"))
	writeTestFile(t, docs, "work/TASK-AUTH-102.md", withMetadata("Ready", "TASK-AUTH-102", "- Parent: TASK-AUTH-100\n- Dependencies: TASK-AUTH-101\n"))
	writeTestFile(t, docs, "work/TASK-AUTH-103.md", strings.Replace(withMetadata("Ready", "TASK-AUTH-103", "- Parent: TASK-AUTH-100\n"), "- Module: MOD-AUTH\n", "", 1))
	writeTestFile(t, docs, "work/TASK-AUTH-104.md", withMetadata("Done", "TASK-AUTH-104", ""))

	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	report, err := BuildTaskCandidates(model, "TASK-AUTH-100", false)
	if err != nil || report.Kind != "task-candidates" || len(report.Candidates) != 3 {
		t.Fatalf("candidates: %#v %v", report, err)
	}
	byID := map[string]TaskCandidate{}
	for _, candidate := range report.Candidates {
		byID[candidate.ID] = candidate
	}
	if !byID["TASK-AUTH-101"].ReadyForWork || byID["TASK-AUTH-101"].Priority != "high" {
		t.Fatalf("ready child: %#v", byID["TASK-AUTH-101"])
	}
	waiting := byID["TASK-AUTH-102"]
	if waiting.DependenciesSatisfied || waiting.ReadyForWork || len(waiting.BlockedBy) != 1 || waiting.BlockedBy[0].ID != "TASK-AUTH-101" {
		t.Fatalf("waiting child: %#v", waiting)
	}
	if byID["TASK-AUTH-103"].ContractComplete || byID["TASK-AUTH-103"].ReadyForWork {
		t.Fatalf("incomplete child: %#v", byID["TASK-AUTH-103"])
	}
	var text strings.Builder
	printTaskCandidatesText(&text, TaskCandidatesReport{Candidates: []TaskCandidate{{
		ID: "TASK-AUTH-105", Status: "draft", ContractComplete: false, DependenciesSatisfied: false,
		BlockedBy: []TaskCandidateBlocker{{ID: "TASK-AUTH-101", Status: "ready"}},
	}}})
	for _, expected := range []string{"BLOCKED", "status=draft", "contract incomplete", "change status to Ready", "depends on TASK-AUTH-101"} {
		if !strings.Contains(text.String(), expected) {
			t.Fatalf("candidate text omitted %q: %s", expected, text.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"task", "candidates", docs, "--repository-root", root, "--parent", "TASK-AUTH-100", "--format", "json", "--stale-days", "0"}, &stdout, &stderr); code != 0 {
		t.Fatalf("candidate CLI: code=%d stderr=%s", code, stderr.String())
	}
	var decoded TaskCandidatesReport
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil || len(decoded.Candidates) != 3 {
		t.Fatalf("candidate CLI JSON: %#v %v", decoded, err)
	}

	writeTestFile(t, docs, "work/TASK-AUTH-100-duplicate.md", withMetadata("Ready", "TASK-AUTH-100", ""))
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildTaskCandidates(model, "TASK-AUTH-100", false); err == nil {
		t.Fatal("duplicate --parent ID was accepted")
	}
}

func TestTaskReadyAcceptsSafeDocumentationImpactDirectory(t *testing.T) {
	root, docs, _ := createFixture(t)
	content := strings.Replace(completeTaskFixture("Ready"), "`docs/index.md`", "`docs/`", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", content)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	ready := BuildTaskReady(model, "TASK-AUTH-021", false)
	if !ready.ContractComplete || !ready.ReadyForWork {
		t.Fatalf("safe documentation-impact directory failed readiness: %#v", ready)
	}
}

func TestTaskReadyRejectsDocumentationImpactDirectoryOutsideRoot(t *testing.T) {
	root, docs, _ := createFixture(t)
	content := strings.Replace(completeTaskFixture("Ready"), "`docs/index.md`", "`../../../`", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", content)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	ready := BuildTaskReady(model, "TASK-AUTH-021", false)
	if !hasIssueCode(ready.Issues, "unsafe-documentation-impact-path") {
		t.Fatalf("outside documentation-impact directory was accepted: %#v", ready)
	}
}

func TestTaskVerifyTargetAndUnknownTarget(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", completeTaskFixture("Ready"))
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeCommandRunner{outcomes: map[string]fakeCommandOutcome{"go version": {}}}
	report := executeTaskVerify(model, Options{
		TaskID: "TASK-AUTH-021", VerifyMode: "run", Target: "AC-01", Format: "json",
	}, io.Discard, io.Discard, runner)
	if report.Status != "passed" || report.FullVerification || len(report.Criteria) != 1 || len(report.Targets) != 1 || strings.Join(runner.commands, ",") != "go version" {
		t.Fatalf("targeted verify: %#v commands=%#v", report, runner.commands)
	}
	blocked := executeTaskVerify(model, Options{
		TaskID: "TASK-AUTH-021", VerifyMode: "dry-run", Target: "AC-99", Format: "json",
	}, io.Discard, io.Discard, runner)
	if blocked.Status != "blocked" || len(blocked.Commands) != 0 {
		t.Fatalf("unknown target: %#v", blocked)
	}
}
