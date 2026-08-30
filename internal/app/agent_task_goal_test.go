package toudocu

import (
	"strings"
	"testing"
)

func taskTreeGoalModel(t *testing.T, rootStatus, childStatus string) *Model {
	t.Helper()
	root := strings.Replace(completeTaskFixture(rootStatus), "TASK-AUTH-021", "TASK-AUTH-100", 1)
	childFixture := completeTaskFixture(childStatus)
	if childStatus == "Done" {
		childFixture = terminalTaskFixture(childStatus)
	}
	child := strings.Replace(childFixture, "TASK-AUTH-021", "TASK-AUTH-101", 1)
	child = strings.Replace(child, "- Use case: UC-AUTH-01\n", "- Use case: UC-AUTH-01\n- Parent: TASK-AUTH-100\n", 1)
	model, _ := hierarchyModel(t, map[string]string{
		"work/TASK-AUTH-100.md": root,
		"work/TASK-AUTH-101.md": child,
	})
	return model
}

func TestTaskTreeGoalSelection(t *testing.T) {
	model := taskTreeGoalModel(t, "In progress", "Ready")
	root, _ := findWorkItem(model, "TASK-AUTH-100")
	items, tree := taskTree(model, root)
	if next := taskTreeGoalNext(items, tree); next == nil || next.ID != "TASK-AUTH-101" {
		t.Fatalf("next=%+v", next)
	}
	child := tree["TASK-AUTH-101"]
	child.statusName = WorkItemDone
	if next := taskTreeGoalNext(items, tree); next == nil || next.ID != "TASK-AUTH-100" {
		t.Fatalf("parent was not selected after child: %+v", next)
	}
}

func TestTaskTreeGoalLifecycle(t *testing.T) {
	model := taskTreeGoalModel(t, "In progress", "In progress")
	goal, err := newTaskTreeGoal(model, "TASK-AUTH-100")
	if err != nil {
		t.Fatal(err)
	}
	for turn := 0; turn < taskTreeGoalNoProgressLimit; turn++ {
		advanceTaskTreeGoal(model, goal)
	}
	if goal.Status != "blocked" || !strings.Contains(goal.Message, "three turns") {
		t.Fatalf("goal=%+v", goal)
	}

	completed := taskTreeGoalModel(t, "Done", "Done")
	goal = &taskTreeGoal{Status: "active", RootTaskID: "TASK-AUTH-100"}
	advanceTaskTreeGoal(completed, goal)
	if goal.Status != "complete" || goal.CompletedTasks != goal.TotalTasks {
		t.Fatalf("completed goal=%+v", goal)
	}
}

func TestTaskTreeGoalSession(t *testing.T) {
	console := &agentConsole{}
	console.setGoal(&taskTreeGoal{Status: "active", RootTaskID: "TASK-AUTH-100"})
	console.blockGoal("session stopped")
	goal := console.goalSnapshot()
	if goal == nil || goal.Status != "blocked" || goal.Message != "session stopped" {
		t.Fatalf("goal=%+v", goal)
	}
	console.clearGoal()
	if console.goalSnapshot() != nil {
		t.Fatal("goal was not cleared")
	}
}
