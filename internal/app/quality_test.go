package toudocu

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQualityCanonicalMetadataAndValidationBoundaries(t *testing.T) {
	for _, value := range []string{"draft", "active", "obsolete", "superseded"} {
		if _, ok := standardStatus(value); !ok {
			t.Errorf("standard status %q was not recognized", value)
		}
	}
	for _, value := range []string{"draft", "active", "review-required", "obsolete"} {
		if _, ok := runbookStatus(value); !ok {
			t.Errorf("runbook status %q was not recognized", value)
		}
	}
	for _, value := range []string{"Черновик", "Draft", "Действует", "Active"} {
		if _, ok := standardStatus(value); ok {
			t.Errorf("localized standard status %q was recognized", value)
		}
	}
	parsed := analyzeMarkdown(`<!-- toudocu
scope: code
environment: production
risk: high
lastVerified: 2026-07-31
supersededBy: STD-NEXT-001
-->

# Entity
`)
	for key, expected := range map[string]string{
		"scope": "code", "environment": "production", "risk": "high",
		"lastVerified": "2026-07-31", "supersededBy": "STD-NEXT-001",
	} {
		if parsed.Metadata[key] != expected {
			t.Errorf("%s: got %q, want %q", key, parsed.Metadata[key], expected)
		}
	}

	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Project\n\nBoundaries.\n")
	writeTestFile(t, docs, "quality/index.md", "# Quality\n\nStandards.\n")
	writeTestFile(t, docs, "quality/STD-NEXT-001.md", validStandard("STD-NEXT-001"))
	superseded := strings.Replace(validStandard("STD-OLD-001"), "- Status: Active", "- Status: Superseded\n- Superseded by: STD-NEXT-001", 1)
	writeTestFile(t, docs, "quality/STD-OLD-001.md", superseded)
	writeTestFile(t, docs, "quality/STD-DUPLICATE.md", validStandard("STD-NEXT-001"))
	writeTestFile(t, docs, "quality/STD-WARN-001.md", "# STD-WARN-001: Incomplete\n\n- Identifier: STD-WARN-001\n- Status: Unknown\n\n## Rules\n\n## Automated checks\n")
	self := strings.Replace(validStandard("STD-SELF-001"), "- Status: Active", "- Status: Superseded\n- Superseded by: STD-SELF-001", 1)
	writeTestFile(t, docs, "quality/STD-SELF-001.md", self)
	dangling := strings.Replace(validStandard("STD-DANGLING-001"), "- Status: Active", "- Status: Superseded\n- Superseded by: STD-MISSING-001", 1)
	writeTestFile(t, docs, "quality/STD-DANGLING-001.md", dangling)
	writeTestFile(t, docs, "runbooks/index.md", "# Runbooks\n\nProcedures.\n")
	missingSections := "# RB-OPS-EMPTY: Empty\n\n- Identifier: RB-OPS-EMPTY\n- Status: Active\n- Environment: Production\n- Risk: Critical\n- Last verified: invalid\n\nNo procedure.\n"
	writeTestFile(t, docs, "runbooks/RB-OPS-EMPTY.md", missingSections)
	writeTestFile(t, docs, "runbooks/RB-INVALID.md", strings.Replace(validRunbook("RB-OPS-INVALID", "Active", "2026-07-20", "Low"), "RB-OPS-INVALID", "INVALID", 2))
	writeTestFile(t, docs, "runbooks/RB-DUPLICATE.md", validRunbook("RB-OPS-EMPTY", "Active", "2026-07-20", "Low"))
	writeTestFile(t, docs, "runbooks/RB-OPS-WARN.md", "# RB-OPS-WARN: Incomplete\n\n- Identifier: RB-OPS-WARN\n- Status: Unknown\n- Risk: Unknown\n- Last verified: 2026-07-20\n\n## Prerequisites\n\nKnown.\n\n## Procedure\n\n1. Act.\n\n## Verification\n\nVerify.\n\n## Rollback\n\nRollback.\n")
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	writeTestFile(t, docs, "runbooks/RB-OPS-BOUNDARY.md", validRunbook("RB-OPS-BOUNDARY", "Active", now.AddDate(0, 0, -90).Format("2006-01-02"), "Low"))
	writeTestFile(t, docs, "runbooks/RB-OPS-OVERDUE.md", validRunbook("RB-OPS-OVERDUE", "Active", now.AddDate(0, 0, -91).Format("2006-01-02"), "Low"))
	writeTestFile(t, docs, "runbooks/RB-OPS-FUTURE.md", validRunbook("RB-OPS-FUTURE", "Active", now.AddDate(0, 0, 1).Format("2006-01-02"), "Low"))
	writeTestFile(t, docs, "runbooks/RB-OPS-OBSOLETE.md", validRunbook("RB-OPS-OBSOLETE", "Obsolete", "2026-07-20", "Low"))
	writeTestFile(t, docs, "custom-bad/index.md", "# Incomplete custom\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 90, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"duplicate-standard-id", "missing-standard-scope",
		"invalid-standard-status", "invalid-standard-updated", "missing-standard-rules",
		"missing-standard-automatic-checks", "self-standard-replacement", "dangling-standard-replacement",
		"invalid-runbook-id", "duplicate-runbook-id",
		"missing-runbook-environment", "invalid-runbook-status", "invalid-runbook-risk",
		"missing-runbook-section", "runbook-procedure-not-numbered",
		"missing-runbook-stop-conditions", "runbook-review-required", "stale-runbook",
		"missing-custom-description",
	} {
		if !hasIssueCode(model.Issues, code) {
			t.Errorf("missing validation code %s: %#v", code, model.Issues)
		}
	}
	freshness := map[string]string{}
	for _, runbook := range model.Knowledge.Runbooks {
		freshness[runbook.ID] = runbook.Freshness
	}
	if freshness["RB-OPS-BOUNDARY"] != "recent" || freshness["RB-OPS-OVERDUE"] != "overdue" || freshness["RB-OPS-FUTURE"] != "review-required" {
		t.Fatalf("boundary freshness: %#v", freshness)
	}
	if freshness["RB-OPS-OBSOLETE"] != "not-applicable" {
		t.Fatalf("obsolete runbook freshness: %#v", freshness)
	}
}

func validStandard(id string) string {
	return "# " + id + `: Go quality

- Identifier: ` + id + `
- Status: Active
- Scope: Go sources
- Last updated: 2026-07-01

Rules used by contributors.

## Rules

Use gofmt and keep the dependency surface small.

## Automated checks

Run ` + "`go test ./...`" + `.
`
}

func validRunbook(id, status, verified, risk string) string {
	return "# " + id + `: Recover service

- Identifier: ` + id + `
- Status: ` + status + `
- Environment: Production
- Risk: ` + risk + `
- Last verified: ` + verified + `

Recover the service after a failed deployment.

## Prerequisites

Confirm access and the incident identifier.

## Procedure

1. Stop the rollout.
2. Restore the previous release.

## Verification

Confirm the health endpoint.

## Rollback

Resume the previous release.

## Stop conditions

Stop if data integrity cannot be confirmed.
`
}

func TestStandardsRunbooksAndFreshness(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Project\n\nTyped knowledge.\n")
	writeTestFile(t, docs, "quality/index.md", "# Quality\n\nProject standards.\n")
	writeTestFile(t, docs, "quality/STD-GO-001.md", validStandard("STD-GO-001"))
	writeTestFile(t, docs, "runbooks/index.md", "# Operations\n\nOperational procedures.\n")
	writeTestFile(t, docs, "runbooks/RB-OPS-001.md", validRunbook("RB-OPS-001", "Active", "2026-07-20", "High"))
	writeTestFile(t, docs, "runbooks/RB-OPS-002.md", validRunbook("RB-OPS-002", "Active", "2026-01-01", "Low"))
	writeTestFile(t, docs, "runbooks/RB-OPS-003.md", validRunbook("RB-OPS-003", "Requires review", "2026-07-20", "Medium"))

	model, err := BuildDocumentationModel(Options{
		InputDirectory: docs, RepositoryRoot: root, StaleDays: 90,
		Now: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Knowledge.Standards) != 1 || len(model.Knowledge.Runbooks) != 3 {
		t.Fatalf("typed collections: %#v", model.Knowledge)
	}
	if model.Stats.RunbooksTotal != 3 || model.Stats.RunbooksRecent != 1 ||
		model.Stats.RunbooksOverdue != 1 || model.Stats.RunbooksReviewRequired != 1 {
		t.Fatalf("freshness stats: %#v", model.Stats)
	}
	if !hasIssueCode(model.Issues, "stale-runbook") || !hasIssueCode(model.Issues, "runbook-review-required") {
		t.Fatalf("freshness warnings missing: %#v", model.Issues)
	}

	disabled, err := BuildDocumentationModel(Options{
		InputDirectory: docs, RepositoryRoot: root, StaleDays: 0,
		Now: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Stats.RunbooksOverdue != 0 || disabled.Stats.RunbooksRecent != 2 {
		t.Fatalf("stale-days=0 did not disable age-based overdue: %#v", disabled.Stats)
	}
}

func TestTypedKnowledgeErrorsAndCustomManifest(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Project\n\nTyped knowledge.\n")
	writeTestFile(t, docs, "quality/index.md", "# Quality\n\nStandards.\n")
	writeTestFile(t, docs, "quality/STD-BAD.md", "# Bad\n\n- Identifier: std-bad\n- Status: Superseded\n\n## Rules\n\nRule.\n\n## Automated checks\n\nCheck.\n")
	writeTestFile(t, docs, "runbooks/index.md", "# Runbooks\n\nOperational procedures.\n")
	writeTestFile(t, docs, "runbooks/RB-BAD-001.md", validRunbook("RB-BAD-001", "Active", "2026-07-20", "Low")+"\n[Missing target](missing.md)\n")
	writeTestFile(t, docs, "handbook/index.md", "# Team handbook\n\nTeam-specific guidance.\n")
	writeTestFile(t, docs, "handbook/start.md", "# Start\n\nRead this first.\n")
	writeTestFile(t, docs, "misc/note.md", "# Note\n\nNo manifest exists.\n")

	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssueCode(model.Issues, "invalid-standard-id") ||
		!hasIssueCode(model.Issues, "missing-standard-superseded-by") ||
		!hasIssueCode(model.Issues, "missing-section-manifest") ||
		!hasIssueCode(model.Issues, "invalid-runbook-link") {
		t.Fatalf("expected diagnostics missing: %#v", model.Issues)
	}
	for _, code := range []string{"invalid-custom-manifest-type", "missing-custom-description"} {
		if hasIssueCode(model.Issues, code) {
			t.Fatalf("valid custom manifest received %s: %#v", code, model.Issues)
		}
	}
}

func TestQualityTaskContextAndConditionalVerification(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "quality/index.md", "# Quality\n\nProject standards.\n")
	writeTestFile(t, docs, "quality/STD-GO-001.md", validStandard("STD-GO-001"))
	writeTestFile(t, docs, "runbooks/index.md", "# Runbooks\n\nOperational procedures.\n")
	writeTestFile(t, docs, "runbooks/RB-OPS-001.md", validRunbook("RB-OPS-001", "Active", "2026-07-20", "Low"))
	task := strings.Replace(completeTaskFixture("Ready"), "- Use case: UC-AUTH-01", "- Use case: UC-AUTH-01\n- Standards: STD-GO-001\n- Affected runbooks: RB-OPS-001", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", task)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0, Now: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	ready := BuildTaskReady(model, "TASK-AUTH-021", false)
	if !hasIssueCode(ready.Issues, "missing-verification-target") {
		t.Fatalf("QUALITY was not required: %#v", ready)
	}
	task = strings.Replace(task, "- `DOCS` -> `go version`", "- `DOCS` -> `go version`\n- `QUALITY` -> `go version`", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", task)
	model, err = BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0, Now: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	context, err := BuildTaskContext(model, "TASK-AUTH-021")
	if err != nil {
		t.Fatal(err)
	}
	if !context.FullVerification || len(context.Standards) != 1 || len(context.Runbooks) != 1 ||
		!containsString(context.RequiredReads, "quality/STD-GO-001.md") || !containsString(context.RequiredReads, "runbooks/RB-OPS-001.md") {
		t.Fatalf("typed task context: %#v", context)
	}
}

func TestQualityDanglingReferencesAndAdditiveJSON(t *testing.T) {
	root, docs, _ := createFixture(t)
	task := strings.Replace(completeTaskFixture("Ready"), "- Use case: UC-AUTH-01", "- Use case: UC-AUTH-01\n- Standards: STD-MISSING-001\n- Affected runbooks: RB-MISSING-001", 1)
	task = strings.Replace(task, "- `DOCS` -> `go version`", "- `DOCS` -> `go version`\n- `QUALITY` -> `go version`", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", task)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if !hasIssueCode(model.Issues, "dangling-standard-reference") || !hasIssueCode(model.Issues, "dangling-runbook-reference") {
		t.Fatalf("dangling typed references were accepted: %#v", model.Issues)
	}

	minimalRoot := t.TempDir()
	minimalDocs := filepath.Join(minimalRoot, "docs")
	writeTestFile(t, minimalDocs, "index.md", "# Minimal\n\nOnly an index.\n")
	minimal, err := BuildDocumentationModel(Options{InputDirectory: minimalDocs, RepositoryRoot: minimalRoot, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(BuildReport(minimal))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"standards":[]`, `"runbooks":[]`, `"runbooksTotal":0`, `"runbooksRecent":0`, `"runbooksReviewRequired":0`, `"runbooksOverdue":0`} {
		if !bytes.Contains(data, []byte(expected)) {
			t.Fatalf("additive empty collection missing %s: %s", expected, data)
		}
	}
	reportData, err := json.Marshal(BuildReport(model))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"standardIds":["STD-MISSING-001"]`, `"runbookIds":["RB-MISSING-001"]`} {
		if !bytes.Contains(reportData, []byte(expected)) {
			t.Fatalf("work item typed ID collection missing %s: %s", expected, reportData)
		}
	}
	context, err := BuildTaskContext(model, "TASK-AUTH-021")
	if err != nil {
		t.Fatal(err)
	}
	contextData, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"standards":[]`, `"runbooks":[]`} {
		if !bytes.Contains(contextData, []byte(expected)) {
			t.Fatalf("task context empty typed collection missing %s: %s", expected, contextData)
		}
	}
}

func TestStandardAndRunbookScaffoldsAndCatalogs(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	writeTestFile(t, docs, "index.md", "# Project\n\nScaffolds.\n")
	for _, test := range []struct {
		kind, id, language string
	}{
		{"standard", "STD-GO-001", "ru"},
		{"standard", "STD-DOCS-001", "en"},
		{"runbook", "RB-OPS-002", "ru"},
		{"runbook", "RB-OPS-001", "en"},
	} {
		report, err := Scaffold(Options{InputDirectory: docs, EntityKind: test.kind, EntityID: test.id, Title: "Title", Language: test.language, Now: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(docs, filepath.FromSlash(report.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "Last verified:") || strings.Contains(string(data), "Последняя проверка:") {
			t.Fatalf("scaffold invented review date:\n%s", data)
		}
		if _, err := Scaffold(Options{InputDirectory: docs, EntityKind: test.kind, EntityID: test.id, Title: "Overwrite", Language: test.language}); err == nil {
			t.Fatal("scaffold overwrote an existing file")
		}
	}
	if _, _, _, err := ParseArguments([]string{"scaffold", "standard", "STD-CLI-001", docs, "--title", "CLI", "--lang", "en"}); err != nil {
		t.Fatalf("standard CLI parsing: %v", err)
	}
	if _, _, _, err := ParseArguments([]string{"scaffold", "runbook", "RB-CLI-001", docs, "--title", "CLI", "--lang", "ru"}); err != nil {
		t.Fatalf("runbook CLI parsing: %v", err)
	}
	if _, _, _, err := ParseArguments([]string{"scaffold", "runbook", "STD-WRONG-001", docs, "--title", "Wrong"}); err == nil {
		t.Fatal("CLI accepted a runbook with an invalid prefix")
	}
	for _, args := range [][]string{
		{"scaffold", "standard", "STD-CLI-001", docs, "--title", "CLI standard", "--lang", "en", "--format", "json"},
		{"scaffold", "runbook", "RB-CLI-001", docs, "--title", "CLI runbook", "--lang", "ru", "--format", "json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := RunCLI(args, &stdout, &stderr); code != 0 {
			t.Fatalf("CLI scaffold failed for %v: %s", args, stderr.String())
		}
		var report ScaffoldReport
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.ID != args[2] {
			t.Fatalf("CLI scaffold report for %v: %#v %v", args, report, err)
		}
	}
	failureRoot := t.TempDir()
	failureDocs := filepath.Join(failureRoot, "docs")
	if err := os.MkdirAll(failureDocs, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, failureDocs, "quality", "not a directory")
	if _, err := Scaffold(Options{InputDirectory: failureDocs, EntityKind: "standard", EntityID: "STD-FAIL-001", Title: "Fail", Language: "en"}); err == nil {
		t.Fatal("scaffold succeeded when the parent path was a file")
	}

	writeTestFile(t, docs, "quality/index.md", "# Engineering quality\n\nStandards catalog.\n")
	writeTestFile(t, docs, "quality/STD-GO-001.md", validStandard("STD-GO-001"))
	writeTestFile(t, docs, "runbooks/index.md", "# Operations\n\nRunbook catalog.\n")
	writeTestFile(t, docs, "runbooks/RB-OPS-001.md", validRunbook("RB-OPS-001", "Active", "2026-07-20", "Low"))
	writeTestFile(t, docs, "handbook/index.md", "# Team handbook\n\n- Type: Custom\n\nTeam guidance.\n")
	writeTestFile(t, docs, "handbook/start.md", "# Start\n\nStart here.\n")
	writeTestFile(t, docs, "modules/core.md", "# MOD-CORE: Core\n\n- Identifier: MOD-CORE\n- Status: Planned\n\nCore module.\n")
	writeTestFile(t, docs, "use-cases/core.md", "# UC-CORE-01: Continue\n\n- Identifier: UC-CORE-01\n- Status: Planned\n- Module: MOD-CORE\n\nContinue.\n")
	writeTestFile(t, docs, "flows/core.md", "# FLOW-CORE-01: Continue\n\n- Identifier: FLOW-CORE-01\n- Scenario: UC-CORE-01\n- Module: MOD-CORE\n\n## Process\n\n```mermaid\nflowchart TD\n  A --> B\n```\n\n[Use case](../use-cases/core.md)\n")
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 90, Now: time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "site")
	if _, err := GenerateSite(model, Options{OutputDirectory: output}); err != nil {
		t.Fatal(err)
	}
	quality, _ := os.ReadFile(filepath.Join(output, "quality", "index.html"))
	runbooks, _ := os.ReadFile(filepath.Join(output, "runbooks", "index.html"))
	rootPage, _ := os.ReadFile(filepath.Join(output, "index.html"))
	if !strings.Contains(string(quality), "Quality") || !strings.Contains(string(quality), "STD-GO-001") {
		t.Fatalf("quality catalog missing content")
	}
	if !strings.Contains(string(quality), `data-filter-control="status"`) {
		t.Fatalf("quality catalog status filter missing")
	}
	for _, expected := range []string{"Total", "Recent", "Review required", "Overdue", `data-filter-control="freshness"`} {
		if !strings.Contains(string(runbooks), expected) {
			t.Fatalf("runbook catalog missing %q", expected)
		}
	}
	if !strings.Contains(string(rootPage), "Quality") || !strings.Contains(string(rootPage), "Runbooks") {
		t.Fatalf("built-in fallback titles missing from navigation")
	}
	if !strings.Contains(string(rootPage), "Team handbook") || !strings.Contains(string(rootPage), "processes/index.html") {
		t.Fatalf("custom H1 or processes route missing from navigation")
	}
	if strings.Index(string(rootPage), "Quality") > strings.Index(string(rootPage), "Runbooks") {
		t.Fatalf("quality must precede runbooks in navigation")
	}
	processes, err := os.ReadFile(filepath.Join(output, "processes", "index.html"))
	if err != nil || !strings.Contains(string(processes), "FLOW-CORE-01") {
		t.Fatalf("processes catalog was not preserved: %v", err)
	}
	app, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "core", "portal.ts"))
	if err != nil || !strings.Contains(string(app), "item.dataset[key]") {
		t.Fatalf("generic catalog filter behavior is missing: %v", err)
	}
}
