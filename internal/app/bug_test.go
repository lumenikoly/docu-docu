package toudocu

import (
	"io"
	"strings"
	"testing"
)

func completeBugFixture(status string) string {
	criterion := "- [ ] `AC-01` A regression test protects the original session-expiration behavior."
	if status == "Done" {
		criterion = "- [x] `AC-01` A regression test protects the original session-expiration behavior."
	}
	return `# BUG-AUTH-021: Session does not expire

- Type: Bug
- Status: ` + status + `
- Severity: High
- Priority: High
- Reproducibility: Always
- Regression: Yes
- Observed in: 0.8.2
- Module: MOD-AUTH
- Use case: UC-AUTH-01
- Last updated: 2026-07-30

## Symptom

The account remains visible after session expiration.

## Expected behavior

The user is redirected to the login screen.

## Actual behavior

The request fails while the protected screen remains visible.

## Steps to reproduce

1. Sign in.
2. Wait for session expiration.
3. Send an authenticated request.

## Environment

- Version: 0.8.2

## Evidence

- API error: SESSION_EXPIRED

## Cause

The expiration handler clears the token but does not navigate to login.

## Scope

- ` + "`docs/index.md`" + `

## Out of scope

Changing the session lifetime.

## Plan

1. Reproduce the defect.
2. Add a failing regression test.
3. Fix the root cause.
4. Run focused and complete checks.
5. Review documentation impact.

## Acceptance criteria

` + criterion + `

## Verification

- ` + "`AC-01`" + ` -> ` + "`go test ./...`" + `
- ` + "`ALL`" + ` -> ` + "`go test ./...`" + `
- ` + "`DOCS`" + ` -> ` + "`go run ./cmd/toudocu check ./docs --strict`" + `

## Regression test

Covered by AC-01.

## Documentation impact

Not required: the fix restores documented behavior.
`
}

func issuesForDocument(model *Model, path string) []Issue {
	result := []Issue{}
	for _, issue := range model.Issues {
		if issue.DocumentPath == path {
			result = append(result, issue)
		}
	}
	return result
}

func bugIssueCodes(issues []Issue) string {
	codes := make([]string, 0, len(issues))
	for _, issue := range issues {
		codes = append(codes, issue.Code)
	}
	return strings.Join(codes, ",")
}

func TestBugWorkItemValidationAndPortalFilters(t *testing.T) {
	root, docs, _ := createFixture(t)
	writeTestFile(t, docs, "work/BUG-AUTH-021.md", completeBugFixture("Ready"))
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if issues := issuesForDocument(model, "work/BUG-AUTH-021.md"); len(issues) != 0 {
		t.Fatalf("valid bug issues: %s", bugIssueCodes(issues))
	}
	if len(model.Knowledge.WorkItems) != 1 {
		t.Fatalf("work items: %#v", model.Knowledge.WorkItems)
	}
	item := model.Knowledge.WorkItems[0]
	if item.ID != "BUG-AUTH-021" || item.Type != "bug" || item.Severity != "high" || item.Reproducibility != "always" || item.Regression != "true" {
		t.Fatalf("bug item: %#v", item)
	}
	context, err := BuildTaskContext(model, "BUG-AUTH-021")
	if err != nil || context.Task.ID != "BUG-AUTH-021" || context.UseCase == nil {
		t.Fatalf("bug context: %#v %v", context, err)
	}
	if model.Stats.OpenBugs != 1 || model.Stats.HighSeverityBugs != 1 || model.Stats.RegressionBugs != 1 {
		t.Fatalf("bug stats: %#v", model.Stats)
	}
	ready := BuildTaskReady(model, "BUG-AUTH-021", false)
	if ready.Status != "ready" || !ready.ContractComplete || !ready.ReadyForWork {
		t.Fatalf("valid bug readiness: %#v", ready)
	}
	verify := executeTaskVerify(model, Options{TaskID: "BUG-AUTH-021", VerifyMode: "dry-run"}, io.Discard, io.Discard, &fakeCommandRunner{})
	if verify.Status != "planned" || len(verify.ValidationIssues) != 0 {
		t.Fatalf("valid bug verification: %#v", verify)
	}
	catalog := renderDirectoryPage(model, "work")
	for _, expected := range []string{`data-task-workspace-item`, `data-workspace-filter="type"`, "High", "BUG-AUTH-021"} {
		if !strings.Contains(catalog, expected) {
			t.Fatalf("bug catalog missing %q", expected)
		}
	}
}

func TestBugValidationRejectsMissingContractAndWrongPrefix(t *testing.T) {
	root, docs, _ := createFixture(t)
	content := completeBugFixture("Done")
	content = strings.Replace(content, "BUG-AUTH-021", "TASK-AUTH-021", 1)
	content = strings.Replace(content, "- Severity: High\n", "", 1)
	content = strings.Replace(content, "- [x] `AC-01`", "- [ ] `AC-01`", 1)
	content = strings.Replace(content, "The expiration handler clears the token but does not navigate to login.", "Not established.", 1)
	content = strings.Replace(content, "A regression test protects the original session-expiration behavior.", "The redirect works.", 1)
	content = strings.Replace(content, "Not established.\n\n## Scope", "\n## Scope", 1)
	content = strings.Replace(content, "Covered by AC-01.", "", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", content)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	codes := bugIssueCodes(issuesForDocument(model, "work/TASK-AUTH-021.md"))
	for _, expected := range []string{"invalid-bug-id", "missing-bug-field", "missing-bug-regression-test", "missing-completed-bug-cause", "incomplete-completed-task"} {
		if !strings.Contains(codes, expected) {
			t.Fatalf("missing %s in %s", expected, codes)
		}
	}
}

func TestTechnicalBugMayExplainMissingUseCase(t *testing.T) {
	root, docs, _ := createFixture(t)
	content := strings.Replace(completeBugFixture("Ready"), "- Use case: UC-AUTH-01", "- Use case: Not applicable", 1)
	content += "\n## Relationship to user behavior\n\nNot applicable: resource cleanup is internal and does not change observable behavior.\n"
	writeTestFile(t, docs, "work/BUG-AUTH-021.md", content)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	if issues := issuesForDocument(model, "work/BUG-AUTH-021.md"); len(issues) != 0 {
		t.Fatalf("technical bug issues: %s", bugIssueCodes(issues))
	}
	ready := BuildTaskReady(model, "BUG-AUTH-021", false)
	if ready.Status != "ready" || !ready.ReadyForWork {
		t.Fatalf("technical bug readiness: %#v", ready)
	}
}
