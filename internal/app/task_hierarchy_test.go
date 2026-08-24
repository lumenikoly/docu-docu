package toudocu

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskVerifyDoesNotRunChildCommands(t *testing.T) {
	root, docs, _ := createFixture(t)
	parentCommands := map[string]string{"AC-01": "parent-1", "AC-02": "parent-2", "ALL": "parent-all", "DOCS": "parent-docs"}
	childCommands := map[string]string{"AC-01": "child-1", "AC-02": "child-2", "ALL": "child-all", "DOCS": "child-docs"}
	parent := taskVerifyFixture("Готово к работе", false, parentCommands, "")
	child := strings.Replace(taskVerifyFixture("Готово к работе", false, childCommands, ""), "TASK-AUTH-020", "TASK-AUTH-021", 1)
	child = strings.Replace(child, "- Сценарий: UC-AUTH-01\n", "- Сценарий: UC-AUTH-01\n- Родительская задача: TASK-AUTH-020\n", 1)
	writeTestFile(t, docs, "work/TASK-AUTH-020.md", parent)
	writeTestFile(t, docs, "work/TASK-AUTH-021.md", child)
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeCommandRunner{outcomes: map[string]fakeCommandOutcome{}}
	report := executeTaskVerify(model, Options{TaskID: "TASK-AUTH-020", VerifyMode: "run", Timeout: time.Second}, io.Discard, io.Discard, runner)
	if report.Status != "passed" {
		t.Fatalf("parent verification failed: %#v", report)
	}
	for _, command := range runner.commands {
		if strings.HasPrefix(command, "child-") {
			t.Fatalf("child command executed: %s", command)
		}
	}
}

func TestTaskInitWithParent(t *testing.T) {
	model, docs := hierarchyModel(t, map[string]string{
		"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "", ""),
	})
	report, err := InitTask(Options{InputDirectory: docs, RepositoryRoot: model.RepositoryRoot, Area: "AUTH", Title: "Child", TaskType: "Maintenance", Language: "ru", ParentTaskID: "TASK-AUTH-100"})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(docs, filepath.FromSlash(report.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if report.ParentID == nil || *report.ParentID != "TASK-AUTH-100" || !strings.Contains(string(content), "parentTask: TASK-AUTH-100") {
		t.Fatalf("parent missing from scaffold/report: %#v\n%s", report, content)
	}
	if _, err := InitTask(Options{InputDirectory: docs, RepositoryRoot: model.RepositoryRoot, Area: "AUTH", Title: "Bug", TaskType: "Bug", Language: "en", ParentTaskID: "TASK-AUTH-100"}); err == nil {
		t.Fatal("BUG-* accepted --parent")
	}
}

func hierarchyTask(id, status, parent, dependencies string) string {
	metadata := ""
	if parent != "" {
		metadata += "- Parent: " + parent + "\n"
	}
	if dependencies != "" {
		metadata += "- Dependencies: " + dependencies + "\n"
	}
	return "# " + id + ": " + id + " title\n\n- Status: " + status + "\n- Type: Maintenance\n" + metadata + "\n## Result\n\nObservable result.\n"
}

func hierarchyModel(t *testing.T, files map[string]string) (*Model, string) {
	t.Helper()
	root, docs, _ := createFixture(t)
	for path, content := range files {
		writeTestFile(t, docs, path, content)
	}
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: root, StaleDays: 0})
	if err != nil {
		t.Fatal(err)
	}
	return model, docs
}

func hasIssue(model *Model, code string) bool {
	for _, issue := range model.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestTaskHierarchyBuildsComputedChildrenAndCompatibleJSON(t *testing.T) {
	model, _ := hierarchyModel(t, map[string]string{
		"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "", ""),
		"work/TASK-AUTH-101.md": strings.Replace(hierarchyTask("TASK-AUTH-101", "Draft", "", ""), "- Type: Maintenance\n", "- Type: Maintenance\n- Родительская задача: TASK-AUTH-100\n", 1),
		"work/TASK-AUTH-102.md": hierarchyTask("TASK-AUTH-102", "Draft", "TASK-AUTH-100", "TASK-AUTH-101"),
		"work/TASK-AUTH-111.md": hierarchyTask("TASK-AUTH-111", "Draft", "TASK-AUTH-101", ""),
	})
	root, _ := findWorkItem(model, "TASK-AUTH-100")
	middle, _ := findWorkItem(model, "TASK-AUTH-101")
	if root.ParentID != nil || strings.Join(root.ChildIDs, ",") != "TASK-AUTH-101,TASK-AUTH-102" || middle.ParentID == nil || *middle.ParentID != root.ID || strings.Join(middle.ChildIDs, ",") != "TASK-AUTH-111" {
		t.Fatalf("unexpected hierarchy: root=%#v middle=%#v", root, middle)
	}
	data, err := json.Marshal(BuildReport(model))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"schemaVersion":1`, `"parentId":null`, `"childIds":["TASK-AUTH-101","TASK-AUTH-102"]`} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("project report missing %s: %s", expected, data)
		}
	}
}

func TestTaskHierarchyDiagnostics(t *testing.T) {
	bugParent := strings.Replace(completeTaskFixture("Draft"), "TASK-AUTH-021", "BUG-AUTH-999", 1)
	bugParent = strings.Replace(bugParent, "- Type: Feature", "- Type: Bug", 1)
	tests := []struct {
		name  string
		files map[string]string
		code  string
	}{
		{"unknown", map[string]string{"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-999", "")}, "TASK_PARENT_UNKNOWN"},
		{"self", map[string]string{"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-100", "")}, "TASK_PARENT_SELF"},
		{"invalid", map[string]string{"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-101, TASK-AUTH-102", "")}, "TASK_PARENT_INVALID"},
		{"unsupported-type", map[string]string{
			"work/BUG-AUTH-999.md":  bugParent,
			"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "BUG-AUTH-999", ""),
		}, "TASK_PARENT_TYPE_UNSUPPORTED"},
		{"two-node-cycle", map[string]string{
			"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-101", ""),
			"work/TASK-AUTH-101.md": hierarchyTask("TASK-AUTH-101", "Draft", "TASK-AUTH-100", ""),
		}, "TASK_PARENT_CYCLE"},
		{"multi-level-cycle", map[string]string{
			"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-102", ""),
			"work/TASK-AUTH-101.md": hierarchyTask("TASK-AUTH-101", "Draft", "TASK-AUTH-100", ""),
			"work/TASK-AUTH-102.md": hierarchyTask("TASK-AUTH-102", "Draft", "TASK-AUTH-101", ""),
		}, "TASK_PARENT_CYCLE"},
		{"combined-cycle", map[string]string{
			"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "", ""),
			"work/TASK-AUTH-101.md": hierarchyTask("TASK-AUTH-101", "Draft", "TASK-AUTH-100", "TASK-AUTH-100"),
		}, "TASK_COMPLETION_CYCLE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model, _ := hierarchyModel(t, test.files)
			if !hasIssue(model, test.code) {
				t.Fatalf("missing %s: %#v", test.code, model.Issues)
			}
		})
	}
	model, _ := hierarchyModel(t, map[string]string{"work/TASK-AUTH-100.md": hierarchyTask("TASK-AUTH-100", "Draft", "TASK-AUTH-999", "")})
	parentLine := fileLineContaining(t, filepath.Join(model.RootDirectory, "work", "TASK-AUTH-100.md"), "parentTask:")
	for _, issue := range model.Issues {
		if issue.Code == "TASK_PARENT_UNKNOWN" && (issue.Line != parentLine || issue.TaskID != "TASK-AUTH-100" || issue.RelatedID != "TASK-AUTH-999") {
			t.Fatalf("diagnostic location/IDs lost: %#v", issue)
		}
	}
}

func TestTaskHierarchyAllowsDependencyAcrossTrees(t *testing.T) {
	model, _ := hierarchyModel(t, map[string]string{
		"work/TASK-AUTH-100.md":    hierarchyTask("TASK-AUTH-100", "Draft", "", ""),
		"work/TASK-AUTH-101.md":    hierarchyTask("TASK-AUTH-101", "Draft", "TASK-AUTH-100", "TASK-BILLING-201"),
		"work/TASK-BILLING-200.md": hierarchyTask("TASK-BILLING-200", "Draft", "", ""),
		"work/TASK-BILLING-201.md": hierarchyTask("TASK-BILLING-201", "Draft", "TASK-BILLING-200", ""),
	})
	child, err := findWorkItem(model, "TASK-AUTH-101")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(child.DependsOn, ",") != "TASK-BILLING-201" || hasIssue(model, "TASK_COMPLETION_CYCLE") {
		t.Fatalf("cross-tree dependency rejected: child=%#v issues=%#v", child, model.Issues)
	}
}

func TestTaskHierarchyLifecycle(t *testing.T) {
	parent := strings.Replace(terminalTaskFixture("Done"), "TASK-AUTH-021", "TASK-AUTH-100", 1)
	readyChild := strings.Replace(completeTaskFixture("Ready"), "TASK-AUTH-021", "TASK-AUTH-101", 1)
	readyChild = strings.Replace(readyChild, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	model, _ := hierarchyModel(t, map[string]string{"work/TASK-AUTH-100.md": parent, "work/TASK-AUTH-101.md": readyChild})
	if !hasIssue(model, "TASK_CHILD_INCOMPLETE") {
		t.Fatalf("Done parent accepted incomplete child: %#v", model.Issues)
	}
	ready := BuildTaskReady(model, "TASK-AUTH-101", false)
	if ready.Status == "ready" || !reportHasIssue(ready.Issues, "TASK_CHILD_INCOMPLETE") {
		t.Fatalf("task ready missed done-parent lifecycle error: %#v", ready)
	}

	cancelled := strings.Replace(terminalTaskFixture("Cancelled"), "TASK-AUTH-021", "TASK-AUTH-100", 1)
	model, _ = hierarchyModel(t, map[string]string{"work/TASK-AUTH-100.md": cancelled, "work/TASK-AUTH-101.md": readyChild})
	if !hasIssue(model, "TASK_CANCELLED_PARENT_ACTIVE_CHILD") {
		t.Fatalf("Cancelled parent accepted active child: %#v", model.Issues)
	}
	ready = BuildTaskReady(model, "TASK-AUTH-101", false)
	if ready.Status == "ready" || !reportHasIssue(ready.Issues, "TASK_CANCELLED_PARENT_ACTIVE_CHILD") {
		t.Fatalf("task ready missed cancelled-parent lifecycle error: %#v", ready)
	}

	cancelledChild := strings.Replace(terminalTaskFixture("Cancelled"), "TASK-AUTH-021", "TASK-AUTH-101", 1)
	cancelledChild = strings.Replace(cancelledChild, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	model, _ = hierarchyModel(t, map[string]string{"work/TASK-AUTH-100.md": parent, "work/TASK-AUTH-101.md": cancelledChild})
	if !hasIssue(model, "TASK_CHILD_INCOMPLETE") {
		t.Fatalf("Cancelled child counted as completed: %#v", model.Issues)
	}
}

func reportHasIssue(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestTaskHierarchyIncludesArchivedDoneChild(t *testing.T) {
	parent := strings.Replace(terminalTaskFixture("Done"), "TASK-AUTH-021", "TASK-AUTH-100", 1)
	child := strings.Replace(terminalTaskFixture("Done"), "TASK-AUTH-021", "TASK-AUTH-101", 1)
	child = strings.Replace(child, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	model, _ := hierarchyModel(t, map[string]string{"work/TASK-AUTH-100.md": parent, "work/archive/2026/TASK-AUTH-101.md": child})
	root, _ := findWorkItem(model, "TASK-AUTH-100")
	if strings.Join(root.ChildIDs, ",") != "TASK-AUTH-101" || hasIssue(model, "TASK_CHILD_INCOMPLETE") {
		t.Fatalf("archived Done child not counted: %#v %#v", root, model.Issues)
	}
}

func TestTaskTreeContextAndPortalUseSharedHierarchy(t *testing.T) {
	rootTask := strings.Replace(completeTaskFixture("Ready"), "TASK-AUTH-021", "TASK-AUTH-100", 1)
	child := strings.Replace(completeTaskFixture("In progress"), "TASK-AUTH-021", "TASK-AUTH-101", 1)
	child = strings.Replace(child, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	secondChild := strings.Replace(completeTaskFixture("Ready"), "TASK-AUTH-021", "TASK-AUTH-102", 1)
	secondChild = strings.Replace(secondChild, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	grandchild := strings.Replace(completeTaskFixture("Draft"), "TASK-AUTH-021", "TASK-AUTH-111", 1)
	grandchild = strings.Replace(grandchild, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-101\n", 1)
	model, _ := hierarchyModel(t, map[string]string{
		"work/TASK-AUTH-100.md": rootTask,
		"work/TASK-AUTH-101.md": child,
		"work/TASK-AUTH-102.md": secondChild,
		"work/TASK-AUTH-111.md": grandchild,
	})

	tree, err := BuildTaskTree(model, "TASK-AUTH-100")
	if err != nil || tree.Tree.Status != "ready" || len(tree.Tree.Children) != 2 || tree.Tree.Children[0].ID != "TASK-AUTH-101" || tree.Tree.Children[1].ID != "TASK-AUTH-102" || len(tree.Tree.Children[0].Children) != 1 || tree.Tree.Children[0].Children[0].ID != "TASK-AUTH-111" || tree.Tree.Children[0].Children[0].Children == nil {
		t.Fatalf("tree=%#v err=%v", tree, err)
	}
	var stdout, stderr strings.Builder
	if code := RunCLI([]string{"task", "tree", "TASK-AUTH-100", model.RootDirectory}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "├── TASK-AUTH-101  in-progress") || !strings.Contains(stdout.String(), "│   └── TASK-AUTH-111  draft") || !strings.Contains(stdout.String(), "└── TASK-AUTH-102  ready") {
		t.Fatalf("task tree text failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := RunCLI([]string{"task", "tree", "TASK-AUTH-100", model.RootDirectory, "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("task tree JSON failed: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var decoded TaskTreeReport
	if err := json.Unmarshal([]byte(stdout.String()), &decoded); err != nil || decoded.SchemaVersion != 1 || decoded.TaskID != "TASK-AUTH-100" || decoded.Tree.Status != "ready" || len(decoded.Tree.Children) != 2 || len(decoded.Tree.Children[0].Children) != 1 || decoded.Tree.Children[0].Children[0].Children == nil {
		t.Fatalf("task tree JSON contract incomplete: report=%#v err=%v", decoded, err)
	}
	context, err := BuildTaskContext(model, "TASK-AUTH-101")
	if err != nil || context.Hierarchy.Parent == nil || context.Hierarchy.Parent.ID != "TASK-AUTH-100" || len(context.Documents) > 4 {
		t.Fatalf("context hierarchy is not compact: %#v err=%v", context, err)
	}
	rootContext, err := BuildTaskContext(model, "TASK-AUTH-100")
	if err != nil || len(rootContext.Hierarchy.Children) != 2 || rootContext.Hierarchy.Descendants.Total != 3 || rootContext.Hierarchy.Descendants.Ready != 1 || rootContext.Hierarchy.Descendants.InProgress != 1 || rootContext.Hierarchy.Descendants.Draft != 1 {
		t.Fatalf("root context summary missing: %#v err=%v", rootContext.Hierarchy, err)
	}
	for _, document := range rootContext.Documents {
		if document.Path == "work/TASK-AUTH-101.md" {
			t.Fatal("root context included the full child work item")
		}
	}
	model.SiteConfig.Project.Locale = "ru"
	html := renderDocumentPage(model, model.DocByPath["work/TASK-AUTH-101.md"])
	for _, expected := range []string{"task-decomposition", "TASK-AUTH-100", "task-hierarchy-breadcrumb"} {
		if !strings.Contains(html, expected) {
			t.Fatalf("portal missing %q", expected)
		}
	}
	rootHTML := renderDocumentPage(model, model.DocByPath["work/TASK-AUTH-100.md"])
	for _, expected := range []string{"task-decomposition", "task-tree", "TASK-AUTH-100", "TASK-AUTH-101", "TASK-AUTH-102", "TASK-AUTH-111", "status-in-progress", "status-not-started", "В работе", "Не начато"} {
		if !strings.Contains(rootHTML, expected) {
			t.Fatalf("parent portal tree missing %q", expected)
		}
	}
	if strings.Index(rootHTML, "TASK-AUTH-101") >= strings.Index(rootHTML, "TASK-AUTH-102") {
		t.Fatal("parent portal tree did not preserve natural task ID order")
	}
	navigation := renderNavigation(model, "work/TASK-AUTH-111.html")
	for _, expected := range []string{`nav-task-folder`, `data-nav-folder="task-task-auth-100"`, `data-nav-folder="task-task-auth-101"`, `aria-label="Свернуть подзадачи Add verification workflow"`, "TASK-AUTH-100", "TASK-AUTH-101", "TASK-AUTH-102", "TASK-AUTH-111"} {
		if !strings.Contains(navigation, expected) {
			t.Fatalf("task navigation missing %q: %s", expected, navigation)
		}
	}
	if strings.Index(navigation, "TASK-AUTH-101") >= strings.Index(navigation, "TASK-AUTH-102") {
		t.Fatal("task navigation did not preserve child order")
	}
	childHTML := renderDocumentPage(model, model.DocByPath["work/TASK-AUTH-101.md"])
	treeStart := strings.Index(childHTML, `<ul class="task-tree"`)
	if treeStart < 0 {
		t.Fatal("nested parent portal page omitted its task tree")
	}
	treeEnd := strings.Index(childHTML[treeStart:], `</section>`)
	if treeEnd < 0 {
		t.Fatal("nested parent portal page omitted its hierarchy boundary")
	}
	childTree := childHTML[treeStart : treeStart+treeEnd]
	if !strings.Contains(childTree, "TASK-AUTH-111") || strings.Contains(childTree, "TASK-AUTH-102") {
		t.Fatal("nested parent portal page did not render only its own subtree")
	}
	deepHTML := renderDocumentPage(model, model.DocByPath["work/TASK-AUTH-111.md"])
	if !strings.Contains(deepHTML, "TASK-AUTH-100") || !strings.Contains(deepHTML, "TASK-AUTH-101") || !strings.Contains(deepHTML, "task-hierarchy-breadcrumb") || strings.Contains(deepHTML, "class=\"task-tree\"") {
		t.Fatal("deep portal page did not render ancestor navigation")
	}
	model.serveMode = true
	serveHTML := renderDocumentPage(model, model.DocByPath["work/TASK-AUTH-101.md"])
	if !strings.Contains(serveHTML, "task-decomposition") || !strings.Contains(serveHTML, "TASK-AUTH-100") || !strings.Contains(serveHTML, "TASK-AUTH-111") {
		t.Fatal("serve portal did not use the shared hierarchy")
	}
}

func TestTaskContextTextHierarchy(t *testing.T) {
	report := TaskContextReport{
		Task: WorkItem{ID: "TASK-AUTH-101", Title: "Child", Document: "work/TASK-AUTH-101.md", Status: StatusFor("Ready")},
		Hierarchy: TaskHierarchy{
			Parent: &TaskHierarchyRef{ID: "TASK-AUTH-100", Title: "Parent", Status: "in-progress"},
			Ancestors: []TaskHierarchyRef{
				{ID: "TASK-AUTH-001", Title: "Program", Status: "in-progress"},
				{ID: "TASK-AUTH-100", Title: "Parent", Status: "in-progress"},
			},
			Children:    []TaskHierarchyRef{{ID: "TASK-AUTH-111", Title: "Blocked child", Status: "blocked", HasBlocker: true}},
			Descendants: TaskHierarchySummary{Total: 2, Ready: 1, Blocked: 1},
		},
		Dependencies: []WorkItem{}, Dependents: []WorkItem{}, Issues: []Issue{}, RequiredReads: []string{},
	}
	var output strings.Builder
	printTaskContextText(&output, report)
	for _, expected := range []string{
		"Ancestors: TASK-AUTH-001 — Program [in-progress; blocker: no] / TASK-AUTH-100 — Parent [in-progress; blocker: no]",
		"Parent task: TASK-AUTH-100 — Parent [in-progress; blocker: no]",
		"- TASK-AUTH-111 — Blocked child [blocked; blocker: yes]",
		"Descendants: total 2; ready: 1; blocked: 1",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("text hierarchy missing %q:\n%s", expected, output.String())
		}
	}
}
