package toudocu

import (
	"fmt"
	"io"
	"strings"
)

func BuildTaskCandidates(model *Model, parentTaskID string, strict bool) (TaskCandidatesReport, error) {
	if err := rejectTranslationTaskModel(model); err != nil {
		return TaskCandidatesReport{}, err
	}
	byID := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		byID[item.ID] = item
	}
	allowed := map[string]bool{}
	if parentTaskID != "" {
		parent, err := findWorkItem(model, parentTaskID)
		if err != nil {
			return TaskCandidatesReport{}, err
		}
		if !strings.HasPrefix(parent.ID, "TASK-") {
			return TaskCandidatesReport{}, fmt.Errorf("task candidates --parent requires a TASK-* work item")
		}
		var addDescendants func(*WorkItem)
		addDescendants = func(item *WorkItem) {
			for _, childID := range item.ChildIDs {
				if child := byID[childID]; child != nil && !allowed[childID] {
					allowed[childID] = true
					addDescendants(child)
				}
			}
		}
		addDescendants(parent)
	}
	report := TaskCandidatesReport{
		SchemaVersion: 1, Kind: "task-candidates", Generator: GeneratorInfo{Name: "Toudocu", Version: Version},
		ParentTaskID: parentTaskID, Candidates: []TaskCandidate{},
	}
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		if item.Archived || item.statusName != "draft" && item.statusName != "ready" || parentTaskID != "" && !allowed[item.ID] {
			continue
		}
		_, issues := taskReadiness(model, item.ID, strict)
		contractComplete := len(blockingReadinessIssues(issues, strict)) == 0
		blockedBy := []TaskCandidateBlocker{}
		for _, dependencyID := range item.DependsOn {
			status := "unknown"
			if dependency := byID[dependencyID]; dependency != nil {
				status = string(dependency.statusName)
				if dependency.statusName == "done" {
					continue
				}
			}
			blockedBy = append(blockedBy, TaskCandidateBlocker{ID: dependencyID, Status: status})
		}
		dependenciesSatisfied := len(blockedBy) == 0
		report.Candidates = append(report.Candidates, TaskCandidate{
			ID: item.ID, Title: item.Title, Status: string(item.statusName), Priority: item.Priority, ParentID: item.ParentID,
			ContractComplete: contractComplete, DependenciesSatisfied: dependenciesSatisfied,
			ReadyForWork: item.statusName == "ready" && contractComplete && dependenciesSatisfied,
			BlockedBy:    blockedBy, Issues: issues,
		})
	}
	return report, nil
}

func printTaskCandidatesText(w io.Writer, report TaskCandidatesReport) {
	if len(report.Candidates) == 0 {
		_, _ = fmt.Fprintln(w, "No task candidates.")
		return
	}
	for _, candidate := range report.Candidates {
		state := "DRAFT"
		switch {
		case candidate.ReadyForWork:
			state = "READY"
		case !candidate.ContractComplete:
			state = "BLOCKED"
		case !candidate.DependenciesSatisfied:
			state = "WAITING"
		}
		conditions := []string{"status=" + candidate.Status}
		if !candidate.ContractComplete {
			conditions = append(conditions, "contract incomplete")
		}
		if candidate.Status == "draft" {
			conditions = append(conditions, "change status to Ready")
		}
		if !candidate.DependenciesSatisfied {
			ids := make([]string, 0, len(candidate.BlockedBy))
			for _, blocker := range candidate.BlockedBy {
				ids = append(ids, blocker.ID)
			}
			conditions = append(conditions, "depends on "+strings.Join(ids, ", "))
		}
		if candidate.ReadyForWork {
			conditions = append(conditions, "executable")
		}
		priority := candidate.Priority
		if priority == "" {
			priority = "-"
		}
		_, _ = fmt.Fprintf(w, "%-16s %-8s %-7s %s\n", candidate.ID, state, priority, strings.Join(conditions, "; "))
	}
}
