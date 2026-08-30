package toudocu

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const taskTreeGoalNoProgressLimit = 3

type taskTreeGoal struct {
	Status         string `json:"status"`
	RootTaskID     string `json:"rootTaskID"`
	CurrentTaskID  string `json:"currentTaskID,omitempty"`
	CompletedTasks int    `json:"completedTasks"`
	TotalTasks     int    `json:"totalTasks"`
	Message        string `json:"message,omitempty"`

	documentationRoot string
	noProgressTurns  int
	fingerprint      string
}

func taskTree(model *Model, root *WorkItem) ([]*WorkItem, map[string]*WorkItem) {
	byID := workItemsByID(model)
	tree := map[string]*WorkItem{}
	var add func(*WorkItem)
	add = func(item *WorkItem) {
		if item == nil || tree[item.ID] != nil {
			return
		}
		tree[item.ID] = item
		for _, childID := range item.ChildIDs {
			add(byID[childID])
		}
	}
	add(root)
	items := make([]*WorkItem, 0, len(tree))
	for _, item := range tree {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, tree
}

func validateTaskTree(model *Model, root *WorkItem, items []*WorkItem, tree map[string]*WorkItem) error {
	if !strings.HasPrefix(root.ID, "TASK-") || len(root.ChildIDs) == 0 {
		return errors.New("task-tree goal requires a parent TASK-* with descendants")
	}
	if root.statusName != WorkItemReady && root.statusName != WorkItemInProgress {
		return fmt.Errorf("parent task %s must be Ready or In progress", root.ID)
	}
	byID := workItemsByID(model)
	for _, item := range items {
		switch item.statusName {
		case WorkItemReady, WorkItemInProgress:
			_, issues := taskReadiness(model, item.ID, model.strictPolicy)
			if blocking := blockingReadinessIssues(issues, model.strictPolicy); len(blocking) > 0 {
				return fmt.Errorf("task %s is not ready: %s", item.ID, blocking[0].Message)
			}
		case WorkItemDone:
			for _, criterion := range item.Criteria {
				if !criterion.Completed {
					return fmt.Errorf("task %s is Done with unchecked acceptance criteria", item.ID)
				}
			}
		default:
			return fmt.Errorf("task %s has unsupported state %s", item.ID, item.statusName)
		}
		for _, dependencyID := range item.DependsOn {
			dependency := byID[dependencyID]
			if dependency == nil {
				return fmt.Errorf("task %s depends on missing task %s", item.ID, dependencyID)
			}
			if tree[dependencyID] == nil && dependency.statusName != WorkItemDone {
				return fmt.Errorf("task %s waits for external dependency %s", item.ID, dependencyID)
			}
		}
	}
	return nil
}

func taskTreeGoalCounts(items []*WorkItem) (completed int) {
	for _, item := range items {
		if item.statusName == WorkItemDone {
			completed++
		}
	}
	return completed
}

func taskTreeGoalNext(items []*WorkItem, tree map[string]*WorkItem) *WorkItem {
	for _, item := range items {
		if item.statusName == WorkItemDone {
			continue
		}
		ready := true
		for _, dependencyID := range item.DependsOn {
			dependency := tree[dependencyID]
			if dependency != nil && dependency.statusName != WorkItemDone {
				ready = false
			}
		}
		for _, childID := range item.ChildIDs {
			if child := tree[childID]; child != nil && child.statusName != WorkItemDone {
				ready = false
			}
		}
		if ready {
			return item
		}
	}
	return nil
}

func taskTreeGoalFingerprint(item *WorkItem) string {
	completed := make([]string, 0, len(item.Criteria))
	for _, criterion := range item.Criteria {
		if criterion.Completed {
			completed = append(completed, criterion.Text)
		}
	}
	return string(item.statusName) + "\x00" + strings.Join(completed, "\x00")
}

func newTaskTreeGoal(model *Model, rootID string) (*taskTreeGoal, error) {
	root, err := findWorkItem(model, rootID)
	if err != nil {
		return nil, err
	}
	items, tree := taskTree(model, root)
	if err := validateTaskTree(model, root, items, tree); err != nil {
		return nil, err
	}
	next := taskTreeGoalNext(items, tree)
	if next == nil {
		return nil, errors.New("task tree has no executable task")
	}
	return &taskTreeGoal{Status: "active", RootTaskID: rootID, CurrentTaskID: next.ID, CompletedTasks: taskTreeGoalCounts(items), TotalTasks: len(items), documentationRoot: model.RootDirectory, fingerprint: taskTreeGoalFingerprint(next)}, nil
}

func advanceTaskTreeGoal(model *Model, goal *taskTreeGoal) (string, bool) {
	root, err := findWorkItem(model, goal.RootTaskID)
	if err != nil {
		goal.Status, goal.Message = "blocked", err.Error()
		return "", false
	}
	items, tree := taskTree(model, root)
	goal.CompletedTasks, goal.TotalTasks = taskTreeGoalCounts(items), len(items)
	if root.statusName == WorkItemDone && goal.CompletedTasks == goal.TotalTasks {
		goal.Status, goal.CurrentTaskID, goal.Message = "complete", "", ""
		return "", false
	}
	if err := validateTaskTree(model, root, items, tree); err != nil {
		goal.Status, goal.Message = "blocked", err.Error()
		return "", false
	}
	current := tree[goal.CurrentTaskID]
	if current == nil {
		goal.Status, goal.Message = "blocked", "current task is no longer part of the tree"
		return "", false
	}
	if current.statusName != WorkItemDone {
		fingerprint := taskTreeGoalFingerprint(current)
		if fingerprint == goal.fingerprint {
			goal.noProgressTurns++
		} else {
			goal.noProgressTurns = 0
			goal.fingerprint = fingerprint
		}
		if goal.noProgressTurns >= taskTreeGoalNoProgressLimit {
			goal.Status, goal.Message = "blocked", fmt.Sprintf("task %s made no status or acceptance-criteria progress in three turns", current.ID)
			return "", false
		}
		return taskTreeGoalPrompt(goal.RootTaskID, current.ID), true
	}
	next := taskTreeGoalNext(items, tree)
	if next == nil {
		goal.Status, goal.Message = "blocked", "task tree has no executable task"
		return "", false
	}
	goal.CurrentTaskID, goal.noProgressTurns, goal.fingerprint = next.ID, 0, taskTreeGoalFingerprint(next)
	return taskTreeGoalPrompt(goal.RootTaskID, next.ID), true
}

func taskTreeGoalPrompt(rootID, currentID string) string {
	return fmt.Sprintf("Continue the Toudocu task-tree goal rooted at %s. Work only on current task %s in this turn. Read its current Toudocu task context and repository state. If it is Ready, move it to In progress before implementation. Complete its scoped implementation and documentation, run every declared verification command, check only acceptance criteria proven by those results, and set it to Done only after all checks pass. If it cannot continue, set it to Blocked with a concrete blocker. Toudocu will select the next dependency-ready task after this turn.", rootID, currentID)
}

func (c *agentConsole) goalSnapshot() *taskTreeGoal {
	c.goalMu.Lock()
	defer c.goalMu.Unlock()
	if c.goal == nil {
		return nil
	}
	copy := *c.goal
	return &copy
}

func (c *agentConsole) setGoal(goal *taskTreeGoal) {
	c.goalMu.Lock()
	c.goal = goal
	c.goalMu.Unlock()
}

func (c *agentConsole) clearGoal() {
	c.goalMu.Lock()
	c.goal = nil
	c.goalMu.Unlock()
}

func (c *agentConsole) blockGoal(message string) {
	c.goalMu.Lock()
	if c.goal != nil && c.goal.Status == "active" {
		c.goal.Status, c.goal.Message = "blocked", message
	}
	c.goalMu.Unlock()
}

func (s *documentationServer) continueTaskTreeGoal() {
	console := s.agentConsole
	if console == nil {
		return
	}
	console.goalMu.Lock()
	defer console.goalMu.Unlock()
	if console.goal == nil || console.goal.Status != "active" {
		return
	}
	snapshot, active := console.manager.Snapshot()
	if !active || snapshot.Status == AgentSessionFailed || !agentTaskMatches(snapshot, console.goal.RootTaskID, console.goal.documentationRoot) {
		console.goal.Status, console.goal.Message = "blocked", "agent session is unavailable"
		console.publishState()
		return
	}
	if snapshot.Status != AgentSessionIdle || len(snapshot.Approvals) > 0 {
		return
	}
	options := s.options
	options.InputDirectory = console.goal.documentationRoot
	model, err := BuildDocumentationModel(options)
	if err != nil {
		console.goal.Status, console.goal.Message = "blocked", err.Error()
		console.publishState()
		return
	}
	prompt, send := advanceTaskTreeGoal(model, console.goal)
	if send {
		prompt = s.taskActionPrompt(model, prompt)
		if err := console.manager.Send(context.Background(), prompt, AgentTurnNormal); err != nil {
			console.goal.Status, console.goal.Message = "blocked", err.Error()
		}
	}
	console.publishState()
}
