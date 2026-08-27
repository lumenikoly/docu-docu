package toudocu

import (
	"fmt"
	"strings"
)

func processTitle(id, title string) string {
	for _, separator := range []string{":", "—"} {
		if strings.HasPrefix(title, id+separator) {
			return strings.TrimSpace(strings.TrimPrefix(title, id+separator))
		}
	}
	return title
}

func findKnowledgeFlow(model *Model, id string) *KnowledgeFlow {
	for index := range model.Knowledge.Flows {
		if model.Knowledge.Flows[index].ID == id {
			return &model.Knowledge.Flows[index]
		}
	}
	return nil
}

func findPlayableFlow(model *Model, id string) *PlayableFlow {
	for index := range model.Knowledge.PlayableFlows {
		if model.Knowledge.PlayableFlows[index].UseCaseID == id {
			return &model.Knowledge.PlayableFlows[index]
		}
	}
	return nil
}

func processModuleOptions(model *Model) string {
	var options strings.Builder
	for _, module := range model.Knowledge.Modules {
		_, _ = fmt.Fprintf(&options, `<option value="%s">%s</option>`,
			escapeAttr(module.ID), escapeHTML(module.ID+" · "+processTitle(module.ID, module.Title)))
	}
	return options.String()
}

func processUseCaseOptions(model *Model) string {
	var options strings.Builder
	for _, useCase := range model.Knowledge.UseCases {
		_, _ = fmt.Fprintf(&options, `<option value="%s">%s</option>`,
			escapeAttr(useCase.ID), escapeHTML(useCase.ID+" · "+screenTitleForUseCase(useCase)))
	}
	return options.String()
}

func processRelations(current string, model *Model, ids []string, kind string) string {
	if len(ids) == 0 {
		return `<span class="process-empty-value">—</span>`
	}
	var values strings.Builder
	for _, id := range ids {
		var document *Document
		if kind == "use-case" {
			if useCase := findUseCase(model, id); useCase != nil {
				document = model.DocByPath[useCase.Document]
			}
		} else if flow := findKnowledgeFlow(model, id); flow != nil {
			document = model.DocByPath[flow.Document]
		}
		label := `<span class="process-relation-chip">` + escapeHTML(id) + `</span>`
		if document != nil {
			label = `<a class="process-relation-chip" href="` + escapeAttr(relativeURL(current, document.OutputPath)) + `">` + escapeHTML(id) + `</a>`
		}
		values.WriteString(label)
	}
	return `<div class="process-relations">` + values.String() + `</div>`
}

func renderProcessRows(model *Model, current, onlyType string) string {
	ui := portalUI(model)
	var rows strings.Builder
	if onlyType == "" || onlyType == "use-case" {
		for _, useCase := range model.Knowledge.UseCases {
			document := model.DocByPath[useCase.Document]
			if document == nil {
				continue
			}
			views := `<a href="` + escapeAttr(relativeURL(current, document.OutputPath)+"#overview") + `">` + escapeHTML(ui.Text("process.overview")) + `</a>`
			if flow := findPlayableFlow(model, useCase.ID); flow != nil {
				views += `<a href="` + escapeAttr(relativeURL(current, document.OutputPath)+"#map") + `">` + escapeHTML(ui.Text("process.map")) + `</a>`
				if flow.Valid {
					views += `<a href="` + escapeAttr(relativeURL(current, document.OutputPath)+"#play") + `">` + escapeHTML(ui.Text("process.play")) + `</a>`
				}
			}
			search := strings.Join([]string{useCase.ID, useCase.Title, useCase.ModuleID, strings.Join(useCase.FlowIDs, " ")}, " ")
			_, _ = fmt.Fprintf(&rows, `<tr data-filter-item data-search="%s" data-type="use-case" data-module="%s" data-status="%s" data-usecase="%s"><td><a class="process-title" href="%s">%s</a><code>%s</code></td><td><span class="process-kind process-kind-user">%s</span></td><td><code>%s</code></td><td>%s</td><td>%s</td><td><div class="process-view-links">%s</div></td></tr>`,
				escapeAttr(search), escapeAttr(useCase.ModuleID), escapeAttr(useCase.Status.Kind), escapeAttr(useCase.ID),
				escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(screenTitleForUseCase(useCase)), escapeHTML(useCase.ID),
				escapeHTML(ui.Text("process.useCase")), escapeHTML(useCase.ModuleID), renderStatusChip(model, useCase.Status), processRelations(current, model, useCase.FlowIDs, "flow"), views)
		}
	}
	if onlyType == "" || onlyType == "flow" {
		for _, flow := range model.Knowledge.Flows {
			document := model.DocByPath[flow.Document]
			if document == nil {
				continue
			}
			search := strings.Join([]string{flow.ID, flow.Title, flow.ModuleID, strings.Join(flow.UseCaseIDs, " ")}, " ")
			_, _ = fmt.Fprintf(&rows, `<tr data-filter-item data-search="%s" data-type="flow" data-module="%s" data-status="%s" data-usecase="%s"><td><a class="process-title" href="%s">%s</a><code>%s</code></td><td><span class="process-kind process-kind-system">%s</span></td><td><code>%s</code></td><td>%s</td><td>%s</td><td><div class="process-view-links"><a href="%s">%s</a></div></td></tr>`,
				escapeAttr(search), escapeAttr(flow.ModuleID), escapeAttr(document.Status.Kind), escapeAttr(strings.Join(flow.UseCaseIDs, "|")),
				escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(processTitle(flow.ID, flow.Title)), escapeHTML(flow.ID),
				escapeHTML(ui.Text("process.visual")), escapeHTML(flow.ModuleID), renderStatusChip(model, document.Status), processRelations(current, model, flow.UseCaseIDs, "use-case"),
				escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(ui.Text("process.diagram")))
		}
	}
	return rows.String()
}

func renderProcessCatalogPage(model *Model, current, onlyType string) string {
	ui := portalUI(model)
	title := modelDirectoryLabel(model, "flows")
	description := ui.Text("process.description")
	badge := title
	badgeTarget := sectionCatalogOutput(SectionFlows)
	summary := ui.Text("process.flowSummary")
	if onlyType == "use-case" {
		title = ui.Text("process.useCases")
		description = ui.Text("process.useCaseDescription")
		badge = ui.Text("process.useCases")
		badgeTarget = "use-cases/index.html"
		summary = ui.Text("process.useCaseSummary")
	}
	rows := renderProcessRows(model, current, onlyType)
	content := breadcrumbs(model, current, title) +
		`<header class="page-header"><div class="page-kicker"><a class="badge" href="` + escapeAttr(relativeURL(current, badgeTarget)) + `">` + escapeHTML(badge) + `</a></div><h1>` +
		escapeHTML(title) + `</h1><p class="page-lead">` + escapeHTML(description) + `</p></header>` +
		`<section class="process-catalog" data-filter-scope><div class="process-filterbar">` +
		`<label class="screen-filter-field process-filter-search"><span>` + escapeHTML(ui.Text("process.search")) + `</span><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("process.searchPlaceholder")) + `"></label>` +
		`<label class="screen-filter-field"><span>` + escapeHTML(ui.Text("process.module")) + `</span><select data-filter-control="module"><option value="all">` + escapeHTML(ui.Text("process.allModules")) + `</option>` + processModuleOptions(model) + `</select></label>` +
		`<label class="screen-filter-field"><span>` + escapeHTML(ui.Text("process.relatedUseCase")) + `</span><select data-filter-control="usecase"><option value="all">` + escapeHTML(ui.Text("process.allUseCases")) + `</option>` + processUseCaseOptions(model) + `</select></label>` +
		`<button class="toolbar-button screen-filter-reset" type="button" data-filter-reset>` + escapeHTML(ui.Text("process.reset")) + `</button></div>` +
		`<div class="screen-catalog-summary" aria-live="polite"><span>` + escapeHTML(ui.Text("process.found")) + ` <strong data-filter-count></strong></span><span>` + escapeHTML(summary) + `</span></div>` +
		`<div class="process-table"><table><thead><tr><th scope="col">` + escapeHTML(ui.Text("process.process")) + `</th><th scope="col">` + escapeHTML(ui.Text("process.type")) + `</th><th scope="col">` + escapeHTML(ui.Text("process.module")) + `</th><th scope="col">` + escapeHTML(ui.Text("process.status")) + `</th><th scope="col">` + escapeHTML(ui.Text("process.relations")) + `</th><th scope="col">` + escapeHTML(ui.Text("process.views")) + `</th></tr></thead><tbody>` +
		rows + `</tbody></table></div><div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("process.none")) + `</div></section>`
	return pageShell(model, current, title, description, content, "")
}

func renderUseCaseRelations(model *Model, useCase KnowledgeUseCase, current string) string {
	ui := portalUI(model)
	var module, screens, activeTasks, archivedTasks, repositoryPaths, traceability strings.Builder
	for _, candidate := range model.Knowledge.Modules {
		if candidate.ID != useCase.ModuleID {
			continue
		}
		if document := model.DocByPath[candidate.Document]; document != nil {
			_, _ = fmt.Fprintf(&module, `<li><a href="%s"><code>%s</code> · %s</a></li>`,
				escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(candidate.ID), escapeHTML(processTitle(candidate.ID, candidate.Title)))
		}
	}
	for _, screenID := range useCase.ScreenIDs {
		for _, screen := range model.Knowledge.Screens {
			if screen.ID != screenID {
				continue
			}
			if document := model.DocByPath[screen.Document]; document != nil {
				_, _ = fmt.Fprintf(&screens, `<li><a href="%s"><code>%s</code> · %s</a></li>`,
					escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(screen.ID), escapeHTML(screen.Title))
			}
		}
	}
	for _, item := range model.Knowledge.WorkItems {
		if item.UseCaseID == useCase.ID {
			if document := model.DocByPath[item.Document]; document != nil {
				rows := &activeTasks
				if item.Archived {
					rows = &archivedTasks
				}
				_, _ = fmt.Fprintf(rows, `<li><a href="%s"><code>%s</code> · %s</a>`,
					escapeAttr(relativeURL(current, document.OutputPath)), escapeHTML(item.ID), escapeHTML(item.Title))
				if item.Archived {
					_, _ = fmt.Fprintf(rows, ` <span class="badge">%s</span>`, escapeHTML(ui.Text("archive.year", item.ArchiveYear)))
				}
				rows.WriteString(`</li>`)
			}
		}
	}
	tasks := activeTasks.String() + archivedTasks.String()
	for _, repositoryPath := range useCase.RepositoryPaths {
		writeStrings(&repositoryPaths, `<li><code>`, escapeHTML(repositoryPath), `</code></li>`)
	}
	for _, row := range model.Knowledge.Traceability {
		if row.UseCaseID == useCase.ID {
			writeStrings(&traceability, `<tr><td><code>`, escapeHTML(row.TransitionID), `</code></td><td>`, escapeHTML(row.TaskID), `</td><td>`, escapeHTML(row.CriterionID), `</td><td>`, escapeHTML(row.Verification), `</td></tr>`)
		}
	}
	processLinks := processRelations(current, model, useCase.FlowIDs, "flow")
	return `<div class="usecase-relations-grid"><section><h2>` + escapeHTML(ui.Text("process.relatedFlows")) + `</h2>` + processLinks +
		`</section><section><h2>` + escapeHTML(ui.Text("process.module")) + `</h2><ul class="related-list">` + fallbackList(model, module.String()) +
		`</ul></section><section><h2>` + escapeHTML(ui.Text("process.screens")) + `</h2><ul class="related-list">` + fallbackList(model, screens.String()) +
		`</ul></section><section><h2>` + escapeHTML(ui.Text("process.work")) + `</h2><ul class="related-list">` + fallbackList(model, tasks) +
		`</ul></section><section><h2>` + escapeHTML(ui.Text("process.code")) + `</h2><ul class="related-list">` + fallbackList(model, repositoryPaths.String()) +
		`</ul></section></div><section class="dashboard-section"><h2>` + escapeHTML(ui.Text("process.traceability")) + `</h2><div class="data-table"><table><thead><tr><th>` + escapeHTML(ui.Text("process.transition")) + `</th><th>` + escapeHTML(ui.Text("process.task")) + `</th><th>` + escapeHTML(ui.Text("process.criterion")) + `</th><th>` + escapeHTML(ui.Text("process.verification")) + `</th></tr></thead><tbody>` +
		fallbackTraceability(model, traceability.String()) + `</tbody></table></div></section>`
}

func fallbackList(model *Model, value string) string {
	if value == "" {
		return `<li class="process-empty-value">` + escapeHTML(portalUI(model).Text("process.noRelations")) + `</li>`
	}
	return value
}

func fallbackTraceability(model *Model, value string) string {
	if value == "" {
		return `<tr><td colspan="4" class="process-empty-value">` + escapeHTML(portalUI(model).Text("process.noCriteria")) + `</td></tr>`
	}
	return value
}

func renderUseCasePage(model *Model, document *Document) string {
	ui := portalUI(model)
	useCase := findUseCase(model, document.Metadata["id"])
	if useCase == nil {
		return renderDocumentPage(model, document)
	}
	current := document.OutputPath
	body := renderDocumentBody(model, document, linkResolverFor(model, document), nil)
	flow := findPlayableFlow(model, useCase.ID)
	mapPanel := `<div class="empty-state"><strong>` + escapeHTML(ui.Text("process.mapUnavailable")) + `</strong><p>` + escapeHTML(ui.Text("process.mapUnavailableHelp")) + `</p></div>`
	playPanel := `<div class="empty-state"><strong>` + escapeHTML(ui.Text("process.playUnavailable")) + `</strong><p>` + escapeHTML(ui.Text("process.playUnavailableHelp")) + `</p></div>`
	if flow != nil && len(model.Knowledge.Screens) > 0 {
		mapPanel = renderScreenMapWorkspace(model, current, useCase.ID, true)
		playPanel = renderPlayableFlowComponent(model, *flow, current, true)
	}
	tabs := []struct {
		id    string
		label string
	}{
		{"overview", ui.Text("process.overview")},
		{"map", ui.Text("process.map")},
		{"play", ui.Text("process.play")},
		{"links", ui.Text("process.links")},
	}
	var tabLinks strings.Builder
	for index, tab := range tabs {
		selected := "false"
		tabIndex := `-1`
		active := ""
		if index == 0 {
			selected = "true"
			tabIndex = "0"
			active = " is-active"
		}
		_, _ = fmt.Fprintf(&tabLinks, `<a class="usecase-tab%s" id="tab-%s" role="tab" aria-selected="%s" aria-controls="%s" tabindex="%s" href="#%s" data-usecase-tab="%s">%s</a>`,
			active, tab.id, selected, tab.id, tabIndex, tab.id, tab.id, tab.label)
	}
	content := breadcrumbs(model, current, useCase.ID) +
		`<header class="page-header usecase-header"><div class="page-kicker">` + renderStatusChip(model, useCase.Status) + `<span class="badge">` + escapeHTML(ui.Text("process.useCase")) + `</span></div><h1>` +
		escapeHTML(useCase.ID+" · "+screenTitleForUseCase(*useCase)) + `</h1><p class="page-lead">` + escapeHTML(document.Description) + `</p>` +
		renderMetadata(model, document) + `<div class="page-actions">` + renderDocumentContextButton(model, document) + `</div></header><div class="usecase-workspace" data-usecase-tabs><nav class="usecase-tabs" role="tablist" aria-label="` + escapeAttr(ui.Text("process.viewsAria")) + `">` +
		tabLinks.String() + `</nav><section class="usecase-panel" id="overview" role="tabpanel" aria-labelledby="tab-overview" data-usecase-panel><article class="doc-content">` + body +
		`</article></section><section class="usecase-panel" id="map" role="tabpanel" aria-labelledby="tab-map" data-usecase-panel>` + mapPanel +
		`</section><section class="usecase-panel" id="play" role="tabpanel" aria-labelledby="tab-play" data-usecase-panel>` + playPanel +
		`</section><section class="usecase-panel" id="links" role="tabpanel" aria-labelledby="tab-links" data-usecase-panel>` +
		renderUseCaseRelations(model, *useCase, current) + renderRelated(model, document) + `</section></div>`
	return pageShell(model, current, document.Title, document.Description, content, "")
}

func renderFlowConnections(model *Model, document *Document) string {
	flow := findKnowledgeFlow(model, document.Metadata["id"])
	if flow == nil {
		return ""
	}
	return `<section class="dashboard-section dashboard-support-panel flow-connections"><h2>` + escapeHTML(portalUI(model).Text("process.relatedUseCases")) + `</h2>` +
		processRelations(document.OutputPath, model, flow.UseCaseIDs, "use-case") + `</section>`
}
