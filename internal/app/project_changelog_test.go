package toudocu

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectChangelogBuildsPortalPageAndSearchEntry(t *testing.T) {
	root, docs, output := createFixture(t)
	writeTestFile(t, root, projectChangelogFile, "# Changelog\n\n## Unreleased\n\n- Added OpenAPI compatibility reports.\n")
	writeTestFile(t, docs, "changelog.md", "# Local notes\n\nThis is an ordinary document.\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if model.ProjectChangelog == nil || model.ProjectChangelog.OutputPath != projectChangelogOutput {
		t.Fatalf("project changelog=%#v", model.ProjectChangelog)
	}
	if local := model.DocByPath["changelog.md"]; local == nil || local.Type != "document" || local.OutputPath != "changelog.html" {
		t.Fatalf("local changelog must remain an ordinary document: %#v", local)
	}
	for _, document := range BuildReport(model).Documents {
		if document.SourcePath == projectChangelogFile {
			t.Fatal("project changelog must not enter ProjectReport")
		}
	}
	if report, err := SearchDocumentation(model, "OpenAPI", 10); err != nil || report.Total != 0 {
		t.Fatalf("CLI search must not include project changelog: %#v, %v", report, err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(output, projectChangelogOutput))
	if err != nil || !strings.Contains(string(page), "Added OpenAPI compatibility reports.") || !strings.Contains(string(page), "Project changelog") {
		t.Fatalf("project changelog page: %v\n%s", err, page)
	}
	localPage, err := os.ReadFile(filepath.Join(output, "changelog.html"))
	if err != nil || !strings.Contains(string(localPage), "This is an ordinary document.") {
		t.Fatalf("local changelog page: %v\n%s", err, localPage)
	}
	search, err := os.ReadFile(filepath.Join(output, "data", "search-index.json"))
	if err != nil || !strings.Contains(string(search), "OpenAPI compatibility reports") || !strings.Contains(string(search), projectChangelogOutput) || !strings.Contains(string(search), `"path": "CHANGELOG.md"`) {
		t.Fatalf("portal search index: %v\n%s", err, search)
	}
	if _, err := generateServeSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	servePage, err := os.ReadFile(filepath.Join(output, projectChangelogOutput))
	if err != nil || strings.Contains(string(servePage), `path=CHANGELOG.md`) || strings.Contains(string(servePage), `data-copy-document-context`) || strings.Contains(string(servePage), "Открыть исходник") {
		t.Fatalf("project changelog must not expose editor controls: %v\n%s", err, servePage)
	}
}

func TestProjectChangelogFallbackTitleUsesUILocale(t *testing.T) {
	for _, test := range []struct {
		locale string
		title  string
	}{
		{locale: "en", title: "Project changelog"},
		{locale: "ru-RU", title: "Журнал изменений проекта"},
	} {
		t.Run(test.locale, func(t *testing.T) {
			root, docs, _ := createFixture(t)
			writeSiteConfig(t, root, localeConfigForTest(test.locale))
			writeTestFile(t, root, projectChangelogFile, "- Entry.\n")
			model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
			if err != nil {
				t.Fatal(err)
			}
			if model.ProjectChangelog == nil || model.ProjectChangelog.Title != test.title {
				t.Fatalf("fallback title = %q, want %q", model.ProjectChangelog.Title, test.title)
			}
		})
	}
}

func TestProjectChangelogIsOptionalAndUnsafeFilesWarn(t *testing.T) {
	root, docs, output := createFixture(t)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if model.ProjectChangelog != nil || strings.Contains(renderNavigation(model, "index.html"), "Журнал изменений проекта") {
		t.Fatal("missing project changelog must not create a portal tab")
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, projectChangelogOutput)); !os.IsNotExist(err) {
		t.Fatalf("unexpected project changelog page: %v", err)
	}
	writeTestFile(t, root, projectChangelogFile, "# Changelog\n\n- Present.\n")
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, projectChangelogFile)); err != nil {
		t.Fatal(err)
	}
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(output, projectChangelogOutput)); !os.IsNotExist(err) {
		t.Fatalf("stale project changelog page: %v", err)
	}

	writeTestFile(t, root, "outside.md", "# Outside\n")
	if err := os.Symlink(filepath.Join(root, "outside.md"), filepath.Join(root, projectChangelogFile)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if model.ProjectChangelog != nil || !hasIssueCode(model.Issues, "project-changelog-unavailable") {
		t.Fatalf("unsafe project changelog=%#v issues=%#v", model.ProjectChangelog, model.Issues)
	}
}

func TestProjectChangelogIsExcludedWhenInputIsRepositoryRoot(t *testing.T) {
	root, _, output := createFixture(t)
	writeTestFile(t, root, projectChangelogFile, "# Changelog\n\n- Root source.\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: root, RepositoryRoot: root, StaleDays: 0})
	if err == nil || !strings.Contains(err.Error(), "LOCALE_ROOT_NOT_CONFIGURED") {
		t.Fatalf("repository root must not combine locales: %#v %v", model, err)
	}
	if err != nil {
		return
	}
	options := Options{InputDirectory: root, OutputDirectory: output, RepositoryRoot: root, StaleDays: 0}
	workspace, err := newEditorWorkspace(options)
	if err != nil {
		t.Fatal(err)
	}
	files, _, err := workspace.scan(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Path == projectChangelogFile {
			t.Fatal("project changelog must not enter editor workspace")
		}
	}
	server, _, _, err := newDocumentationServer(options, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	get := performEditorRequest(server, editorRequest(http.MethodGet, editorAPIBase+"/file?path="+projectChangelogFile, "", nil))
	if get.Code != http.StatusForbidden || !strings.Contains(get.Body.String(), `"code":"path_forbidden"`) {
		t.Fatalf("project changelog GET: status=%d body=%s", get.Code, get.Body.String())
	}
	put := performEditorRequest(server, editorRequest(http.MethodPut, editorAPIBase+"/file", "save", map[string]any{"path": projectChangelogFile, "content": "# Changed\n", "expectedDigest": "ignored", "confirmOverwrite": false}))
	if put.Code != http.StatusForbidden || !strings.Contains(put.Body.String(), `"code":"path_forbidden"`) {
		t.Fatalf("project changelog PUT: status=%d body=%s", put.Code, put.Body.String())
	}
}

func TestServeRevisionIncludesProjectChangelog(t *testing.T) {
	root, docs, output := createFixture(t)
	workspace, err := newEditorWorkspace(Options{InputDirectory: docs, OutputDirectory: output, RepositoryRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	server := &documentationServer{workspace: workspace}
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	missing, err := server.workspaceRevision(model)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, projectChangelogFile, "# Changelog\n\n- First release note.\n")
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	created, err := server.workspaceRevision(model)
	if err != nil || created == missing {
		t.Fatalf("creation must change revision: %q %q %v", missing, created, err)
	}
	writeTestFile(t, root, projectChangelogFile, "# Changelog\n\n- Updated release note.\n")
	updated, err := server.workspaceRevision(model)
	if err != nil || updated == created {
		t.Fatalf("update must change revision: %q %q %v", created, updated, err)
	}
	if err := os.Remove(filepath.Join(root, projectChangelogFile)); err != nil {
		t.Fatal(err)
	}
	removed, err := server.workspaceRevision(model)
	if err != nil || removed == updated {
		t.Fatalf("removal must change revision: %q %q %v", updated, removed, err)
	}
}
