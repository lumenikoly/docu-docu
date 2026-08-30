package toudocu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocaleRootAllowsTaskScaffoldAndServeWorkspace(t *testing.T) {
	root := t.TempDir()
	docs, target := filepath.Join(root, "docs"), filepath.Join(root, "docs-en")
	writeTestFile(t, docs, "index.md", "# Русский\n")
	writeTestFile(t, target, "index.md", "# English\n")
	writeSiteConfig(t, root, `project:
  locale: ru
  sections:
    architecture: Архитектура
    modules: Модули
    use-cases: Сценарии
    flows: Процессы
    screens: Экраны
    decisions: Решения
    contracts: Контракты
    quality: Качество
    runbooks: Runbooks
    reference: Справочник
    work: Задачи
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
      decisions: Decisions
      contracts: Contracts
      quality: Quality
      runbooks: Runbooks
      reference: Reference
      work: Work
      guides: Guides
`)
	options := Options{InputDirectory: target, RepositoryRoot: root, StaleDays: 0}
	model, err := BuildDocumentationModel(options)
	if err != nil || model.SiteConfig.Project.Locale != "en" {
		t.Fatalf("locale model: %#v %v", model, err)
	}
	if _, err = Scaffold(Options{InputDirectory: target, RepositoryRoot: root, EntityKind: "module", EntityID: "MOD-LOCAL", Title: "Local", Language: "en"}); err != nil {
		t.Fatal(err)
	}

	options.Command, options.OutputDirectory, options.Host = "serve", filepath.Join(root, "site"), "127.0.0.1"
	handler, _, _, err := newDocumentationServer(options, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	if home.Code != http.StatusOK || !strings.Contains(home.Body.String(), "/_toudocu/editor/") {
		t.Fatalf("workspace unavailable: %d", home.Code)
	}
}
