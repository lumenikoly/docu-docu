package toudocu

type TaskWorkState string

const (
	TaskWorkStateDraft          TaskWorkState = "draft"
	TaskWorkStateReadyCandidate TaskWorkState = "ready_candidate"
	TaskWorkStateReady          TaskWorkState = "ready"
	TaskWorkStateWaiting        TaskWorkState = "waiting"
	TaskWorkStateNeedsAttention TaskWorkState = "needs_attention"
	TaskWorkStateInProgress     TaskWorkState = "in_progress"
	TaskWorkStateBlocked        TaskWorkState = "blocked"
	TaskWorkStateDone           TaskWorkState = "done"
	TaskWorkStateCancelled      TaskWorkState = "cancelled"
)

type TaskWorkStateCounts struct {
	Draft          int `json:"draft"`
	ReadyCandidate int `json:"readyCandidate"`
	Ready          int `json:"ready"`
	Waiting        int `json:"waiting"`
	NeedsAttention int `json:"needsAttention"`
	InProgress     int `json:"inProgress"`
	Blocked        int `json:"blocked"`
	Done           int `json:"done"`
	Cancelled      int `json:"cancelled"`
}

type TaskDescendantsSummary struct {
	Total    int                 `json:"total"`
	Counts   TaskWorkStateCounts `json:"counts"`
	Started  bool                `json:"started"`
	Complete bool                `json:"complete"`
}

func taskWorkState(status WorkItemStatus, contractComplete, dependenciesSatisfied bool) TaskWorkState {
	switch status {
	case WorkItemDraft:
		if contractComplete {
			return TaskWorkStateReadyCandidate
		}
		return TaskWorkStateDraft
	case WorkItemReady:
		if !contractComplete {
			return TaskWorkStateNeedsAttention
		}
		if !dependenciesSatisfied {
			return TaskWorkStateWaiting
		}
		return TaskWorkStateReady
	case WorkItemInProgress:
		return TaskWorkStateInProgress
	case WorkItemBlocked:
		return TaskWorkStateBlocked
	case WorkItemDone:
		return TaskWorkStateDone
	case WorkItemCancelled:
		return TaskWorkStateCancelled
	default:
		return TaskWorkState(status)
	}
}

func addTaskWorkState(counts *TaskWorkStateCounts, state TaskWorkState) {
	switch state {
	case TaskWorkStateDraft:
		counts.Draft++
	case TaskWorkStateReadyCandidate:
		counts.ReadyCandidate++
	case TaskWorkStateReady:
		counts.Ready++
	case TaskWorkStateWaiting:
		counts.Waiting++
	case TaskWorkStateNeedsAttention:
		counts.NeedsAttention++
	case TaskWorkStateInProgress:
		counts.InProgress++
	case TaskWorkStateBlocked:
		counts.Blocked++
	case TaskWorkStateDone:
		counts.Done++
	case TaskWorkStateCancelled:
		counts.Cancelled++
	}
}

func taskDescendantsSummary(item *WorkItem, byID map[string]*WorkItem, state func(*WorkItem) TaskWorkState) TaskDescendantsSummary {
	result := TaskDescendantsSummary{}
	seen := map[string]bool{}
	var visit func(*WorkItem)
	visit = func(current *WorkItem) {
		for _, id := range current.ChildIDs {
			child := byID[id]
			if child == nil || seen[id] {
				continue
			}
			seen[id] = true
			childState := state(child)
			result.Total++
			addTaskWorkState(&result.Counts, childState)
			if childState == TaskWorkStateInProgress || childState == TaskWorkStateBlocked || childState == TaskWorkStateDone {
				result.Started = true
			}
			visit(child)
		}
	}
	visit(item)
	result.Complete = result.Total > 0 && result.Counts.Done == result.Total
	return result
}
