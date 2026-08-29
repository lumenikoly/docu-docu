package toudocu

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func findWorkItem(model *Model, taskID string) (*WorkItem, error) {
	var found *WorkItem
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		if item.ID != taskID {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("task identifier %s is ambiguous", taskID)
		}
		found = item
	}
	if found == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	return found, nil
}

func taskContextDocument(document *Document) TaskContextDocument {
	sections := []TaskContextSection{}
	full := containsType([]string{"work", "contract", "guide", "reference", "document", "flow", "screen"}, document.Type)
	if full {
		sections = append(sections, TaskContextSection{Title: document.Title, Markdown: document.Content})
	} else {
		selected := map[SectionKind]bool{}
		switch document.Type {
		case "module":
			for _, kind := range []SectionKind{SectionKindBusinessRules, SectionKindInvariants, SectionKindStableInterfaces} {
				selected[kind] = true
			}
		case "use-case":
			for _, kind := range []SectionKind{SectionKindMainScenario, SectionKindPostconditions, SectionKindBusinessRules} {
				selected[kind] = true
			}
		case "standard":
			for _, kind := range []SectionKind{SectionKindRules, SectionKindAutomatedChecks} {
				selected[kind] = true
			}
		case "runbook":
			for _, kind := range []SectionKind{SectionKindPrerequisites, SectionKindProcedure, SectionKindVerification, SectionKindRollback, SectionKindStopConditions} {
				selected[kind] = true
			}
		}
		for _, section := range document.Sections {
			if selected[section.Kind] {
				sections = append(sections, TaskContextSection{Title: section.Title, Markdown: section.Markdown})
			}
		}
	}
	return TaskContextDocument{
		ID: document.Metadata["id"], Path: document.SourcePath, Type: document.Type,
		Title: document.Title, Description: document.Description, Status: document.Status, Sections: sections,
	}
}

func taskContextExternalDocument(model *Model, relativePath string) (TaskContextDocument, bool) {
	absolute := filepath.Join(model.RepositoryRoot, filepath.FromSlash(relativePath))
	if !ensureInside(model.RepositoryRoot, absolute) || ensureInside(model.RootDirectory, absolute) {
		return TaskContextDocument{}, false
	}
	resolvedRoot, rootErr := resolvePathForSafety(model.RepositoryRoot)
	resolvedPath, pathErr := resolvePathForSafety(absolute)
	if rootErr != nil || pathErr != nil || !ensureInside(resolvedRoot, resolvedPath) {
		return TaskContextDocument{}, false
	}
	content, err := os.ReadFile(absolute)
	if err != nil {
		return TaskContextDocument{}, false
	}
	parsed := analyzeMarkdown(string(content))
	return TaskContextDocument{
		Path: relativePath, Type: "document", Title: parsed.Title, Description: parsed.Description,
		Status:   StatusFor(parsed.Metadata["status"]),
		Sections: []TaskContextSection{{Title: parsed.Title, Markdown: string(content)}},
	}, true
}

// BuildTaskContext returns compact, read-only implementation context for one task.
func BuildTaskContext(model *Model, taskID string) (TaskContextReport, error) {
	item, err := findWorkItem(model, taskID)
	if err != nil {
		return TaskContextReport{}, err
	}
	if item.statusName != "ready" && item.statusName != "in-progress" && item.statusName != "blocked" && item.statusName != "done" {
		return TaskContextReport{}, fmt.Errorf("task context is available only for Ready, In Progress, Blocked, or Done tasks")
	}
	report := TaskContextReport{
		SchemaVersion: 1, Kind: "task-context",
		Generator:         GeneratorInfo{Name: "Toudocu", Version: Version},
		Task:              *item,
		Hierarchy:         taskHierarchy(model, item),
		Screens:           []KnowledgeScreen{},
		ScreenTransitions: []ScreenTransition{},
		BusinessRules:     []BusinessRule{},
		Standards:         []KnowledgeStandard{},
		Runbooks:          []KnowledgeRunbook{},
		Dependencies:      []WorkItem{},
		Dependents:        []WorkItem{},
		Documents:         []TaskContextDocument{},
		Issues:            []Issue{},
		RequiredReads:     []string{},
	}
	_, report.FullVerification = planTaskCommands(*item)

	ruleIDs := map[string]bool{}
	if item.ModuleID != "" {
		for index := range model.Knowledge.Modules {
			module := &model.Knowledge.Modules[index]
			if module.ID == item.ModuleID {
				report.Module = module
				for _, id := range module.BusinessRuleIDs {
					ruleIDs[id] = true
				}
				break
			}
		}
	}
	if item.UseCaseID != "" {
		for index := range model.Knowledge.UseCases {
			useCase := &model.Knowledge.UseCases[index]
			if useCase.ID == item.UseCaseID {
				report.UseCase = useCase
				for _, id := range useCase.BusinessRuleIDs {
					ruleIDs[id] = true
				}
				break
			}
		}
	}
	for _, rule := range model.Knowledge.BusinessRules {
		if ruleIDs[rule.ID] {
			report.BusinessRules = append(report.BusinessRules, rule)
		}
	}
	for _, candidate := range model.Knowledge.WorkItems {
		if containsString(item.DependsOn, candidate.ID) {
			report.Dependencies = append(report.Dependencies, candidate)
		}
		if containsString(candidate.DependsOn, item.ID) {
			report.Dependents = append(report.Dependents, candidate)
		}
	}

	documentPaths := map[string]bool{item.Document: true}
	if report.Module != nil {
		documentPaths[report.Module.Document] = true
	}
	if report.UseCase != nil {
		documentPaths[report.UseCase.Document] = true
	}
	if item.FlowID != "" {
		for index := range model.Knowledge.Flows {
			flow := &model.Knowledge.Flows[index]
			if flow.ID == item.FlowID {
				report.Flow = flow
				documentPaths[flow.Document] = true
				break
			}
		}
	}
	for _, screenID := range item.ScreenIDs {
		for _, screen := range model.Knowledge.Screens {
			if screen.ID != screenID {
				continue
			}
			report.Screens = append(report.Screens, screen)
			if screen.Document != "" {
				documentPaths[screen.Document] = true
			}
			break
		}
	}
	selectedScreens := map[string]bool{}
	for _, screenID := range item.ScreenIDs {
		selectedScreens[screenID] = true
	}
	selectedTransitions := map[string]bool{}
	for _, transitionID := range item.TransitionIDs {
		selectedTransitions[transitionID] = true
	}
	for _, transition := range model.Knowledge.Transitions {
		if selectedTransitions[transition.ID] || selectedScreens[transition.FromID] || selectedScreens[transition.ToID] {
			report.ScreenTransitions = append(report.ScreenTransitions, transition)
			if transition.Document != "" {
				documentPaths[transition.Document] = true
			}
		}
	}
	for _, rule := range report.BusinessRules {
		documentPaths[rule.Document] = true
	}
	for _, standard := range model.Knowledge.Standards {
		if containsString(item.StandardIDs, standard.ID) {
			report.Standards = append(report.Standards, standard)
			documentPaths[standard.Document] = true
		}
	}
	for _, runbook := range model.Knowledge.Runbooks {
		if containsString(item.RunbookIDs, runbook.ID) {
			report.Runbooks = append(report.Runbooks, runbook)
			documentPaths[runbook.Document] = true
		}
	}
	for _, path := range item.DocumentationPaths {
		documentPaths[path] = true
	}
	for _, document := range model.Documents {
		if documentPaths[document.SourcePath] {
			report.Documents = append(report.Documents, taskContextDocument(document))
			report.RequiredReads = append(report.RequiredReads, document.SourcePath)
		}
	}
	for _, path := range item.DocumentationPaths {
		if model.DocByPath[path] != nil {
			continue
		}
		if document, ok := taskContextExternalDocument(model, path); ok {
			report.Documents = append(report.Documents, document)
			report.RequiredReads = append(report.RequiredReads, path)
		}
	}
	_, report.Issues = taskReadiness(model, taskID, false)
	sort.SliceStable(report.BusinessRules, func(i, j int) bool {
		return naturalCompare(report.BusinessRules[i].ID, report.BusinessRules[j].ID) < 0
	})
	sort.SliceStable(report.Dependencies, func(i, j int) bool {
		return naturalCompare(report.Dependencies[i].ID, report.Dependencies[j].ID) < 0
	})
	sort.SliceStable(report.Dependents, func(i, j int) bool {
		return naturalCompare(report.Dependents[i].ID, report.Dependents[j].ID) < 0
	})
	sort.SliceStable(report.RequiredReads, func(i, j int) bool {
		if report.RequiredReads[i] == item.Document {
			return true
		}
		if report.RequiredReads[j] == item.Document {
			return false
		}
		return naturalCompare(report.RequiredReads[i], report.RequiredReads[j]) < 0
	})
	return report, nil
}

func printTaskContextText(w io.Writer, report TaskContextReport) {
	_, _ = fmt.Fprintf(w, "Task: %s — %s\nDocument: %s\nStatus: %s\n", report.Task.ID, report.Task.Title, report.Task.Document, report.Task.Status.Label)
	if report.Module != nil {
		_, _ = fmt.Fprintf(w, "Module: %s — %s\n", report.Module.ID, report.Module.Title)
	}
	if report.UseCase != nil {
		_, _ = fmt.Fprintf(w, "Use case: %s — %s\n", report.UseCase.ID, report.UseCase.Title)
	}
	if report.Task.FlowID != "" {
		_, _ = fmt.Fprintf(w, "Flow: %s\n", report.Task.FlowID)
	}
	if report.Hierarchy.Parent != nil || len(report.Hierarchy.Ancestors) > 0 || len(report.Hierarchy.Children) > 0 || report.Hierarchy.Descendants.Total > 0 {
		refText := func(ref TaskHierarchyRef) string {
			blocker := "no"
			if ref.HasBlocker {
				blocker = "yes"
			}
			return fmt.Sprintf("%s — %s [%s; blocker: %s]", ref.ID, ref.Title, ref.Status, blocker)
		}
		if len(report.Hierarchy.Ancestors) > 0 {
			ancestors := make([]string, 0, len(report.Hierarchy.Ancestors))
			for _, ancestor := range report.Hierarchy.Ancestors {
				ancestors = append(ancestors, refText(ancestor))
			}
			_, _ = fmt.Fprintf(w, "Ancestors: %s\n", strings.Join(ancestors, " / "))
		}
		if report.Hierarchy.Parent != nil {
			_, _ = fmt.Fprintf(w, "Parent task: %s\n", refText(*report.Hierarchy.Parent))
		}
		if len(report.Hierarchy.Children) > 0 {
			_, _ = fmt.Fprintln(w, "Child tasks:")
			for _, child := range report.Hierarchy.Children {
				_, _ = fmt.Fprintf(w, "- %s\n", refText(child))
			}
		}
		summary := report.Hierarchy.Descendants
		statuses := []string{}
		for _, status := range []struct {
			label string
			count int
		}{
			{"draft", summary.Counts.Draft}, {"ready candidate", summary.Counts.ReadyCandidate},
			{"ready", summary.Counts.Ready}, {"waiting", summary.Counts.Waiting},
			{"needs attention", summary.Counts.NeedsAttention}, {"in progress", summary.Counts.InProgress},
			{"blocked", summary.Counts.Blocked}, {"done", summary.Counts.Done}, {"cancelled", summary.Counts.Cancelled},
		} {
			if status.count > 0 {
				statuses = append(statuses, fmt.Sprintf("%s: %d", status.label, status.count))
			}
		}
		detail := ""
		if len(statuses) > 0 {
			detail = "; " + strings.Join(statuses, "; ")
		}
		_, _ = fmt.Fprintf(w, "Descendants: total %d%s; started: %t; complete: %t\n", summary.Total, detail, summary.Started, summary.Complete)
	}
	if len(report.Task.ScreenIDs) > 0 {
		_, _ = fmt.Fprintf(w, "Screens: %s\n", strings.Join(report.Task.ScreenIDs, ", "))
	}
	if len(report.Task.RepositoryPaths) > 0 {
		_, _ = fmt.Fprintf(w, "Scope: %s\n", strings.Join(report.Task.RepositoryPaths, ", "))
	}
	_, _ = fmt.Fprintf(w, "Criteria: %d\nChecks: %d\nDependencies: %d\nDependents: %d\nContext issues: %d\n",
		len(report.Task.Criteria), len(report.Task.Checks), len(report.Dependencies), len(report.Dependents), len(report.Issues))
	_, _ = fmt.Fprintf(w, "Required documents: %s\n", strings.Join(report.RequiredReads, ", "))
}
