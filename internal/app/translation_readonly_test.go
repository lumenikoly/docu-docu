package toudocu

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const translationReadOnlyConfig = `project:
  locale: ru
  sections:
    architecture: Архитектура
    modules: Модули
    use-cases: Пользовательские сценарии
    flows: Процессы
    screens: Экраны
    decisions: Архитектурные решения
    contracts: Контракты
    quality: Стандарты качества
    runbooks: Runbooks
    reference: Справочник
    work: Рабочие задачи
    guides: Руководства
translations:
  en:
    root: docs-en
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use Cases
      flows: Processes
      screens: Screens
      decisions: Architecture Decisions
      contracts: Contracts
      quality: Quality Standards
      runbooks: Runbooks
      reference: Reference
      work: Work Items
      guides: Guides
`

func translationReadOnlyFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	target := filepath.Join(root, "docs-en")
	writeTestFile(t, docs, "index.md", "# Canonical\n")
	writeTestFile(t, target, "index.md", "# English mirror\n")
	writeTestFile(t, target, "work/TASK-DOCS-001.md", "# TASK-DOCS-001: Mirror\n\n- Status: Draft\n- Type: Documentation\n\n## Result\n\nMirror.\n")
	writeTestFile(t, target, "work/TASK-DOCS-002.md", "# TASK-DOCS-002: Child mirror\n\n- Status: Draft\n- Type: Documentation\n- Parent: TASK-DOCS-001\n\n## Result\n\nChild mirror.\n")
	writeSiteConfig(t, root, translationReadOnlyConfig)
	return root, docs, target
}

func TestTranslationRootRejectsTaskAndScaffoldOperations(t *testing.T) {
	root, _, target := translationReadOnlyFixture(t)
	options := Options{InputDirectory: target, RepositoryRoot: root, StaleDays: 0}
	model, err := BuildDocumentationModel(options)
	if err != nil {
		t.Fatal(err)
	}
	if model.translationLocale != "en" {
		t.Fatalf("translation locale = %q", model.translationLocale)
	}
	assertReadOnly := func(err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), translationRootReadOnlyCode) {
			t.Fatalf("read-only error = %v", err)
		}
	}
	_, err = BuildTaskContext(model, "TASK-DOCS-001")
	assertReadOnly(err)
	_, err = BuildTaskTree(model, "TASK-DOCS-001")
	assertReadOnly(err)
	portal := renderDocumentPage(model, model.DocByPath["work/TASK-DOCS-001.md"])
	if !strings.Contains(portal, "task-tree") || !strings.Contains(portal, "TASK-DOCS-002") {
		t.Fatalf("translation portal omitted the read-only task tree: %s", portal)
	}
	ready := BuildTaskReady(model, "TASK-DOCS-001", false)
	if ready.Status != "blocked" || !hasIssueCode(ready.Issues, "translation-root-read-only") {
		t.Fatalf("ready report = %#v", ready)
	}
	_, err = MoveTask(model, Options{TaskID: "TASK-DOCS-001"}, "archive")
	assertReadOnly(err)
	runner := &fakeCommandRunner{}
	verify := executeTaskVerify(model, Options{TaskID: "TASK-DOCS-001", VerifyMode: "run"}, io.Discard, io.Discard, runner)
	if verify.Status != "blocked" || !hasIssueCode(verify.ValidationIssues, "translation-root-read-only") || len(runner.commands) != 0 {
		t.Fatalf("verify report = %#v, commands = %#v", verify, runner.commands)
	}
	_, err = InitTask(Options{InputDirectory: target, RepositoryRoot: root, Area: "DOCS", Title: "Blocked", TaskType: "Documentation", Language: "en"})
	assertReadOnly(err)
	_, err = Scaffold(Options{InputDirectory: target, RepositoryRoot: root, EntityKind: "module", EntityID: "MOD-BLOCKED", Title: "Blocked", Language: "en"})
	assertReadOnly(err)
	_, err = BuildDocumentationChanges(Options{InputDirectory: target, RepositoryRoot: root, ChangeTaskID: "TASK-DOCS-001"})
	assertReadOnly(err)
	for _, relative := range []string{"work/TASK-DOCS-003.md", "modules/MOD-BLOCKED.md"} {
		if _, statErr := os.Stat(filepath.Join(target, filepath.FromSlash(relative))); !os.IsNotExist(statErr) {
			t.Fatalf("blocked operation created %s", relative)
		}
	}

	var stdout, stderr bytes.Buffer
	code := RunCLI([]string{"task", "context", "TASK-DOCS-001", target, "--repository-root", root, "--format", "json"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), translationRootReadOnlyCode) {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestTranslationRootAllowsReadOnlyPortalWithoutEditor(t *testing.T) {
	root, _, target := translationReadOnlyFixture(t)
	options := Options{
		Command: "serve", InputDirectory: target, OutputDirectory: filepath.Join(root, "site-en"), RepositoryRoot: root,
		RepositoryRef: "main", StaleDays: 0, Host: "127.0.0.1", Port: 8080,
	}
	handler, model, result, err := newDocumentationServer(options, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !handler.translationReadOnly || model.translationLocale != "en" || result.Pages == 0 {
		t.Fatalf("server state: readOnly=%t locale=%q result=%#v", handler.translationReadOnly, model.translationLocale, result)
	}
	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	if home.Code != http.StatusOK || strings.Contains(home.Body.String(), "/_toudocu/editor/") {
		t.Fatalf("home: %d %s", home.Code, home.Body.String())
	}
	if strings.Contains(home.Body.String(), `"review":true`) || strings.Contains(home.Body.String(), reviewAPIBase) {
		t.Fatalf("translation home exposed review capability: %s", home.Body.String())
	}
	for _, route := range []string{editorUIPath, editorAPIBase + "/files", reviewAPIBase + "/repository/changes", rebuildEndpoint} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d", route, response.Code)
		}
	}
}
