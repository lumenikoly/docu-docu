package toudocu

import "testing"

func TestTaskWorkState(t *testing.T) {
	tests := []struct {
		status                          WorkItemStatus
		contract, dependenciesSatisfied bool
		want                            TaskWorkState
	}{
		{WorkItemDraft, false, false, TaskWorkStateDraft},
		{WorkItemDraft, true, false, TaskWorkStateReadyCandidate},
		{WorkItemReady, true, true, TaskWorkStateReady},
		{WorkItemReady, true, false, TaskWorkStateWaiting},
		{WorkItemReady, false, true, TaskWorkStateNeedsAttention},
		{WorkItemInProgress, false, false, TaskWorkStateInProgress},
		{WorkItemBlocked, false, false, TaskWorkStateBlocked},
		{WorkItemDone, false, false, TaskWorkStateDone},
		{WorkItemCancelled, false, false, TaskWorkStateCancelled},
	}
	for _, test := range tests {
		if got := taskWorkState(test.status, test.contract, test.dependenciesSatisfied); got != test.want {
			t.Errorf("taskWorkState(%q, %t, %t) = %q, want %q", test.status, test.contract, test.dependenciesSatisfied, got, test.want)
		}
	}
}

func TestTaskWorkStateDraft(t *testing.T) {
	if taskWorkState(WorkItemDraft, false, true) != TaskWorkStateDraft || taskWorkState(WorkItemDraft, true, false) != TaskWorkStateReadyCandidate {
		t.Fatal("draft classification depends on more than contract completeness")
	}
}

func TestTaskWorkStateReady(t *testing.T) {
	if taskWorkState(WorkItemReady, true, true) != TaskWorkStateReady || taskWorkState(WorkItemReady, true, false) != TaskWorkStateWaiting || taskWorkState(WorkItemReady, false, true) != TaskWorkStateNeedsAttention {
		t.Fatal("ready classification is incorrect")
	}
}

func TestTaskWorkStateStoredStatuses(t *testing.T) {
	for status, want := range map[WorkItemStatus]TaskWorkState{WorkItemInProgress: TaskWorkStateInProgress, WorkItemBlocked: TaskWorkStateBlocked, WorkItemDone: TaskWorkStateDone, WorkItemCancelled: TaskWorkStateCancelled} {
		if got := taskWorkState(status, false, false); got != want {
			t.Errorf("%s = %s, want %s", status, got, want)
		}
	}
}

func TestTaskDescendantsSummary(t *testing.T) {
	root := &WorkItem{ID: "TASK-ROOT", ChildIDs: []string{"TASK-A", "TASK-B"}}
	items := map[string]*WorkItem{
		"TASK-A": {ID: "TASK-A", ChildIDs: []string{"TASK-C"}},
		"TASK-B": {ID: "TASK-B"},
		"TASK-C": {ID: "TASK-C"},
	}
	states := map[string]TaskWorkState{"TASK-A": TaskWorkStateDraft, "TASK-B": TaskWorkStateDone, "TASK-C": TaskWorkStateInProgress}
	got := taskDescendantsSummary(root, items, func(item *WorkItem) TaskWorkState { return states[item.ID] })
	if got.Total != 3 || got.Counts.Draft != 1 || got.Counts.Done != 1 || got.Counts.InProgress != 1 || !got.Started || got.Complete {
		t.Fatalf("summary = %#v", got)
	}
}

func TestTaskDescendantsStarted(t *testing.T) {
	for state, want := range map[TaskWorkState]bool{TaskWorkStateDraft: false, TaskWorkStateReadyCandidate: false, TaskWorkStateReady: false, TaskWorkStateWaiting: false, TaskWorkStateNeedsAttention: false, TaskWorkStateCancelled: false, TaskWorkStateInProgress: true, TaskWorkStateBlocked: true, TaskWorkStateDone: true} {
		child := &WorkItem{ID: "TASK-CHILD"}
		got := taskDescendantsSummary(&WorkItem{ChildIDs: []string{child.ID}}, map[string]*WorkItem{child.ID: child}, func(*WorkItem) TaskWorkState { return state })
		if got.Started != want {
			t.Errorf("state %s: started = %t, want %t", state, got.Started, want)
		}
	}
}

func TestTaskDescendantsComplete(t *testing.T) {
	child := &WorkItem{ID: "TASK-CHILD"}
	root := &WorkItem{ChildIDs: []string{child.ID}}
	items := map[string]*WorkItem{child.ID: child}
	if !taskDescendantsSummary(root, items, func(*WorkItem) TaskWorkState { return TaskWorkStateDone }).Complete {
		t.Fatal("all-done descendants are not complete")
	}
	if taskDescendantsSummary(root, items, func(*WorkItem) TaskWorkState { return TaskWorkStateCancelled }).Complete || taskDescendantsSummary(&WorkItem{}, items, func(*WorkItem) TaskWorkState { return TaskWorkStateDone }).Complete {
		t.Fatal("cancelled or empty descendants are complete")
	}
}

func TestChildStateDoesNotChangeParentWorkState(t *testing.T) {
	parent := taskWorkState(WorkItemDraft, false, true)
	child := &WorkItem{ID: "TASK-CHILD"}
	_ = taskDescendantsSummary(&WorkItem{ChildIDs: []string{child.ID}}, map[string]*WorkItem{child.ID: child}, func(*WorkItem) TaskWorkState { return TaskWorkStateDone })
	if taskWorkState(WorkItemDraft, false, true) != parent {
		t.Fatal("descendant calculation changed parent state")
	}
}
