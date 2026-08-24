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

func TestTaskWorkspaceTreeKeepsRootSiblingsOutsideBranches(t *testing.T) {
	html := renderTaskWorkspaceTree(nil, TaskWorkspaceData{Items: []TaskWorkspaceItem{
		{ID: "TASK-MEDIA-002", Title: "Media", WorkspaceState: "done"},
		{ID: "TASK-MEDIA-003", Title: "Media child", ParentID: "TASK-MEDIA-002", WorkspaceState: "draft"},
		{ID: "TASK-V1-001", Title: "V1", WorkspaceState: "draft"},
	}})
	boundary := `data-task-id="TASK-MEDIA-003" data-parent-id="TASK-MEDIA-002"`
	child := strings.Index(html, boundary)
	root := strings.Index(html, `data-task-id="TASK-V1-001" data-parent-id=""`)
	if child < 0 || root < 0 || !strings.Contains(html[child:root], `</ul></li><li data-task-workspace-item`) {
		t.Fatalf("root sibling rendered inside previous branch: %s", html)
	}
}
