package toudocu

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitTestRun(t *testing.T, directory string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeChangesTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	relative := filepath.ToSlash(path)
	if index := strings.LastIndex(relative, "/docs/"); index >= 0 {
		relative = relative[index+len("/docs/"):]
	}
	if err := os.WriteFile(path, []byte(canonicalizeTestMarkdown(relative, content)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newChangesRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	gitTestRun(t, root, "init", "-q")
	gitTestRun(t, root, "config", "user.email", "test@example.test")
	gitTestRun(t, root, "config", "user.name", "Toudocu Test")
	writeSiteConfig(t, root, "")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\n- Status: Active\n\n## Rules\n\nOriginal.\n")
	writeChangesTestFile(t, filepath.Join(docs, "use-cases", "UC-CORE-01.md"), "# UC-CORE-01: Open\n\n## Result\n\nOpened.\n")
	gitTestRun(t, root, "add", "docs", ".toudocu")
	gitTestRun(t, root, "commit", "-q", "-m", "baseline")
	return root, docs
}

func TestParseNameStatusNULPaths(t *testing.T) {
	changes, err := parseNameStatus([]byte("R087\x00docs/old name.md\x00docs/новое имя.md\x00T\x00docs/type.yaml\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[0].status != "renamed" || changes[0].oldPath != "docs/old name.md" || changes[0].path != "docs/новое имя.md" || changes[1].status != "type-changed" {
		t.Fatalf("unexpected changes: %#v", changes)
	}
}

func TestStatusStatesPreservePathsWithSpaces(t *testing.T) {
	root, docs := newChangesRepository(t)
	path := filepath.Join(docs, "modules", "file with spaces.md")
	writeChangesTestFile(t, path, "# Before\n")
	gitTestRun(t, root, "add", "docs/modules/file with spaces.md")
	gitTestRun(t, root, "commit", "-q", "-m", "add spaced path")
	writeChangesTestFile(t, path, "# Unstaged\n")

	source, err := openGitChangeSource(docs, 60)
	if err != nil {
		t.Fatal(err)
	}
	states, err := source.statusStates()
	if err != nil {
		t.Fatal(err)
	}
	state := states["docs/modules/file with spaces.md"]
	if !state.Unstaged || state.Staged {
		t.Fatalf("unstaged path state lost: %#v in %#v", state, states)
	}

	gitTestRun(t, root, "add", "docs/modules/file with spaces.md")
	states, err = source.statusStates()
	if err != nil {
		t.Fatal(err)
	}
	state = states["docs/modules/file with spaces.md"]
	if !state.Staged || state.Unstaged {
		t.Fatalf("staged path state lost: %#v in %#v", state, states)
	}

	renamed := filepath.Join(docs, "modules", "renamed with spaces.md")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	gitTestRun(t, root, "add", "-A", "docs/modules")
	states, err = source.statusStates()
	if err != nil {
		t.Fatal(err)
	}
	state = states["docs/modules/renamed with spaces.md"]
	if !state.Staged {
		t.Fatalf("renamed path state lost: %#v in %#v", state, states)
	}
}

func TestChangesForceIncludeAssetsOverridesConfigOnly(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeSiteConfig(t, root, "changes:\n  includeAssets: false\n")
	asset := filepath.Join(docs, "images", "diagram.png")
	writeChangesTestFile(t, asset, "\x89PNG\r\n\x1a\nasset")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "working-tree"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 0 {
		t.Fatalf("assets must follow config by default: %#v", report.Changes)
	}
	report, err = BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "working-tree", ChangeForceIncludeAssets: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || report.Changes[0].Path != "docs/images/diagram.png" || !report.Changes[0].Binary {
		t.Fatalf("asset override lost: %#v", report.Changes)
	}

	var stdout, stderr strings.Builder
	code := RunCLI([]string{
		"changes", docs, "--base", "HEAD", "--target", "working-tree",
		"--include-assets", "--format", "json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("changes --include-assets exit=%d stderr=%s", code, stderr.String())
	}
	var cliReport ChangeSetReport
	if err := json.Unmarshal([]byte(stdout.String()), &cliReport); err != nil {
		t.Fatal(err)
	}
	if len(cliReport.Changes) != 1 || cliReport.Changes[0].Path != "docs/images/diagram.png" || !cliReport.Changes[0].Binary {
		t.Fatalf("CLI asset override lost: %#v", cliReport.Changes)
	}
}

func TestChangesTranslationInputOverridesReaderFacingFilters(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeSiteConfig(t, root, "changes:\n  includeTaskArtifacts: false\n  includeAssets: false\n  exclude:\n    - docs/guides/**\n    - docs/images/**\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-CLI-001.md"), "# TASK-CLI-001: Translate\n")
	writeChangesTestFile(t, filepath.Join(docs, "guides", "reader.md"), "# Reader guide\n")
	writeChangesTestFile(t, filepath.Join(docs, "images", "diagram.png"), "\x89PNG\r\n\x1a\nasset")
	writeChangesTestFile(t, filepath.Join(docs, "generated", "ignored.md"), "# Generated\n")

	report, err := BuildDocumentationChanges(Options{
		InputDirectory: docs, RepositoryRoot: root, ChangeBase: "HEAD",
		ChangeTarget: "working-tree", ChangeTranslationInput: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]bool{}
	for _, change := range report.Changes {
		paths[change.Path] = true
	}
	for _, expected := range []string{"docs/work/TASK-CLI-001.md", "docs/guides/reader.md", "docs/images/diagram.png"} {
		if !paths[expected] {
			t.Errorf("translation input omitted %s: %#v", expected, paths)
		}
	}
	if paths["docs/generated/ignored.md"] {
		t.Fatalf("translation input included generated output: %#v", paths)
	}

	var stdout, stderr strings.Builder
	code := RunCLI([]string{
		"changes", docs, "--repository-root", root, "--base", "HEAD",
		"--target", "working-tree", "--translation-input", "--format", "json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("changes --translation-input exit=%d stderr=%s", code, stderr.String())
	}
	var cliReport ChangeSetReport
	if err := json.Unmarshal([]byte(stdout.String()), &cliReport); err != nil {
		t.Fatal(err)
	}
	if len(cliReport.Changes) != len(report.Changes) {
		t.Fatalf("CLI translation input differs from API: %#v", cliReport.Changes)
	}
}

func TestChangesUsesNestedRepositoryRootConfiguration(t *testing.T) {
	gitRoot, _ := newChangesRepository(t)
	projectRoot := filepath.Join(gitRoot, "packages", "app")
	docs := filepath.Join(projectRoot, "docs")
	writeChangesTestFile(t, filepath.Join(docs, "index.md"), "# Nested\n")
	gitTestRun(t, gitRoot, "add", "packages/app/docs")
	gitTestRun(t, gitRoot, "commit", "-q", "-m", "nested baseline")
	writeSiteConfig(t, gitRoot, "changes:\n  includeAssets: true\n")
	writeSiteConfig(t, projectRoot, "changes:\n  includeAssets: true\n  exclude:\n    - docs/images/**\n")
	writeChangesTestFile(t, filepath.Join(docs, "images", "diagram.png"), "\x89PNG\r\n\x1a\nasset")

	nested, err := BuildDocumentationChanges(Options{
		InputDirectory: docs, RepositoryRoot: projectRoot,
		ChangeBase: "HEAD", ChangeTarget: "working-tree",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nested.Changes) != 0 {
		t.Fatalf("nested repository config was not applied relative to its root: %#v", nested.Changes)
	}
	gitLevel, err := BuildDocumentationChanges(Options{
		InputDirectory: docs, RepositoryRoot: gitRoot,
		ChangeBase: "HEAD", ChangeTarget: "working-tree",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(gitLevel.Changes) != 1 || gitLevel.Changes[0].Path != "packages/app/docs/images/diagram.png" {
		t.Fatalf("Git-level configuration was not distinguished: %#v", gitLevel.Changes)
	}
}

func TestOpenGitChangeSourceResolvesRepositoryAlias(t *testing.T) {
	root, _ := newChangesRepository(t)
	alias := filepath.Join(t.TempDir(), "repository")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("symlink is unavailable: %v", err)
	}

	source, err := openGitChangeSource(filepath.Join(alias, "docs"), 60)
	if err != nil {
		t.Fatal(err)
	}
	if source.docsRel != "docs" {
		t.Fatalf("unexpected documentation path: %q", source.docsRel)
	}
}

func TestSourceDiffHunksKeepLocationsAndPatch(t *testing.T) {
	patch := "diff --git a/docs/a.md b/docs/a.md\n--- a/docs/a.md\n+++ b/docs/a.md\n@@ -2,2 +2,3 @@ heading\n old\n+new\n@@ -20 +21,0 @@ tail\n-gone\n"
	hunks := parseSourceDiffHunks(patch)
	if len(hunks) != 2 || hunks[0].ID != "hunk-2-2" || hunks[0].OldLines != 2 || hunks[0].NewLines != 3 || !strings.Contains(hunks[0].Patch, "+new") || hunks[1].NewLines != 0 {
		t.Fatalf("unexpected hunks: %#v", hunks)
	}
}

func TestRenderedSectionsAndTypedSemanticDiff(t *testing.T) {
	oldContent := []byte(canonicalizeTestMarkdown("modules/MOD-AUTH.md", "# MOD-AUTH: Auth\n\n- Status: Active\n\n## Business rules\n\n- `BR-AUTH-001` Login is allowed.\n\n## Responsibility\n\nOwn auth.\n"))
	newContent := []byte(canonicalizeTestMarkdown("modules/MOD-AUTH.md", "# MOD-AUTH: Auth\n\n- Status: Deprecated\n\n## Responsibility\n\nOwn identity.\n\n## Business rules\n\n- `BR-AUTH-001` Login requires MFA.\n- `BR-AUTH-002` Lock after five attempts.\n"))
	sections, diagnostics := renderedSectionDiff(oldContent, newContent, "docs/modules/MOD-AUTH.md", "docs/modules/MOD-AUTH.md", nil)
	states := map[string]string{}
	for _, section := range sections {
		states[section.ID] = section.Status
	}
	if len(diagnostics) != 0 || states["business-rules"] != "modified-section" || states["responsibility"] != "modified-section" {
		t.Fatalf("sections=%#v diagnostics=%#v", sections, diagnostics)
	}
	entity := []ChangeEntity{{ID: "MOD-AUTH", Type: "module", Title: "Auth"}}
	semantic := semanticMarkdownDiff(oldContent, newContent, "docs/modules/MOD-AUTH.md", "docs/modules/MOD-AUTH.md", entity, entity)
	kinds := map[string]bool{}
	for _, change := range semantic {
		kinds[change.Kind+":"+change.Field] = true
		if change.Kind == "rule-added" && change.Field == "BR-AUTH-002" && (change.SourceAfter == nil || change.SourceAfter.Line == 0) {
			t.Fatalf("missing source location: %#v", change)
		}
	}
	if !kinds["status-changed:status"] || !kinds["rule-changed:BR-AUTH-001"] || !kinds["rule-added:BR-AUTH-002"] {
		t.Fatalf("typed semantic changes missing: %#v", semantic)
	}
}

func TestMermaidBlockDiffMatchesStableIDsAndKeepsBothSides(t *testing.T) {
	oldContent := []byte("# Flow\n\n## Diagram\n\n```mermaid\n%% id: login\nflowchart LR\nA --> B\n```\n\n## Other\n\n```mermaid\nsequenceDiagram\nA->>B: old\n```\n")
	newContent := []byte("# Flow\n\n## Diagram\n\n```mermaid\n%% id: login\nflowchart LR\nA --> C\n```\n\n## Other\n\n```mermaid\nsequenceDiagram\nA->>B: new\n```\n\n```mermaid\n%% id: extra\nflowchart LR\nC --> D\n```\n")
	blocks, diagnostics := mermaidBlockDiff(oldContent, newContent, "docs/flows/FLOW.md", "docs/flows/FLOW.md", nil)
	if len(diagnostics) != 0 || len(blocks) != 3 {
		t.Fatalf("blocks=%#v diagnostics=%#v", blocks, diagnostics)
	}
	byID := map[string]MermaidBlockChange{}
	for _, block := range blocks {
		byID[block.ID] = block
	}
	login := byID["id-login"]
	if login.Status != "modified" || !strings.Contains(login.Before, "A --> B") || !strings.Contains(login.After, "A --> C") || login.SourceBefore == nil || login.SourceAfter == nil {
		t.Fatalf("login=%#v", login)
	}
	if byID["id-extra"].Status != "added" || byID["id-extra"].Before != "" {
		t.Fatalf("extra=%#v", byID["id-extra"])
	}
}

func TestMermaidBlockDiffReportsInvalidSideWithoutDroppingDiff(t *testing.T) {
	changes, diagnostics := mermaidBlockDiff([]byte("# A\n```mermaid\nnot-a-diagram\n```"), []byte("# A\n```mermaid\nflowchart TD\n  A-->B\n```"), "docs/a.md", "docs/a.md", nil)
	if len(changes) != 1 || changes[0].Before == "" || changes[0].After == "" {
		t.Fatalf("Mermaid diff was lost: %#v", changes)
	}
	found := false
	for _, diagnostic := range diagnostics {
		found = found || diagnostic.Code == "mermaid-old-version-invalid"
	}
	if !found {
		t.Fatalf("expected invalid old Mermaid diagnostic: %#v", diagnostics)
	}
}

func TestSemanticAddedUntypedDocumentHasReadableSummary(t *testing.T) {
	entity := []ChangeEntity{{Type: "guide", Title: "How to review"}}
	changes := semanticMarkdownDiff(nil, []byte("# How to review\n"), "", "docs/guides/review.md", nil, entity)
	if len(changes) != 1 || changes[0].Summary != "Added document How to review." {
		t.Fatalf("summary: %#v", changes)
	}
}

func TestAssetMetadataForPNGSVGAndWebP(t *testing.T) {
	pngHeader := make([]byte, 26)
	copy(pngHeader, []byte("\x89PNG\r\n\x1a\n"))
	pngHeader[25] = 6
	png := inspectAsset(pngHeader)
	if png.MediaType != "image/png" || png.Transparency == nil || !*png.Transparency {
		t.Fatalf("png metadata: %#v", png)
	}
	svg := inspectAsset([]byte(`<svg viewBox="0 0 640 360" xmlns="http://www.w3.org/2000/svg"></svg>`))
	if svg.MediaType != "image/svg+xml" || svg.Width != 640 || svg.Height != 360 || svg.AspectRatio == 0 {
		t.Fatalf("svg metadata: %#v", svg)
	}
	webp := make([]byte, 30)
	copy(webp, []byte("RIFF\x16\x00\x00\x00WEBPVP8X"))
	webp[20] = 0x10
	webp[24], webp[27] = 0xff, 0x7f
	binary.LittleEndian.PutUint32(webp[4:8], uint32(len(webp)-8))
	metadata := inspectAsset(webp)
	if metadata.MediaType != "image/webp" || metadata.Width != 256 || metadata.Height != 128 || metadata.Transparency == nil || !*metadata.Transparency {
		t.Fatalf("webp metadata: %#v", metadata)
	}
}

func TestScreenDiffKeepsGhostEndpoints(t *testing.T) {
	oldContent := []byte("<!-- toudocu\nid: SC-AUTH-LOGIN\nroute: /login\n-->\n# SC-AUTH-LOGIN: Login\n\n## Transitions\n\n<!-- toudocu:table transitions columns=id,useCase,action,condition,target,kind -->\n| ID | Scenario | Action | Condition | Target | Type |\n|---|---|---|---|---|---|\n| TR-AUTH-001 | UC-AUTH-01 | Submit | Invalid | SC-AUTH-ERROR | navigation |\n| TR-AUTH-002 | UC-AUTH-01 | Cancel | Always | SC-HOME | navigation |\n")
	newContent := []byte("<!-- toudocu\nid: SC-AUTH-LOGIN\nroute: /sign-in\n-->\n# SC-AUTH-LOGIN: Login\n\n## Transitions\n\n<!-- toudocu:table transitions columns=id,useCase,action,condition,target,kind -->\n| ID | Scenario | Action | Condition | Target | Type |\n|---|---|---|---|---|---|\n| TR-AUTH-001 | UC-AUTH-01 | Submit | Locked | SC-AUTH-LOCKED | navigation |\n| TR-AUTH-003 | UC-AUTH-01 | Help | Always | SC-AUTH-HELP | navigation |\n")
	diff := buildScreenDiffMetadata(oldContent, newContent, "docs/screens/SC-AUTH-LOGIN.md", "docs/screens/SC-AUTH-LOGIN.md")
	if diff.Before == nil || diff.After == nil || diff.Before.Route != "/login" || diff.After.Route != "/sign-in" || len(diff.Transitions) != 3 {
		t.Fatalf("screen diff: %#v", diff)
	}
	states := map[string]ScreenTransitionChange{}
	for _, transition := range diff.Transitions {
		states[transition.ID] = transition
	}
	if states["TR-AUTH-001"].Status != "modified" || states["TR-AUTH-001"].Before.Target != "SC-AUTH-ERROR" || states["TR-AUTH-001"].After.Target != "SC-AUTH-LOCKED" || states["TR-AUTH-002"].Status != "removed" || states["TR-AUTH-002"].After != nil || states["TR-AUTH-003"].Status != "added" {
		t.Fatalf("transition states: %#v", states)
	}
	report := &ChangeSetReport{ChangeSetDigest: "sha256:test", Changes: []DocumentationChange{{Status: "modified", Path: "docs/screens/SC-AUTH-LOGIN.md", Screen: diff}}}
	mapDiff := buildScreenMapChanges(report)
	if len(mapDiff["nodes"].([]map[string]any)) != 1 || len(mapDiff["edges"].([]map[string]any)) != 3 {
		t.Fatalf("screen map: %#v", mapDiff)
	}
}

func TestDocumentationChangesWorkingTreeIncludesGitStates(t *testing.T) {
	root, docs := newChangesRepository(t)
	module := filepath.Join(docs, "modules", "MOD-CORE.md")
	writeChangesTestFile(t, module, "# MOD-CORE: Core\n\n- Status: Active\n\n## Rules\n\nStaged.\n")
	gitTestRun(t, root, "add", "docs/modules/MOD-CORE.md")
	writeChangesTestFile(t, module, "# MOD-CORE: Core\n\n- Status: Active\n\n## Rules\n\nWorking tree.\n")
	writeChangesTestFile(t, filepath.Join(docs, "guides", "новый файл.md"), "# Unicode guide\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs})
	if err != nil {
		t.Fatal(err)
	}
	if report.SchemaVersion != 1 || report.Comparison.Base.Revision != "HEAD" || report.Comparison.Target.Type != "working-tree" || !strings.HasPrefix(report.ChangeSetDigest, "sha256:") {
		t.Fatalf("invalid report header: %#v", report)
	}
	byPath := map[string]DocumentationChange{}
	for _, change := range report.Changes {
		byPath[change.Path] = change
	}
	modified := byPath["docs/modules/MOD-CORE.md"]
	if !modified.GitState.Staged || !modified.GitState.Unstaged || !modified.SourceDiffAvailable || modified.Lines.Added == 0 || !modified.SemanticDiffAvailable {
		t.Fatalf("invalid modified change: %#v", modified)
	}
	untracked := byPath["docs/guides/новый файл.md"]
	if untracked.Status != "untracked" || !untracked.GitState.Untracked || !strings.Contains(untracked.SourceDiff, "+# Unicode guide") {
		t.Fatalf("invalid untracked change: %#v", untracked)
	}
}

func TestDocumentationChangesIndexExcludesUnstaged(t *testing.T) {
	root, docs := newChangesRepository(t)
	module := filepath.Join(docs, "modules", "MOD-CORE.md")
	writeChangesTestFile(t, module, "# MOD-CORE: Core\n\n## Rules\n\nStaged.\n")
	gitTestRun(t, root, "add", "docs/modules/MOD-CORE.md")
	writeChangesTestFile(t, filepath.Join(docs, "use-cases", "UC-CORE-01.md"), "# UC-CORE-01: Changed only in worktree\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTarget: "index"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || report.Changes[0].Path != "docs/modules/MOD-CORE.md" {
		t.Fatalf("index included wrong files: %#v", report.Changes)
	}
}

func TestDocumentationChangesFileScopeBuildsOnlyRequestedPath(t *testing.T) {
	_, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")
	writeChangesTestFile(t, filepath.Join(docs, "use-cases", "UC-CORE-01.md"), "# UC-CORE-01: Changed\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeFile: "docs/modules/MOD-CORE.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || report.Changes[0].Path != "docs/modules/MOD-CORE.md" || len(report.Changes[0].SourceDiffHunks) == 0 {
		t.Fatalf("file-scoped report: %#v", report.Changes)
	}
}

func TestDocumentationChangesUnstagedUsesIndexAsBase(t *testing.T) {
	root, docs := newChangesRepository(t)
	module := filepath.Join(docs, "modules", "MOD-CORE.md")
	writeChangesTestFile(t, module, "# MOD-CORE: Core\n\nStaged.\n")
	gitTestRun(t, root, "add", "docs/modules/MOD-CORE.md")
	writeChangesTestFile(t, module, "# MOD-CORE: Core\n\nUnstaged.\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "index", ChangeTarget: "working-tree"})
	if err != nil {
		t.Fatal(err)
	}
	if report.Comparison.Base.Type != "index" || len(report.Changes) != 1 || !strings.Contains(report.Changes[0].SourceDiff, "+Unstaged.") || strings.Contains(report.Changes[0].SourceDiff, "-Original.") {
		t.Fatalf("wrong index comparison: %#v", report)
	}
}

func TestDocumentationChangesRevisionExpression(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nSecond.\n")
	gitTestRun(t, root, "add", "docs")
	gitTestRun(t, root, "commit", "-q", "-m", "second")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD~1", ChangeTarget: "HEAD"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("revision expression ignored: %#v", report.Changes)
	}
}

func TestDocumentationChangesRenameKeepsEntity(t *testing.T) {
	root, docs := newChangesRepository(t)
	if err := os.Rename(filepath.Join(docs, "modules", "MOD-CORE.md"), filepath.Join(docs, "modules", "MOD-RENAMED.md")); err != nil {
		t.Fatal(err)
	}
	gitTestRun(t, root, "add", "-A")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, change := range report.Changes {
		if change.Path == "docs/modules/MOD-RENAMED.md" {
			found = true
			if change.Status != "renamed" || change.OldPath != "docs/modules/MOD-CORE.md" || len(change.EntitiesAfter) == 0 || change.EntitiesAfter[0].ID != "MOD-CORE" {
				t.Fatalf("invalid rename: %#v", change)
			}
		}
	}
	if !found {
		t.Fatal("rename not found")
	}
}

func TestDocumentationChangesCoalescedMoveRetainsSemanticDiff(t *testing.T) {
	root, docs := newChangesRepository(t)
	oldPath := "docs/modules/MOD-CORE.md"
	newPath := "docs/modules/MOD-MOVED.md"
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(oldPath))); err != nil {
		t.Fatal(err)
	}
	writeChangesTestFile(t, filepath.Join(root, filepath.FromSlash(newPath)), "# MOD-CORE: Core\n\n- Status: Deprecated\n\n## Rules\n\nMoved and changed.\n")
	gitTestRun(t, root, "add", "-A")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeRenameSimilarity: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("coalesced move was not retained: %#v", report.Changes)
	}
	change := report.Changes[0]
	if change.Status != "renamed" || change.OldPath != oldPath || change.Path != newPath {
		t.Fatalf("unexpected move: %#v", change)
	}
	kinds := map[string]bool{}
	for _, semantic := range change.SemanticChanges {
		kinds[semantic.Kind+":"+semantic.Field] = true
	}
	if !kinds["status-changed:status"] || !kinds["entity-moved:"] {
		t.Fatalf("move lost semantic changes: %#v", change.SemanticChanges)
	}
}

func TestDocumentationChangesOldPathFilterKeepsGitRename(t *testing.T) {
	root, docs := newChangesRepository(t)
	oldPath := "docs/modules/MOD-CORE.md"
	newPath := "docs/modules/MOD-RENAMED.md"
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(oldPath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(oldPath))); err != nil {
		t.Fatal(err)
	}
	writeChangesTestFile(t, filepath.Join(root, filepath.FromSlash(newPath)), string(content))
	gitTestRun(t, root, "add", "-A")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeFile: oldPath})
	if err != nil {
		t.Fatal(err)
	}
	filterDocumentationChanges(report, Options{ChangeFile: oldPath})
	if len(report.Changes) != 1 || report.Changes[0].Path != newPath || report.Changes[0].OldPath != oldPath {
		t.Fatalf("old path did not retain Git rename: %#v", report.Changes)
	}
}

func TestDocumentationChangesFilteredDigestMatchesFilteredReport(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(root, "docs/modules/MOD-CORE.md"), "# MOD-CORE: Core\n\nModified.\n")
	writeChangesTestFile(t, filepath.Join(root, "docs/guides/new.md"), "# New guide\n")
	gitTestRun(t, root, "add", "docs/guides/new.md")
	modified, err := BuildDocumentationChanges(Options{InputDirectory: docs})
	if err != nil {
		t.Fatal(err)
	}
	filterDocumentationChanges(modified, Options{ChangeStatus: "modified"})
	added, err := BuildDocumentationChanges(Options{InputDirectory: docs})
	if err != nil {
		t.Fatal(err)
	}
	filterDocumentationChanges(added, Options{ChangeStatus: "added"})
	if len(modified.Changes) != 1 || len(added.Changes) != 1 || modified.ChangeSetDigest == added.ChangeSetDigest {
		t.Fatalf("filtered reports have incorrect digests: modified=%#v added=%#v", modified, added)
	}
	if modified.ChangeSetDigest != digestChangeSet(modified) || added.ChangeSetDigest != digestChangeSet(added) {
		t.Fatalf("digest does not match filtered response: modified=%s added=%s", modified.ChangeSetDigest, added.ChangeSetDigest)
	}
}

func TestChangesCLIJSONAndExitCodes(t *testing.T) {
	_, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")
	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"changes", docs, "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d: %s", code, stderr.String())
	}
	var report ChangeSetReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("unexpected report: %#v", report.Changes)
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"changes", docs, "--base", "does-not-exist"}, &stdout, &stderr); code != 2 {
		t.Fatalf("invalid revision code = %d, stderr=%s", code, stderr.String())
	}
}

func TestChangesWithoutGitExitCode(t *testing.T) {
	docs := filepath.Join(t.TempDir(), "docs")
	writeChangesTestFile(t, filepath.Join(docs, "index.md"), "# Docs\n")
	var stdout, stderr bytes.Buffer
	if code := RunCLI([]string{"changes", docs}, &stdout, &stderr); code != 3 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestOpenAPIDiffCompatibility(t *testing.T) {
	oldSpec := []byte("openapi: 3.0.3\npaths:\n  /login:\n    post:\n      operationId: login\n      responses:\n        '200':\n          description: ok\n        '401':\n          description: invalid\n")
	newSpec := []byte("openapi: 3.0.3\npaths:\n  /login:\n    post:\n      operationId: login\n      responses:\n        '200':\n          description: ok\n        '423':\n          description: locked\n  /logout:\n    post:\n      operationId: logout\n      responses:\n        '204':\n          description: done\n")
	changes, diagnostics, available := openAPIDiff(oldSpec, newSpec, "docs/contracts/auth.yaml", "docs/contracts/auth.yaml")
	if !available || len(changes) < 3 {
		t.Fatalf("changes=%#v diagnostics=%#v", changes, diagnostics)
	}
	compatibility := map[string]string{}
	for _, change := range changes {
		compatibility[change.Field] = change.Compatibility
	}
	if compatibility["POST /login.responses.401"] != "breaking" || compatibility["POST /login.responses.423"] != "non-breaking" || compatibility["POST /logout"] != "non-breaking" {
		t.Fatalf("wrong compatibility: %#v", compatibility)
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != "openapi-breaking-change" {
		t.Fatalf("missing breaking diagnostic: %#v", diagnostics)
	}
}

func TestOpenAPIDiffClassifiesParametersSecurityAndSchemaProperties(t *testing.T) {
	oldSpec := []byte(`openapi: 3.1.0
info: {title: Auth, version: 1.0.0}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
  schemas:
    Login:
      type: object
      required: [email]
      properties:
        email: {type: string}
        role: {type: string, enum: [user, admin]}
paths:
  /login:
    post:
      security: [{bearerAuth: []}, {}]
      parameters:
        - {name: locale, in: query, required: false, schema: {type: string}}
      responses:
        '200':
          description: ok
          headers:
            X-Request-ID: {schema: {type: string}}
`)
	newSpec := []byte(`openapi: 3.1.0
info: {title: Auth, version: 1.1.0}
components:
  securitySchemes: {}
  schemas:
    Login:
      type: object
      required: [email, password]
      properties:
        email: {type: string}
        password: {type: string}
        role: {type: string, enum: [user]}
paths:
  /login:
    post:
      security: [{bearerAuth: []}]
      parameters:
        - {name: locale, in: query, required: false, schema: {type: string}}
        - {name: client, in: header, required: true, schema: {type: string}}
      requestBody:
        required: true
      responses:
        '200':
          description: ok
          headers: {}
`)
	changes, diagnostics, available := openAPIDiff(oldSpec, newSpec, "docs/contracts/auth.yaml", "docs/contracts/auth.yaml")
	if !available {
		t.Fatalf("diff unavailable: %#v", diagnostics)
	}
	compatibility := map[string]string{}
	for _, change := range changes {
		compatibility[change.Field] = change.Compatibility
	}
	for field, want := range map[string]string{
		"POST /login.parameters.header:client":           "breaking",
		"POST /login.requestBody":                        "breaking",
		"POST /login.security":                           "breaking",
		"POST /login.responses.200.headers.X-Request-ID": "potentially-breaking",
		"components.schemas.Login.properties.password":   "non-breaking",
		"components.schemas.Login.required.password":     "breaking",
		"components.schemas.Login.properties.role.enum":  "breaking",
		"components.securitySchemes.bearerAuth":          "potentially-breaking",
	} {
		if got := compatibility[field]; got != want {
			t.Fatalf("%s = %q, want %q; %#v", field, got, want, changes)
		}
	}
	if len(diagnostics) == 0 {
		t.Fatalf("breaking diagnostics missing: %#v", diagnostics)
	}
}

func TestTaskImpactSeparatesTaskAndReportsMismatch(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-CORE-001.md"), "# TASK-CORE-001: Change\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n- `docs/contracts/missing.yaml`\n")
	gitTestRun(t, root, "add", "docs/work/TASK-CORE-001.md")
	gitTestRun(t, root, "commit", "-q", "-m", "task")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")
	writeChangesTestFile(t, filepath.Join(docs, "guides", "extra.md"), "# Extra\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-CORE-001.md"), "# TASK-CORE-001: Change\n\n- Status: Done\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n- `docs/contracts/missing.yaml`\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "TASK-CORE-001"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskImpact == nil || len(report.TaskImpact.TaskChanges) != 1 {
		t.Fatalf("task changes not separated: %#v", report.TaskImpact)
	}
	codes := map[string]bool{}
	for _, diagnostic := range report.TaskImpact.Diagnostics {
		codes[diagnostic.Code] = true
	}
	if !codes["declared-document-not-created"] || !codes["undeclared-document-created"] {
		t.Fatalf("missing task impact diagnostics: %#v", report.TaskImpact.Diagnostics)
	}
}

func TestTaskChangesIgnoresUnrelatedDuplicateTaskIDs(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-CORE-001.md"), "# TASK-CORE-001: Selected\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n")
	duplicate := "# TASK-OTHER-001: Duplicate\n\n## Result\n\nDuplicate.\n"
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-OTHER-001.md"), duplicate)
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-OTHER-copy.md"), duplicate)
	gitTestRun(t, root, "add", "docs/work")
	gitTestRun(t, root, "commit", "-q", "-m", "tasks")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")

	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "TASK-CORE-001"})
	if err != nil || report.TaskImpact == nil || report.TaskImpact.TaskID != "TASK-CORE-001" {
		t.Fatalf("unrelated duplicate broke ordinary task changes: report=%#v err=%v", report, err)
	}
	if _, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "TASK-OTHER-001"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("selected duplicate task ID was not rejected: %v", err)
	}
}

func TestTaskChangesTreeRejectsBugs(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "work", "BUG-CORE-001.md"), "# BUG-CORE-001: Root bug\n\n## Documentation impact\n\n- `docs/use-cases/UC-CORE-01.md`\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "TASK-CORE-100.md"), "# TASK-CORE-100: Parent\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "BUG-CORE-002.md"), "# BUG-CORE-002: Illegal child\n\n- Parent: TASK-CORE-100\n\n## Documentation impact\n\n- `docs/use-cases/UC-CORE-01.md`\n")
	gitTestRun(t, root, "add", "docs/work")
	gitTestRun(t, root, "commit", "-q", "-m", "task and bugs")

	if _, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "BUG-CORE-001", ChangeTaskTree: true}); err == nil || !strings.Contains(err.Error(), "TASK-*") {
		t.Fatalf("BUG root accepted tree mode: %v", err)
	}
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "TASK-CORE-100", ChangeTaskTree: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskImpact == nil || len(report.TaskImpact.Declared) != 1 || report.TaskImpact.Declared[0].Path != "docs/modules/MOD-CORE.md" || strings.Join(report.TaskImpact.Declared[0].DeclaredBy, ",") != "TASK-CORE-100" {
		t.Fatalf("BUG child entered TASK decomposition: %#v", report.TaskImpact)
	}
}

func TestTaskChangesTreeAggregatesDescendantsAndOwnership(t *testing.T) {
	root, docs := newChangesRepository(t)
	parentPath := filepath.Join(docs, "work", "TASK-CORE-100.md")
	childPath := filepath.Join(docs, "work", "TASK-CORE-101.md")
	grandchildPath := filepath.Join(docs, "work", "TASK-CORE-111.md")
	writeChangesTestFile(t, parentPath, "# TASK-CORE-100: Parent\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n")
	writeChangesTestFile(t, childPath, "# TASK-CORE-101: Child\n\n- Parent: TASK-CORE-100\n\n## Documentation impact\n\n- `docs/use-cases/UC-CORE-01.md`\n")
	writeChangesTestFile(t, grandchildPath, "# TASK-CORE-111: Grandchild\n\n- Parent: TASK-CORE-101\n\n## Documentation impact\n\n- `docs/reference/auth.md`\n")
	gitTestRun(t, root, "add", "docs/work")
	gitTestRun(t, root, "commit", "-q", "-m", "task tree")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")
	writeChangesTestFile(t, filepath.Join(docs, "use-cases", "UC-CORE-01.md"), "# UC-CORE-01: Open\n\nChanged.\n")
	writeChangesTestFile(t, filepath.Join(docs, "reference", "auth.md"), "# Auth\n\nChanged.\n")
	writeChangesTestFile(t, childPath, "# TASK-CORE-101: Child\n\n- Parent: TASK-CORE-100\n- Status: Done\n\n## Documentation impact\n\n- `docs/use-cases/UC-CORE-01.md`\n")
	writeChangesTestFile(t, grandchildPath, "# TASK-CORE-111: Grandchild\n\n- Parent: TASK-CORE-101\n- Status: Done\n\n## Documentation impact\n\n- `docs/reference/auth.md`\n")

	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeTaskID: "TASK-CORE-100", ChangeTaskTree: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskImpact == nil || len(report.TaskImpact.Declared) != 3 || len(report.TaskImpact.TaskChanges) != 2 {
		t.Fatalf("tree impact not aggregated: %#v", report.TaskImpact)
	}
	owners := map[string][]string{}
	for _, entry := range report.TaskImpact.Declared {
		owners[entry.Path] = entry.DeclaredBy
	}
	if strings.Join(owners["docs/modules/MOD-CORE.md"], ",") != "TASK-CORE-100" || strings.Join(owners["docs/use-cases/UC-CORE-01.md"], ",") != "TASK-CORE-101" {
		t.Fatalf("ownership lost: %#v", owners)
	}
	if strings.Join(owners["docs/reference/auth.md"], ",") != "TASK-CORE-111" {
		t.Fatalf("deep descendant ownership lost: %#v", owners)
	}
	filterDocumentationChanges(report, Options{ChangeTaskID: "TASK-CORE-100", ChangeTaskTree: true})
	if len(report.Changes) != 5 {
		t.Fatalf("tree filter excluded descendant changes: %#v", report.Changes)
	}
}

func TestTaskImpactUsesSelectedTargetTaskDocument(t *testing.T) {
	root, docs := newChangesRepository(t)
	taskPath := filepath.Join(docs, "work", "TASK-CORE-001.md")
	writeChangesTestFile(t, taskPath, "# TASK-CORE-001: Change\n\n## Documentation impact\n\n- `docs/modules/MOD-CORE.md`\n")
	gitTestRun(t, root, "add", "docs/work/TASK-CORE-001.md")
	gitTestRun(t, root, "commit", "-q", "-m", "add task")
	writeChangesTestFile(t, taskPath, "# TASK-CORE-001: Change\n\n## Documentation impact\n\n- `docs/use-cases/UC-CORE-01.md`\n")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "HEAD", ChangeTaskID: "TASK-CORE-001"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskImpact == nil || len(report.TaskImpact.Declared) != 1 || report.TaskImpact.Declared[0].Path != "docs/modules/MOD-CORE.md" {
		t.Fatalf("task impact used working tree instead of target: %#v", report.TaskImpact)
	}
	filterDocumentationChanges(report, Options{ChangeTaskID: "TASK-CORE-001"})
	if len(report.Changes) != 0 {
		t.Fatalf("task filter used working tree instead of target: %#v", report.Changes)
	}
}

func TestTaskImpactResolvesRelativeDocumentationLinks(t *testing.T) {
	root, docs := newChangesRepository(t)
	taskPath := filepath.Join(docs, "work", "TASK-CORE-001.md")
	writeChangesTestFile(t, taskPath, "# TASK-CORE-001: Change\n\n## Documentation impact\n\nUpdate [the core module](../modules/MOD-CORE.md) and [local support](./support.md).\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "support.md"), "# Support\n\nOriginal.\n")
	gitTestRun(t, root, "add", "docs/work/TASK-CORE-001.md", "docs/work/support.md")
	gitTestRun(t, root, "commit", "-q", "-m", "add task")
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged.\n")
	writeChangesTestFile(t, filepath.Join(docs, "work", "support.md"), "# Support\n\nChanged.\n")

	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "working-tree", ChangeTaskID: "TASK-CORE-001"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TaskImpact == nil || len(report.TaskImpact.Declared) != 2 {
		t.Fatalf("relative documentation impact was not resolved: %#v", report.TaskImpact)
	}
	declared := map[string]TaskImpactEntry{}
	for _, entry := range report.TaskImpact.Declared {
		declared[entry.Path] = entry
	}
	for _, path := range []string{"docs/modules/MOD-CORE.md", "docs/work/support.md"} {
		if entry, ok := declared[path]; !ok || !entry.Changed {
			t.Fatalf("unexpected declared impact for %s: %#v", path, report.TaskImpact.Declared)
		}
	}
	for _, diagnostic := range report.TaskImpact.Diagnostics {
		if diagnostic.Code == "undeclared-document-change" {
			t.Fatalf("resolved relative link was still reported as undeclared: %#v", report.TaskImpact.Diagnostics)
		}
	}
}

func TestTaskChangesDoesNotInferTaskIDFromHeading(t *testing.T) {
	root, docs := newChangesRepository(t)
	exactPath := filepath.Join(docs, "work", "custom-name.md")
	prefixPath := filepath.Join(docs, "work", "TASK-CORE-0010.md")
	writeChangesTestFile(t, exactPath, "# TASK-CORE-001: Exact\n\n## Documentation impact\n\nNo changes.\n")
	writeChangesTestFile(t, prefixPath, "# TASK-CORE-0010: Prefix\n\n## Documentation impact\n\nNo changes.\n")
	gitTestRun(t, root, "add", "docs/work")
	gitTestRun(t, root, "commit", "-q", "-m", "add tasks")
	writeChangesTestFile(t, exactPath, "# TASK-CORE-001: Exact\n\nChanged.\n")
	writeChangesTestFile(t, prefixPath, "# TASK-CORE-0010: Prefix\n\nChanged.\n")

	if _, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD", ChangeTarget: "working-tree", ChangeTaskID: "TASK-CORE-001"}); err == nil {
		t.Fatal("a heading-only task ID must not be accepted")
	}
}

func TestCommittedInBranchGitState(t *testing.T) {
	root, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nCommitted branch change.\n")
	gitTestRun(t, root, "add", "docs/modules/MOD-CORE.md")
	gitTestRun(t, root, "commit", "-q", "-m", "branch change")
	report, err := BuildDocumentationChanges(Options{InputDirectory: docs, ChangeBase: "HEAD~1", ChangeTarget: "working-tree"})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || !report.Changes[0].GitState.CommittedInBranch {
		t.Fatalf("changes=%#v", report.Changes)
	}
}

func TestTaskFilterKeepsDeclaredAndEntityRelatedChanges(t *testing.T) {
	root := t.TempDir()
	writeChangesTestFile(t, filepath.Join(root, "docs/work/TASK-AUTH-1.md"), "# TASK-AUTH-1: Auth change\n\n## Влияние на документацию\n\n- docs/modules/MOD-AUTH.md\n")
	report := &ChangeSetReport{Repository: ChangeRepository{Root: root}, Summary: ChangeSummary{Entities: map[string]int{}, Classifications: map[string]int{}}, Changes: []DocumentationChange{
		{Path: "docs/work/TASK-AUTH-1.md", Classification: "work-artifact"},
		{Path: "docs/modules/MOD-AUTH.md", Classification: "permanent-documentation", EntitiesAfter: []ChangeEntity{{ID: "MOD-AUTH", Type: "module"}}},
		{Path: "docs/screens/SC-AUTH.md", Classification: "permanent-documentation", RelationChanges: []RelationChange{{Source: ChangeEntity{ID: "SC-AUTH"}, Target: ChangeEntity{ID: "MOD-AUTH"}}}},
		{Path: "docs/guides/unrelated.md", Classification: "permanent-documentation"},
	}}
	filterDocumentationChanges(report, Options{ChangeTaskID: "TASK-AUTH-1"})
	if len(report.Changes) != 3 {
		t.Fatalf("task filter kept %d changes, want 3: %#v", len(report.Changes), report.Changes)
	}
}

func TestMarkdownSemanticValidityRejectsUnparseableStructure(t *testing.T) {
	if markdownSemanticValid([]byte("описание без заголовка")) {
		t.Fatal("Markdown without a document title must not expose semantic diff")
	}
	if markdownSemanticValid([]byte("# Документ\n\n```yaml\nkey: value")) {
		t.Fatal("Markdown with an unclosed fence must not expose semantic diff")
	}
	if !markdownSemanticValid([]byte("# Документ\n\nТекст.")) {
		t.Fatal("valid Markdown rejected")
	}
}

func changesHTTPServer(t *testing.T) (*documentationServer, string) {
	t.Helper()
	_, docs := newChangesRepository(t)
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\n<script>alert(1)</script>\n\nChanged.\n")
	return &documentationServer{options: Options{InputDirectory: docs}, stderr: io.Discard}, docs
}

func TestChangesHTTPAPIAndRenderedSafety(t *testing.T) {
	server, _ := changesHTTPServer(t)
	summary := httptest.NewRecorder()
	server.ServeHTTP(summary, httptest.NewRequest(http.MethodGet, changesAPIBase, nil))
	if summary.Code != http.StatusOK || summary.Header().Get("ETag") == "" || !strings.Contains(summary.Body.String(), `"schemaVersion":1`) {
		t.Fatalf("summary: %d %#v %s", summary.Code, summary.Header(), summary.Body.String())
	}
	rendered := httptest.NewRecorder()
	renderedURL := changesAPIBase + "/render?side=after&path=docs/modules/MOD-CORE.md"
	server.ServeHTTP(rendered, httptest.NewRequest(http.MethodGet, renderedURL, nil))
	if rendered.Code != http.StatusOK || strings.Contains(strings.ToLower(rendered.Body.String()), "<script") {
		t.Fatalf("unsafe render: %d %s", rendered.Code, rendered.Body.String())
	}
	content := httptest.NewRecorder()
	server.ServeHTTP(content, httptest.NewRequest(http.MethodGet, changesAPIBase+"/content?side=after&path=docs/modules/MOD-CORE.md", nil))
	if content.Code != http.StatusOK || content.Header().Get("Content-Security-Policy") == "" || content.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("content safety headers: %d %#v", content.Code, content.Header())
	}
	traversal := httptest.NewRecorder()
	server.ServeHTTP(traversal, httptest.NewRequest(http.MethodGet, changesAPIBase+"/content?side=after&path=../go.mod", nil))
	if traversal.Code != http.StatusNotFound {
		t.Fatalf("traversal status=%d body=%s", traversal.Code, traversal.Body.String())
	}
}

func TestChangesHTTPCacheUsesWorkspaceRevision(t *testing.T) {
	server, docs := changesHTTPServer(t)
	server.revision = "revision-one"
	request := httptest.NewRequest(http.MethodGet, changesAPIBase, nil)
	first, err := server.buildChanges(request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := server.buildChanges(request)
	if err != nil || first != second {
		t.Fatalf("report was not reused: err=%v", err)
	}
	writeChangesTestFile(t, filepath.Join(docs, "modules", "MOD-CORE.md"), "# MOD-CORE: Core\n\nChanged twice.\n")
	server.revision = "revision-two"
	third, err := server.buildChanges(request)
	if err != nil {
		t.Fatal(err)
	}
	if third == second || third.ChangeSetDigest == second.ChangeSetDigest {
		t.Fatalf("stale report reused: old=%s new=%s", second.ChangeSetDigest, third.ChangeSetDigest)
	}
}

func TestChangesUIAndMethodContract(t *testing.T) {
	server, _ := changesHTTPServer(t)
	page := httptest.NewRecorder()
	server.ServeHTTP(page, httptest.NewRequest(http.MethodGet, changesUIPath, nil))
	for _, marker := range []string{">Changes<", "data-changes-root", "/assets/changes.js", "/assets/codemirror.js"} {
		if !strings.Contains(page.Body.String(), marker) {
			t.Fatalf("changes UI missing %q", marker)
		}
	}
	workspace, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "features", "changes", "workspace.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	detail, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "features", "changes", "detail.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	discussions, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "features", "discussions", "components.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"data-file-list", "data-branch-base", "data-target-revision", "data-review-composer"} {
		if !strings.Contains(string(workspace)+string(detail)+string(discussions), marker) {
			t.Fatalf("Changes React source missing %q", marker)
		}
	}
	method := httptest.NewRecorder()
	server.ServeHTTP(method, httptest.NewRequest(http.MethodPost, changesAPIBase, nil))
	if method.Code != http.StatusMethodNotAllowed || method.Header().Get("Allow") == "" {
		t.Fatalf("method contract: %d %#v", method.Code, method.Header())
	}
}

func TestChangesHTTPAcceptsBranchBase(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, changesAPIBase+"?base=HEAD&target=HEAD&branchBase=main", nil)
	options := changesOptionsFromRequest(Options{}, request)
	if options.ChangeBranchBase != "main" || options.ChangeBase != "HEAD" || options.ChangeTarget != "HEAD" {
		t.Fatalf("options: %#v", options)
	}
}
