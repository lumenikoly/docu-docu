package toudocu

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	frontend "toudocu/internal/site"
)

// TaskWorkspaceData is the page-local, presentation-safe contract for Work items.
type TaskWorkspaceData struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Items         []TaskWorkspaceItem  `json:"items"`
	Summary       TaskWorkspaceSummary `json:"summary"`
}

type TaskWorkspaceSummary struct {
	Active     int `json:"active"`
	InProgress int `json:"inProgress"`
	Ready      int `json:"ready"`
	Waiting    int `json:"waiting"`
	Blocked    int `json:"blocked"`
}

type TaskWorkspaceDependency struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Href   string `json:"href,omitempty"`
}

type TaskWorkspaceItem struct {
	ID                    string                    `json:"id"`
	Title                 string                    `json:"title"`
	Status                string                    `json:"status"`
	WorkState             TaskWorkState             `json:"workState"`
	WorkspaceState        string                    `json:"workspaceState"`
	Descendants           *TaskDescendantsSummary   `json:"descendants,omitempty"`
	Type                  string                    `json:"type"`
	Priority              string                    `json:"priority,omitempty"`
	Archived              bool                      `json:"archived"`
	ArchiveYear           string                    `json:"archiveYear,omitempty"`
	ContractComplete      bool                      `json:"contractComplete"`
	DependenciesSatisfied bool                      `json:"dependenciesSatisfied"`
	ReadyForWork          bool                      `json:"readyForWork"`
	ReadinessIssueCount   int                       `json:"readinessIssueCount"`
	CompletedCriteria     int                       `json:"completedCriteria"`
	TotalCriteria         int                       `json:"totalCriteria"`
	DependsOn             []TaskWorkspaceDependency `json:"dependsOn"`
	ParentID              string                    `json:"parentId,omitempty"`
	ChildIDs              []string                  `json:"childIds"`
	ModuleID              string                    `json:"moduleId,omitempty"`
	UseCaseID             string                    `json:"useCaseId,omitempty"`
	Blocker               string                    `json:"blocker,omitempty"`
	Document              string                    `json:"document"`
	Href                  string                    `json:"href"`
	Severity              string                    `json:"severity,omitempty"`
	ReadinessIssues       []string                  `json:"readinessIssues"`
	Digest                string                    `json:"digest,omitempty"`
	AgentActions          []AgentTaskAction         `json:"agentActions"`
}

type taskReadinessSummary struct {
	ContractComplete, DependenciesSatisfied bool
	BlockedBy                               []TaskCandidateBlocker
	Issues                                  []Issue
}

func taskWorkspaceReadiness(model *Model, item *WorkItem, strict bool, byID map[string]*WorkItem) taskReadinessSummary {
	_, issues := taskReadiness(model, item.ID, strict)
	result := taskReadinessSummary{ContractComplete: len(blockingReadinessIssues(issues, strict)) == 0, Issues: issues}
	for _, id := range item.DependsOn {
		status := "unknown"
		if dependency := byID[id]; dependency != nil {
			status = string(dependency.statusName)
			if dependency.statusName == WorkItemDone {
				continue
			}
		}
		result.BlockedBy = append(result.BlockedBy, TaskCandidateBlocker{ID: id, Status: status})
	}
	result.DependenciesSatisfied = len(result.BlockedBy) == 0
	return result
}

func taskWorkspaceState(item *WorkItem, readiness taskReadinessSummary) string {
	if item.Archived {
		return "archive"
	}
	return strings.ReplaceAll(string(taskWorkState(item.statusName, readiness.ContractComplete, readiness.DependenciesSatisfied)), "_", "-")
}

func taskWorkspacePriority(value string) int {
	switch strings.ToLower(value) {
	case "urgent":
		return 0
	case "high":
		return 1
	case "normal":
		return 2
	case "low":
		return 3
	case "":
		return 4
	default:
		return 5
	}
}

func buildTaskWorkspaceData(model *Model, current string) TaskWorkspaceData {
	byID := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		byID[item.ID] = item
	}
	data := TaskWorkspaceData{SchemaVersion: 1, Items: []TaskWorkspaceItem{}}
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		readiness := taskReadinessSummary{DependenciesSatisfied: true}
		if !item.Archived && (item.statusName == WorkItemDraft || item.statusName == WorkItemReady) {
			readiness = taskWorkspaceReadiness(model, item, model.strictPolicy, byID)
		}
		completed := 0
		for _, criterion := range item.Criteria {
			if criterion.Completed {
				completed++
			}
		}
		workState := taskWorkState(item.statusName, readiness.ContractComplete, readiness.DependenciesSatisfied)
		workspaceState := strings.ReplaceAll(string(workState), "_", "-")
		if item.Archived {
			workspaceState = "archive"
		}
		view := TaskWorkspaceItem{ID: item.ID, Title: item.Title, Status: string(item.statusName), WorkState: workState, WorkspaceState: workspaceState, Type: item.Type, Priority: item.Priority, Severity: item.Severity, Archived: item.Archived, ArchiveYear: item.ArchiveYear, ContractComplete: readiness.ContractComplete, DependenciesSatisfied: readiness.DependenciesSatisfied, ReadyForWork: item.statusName == WorkItemReady && readiness.ContractComplete && readiness.DependenciesSatisfied, ReadinessIssueCount: len(blockingReadinessIssues(readiness.Issues, model.strictPolicy)), CompletedCriteria: completed, TotalCriteria: len(item.Criteria), DependsOn: []TaskWorkspaceDependency{}, ParentID: taskParentID(item), ChildIDs: append([]string{}, item.ChildIDs...), ModuleID: item.ModuleID, UseCaseID: item.UseCaseID, Blocker: item.Blocker, Document: item.Document, ReadinessIssues: []string{}, AgentActions: []AgentTaskAction{}}
		if len(item.ChildIDs) > 0 {
			summary := taskDescendantsSummary(item, byID, func(child *WorkItem) TaskWorkState {
				childReadiness := taskReadinessSummary{DependenciesSatisfied: true}
				if child.statusName == WorkItemDraft || child.statusName == WorkItemReady {
					childReadiness = taskWorkspaceReadiness(model, child, model.strictPolicy, byID)
				}
				return taskWorkState(child.statusName, childReadiness.ContractComplete, childReadiness.DependenciesSatisfied)
			})
			view.Descendants = &summary
		}
		if document := model.DocByPath[item.Document]; document != nil {
			view.Href = relativeURL(current, document.OutputPath)
			view.Digest = contentDigest([]byte(document.Content))
		}
		if model.serveRevision != "" && model.taskActionsEnabled {
			view.AgentActions = preparedTaskActions(view.WorkspaceState, len(item.ChildIDs) > 0)
		}
		for _, blocked := range readiness.BlockedBy {
			dependency := TaskWorkspaceDependency{ID: blocked.ID, Status: blocked.Status}
			if source := byID[blocked.ID]; source != nil {
				if document := model.DocByPath[source.Document]; document != nil {
					dependency.Href = relativeURL(current, document.OutputPath)
				}
			}
			view.DependsOn = append(view.DependsOn, dependency)
		}
		for _, issue := range blockingReadinessIssues(readiness.Issues, model.strictPolicy) {
			view.ReadinessIssues = append(view.ReadinessIssues, issue.Message)
		}
		data.Items = append(data.Items, view)
		switch view.WorkspaceState {
		case "in-progress":
			data.Summary.InProgress++
			fallthrough
		case "ready", "ready-candidate", "waiting", "draft", "needs-attention", "blocked":
			data.Summary.Active++
			if view.WorkspaceState == "ready" {
				data.Summary.Ready++
			}
			if view.WorkspaceState == "waiting" {
				data.Summary.Waiting++
			}
			if view.WorkspaceState == "blocked" {
				data.Summary.Blocked++
			}
		}
	}
	sort.SliceStable(data.Items, func(i, j int) bool {
		a, b := data.Items[i], data.Items[j]
		if taskWorkspacePriority(a.Priority) != taskWorkspacePriority(b.Priority) {
			return taskWorkspacePriority(a.Priority) < taskWorkspacePriority(b.Priority)
		}
		return naturalCompare(a.ID, b.ID) < 0
	})
	return data
}

func taskWorkspaceStateLabel(ui frontend.UI, state string) string {
	return ui.Text("work.state." + state)
}

func renderTaskWorkspaceCard(model *Model, item TaskWorkspaceItem) string {
	ui := portalUI(model)
	meta := []string{}
	if item.Type == "bug" && item.Severity != "" {
		meta = append(meta, localizedSemanticValue(ui, "severity", item.Severity))
	}
	if item.Priority != "" {
		meta = append(meta, localizedSemanticValue(ui, "priority", item.Priority))
	}
	if item.Type != "" {
		meta = append(meta, localizedSemanticValue(ui, "type", item.Type))
	}
	state := taskWorkspaceStateLabel(ui, item.WorkspaceState)
	reason := ""
	if item.WorkspaceState == "ready" {
		reason = ui.Text("work.workspace.readyForWork")
	} else if item.WorkspaceState == "waiting" {
		if len(item.DependsOn) == 1 && item.DependsOn[0].Href != "" {
			reason = ui.Text("work.workspace.waitingFor") + ` <a href="` + escapeAttr(item.DependsOn[0].Href) + `">` + escapeHTML(item.DependsOn[0].ID) + `</a>`
		} else {
			reason = escapeHTML(ui.Text("work.workspace.waitingForCount", len(item.DependsOn)))
		}
	} else if item.WorkspaceState == "needs-attention" {
		reason = escapeHTML(ui.Text("work.workspace.contractIncomplete") + " · " + ui.Text("work.workspace.issueCount", item.ReadinessIssueCount))
	} else if item.WorkspaceState == "draft" && item.ReadinessIssueCount > 0 {
		reason = escapeHTML(ui.Text("work.workspace.needsDefinition") + " · " + ui.Text("work.workspace.issueCount", item.ReadinessIssueCount))
	} else if item.WorkspaceState == "draft" && item.ContractComplete {
		reason = escapeHTML(ui.Text("work.workspace.contractComplete"))
	} else if item.WorkspaceState == "ready-candidate" {
		reason = escapeHTML(ui.Text("work.workspace.contractComplete"))
	} else if item.WorkspaceState == "blocked" {
		reason = escapeHTML(truncate(item.Blocker, 130))
	}
	progress := ""
	if item.TotalCriteria > 0 {
		percent := item.CompletedCriteria * 100 / item.TotalCriteria
		progress = `<div class="task-workspace-progress"><span>` + escapeHTML(ui.Text("work.workspace.criteria", item.CompletedCriteria, item.TotalCriteria)) + `</span><div role="progressbar" aria-label="` + escapeAttr(ui.Text("work.workspace.criteria", item.CompletedCriteria, item.TotalCriteria)) + `" aria-valuemin="0" aria-valuemax="100" aria-valuenow="` + fmt.Sprint(percent) + `"><i style="width:` + fmt.Sprint(percent) + `%"></i></div></div>`
	}
	if item.Descendants != nil {
		branch := ui.Text("work.workspace.descendants", item.Descendants.Counts.Done, item.Descendants.Total)
		if item.Descendants.Started {
			branch = ui.Text("work.workspace.branchStarted") + " · " + branch
		}
		progress += `<p class="task-workspace-branch-progress">` + escapeHTML(branch) + `</p>`
	}
	details := []string{}
	if item.ParentID != "" {
		details = append(details, `<li>`+escapeHTML(ui.Text("work.workspace.parent"))+`: <code>`+escapeHTML(item.ParentID)+`</code></li>`)
	}
	if len(item.DependsOn) > 0 {
		values := []string{}
		for _, dependency := range item.DependsOn {
			value := `<code>` + escapeHTML(dependency.ID) + `</code> · ` + escapeHTML(taskWorkspaceStateLabel(ui, dependency.Status))
			if dependency.Href != "" {
				value = `<a href="` + escapeAttr(dependency.Href) + `">` + value + `</a>`
			}
			values = append(values, value)
		}
		details = append(details, `<li>`+escapeHTML(ui.Text("work.workspace.dependencies"))+`: `+strings.Join(values, ", ")+`</li>`)
	}
	if len(item.ChildIDs) > 0 {
		details = append(details, `<li>`+escapeHTML(ui.Text("work.workspace.children", len(item.ChildIDs)))+`</li>`)
	}
	if item.UseCaseID != "" {
		details = append(details, `<li>`+escapeHTML(ui.Text("work.workspace.useCase"))+`: <code>`+escapeHTML(item.UseCaseID)+`</code></li>`)
	}
	if item.Blocker != "" && item.WorkspaceState == "blocked" {
		details = append(details, `<li>`+escapeHTML(item.Blocker)+`</li>`)
	}
	if len(item.ReadinessIssues) > 0 && (item.WorkspaceState == "needs-attention" || item.WorkspaceState == "draft") {
		for _, issue := range item.ReadinessIssues {
			details = append(details, `<li>`+escapeHTML(issue)+`</li>`)
		}
	}
	disclosure := ""
	if len(details) > 0 {
		disclosure = `<details class="task-workspace-details"><summary>` + escapeHTML(ui.Text("work.workspace.details")) + `</summary><ul>` + strings.Join(details, "") + `</ul></details>`
	}
	module := ""
	if item.ModuleID != "" {
		module = `<code>` + escapeHTML(item.ModuleID) + `</code>`
	}
	return `<article class="task-workspace-card" data-task-workspace-item data-task-id="` + escapeAttr(item.ID) + `" data-state="` + escapeAttr(item.WorkspaceState) + `"><div class="task-workspace-card-head"><a href="` + escapeAttr(item.Href) + `"><code>` + escapeHTML(item.ID) + `</code></a><span class="task-workspace-state">` + escapeHTML(state) + `</span></div><h3><a href="` + escapeAttr(item.Href) + `">` + escapeHTML(item.Title) + `</a></h3><p class="task-workspace-meta">` + escapeHTML(strings.Join(meta, " · ")) + `</p><p class="task-workspace-reason">` + reason + `</p>` + progress + module + disclosure + renderTaskAgentActions(model, item) + `</article>`
}

func renderTaskAgentActions(model *Model, item TaskWorkspaceItem) string {
	if len(item.AgentActions) == 0 {
		return ""
	}
	rows := ""
	for _, action := range item.AgentActions {
		rows += `<div class="task-agent-action" data-task-agent-action="` + escapeAttr(action.ID) + `"><span>` + escapeHTML(portalUI(model).Text("work.agent."+action.ID)) + `</span></div>`
	}
	return `<div class="task-agent-actions" data-task-actions data-task-id="` + escapeAttr(item.ID) + `" data-task-digest="` + escapeAttr(item.Digest) + `">` + rows + `</div>`
}

func renderTaskPageActions(model *Model, document *Document) string {
	if !model.serveMode || !model.taskActionsEnabled {
		return ""
	}
	id := stableDocumentID(model, document.SourcePath)
	for _, item := range buildTaskWorkspaceData(model, document.OutputPath).Items {
		if item.ID == id {
			ui := portalUI(model)
			return `<section class="task-page-actions" aria-label="` + escapeAttr(ui.Text("work.agent.actions")) + `">` + renderTaskAgentActions(model, item) + `</section>`
		}
	}
	return ""
}

func renderTaskWorkspaceBoard(model *Model, data TaskWorkspaceData, states []string) string {
	ui := portalUI(model)
	columns := ""
	for _, state := range states {
		cards := ""
		count := 0
		for _, item := range data.Items {
			if item.WorkspaceState == state {
				cards += renderTaskWorkspaceCard(model, item)
				count++
			}
		}
		if count > 0 || state != "needs-attention" {
			columns += `<section class="task-workspace-column" data-workspace-column="` + state + `"><h2>` + escapeHTML(taskWorkspaceStateLabel(ui, state)) + ` <span>` + fmt.Sprint(count) + `</span></h2><div>` + cards + `</div></section>`
		}
	}
	return `<section class="task-workspace-board" data-workspace-board>` + columns + `</section>`
}

func renderTaskWorkspaceList(model *Model, data TaskWorkspaceData, states map[string]bool) string {
	ui := portalUI(model)
	rows := ""
	for _, item := range data.Items {
		if !states[item.WorkspaceState] {
			continue
		}
		progress := "—"
		if item.TotalCriteria > 0 {
			progress = ui.Text("work.workspace.criteria", item.CompletedCriteria, item.TotalCriteria)
		}
		deps := "—"
		if len(item.DependsOn) > 0 {
			deps = strings.Join(itemDependencyIDs(item.DependsOn), ", ")
		}
		rows += `<tr data-task-workspace-item data-task-id="` + escapeAttr(item.ID) + `" data-state="` + escapeAttr(item.WorkspaceState) + `"><td><a href="` + escapeAttr(item.Href) + `"><code>` + escapeHTML(item.ID) + `</code></a></td><td><a href="` + escapeAttr(item.Href) + `">` + escapeHTML(item.Title) + `</a></td><td>` + escapeHTML(taskWorkspaceStateLabel(ui, item.WorkspaceState)) + `</td><td>` + escapeHTML(item.Priority) + `</td><td>` + escapeHTML(item.Type) + `</td><td><code>` + escapeHTML(item.ModuleID) + `</code></td><td>` + escapeHTML(progress) + `</td><td>` + escapeHTML(deps) + renderTaskAgentActions(model, item) + `</td></tr>`
	}
	return `<div class="data-table task-workspace-list"><table><thead><tr><th>` + escapeHTML(ui.Text("work.workspace.id")) + `</th><th>` + escapeHTML(ui.Text("work.workspace.titleColumn")) + `</th><th>` + escapeHTML(ui.Text("work.workspace.state")) + `</th><th>` + escapeHTML(ui.Text("work.priority")) + `</th><th>` + escapeHTML(ui.Text("work.workspace.type")) + `</th><th>` + escapeHTML(ui.Text("work.module")) + `</th><th>` + escapeHTML(ui.Text("work.workspace.progress")) + `</th><th>` + escapeHTML(ui.Text("work.workspace.dependencies")) + `</th></tr></thead><tbody>` + rows + `</tbody></table></div>`
}

func itemDependencyIDs(items []TaskWorkspaceDependency) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.ID)
	}
	return values
}

func renderTaskWorkspaceTree(model *Model, data TaskWorkspaceData) string {
	ui := portalUI(model)
	byParent := map[string][]TaskWorkspaceItem{}
	for _, item := range data.Items {
		byParent[item.ParentID] = append(byParent[item.ParentID], item)
	}
	var render func(string) string
	render = func(parent string) string {
		out := ""
		for _, item := range byParent[parent] {
			children := render(item.ID)
			toggle := ""
			if children != "" {
				toggle = `<button type="button" data-task-tree-toggle aria-expanded="true" aria-label="` + escapeAttr(ui.Text("work.workspace.details")) + `"></button>`
			}
			priority := ""
			if item.Priority != "" {
				priority = ` · ` + escapeHTML(item.Priority)
			}
			out += `<li data-task-workspace-item data-task-id="` + escapeAttr(item.ID) + `" data-state="` + escapeAttr(item.WorkspaceState) + `">` + toggle + `<a href="` + escapeAttr(item.Href) + `"><code>` + escapeHTML(item.ID) + `</code> ` + escapeHTML(item.Title) + `</a><span>` + escapeHTML(taskWorkspaceStateLabel(ui, item.WorkspaceState)) + priority + `</span>` + renderTaskAgentActions(model, item)
			if children != "" {
				out += `<ul>` + children + `</ul>`
			}
			out += `</li>`
		}
		return out
	}
	return `<section class="task-workspace-tree" data-workspace-tree><ul>` + render("") + `</ul></section>`
}

func renderTaskWorkspacePage(model *Model) string {
	ui, current := portalUI(model), "work/index.html"
	data := buildTaskWorkspaceData(model, current)
	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	quick := func(key string, count int, states string) string {
		return `<button type="button" data-workspace-quick-filter="` + states + `">` + escapeHTML(ui.Text(key)) + ` <strong>` + fmt.Sprint(count) + `</strong></button>`
	}
	currentWork := ""
	inProgress := []TaskWorkspaceItem{}
	for _, item := range data.Items {
		if item.WorkspaceState == "in-progress" {
			inProgress = append(inProgress, item)
		}
	}
	if len(inProgress) > 0 {
		cards := ""
		for _, item := range inProgress {
			cards += renderTaskWorkspaceCard(model, item)
		}
		currentWork = `<section class="task-workspace-current"><h2>` + escapeHTML(ui.Text("work.current.title")) + `</h2><div>` + cards + `</div></section>`
	}
	filters := renderTaskWorkspaceFilters(ui, data)
	views := `<nav class="task-workspace-views" aria-label="` + escapeAttr(ui.Text("work.workspace.views")) + `"><button type="button" data-workspace-view="board" aria-current="page">` + escapeHTML(ui.Text("work.view.board")) + `</button><button type="button" data-workspace-view="list">` + escapeHTML(ui.Text("work.view.list")) + `</button><button type="button" data-workspace-view="tree">` + escapeHTML(ui.Text("work.view.tree")) + `</button></nav>`
	active := map[string]bool{"in-progress": true, "ready": true, "ready-candidate": true, "waiting": true, "needs-attention": true, "draft": true, "blocked": true}
	done := map[string]bool{"done": true}
	cancelled := map[string]bool{"cancelled": true}
	archive := map[string]bool{"archive": true}
	empty := ""
	if len(data.Items) == 0 {
		empty = `<p class="empty-state">` + escapeHTML(ui.Text("work.workspace.empty")) + `</p>`
	}
	content := breadcrumbs(model, current, modelDirectoryLabel(model, "work")) + `<header class="page-header task-workspace-header"><h1>` + escapeHTML(ui.Text("work.workspace.title")) + `</h1><div class="task-workspace-quick">` + quick("work.filter.allActive", data.Summary.Active, "in-progress,ready,ready-candidate,waiting,draft,needs-attention,blocked") + quick("work.filter.inProgress", data.Summary.InProgress, "in-progress") + quick("work.filter.ready", data.Summary.Ready, "ready") + quick("work.filter.waiting", data.Summary.Waiting, "waiting") + quick("work.filter.blocked", data.Summary.Blocked, "blocked") + `</div></header>` + empty + currentWork + `<div class="task-workspace-toolbar"><p data-task-agent-setup hidden></p>` + views + filters + `</div><div data-workspace-view-panel="board">` + renderTaskWorkspaceBoard(model, data, []string{"in-progress", "ready", "ready-candidate", "waiting", "needs-attention", "draft", "blocked"}) + `</div><div data-workspace-view-panel="list" hidden>` + renderTaskWorkspaceList(model, data, active) + `</div><div data-workspace-view-panel="tree" hidden>` + renderTaskWorkspaceTree(model, data) + `</div><p class="empty-state" data-workspace-empty hidden>` + escapeHTML(ui.Text("work.workspace.emptyFiltered")) + ` <button class="toolbar-button" type="button" data-workspace-reset>` + escapeHTML(ui.Text("work.filter.reset")) + `</button></p><section class="task-workspace-terminal">` + renderTaskWorkspaceSection(model, data, "done", done) + renderTaskWorkspaceSection(model, data, "cancelled", cancelled) + renderTaskWorkspaceArchive(model, data, archive) + `</section><script id="task-workspace-data" type="application/json">` + string(jsonData) + `</script>`
	return pageShell(model, current, ui.Text("work.workspace.title"), ui.Text("work.workspace.description"), content, "")
}

func renderTaskWorkspaceFilters(ui frontend.UI, data TaskWorkspaceData) string {
	return `<section class="task-workspace-controls" data-workspace-controls><label class="task-workspace-search">` + escapeHTML(ui.Text("work.workspace.search")) + `<input type="search" data-workspace-query placeholder="` + escapeAttr(ui.Text("work.workspace.searchPlaceholder")) + `"></label><div class="task-workspace-filter-grid">` + taskWorkspaceSelect(ui, "state", []string{"in-progress", "ready", "ready-candidate", "waiting", "needs-attention", "draft", "blocked", "done", "cancelled", "archive"}) + taskWorkspaceSelect(ui, "priority", []string{"urgent", "high", "normal", "low"}) + taskWorkspaceSelect(ui, "type", []string{"feature", "bug", "maintenance", "documentation", "research"}) + `<label>` + escapeHTML(ui.Text("work.module")) + `<select data-workspace-filter="module"><option value="">` + escapeHTML(ui.Text("work.filter.all")) + `</option>` + taskWorkspaceOptions(data, func(item TaskWorkspaceItem) string { return item.ModuleID }) + `</select></label><label>` + escapeHTML(ui.Text("work.workspace.parent")) + `<select data-workspace-filter="parent"><option value="">` + escapeHTML(ui.Text("work.filter.all")) + `</option>` + taskWorkspaceOptions(data, func(item TaskWorkspaceItem) string { return item.ParentID }) + `</select></label></div><div class="task-workspace-filter-actions"><label><input type="checkbox" data-workspace-filter="completed"> <span>` + escapeHTML(ui.Text("work.filter.completed")) + `</span></label><label><input type="checkbox" data-workspace-filter="archive"> <span>` + escapeHTML(ui.Text("work.filter.archive")) + `</span></label><button class="toolbar-button" type="button" data-workspace-reset>` + escapeHTML(ui.Text("work.filter.reset")) + `</button></div></section>`
}

func taskWorkspaceSelect(ui frontend.UI, name string, values []string) string {
	options := `<option value="">` + escapeHTML(ui.Text("work.filter.all")) + `</option>`
	for _, value := range values {
		options += `<option value="` + escapeAttr(value) + `">` + escapeHTML(taskWorkspaceStateLabel(ui, value)) + `</option>`
	}
	label := ui.Text("work.filter." + name)
	if name == "type" {
		label = ui.Text("work.workspace.type")
	}
	return `<label>` + escapeHTML(label) + `<select data-workspace-filter="` + escapeAttr(name) + `">` + options + `</select></label>`
}
func taskWorkspaceOptions(data TaskWorkspaceData, value func(TaskWorkspaceItem) string) string {
	values := map[string]bool{}
	for _, item := range data.Items {
		if v := value(item); v != "" {
			values[v] = true
		}
	}
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Slice(keys, func(i, j int) bool { return naturalCompare(keys[i], keys[j]) < 0 })
	out := ""
	for _, value := range keys {
		out += `<option value="` + escapeAttr(value) + `">` + escapeHTML(value) + `</option>`
	}
	return out
}
func renderTaskWorkspaceSection(model *Model, data TaskWorkspaceData, state string, states map[string]bool) string {
	count := 0
	for _, item := range data.Items {
		if item.WorkspaceState == state {
			count++
		}
	}
	if count == 0 {
		return ""
	}
	ui := portalUI(model)
	return `<details class="task-workspace-terminal-section"><summary>` + escapeHTML(taskWorkspaceStateLabel(ui, state)) + ` (` + fmt.Sprint(count) + `)</summary>` + renderTaskWorkspaceList(model, data, states) + `</details>`
}

func renderTaskWorkspaceArchive(model *Model, data TaskWorkspaceData, states map[string]bool) string {
	groups := map[string]TaskWorkspaceData{}
	for _, item := range data.Items {
		if item.WorkspaceState != "archive" {
			continue
		}
		year := item.ArchiveYear
		if year == "" {
			year = "—"
		}
		group := groups[year]
		group.Items = append(group.Items, item)
		groups[year] = group
	}
	if len(groups) == 0 {
		return ""
	}
	years := make([]string, 0, len(groups))
	for year := range groups {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(years)))
	ui := portalUI(model)
	body := ""
	for _, year := range years {
		body += `<h3>` + escapeHTML(year) + ` (` + fmt.Sprint(len(groups[year].Items)) + `)</h3>` + renderTaskWorkspaceList(model, groups[year], states)
	}
	count := 0
	for _, group := range groups {
		count += len(group.Items)
	}
	return `<details class="task-workspace-terminal-section"><summary>` + escapeHTML(taskWorkspaceStateLabel(ui, "archive")) + ` (` + fmt.Sprint(count) + `)</summary>` + body + `</details>`
}
