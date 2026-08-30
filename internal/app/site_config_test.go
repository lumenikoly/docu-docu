package toudocu

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	frontend "toudocu/internal/site"
)

func TestWriteStrings(t *testing.T) {
	var builder strings.Builder
	writeStrings(&builder, "one", "", " two")
	if got := builder.String(); got != "one two" {
		t.Fatalf("writeStrings() = %q", got)
	}
}

func configFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "index.md"), []byte("# Index title\n\nProject description.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSiteConfig(t, root, "")
	return root, docs
}

func writeSiteConfig(t *testing.T, root, content string) {
	t.Helper()
	content = upgradeV2TestConfig(content)
	if !strings.Contains(content, "documentationVersion:") {
		content = "documentationVersion: 3\n" + content
	}
	if !strings.Contains(content, "locales:") && !strings.Contains(content, "documentationVersion: 1") && !strings.Contains(content, "documentationVersion: 2") {
		locale := "en"
		if marker := "project:\n  locale: "; strings.Contains(content, marker) {
			rest := strings.SplitN(content, marker, 2)[1]
			locale = strings.SplitN(rest, "\n", 2)[0]
			content = strings.Replace(content, marker+locale+"\n", "", 1)
		}
		content += "project:\n  defaultLocale: " + locale + "\nlocales:\n  " + locale + ":\n    root: docs\n    sections:\n"
		for _, spec := range BuiltinSections {
			content += "      " + string(spec.Type) + ": " + defaultSectionTitle(locale, spec.Type) + "\n"
		}
	}
	directory := filepath.Join(root, ".toudocu")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "config.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func upgradeV2TestConfig(content string) string {
	marker := "project:\n  locale: "
	if !strings.Contains(content, marker) {
		return strings.Replace(content, "translations:\n", "locales:\n", 1)
	}
	rest := strings.SplitN(content, marker, 2)[1]
	locale := strings.SplitN(rest, "\n", 2)[0]
	hasProjectSections := strings.HasPrefix(strings.TrimPrefix(rest, locale+"\n"), "  sections:\n")
	content = strings.Replace(content, marker+locale+"\n", "project:\n  defaultLocale: "+locale+"\nlocales:\n  "+locale+":\n    root: docs\n", 1)
	if hasProjectSections {
		content = strings.Replace(content, "  sections:\n", "    sections:\n", 1)
	}
	lines := strings.Split(content, "\n")
	inSections := false
	for index, line := range lines {
		if hasProjectSections && line == "    sections:" && !inSections {
			inSections = true
			continue
		}
		if inSections && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") {
			lines[index] = "  " + line
			continue
		}
		if inSections && line != "" && !strings.HasPrefix(line, "    ") {
			inSections = false
		}
	}
	return strings.Replace(strings.Join(lines, "\n"), "translations:\n", "", 1)
}

func buildConfigFixture(t *testing.T, root, docs, title string) *Model {
	t.Helper()
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, Title: title, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func TestSiteConfigDefaultsAndMissingFile(t *testing.T) {
	root, docs := configFixture(t)
	model := buildConfigFixture(t, root, docs, "")
	config := model.SiteConfig
	if config.Theme != "classic" || config.ColorScheme != "system" || config.Accent != "indigo" ||
		config.Density != "comfortable" || config.ContentWidth != "standard" || !config.Hero.Enabled {
		t.Fatalf("defaults: %#v", config)
	}
	if model.Project.Title != "Index title" || config.Footer.Text != "" || !config.Footer.defaultText || config.Footer.URL != defaultFooterURL {
		t.Fatalf("default title/footer: %#v / %#v", model.Project, config.Footer)
	}
	if got := renderFooter(frontend.NewUI("ru"), config.Footer); got != `Сгенерировано <a href="https://lumenikoly.github.io/toudocu/" rel="noopener noreferrer">Toudocu</a> `+Version {
		t.Fatalf("rendered default footer: %q", got)
	}
}

func TestDocumentationVersionGatesParsing(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "# TASK-OLD-001: Legacy\n\n- Status: Ready\n- Type: Maintenance\n"
	if err := os.WriteFile(filepath.Join(docs, "TASK-OLD-001.md"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Issues) != 1 || model.Issues[0].Code != "DOCS_MIGRATION_REQUIRED" || model.Issues[0].Migration != "v1-to-v2" || len(model.Documents) != 0 {
		t.Fatalf("missing version must stop before parsing: %#v", model)
	}
	if _, err := InitTask(Options{InputDirectory: docs, RepositoryRoot: root, Area: "OLD", Title: "Must not write", TaskType: "Maintenance"}); err == nil || !strings.Contains(err.Error(), "DOCS_MIGRATION_REQUIRED") {
		t.Fatalf("task init did not honor version gate: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docs, "work")); !os.IsNotExist(err) {
		t.Fatalf("version-gated task init changed the project: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"check", docs, "--repository-root", root, "--format", "json"}, &stdout, &stderr); code != 1 || stderr.Len() != 0 {
		t.Fatalf("check code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var report ProjectReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || len(report.Issues) != 1 || report.Issues[0].Migration != "v1-to-v2" {
		t.Fatalf("migration JSON: %#v (%v)", report.Issues, err)
	}
	stdout.Reset()
	if code := RunCLI([]string{"check", docs, "--repository-root", root}, &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), "DOCS_MIGRATION_REQUIRED\n\nMigration: v1-to-v2\nFile: .toudocu/config.yml") {
		t.Fatalf("migration text: code=%d output=%q", code, stdout.String())
	}

	writeSiteConfig(t, root, "documentationVersion: 1\n")
	if explicit, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root}); err != nil || !hasDocumentationVersionIssue(explicit) {
		t.Fatalf("explicit v1 did not require migration: %#v (%v)", explicit, err)
	}
	writeSiteConfig(t, root, "")
	model = buildConfigFixture(t, root, docs, "")
	if hasDocumentationVersionIssue(model) || len(model.Documents) != 1 {
		t.Fatalf("current version did not use the current parser: %#v", model.Issues)
	}
}

func TestDocumentationVersionValidationAndFutureVersion(t *testing.T) {
	for _, value := range []string{"0", "-1", "nope"} {
		if _, err := parseSiteConfig([]byte("documentationVersion: " + value + "\n")); err == nil {
			t.Fatalf("invalid documentationVersion %q accepted", value)
		}
	}
	config, err := parseSiteConfig([]byte("documentationVersion: 3\nproject:\n  defaultLocale: en\nlocales:\n  en:\n    root: docs\n    sections:\n      architecture: Architecture\n      modules: Modules\n      use-cases: Use Cases\n      flows: Processes\n      screens: Screens\n      decisions: Decisions\n      contracts: Contracts\n      quality: Quality\n      runbooks: Runbooks\n      reference: Reference\n      work: Work\n      drafts: Drafts\n      guides: Guides\n"))
	if err != nil || config.DocumentationVersion != currentDocumentationVersion {
		t.Fatalf("current documentationVersion: %#v (%v)", config, err)
	}

	root, docs := configFixture(t)
	writeSiteConfig(t, root, "documentationVersion: 4\n")
	model := buildConfigFixture(t, root, docs, "")
	if len(model.Issues) != 1 || model.Issues[0].Code != "DOCUMENTATION_VERSION_UNSUPPORTED" || len(model.Documents) != 0 {
		t.Fatalf("future version diagnostic: %#v", model.Issues)
	}
}

func TestCustomFooterTextDoesNotInheritDefaultLink(t *testing.T) {
	config, err := parseSiteConfig([]byte("site:\n  footer:\n    text: Custom footer\n"))
	if err != nil {
		t.Fatal(err)
	}
	if config.Footer.URL != "" {
		t.Fatalf("custom text inherited default footer URL: %q", config.Footer.URL)
	}
	if got := renderFooter(frontend.NewUI("ru"), config.Footer); got != "Custom footer" {
		t.Fatalf("rendered custom footer: %q", got)
	}
}

func TestChangesConfig(t *testing.T) {
	config, err := parseSiteConfig([]byte("changes:\n  defaultBaseRef: main\n  renameSimilarity: 75\n  includeAssets: false\n  semanticDiff: true\n  maxSourceDiffBytes: 2048\n  exclude:\n    - docs/generated/**\n"))
	if err != nil {
		t.Fatal(err)
	}
	if config.Changes.DefaultBaseRef != "main" || config.Changes.RenameSimilarity != 75 || config.Changes.IncludeAssets || !config.Changes.SemanticDiff || config.Changes.MaxSourceDiffBytes != 2048 || len(config.Changes.Exclude) != 1 {
		t.Fatalf("changes config: %#v", config.Changes)
	}
}

func TestTranslationProfileSelectsIndependentLocaleRoot(t *testing.T) {
	root, _ := configFixture(t)
	target := filepath.Join(root, "docs-en")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "index.md"), []byte("# English docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := parseSiteConfig([]byte("documentationVersion: 3\n" + upgradeV2TestConfig(`project:
  locale: ru
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
`)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := selectLocaleProfile(&config, root, target); err != nil {
		t.Fatal(err)
	}
	if config.Project.Locale != "en" || config.Project.Sections[SectionModules] != "Modules" {
		t.Fatalf("translation project config was not selected: %#v", config.Project)
	}
	// A canonical selection is unaffected by configured translations.
	config, err = parseSiteConfig([]byte("documentationVersion: 3\n" + upgradeV2TestConfig(`project:
  locale: ru
translations:
  en:
    root: docs-en
    sections:
      architecture: Architecture
`)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = selectLocaleProfile(&config, root, target); err == nil {
		t.Fatal("incomplete locale profile selected")
	}
}

func TestTranslationProfileRejectsUnsafeRootWhenSelected(t *testing.T) {
	root, _ := configFixture(t)
	if _, err := safeTranslationRoot(root, "../outside"); err == nil {
		t.Fatal("unsafe translation root accepted")
	}
}

func TestSiteConfigFullAndTitlePriority(t *testing.T) {
	root, docs := configFixture(t)
	assets := filepath.Join(root, ".toudocu", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"logo.svg", "favicon.svg", "hero.webp"} {
		if err := os.WriteFile(filepath.Join(assets, name), []byte("asset-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSiteConfig(t, root, `# fixed YAML subset
site:
  title: My Project
  logo: assets/logo.svg
  favicon: assets/favicon.svg
  theme: paper
  colorScheme: dark
  accent: teal
  density: compact
  contentWidth: wide
  footer:
    text: "My Project documentation"
    url: https://example.com/docs
  hero:
    enabled: true
    image: assets/hero.webp
`)
	model := buildConfigFixture(t, root, docs, "")
	if model.Project.Title != "My Project" || model.SiteConfig.Theme != "paper" ||
		model.SiteConfig.Footer.URL != "https://example.com/docs" || len(model.BrandingAssets) != 3 {
		t.Fatalf("full config: %#v / %#v", model.SiteConfig, model.BrandingAssets)
	}
	override := buildConfigFixture(t, root, docs, "CLI title")
	if override.Project.Title != "CLI title" {
		t.Fatalf("--title priority lost: %q", override.Project.Title)
	}
}

func TestProjectBrandMark(t *testing.T) {
	for title, want := range map[string]string{
		"My Project":    "MP",
		"Toudocu":       "TO",
		"API Gateway":   "AG",
		"Мой проект":    "МП",
		"Документация":  "ДО",
		"Project-Alpha": "PA",
		"***":           "T",
	} {
		if got := projectBrandMark(title); got != want {
			t.Errorf("projectBrandMark(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestDraftsIndexUsesConfiguredTitle(t *testing.T) {
	root, docs := configFixture(t)
	writeArchitectureOverview(t, docs, "")
	writeTestFile(t, docs, "drafts/index.md", "# Other title\n")
	writeSiteConfig(t, root, "project:\n  locale: en\n  sections:\n    architecture: Architecture\n    modules: Modules\n    use-cases: Use Cases\n    flows: Processes\n    screens: Screens\n    decisions: Decisions\n    contracts: Contracts\n    quality: Quality\n    runbooks: Runbooks\n    reference: Reference\n    work: Work\n    drafts: Drafts\n    guides: Guides\n")
	model := buildConfigFixture(t, root, docs, "")
	if modelDirectoryLabel(model, "drafts") != "Drafts" {
		t.Fatalf("drafts title: %q", modelDirectoryLabel(model, "drafts"))
	}
	for _, issue := range model.Issues {
		if issue.Code == "builtin-section-title-mismatch" && issue.DocumentPath == "drafts/index.md" {
			return
		}
	}
	t.Fatalf("drafts index mismatch was not reported: %#v", model.Issues)
}

func TestSiteConfigEnums(t *testing.T) {
	tests := map[string][]string{
		"theme":        {"classic", "paper", "terminal"},
		"colorScheme":  {"light", "dark", "system"},
		"accent":       {"indigo", "blue", "teal", "green", "amber", "rose", "violet"},
		"density":      {"compact", "comfortable"},
		"contentWidth": {"narrow", "standard", "wide"},
	}
	for field, values := range tests {
		for _, value := range values {
			t.Run(field+"-"+value, func(t *testing.T) {
				config, err := parseSiteConfig([]byte("site:\n  " + field + ": " + value + "\n"))
				if err != nil {
					t.Fatal(err)
				}
				switch field {
				case "theme":
					if config.Theme != value {
						t.Fatal(config.Theme)
					}
				case "colorScheme":
					if config.ColorScheme != value {
						t.Fatal(config.ColorScheme)
					}
				case "accent":
					if config.Accent != value {
						t.Fatal(config.Accent)
					}
				case "density":
					if config.Density != value {
						t.Fatal(config.Density)
					}
				case "contentWidth":
					if config.ContentWidth != value {
						t.Fatal(config.ContentWidth)
					}
				}
			})
		}
	}
}

func TestSiteConfigRejectsInvalidYAMLSubset(t *testing.T) {
	tests := map[string]string{
		"unknown":        "site:\n  customCSS: x.css\n",
		"duplicate":      "site:\n  theme: classic\n  theme: paper\n",
		"indent":         "site:\n   theme: classic\n",
		"list":           "site:\n  theme: [classic]\n",
		"anchor":         "site:\n  theme: &base classic\n",
		"multiline":      "site:\n  title: |\n    Project\n",
		"wrong boolean":  "site:\n  hero:\n    enabled: yes\n",
		"quoted boolean": "site:\n  hero:\n    enabled: \"true\"\n",
		"wrong string":   "site:\n  title: true\n",
		"wrong enum":     "site:\n  theme: custom\n",
		"unsafe footer":  "site:\n  footer:\n    url: http://example.com\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSiteConfig([]byte(content)); err == nil {
				t.Fatalf("expected error for %q", content)
			}
		})
	}
}

func TestSiteConfigRejectsUnsafeBrandAssets(t *testing.T) {
	for name, assetPath := range map[string]string{
		"absolute":  "/tmp/logo.svg",
		"traversal": "assets/../../logo.svg",
		"missing":   "assets/missing.svg",
	} {
		t.Run(name, func(t *testing.T) {
			root, docs := configFixture(t)
			writeSiteConfig(t, root, "site:\n  logo: "+assetPath+"\n")
			if _, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0}); err == nil {
				t.Fatal("expected unsafe asset error")
			}
		})
	}
}

func TestSiteConfigRejectsBrandSymlink(t *testing.T) {
	root, docs := configFixture(t)
	assets := filepath.Join(root, ".toudocu", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.svg")
	if err := os.WriteFile(outside, []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(assets, "logo.svg")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeSiteConfig(t, root, "site:\n  logo: assets/logo.svg\n")
	if _, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestSiteConfigRejectsAssetDirectorySymlinkEscape(t *testing.T) {
	root, docs := configFixture(t)
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "logo.svg"), []byte("<svg/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".toudocu"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".toudocu", "assets")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeSiteConfig(t, root, "site:\n  logo: assets/logo.svg\n")
	if _, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected asset directory symlink error, got %v", err)
	}
}

func TestGenerateSiteBrandingAndThemeContract(t *testing.T) {
	root, docs := configFixture(t)
	output := filepath.Join(root, "site")
	assets := filepath.Join(root, ".toudocu", "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"logo.svg", "favicon.svg", "hero.webp"} {
		if err := os.WriteFile(filepath.Join(assets, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSiteConfig(t, root, `site:
  title: Branded docs
  logo: assets/logo.svg
  favicon: assets/favicon.svg
  theme: terminal
  colorScheme: system
  accent: rose
  density: compact
  contentWidth: narrow
  footer:
    text: "<strong>Escaped</strong>"
    url: https://example.com
  hero:
    enabled: true
    image: assets/hero.webp
`)
	model := buildConfigFixture(t, root, docs, "")
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	htmlBytes, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(htmlBytes)
	for _, part := range []string{
		`data-site-theme="terminal"`, `data-color-scheme="system"`, `data-accent="rose"`,
		`data-density="compact"`, `data-content-width="narrow"`,
		`assets/branding/logo.svg`, `assets/branding/favicon.svg`, `assets/branding/hero.webp`,
		`assets/portal.js`, `data-color-scheme-select`, `data-site-theme-select`, `&lt;strong&gt;Escaped&lt;/strong&gt;`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("missing %q", part)
		}
	}
	portalSource, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "core", "preferences.ts"))
	if err != nil || !strings.Contains(string(portalSource), "toudocu-site-theme") {
		t.Fatalf("portal theme preference key is missing: %v", err)
	}
	if strings.Count(html, `data-theme="`) != 1 {
		t.Fatalf("portal must render one data-theme attribute, got %d", strings.Count(html, `data-theme="`))
	}
	if strings.Contains(html, "<strong>Escaped</strong>") {
		t.Fatal("footer HTML was not escaped")
	}
	if strings.Contains(html, `class="brand-mark"`) {
		t.Fatal("text brand mark rendered despite configured logo")
	}
	for _, name := range []string{"logo.svg", "favicon.svg", "hero.webp"} {
		if _, err := os.Stat(filepath.Join(output, "assets", "branding", name)); err != nil {
			t.Fatalf("branding asset %s not copied: %v", name, err)
		}
	}
	for _, external := range []string{`src="http://`, `src="https://`, `href="http://`} {
		if strings.Contains(html, external) {
			t.Fatalf("external resource found: %s", external)
		}
	}
}

func TestGenerateSiteUsesProjectTitleBrandMark(t *testing.T) {
	root, docs := configFixture(t)
	output := filepath.Join(root, "site")
	writeSiteConfig(t, root, "site:\n  title: My Project\n")
	model := buildConfigFixture(t, root, docs, "")
	if _, err := GenerateSite(model, Options{OutputDirectory: output, Clean: true}); err != nil {
		t.Fatal(err)
	}
	html, err := os.ReadFile(filepath.Join(output, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), `<span class="brand-mark" aria-hidden="true">MP</span>`) {
		t.Fatal("static portal is missing the project title brand mark")
	}
}

func TestHeroCanBeDisabled(t *testing.T) {
	root, docs := configFixture(t)
	writeSiteConfig(t, root, "site:\n  hero:\n    enabled: false\n")
	model := buildConfigFixture(t, root, docs, "")
	html := renderDashboard(model)
	if strings.Contains(html, `class="hero dashboard-about"`) || strings.Contains(html, `class="hero has-image dashboard-about"`) || !strings.Contains(html, `class="page-header dashboard-about"`) {
		t.Fatalf("disabled hero rendered incorrectly")
	}
}
