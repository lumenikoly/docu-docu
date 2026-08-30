package toudocu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func serveTestOptions(t *testing.T) (Options, string) {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Серверный проект\n\nПервая версия.\n")
	return Options{
		Command:         "serve",
		InputDirectory:  docs,
		OutputDirectory: filepath.Join(root, "site"),
		RepositoryRoot:  root,
		RepositoryRef:   "main",
		StaleDays:       0,
		Clean:           true,
		Host:            "127.0.0.1",
		Port:            8080,
	}, docs
}

func TestServeArguments(t *testing.T) {
	options, _, _, err := ParseArguments([]string{"serve", "./docs", "--host", "0.0.0.0", "--port=9090"})
	if err != nil {
		t.Fatal(err)
	}
	if options.Command != "serve" || options.Host != "0.0.0.0" || options.Port != 9090 {
		t.Fatalf("options: %#v", options)
	}

	defaults, _, _, err := ParseArguments([]string{"serve", "./docs"})
	if err != nil {
		t.Fatal(err)
	}
	if defaults.Host != "127.0.0.1" || defaults.Port != 8080 {
		t.Fatalf("defaults: %#v", defaults)
	}
	if defaults.Open {
		t.Fatal("serve unexpectedly enables auto-open")
	}

	for _, args := range [][]string{
		{"serve", "./docs", "--host="},
		{"serve", "./docs", "--port", "0"},
		{"serve", "./docs", "--port", "65536"},
		{"check", "./docs", "--host", "127.0.0.1"},
		{"build", "./docs", "--port", "8080"},
	} {
		if _, _, _, err := ParseArguments(args); err == nil {
			t.Fatalf("expected arguments to fail: %#v", args)
		}
	}

	unsupportedServeFlags := []string{"--edit", "--no-open"}
	for _, flag := range unsupportedServeFlags {
		if _, _, _, err := ParseArguments([]string{"serve", "./docs", flag}); err == nil {
			t.Fatalf("unsupported serve flag was accepted: %s", flag)
		}
	}
}

func TestDocumentationServerServesAndRebuilds(t *testing.T) {
	options, docs := serveTestOptions(t)
	var stderr strings.Builder
	handler, _, _, err := newDocumentationServer(options, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	assertRequestContains := func(method, target, expected string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, target, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s: status=%d body=%s stderr=%s", method, target, response.Code, response.Body.String(), stderr.String())
		}
		if expected != "" && !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("%s %s does not contain %q", method, target, expected)
		}
		if response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s is cacheable", method, target)
		}
		return response
	}

	assertRequestContains(http.MethodGet, "/", "Первая версия.")
	assertRequestContains(http.MethodGet, "/", "data-server-rebuild")
	assertRequestContains(http.MethodHead, "/", "")
	assertRequestContains(http.MethodGet, "/assets/portal.css", "font-family")
	assertRequestContains(http.MethodGet, "/report.json", `"schemaVersion": 1`)

	writeTestFile(t, docs, "index.md", "# Серверный проект\n\nВторая версия после обновления.\n")
	assertRequestContains(http.MethodGet, "/", "Первая версия.")
}

func TestDocumentationServerRendersNotFoundPage(t *testing.T) {
	options, _ := serveTestOptions(t)
	handler, _, _, err := newDocumentationServer(options, &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing/nested-page", nil))
	if response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "Page not found") || !strings.Contains(response.Body.String(), `href="../index.html"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestDocumentationServerRebuildEndpointRegeneratesSite(t *testing.T) {
	options, docs := serveTestOptions(t)
	var stderr strings.Builder
	handler, _, _, err := newDocumentationServer(options, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, docs, "index.md", "# Серверный проект\n\nВерсия после пересборки.\n")
	request := httptest.NewRequest(http.MethodPost, rebuildEndpoint, nil)
	request.Header.Set("X-Toudocu-Action", "rebuild")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s stderr=%s", response.Code, response.Body.String(), stderr.String())
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content type: %q", response.Header().Get("Content-Type"))
	}
	var result map[string]int
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["pages"] == 0 || result["documents"] == 0 {
		t.Fatalf("unexpected rebuild result: %#v", result)
	}
	generated, err := os.ReadFile(filepath.Join(options.OutputDirectory, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "Версия после пересборки.") {
		t.Fatal("rebuild endpoint did not regenerate HTML")
	}

	wrongMethod := httptest.NewRequest(http.MethodGet, rebuildEndpoint, nil)
	wrongMethodResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongMethodResponse, wrongMethod)
	if wrongMethodResponse.Code != http.StatusMethodNotAllowed || wrongMethodResponse.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("GET status=%d allow=%q", wrongMethodResponse.Code, wrongMethodResponse.Header().Get("Allow"))
	}

	missingHeader := httptest.NewRequest(http.MethodPost, rebuildEndpoint, nil)
	missingHeaderResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingHeaderResponse, missingHeader)
	if missingHeaderResponse.Code != http.StatusForbidden {
		t.Fatalf("POST without action header status=%d", missingHeaderResponse.Code)
	}
}

func TestDocumentationServerDoesNotExposeSourceOrPartialBuild(t *testing.T) {
	options, docs := serveTestOptions(t)
	var stderr strings.Builder
	handler, _, _, err := newDocumentationServer(options, &stderr)
	if err != nil {
		t.Fatal(err)
	}

	sourceRequest := httptest.NewRequest(http.MethodGet, "http://example.test/../docs/index.md", nil)
	sourceResponse := httptest.NewRecorder()
	handler.ServeHTTP(sourceResponse, sourceRequest)
	if sourceResponse.Code == http.StatusOK || strings.Contains(sourceResponse.Body.String(), "Первая версия.") {
		t.Fatalf("source file escaped output root: status=%d body=%s", sourceResponse.Code, sourceResponse.Body.String())
	}

	if err := os.RemoveAll(docs); err != nil {
		t.Fatal(err)
	}
	failedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	failedResponse := httptest.NewRecorder()
	handler.ServeHTTP(failedResponse, failedRequest)
	if failedResponse.Code != http.StatusOK || !strings.Contains(failedResponse.Body.String(), "Первая версия.") {
		t.Fatalf("status=%d body=%s", failedResponse.Code, failedResponse.Body.String())
	}
}

func TestDocumentationServerLocalePortalsAreWritableAndMatched(t *testing.T) {
	options, docs := serveTestOptions(t)
	writeTestFile(t, docs, "guide.md", "# Canonical guide\n\nCanonical text.\n")
	writeTestFile(t, docs, "roadmap.md", "# Roadmap\n\n## Next\n\n- [ ] `DLV-NEXT-001` Next.\n")
	writeTestFile(t, filepath.Join(filepath.Dir(docs), "i18n", "en"), "index.md", "# English home\n\nHome.\n")
	writeTestFile(t, filepath.Join(filepath.Dir(docs), "i18n", "en"), "guide.md", "# English guide\n\nEnglish text.\n")
	writeTestFile(t, filepath.Join(filepath.Dir(docs), "i18n", "en"), "roadmap.md", "# Roadmap\n\n## Next\n\n- [ ] `DLV-NEXT-001` Next.\n")
	writeTestFile(t, filepath.Dir(docs), ".toudocu/config.yml", `project:
  defaultLocale: ru
locales:
  ru:
    root: docs
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use cases
      flows: Flows
      screens: Screens
      decisions: Decisions
      contracts: Contracts
      quality: Quality
      runbooks: Runbooks
      reference: Reference
      work: Work
      drafts: Drafts
      guides: Guides
  en:
    root: i18n/en
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use cases
      flows: Flows
      screens: Screens
      decisions: Decisions
      contracts: Contracts
      quality: Quality
      runbooks: Runbooks
      reference: Reference
      work: Work
      guides: Guides
`)
	handler, _, _, err := newDocumentationServer(options, &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	canonical := httptest.NewRecorder()
	handler.ServeHTTP(canonical, httptest.NewRequest(http.MethodGet, "/guide.html", nil))
	if canonical.Code != http.StatusOK || !strings.Contains(canonical.Body.String(), `value="/_toudocu/locales/en/guide.html"`) {
		t.Fatalf("canonical: %d %s", canonical.Code, canonical.Body.String())
	}
	locale := httptest.NewRecorder()
	handler.ServeHTTP(locale, httptest.NewRequest(http.MethodGet, "/_toudocu/locales/en/guide.html", nil))
	if locale.Code != http.StatusOK || !strings.Contains(locale.Body.String(), "English text.") {
		t.Fatalf("locale: %d %s", locale.Code, locale.Body.String())
	}
	for _, required := range []string{"data-server-rebuild", "/_toudocu/editor/", "/changes/"} {
		if !strings.Contains(locale.Body.String(), required) {
			t.Fatalf("locale missing workspace control %q", required)
		}
	}
	localeRoadmap := httptest.NewRecorder()
	handler.ServeHTTP(localeRoadmap, httptest.NewRequest(http.MethodGet, "/_toudocu/locales/en/roadmap.html", nil))
	for _, required := range []string{"data-roadmap-add"} {
		if !strings.Contains(localeRoadmap.Body.String(), required) {
			t.Fatalf("locale roadmap missing control %q", required)
		}
	}
	for _, target := range []string{"/_toudocu/locales/en/_toudocu/api/editor/file", "/_toudocu/locales/en/../editor/"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code == http.StatusOK && strings.Contains(response.Body.String(), "Canonical text.") {
			t.Fatalf("locale escaped mount: %s", target)
		}
	}
}

func TestDocumentationServerKeepsLastGoodSnapshotAndShowsUnavailableLocale(t *testing.T) {
	options, docs := serveTestOptions(t)
	var stderr strings.Builder
	handler, _, _, err := newDocumentationServer(options, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(docs); err != nil {
		t.Fatal(err)
	}
	if _, _, err := handler.rebuild(); err == nil {
		t.Fatal("expected failed canonical rebuild")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Первая версия.") {
		t.Fatalf("last-known-good snapshot lost: %d %s", response.Code, response.Body.String())
	}

	options, docs = serveTestOptions(t)
	writeTestFile(t, filepath.Dir(docs), ".toudocu/config.yml", `project:
  defaultLocale: ru
locales:
  ru:
    root: docs
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use cases
      flows: Flows
      screens: Screens
      decisions: Decisions
      contracts: Contracts
      quality: Quality
      runbooks: Runbooks
      reference: Reference
      work: Work
      drafts: Drafts
      guides: Guides
  en:
    root: missing/en
    sections:
      architecture: Architecture
      modules: Modules
      use-cases: Use cases
      flows: Flows
      screens: Screens
      decisions: Decisions
      contracts: Contracts
      quality: Quality
      runbooks: Runbooks
      reference: Reference
      work: Work
      guides: Guides
`)
	handler, _, _, err = newDocumentationServer(options, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/_toudocu/locales/en/", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "Unavailable") || strings.Contains(response.Body.String(), filepath.Dir(docs)) {
		t.Fatalf("unavailable locale response is unsafe: %d %s", response.Code, response.Body.String())
	}
}

func TestDocumentationServerSerializesConcurrentRequests(t *testing.T) {
	options, _ := serveTestOptions(t)
	handler, _, _, err := newDocumentationServer(options, &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}

	var wait sync.WaitGroup
	for i := 0; i < 12; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			target := "/assets/portal.css"
			if index%2 == 0 {
				target = "/"
			}
			request := httptest.NewRequest(http.MethodGet, target, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Errorf("%s: status=%d body=%s", target, response.Code, response.Body.String())
			}
		}(i)
	}
	wait.Wait()
}

func TestOutputPageMapIncludesTaskWorkspace(t *testing.T) {
	model := &Model{HealthOutputPath: "health.html", Knowledge: KnowledgeModel{WorkItems: []WorkItem{{ID: "TASK-LOCALE-001"}}}}
	if got := outputPageMap(model)["work/index.html"]; got != "work/index.html" {
		t.Fatalf("task workspace page = %q", got)
	}
}

func TestBrowserURLAndNetworkWarning(t *testing.T) {
	if got := browserURL("0.0.0.0", 8080); got != "http://127.0.0.1:8080/" {
		t.Fatalf("browser URL: %s", got)
	}
	if got := browserURL("::1", 8080); got != "http://[::1]:8080/" {
		t.Fatalf("IPv6 browser URL: %s", got)
	}
	if externallyReachableHost("127.0.0.1") || externallyReachableHost("::1") || externallyReachableHost("localhost") {
		t.Fatal("loopback host marked as externally reachable")
	}
	if !externallyReachableHost("0.0.0.0") || !externallyReachableHost("192.168.1.10") {
		t.Fatal("network host not marked as externally reachable")
	}
}
