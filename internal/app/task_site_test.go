package toudocu

import (
	"strings"
	"testing"
)

func TestTaskWorkspaceState(t *testing.T) {
	cases := []struct {
		name, want string
		item       WorkItem
		readiness  taskReadinessSummary
	}{
		{"in progress", "in-progress", WorkItem{statusName: WorkItemInProgress}, taskReadinessSummary{}},
		{"blocked", "blocked", WorkItem{statusName: WorkItemBlocked}, taskReadinessSummary{}},
		{"draft", "draft", WorkItem{statusName: WorkItemDraft}, taskReadinessSummary{}},
		{"ready", "ready", WorkItem{statusName: WorkItemReady}, taskReadinessSummary{ContractComplete: true, DependenciesSatisfied: true}},
		{"waiting", "waiting", WorkItem{statusName: WorkItemReady}, taskReadinessSummary{ContractComplete: true}},
		{"needs attention", "needs-attention", WorkItem{statusName: WorkItemReady}, taskReadinessSummary{DependenciesSatisfied: true}},
		{"done", "done", WorkItem{statusName: WorkItemDone}, taskReadinessSummary{}},
		{"cancelled", "cancelled", WorkItem{statusName: WorkItemCancelled}, taskReadinessSummary{}},
		{"archive wins", "archive", WorkItem{statusName: WorkItemReady, Archived: true}, taskReadinessSummary{}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := taskWorkspaceState(&test.item, test.readiness); got != test.want {
				t.Fatalf("state = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTaskWorkspacePriorityOrder(t *testing.T) {
	for priority, want := range map[string]int{"urgent": 0, "high": 1, "normal": 2, "low": 3, "": 4, "custom": 5} {
		if got := taskWorkspacePriority(priority); got != want {
			t.Fatalf("%q = %d, want %d", priority, got, want)
		}
	}
}

func TestTaskWorkspaceAgentActionsAreServeOnly(t *testing.T) {
	model, _ := hierarchyModel(t, map[string]string{"work/TASK-AUTH-021.md": completeTaskFixture("Ready")})
	static := buildTaskWorkspaceData(model, "work/index.html")
	if len(static.Items) == 0 || len(static.Items[0].AgentActions) != 0 {
		t.Fatalf("static actions=%+v", static.Items)
	}
	model.serveRevision, model.taskActionsEnabled = "revision", true
	serve := buildTaskWorkspaceData(model, "work/index.html")
	found := false
	for _, item := range serve.Items {
		if item.ID == "TASK-AUTH-021" {
			found = len(item.AgentActions) > 0
		}
	}
	if !found {
		t.Fatalf("serve actions=%+v", serve.Items)
	}
}

func TestTaskPageActionsAreServeOnly(t *testing.T) {
	model, _ := hierarchyModel(t, map[string]string{"work/TASK-AUTH-021.md": completeTaskFixture("Ready")})
	document := model.DocByPath["work/TASK-AUTH-021.md"]
	if html := renderDocumentPage(model, document); strings.Contains(html, "data-task-actions") {
		t.Fatal("static task page contains actions")
	}
	model.serveMode, model.taskActionsEnabled, model.serveRevision = true, true, "revision"
	html := renderDocumentPage(model, document)
	for _, expected := range []string{`data-task-actions`, `data-task-id="TASK-AUTH-021"`, `data-task-agent-action="start-work"`} {
		if !strings.Contains(html, expected) {
			t.Fatalf("serve task page missing %q", expected)
		}
	}
}
