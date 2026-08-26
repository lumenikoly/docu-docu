package toudocu

import (
	"encoding/json"
	"fmt"
	"strings"
)

type screenMapPayload struct {
	Screens     []KnowledgeScreen  `json:"screens"`
	Transitions []ScreenTransition `json:"transitions"`
	Flows       []PlayableFlow     `json:"flows"`
	Hotspots    []Hotspot          `json:"hotspots"`
	Modules     map[string]string  `json:"modules"`
	ScreenURLs  map[string]string  `json:"screenUrls"`
	FlowURLs    map[string]string  `json:"flowUrls"`
}

func screenMapData(model *Model, current string) screenMapPayload {
	ui := portalUI(model)
	modules := map[string]string{}
	for _, module := range model.Knowledge.Modules {
		modules[module.ID] = module.Title
	}
	screenURLs := map[string]string{}
	for _, screen := range model.Knowledge.Screens {
		if document := model.DocByPath[screen.Document]; document != nil {
			screenURLs[screen.ID] = relativeURL(current, document.OutputPath)
		}
	}
	flowURLs := map[string]string{}
	for _, flow := range model.Knowledge.PlayableFlows {
		if useCase := findUseCase(model, flow.UseCaseID); useCase != nil {
			if document := model.DocByPath[useCase.Document]; document != nil {
				flowURLs[flow.UseCaseID] = relativeURL(current, document.OutputPath) + "#play"
			}
		}
	}
	screens := make([]KnowledgeScreen, len(model.Knowledge.Screens))
	copy(screens, model.Knowledge.Screens)
	for index := range screens {
		if label := ui.Text("status." + screens[index].Status.Kind); !strings.HasPrefix(label, "status.") {
			screens[index].Status.Label = label
		}
		if screens[index].Preview != "" {
			screens[index].Preview = relativeURL(current, screens[index].Preview)
		}
		screens[index].States = append([]ScreenState{}, screens[index].States...)
		for stateIndex := range screens[index].States {
			if screens[index].States[stateIndex].Preview != "" {
				screens[index].States[stateIndex].Preview = relativeURL(current, screens[index].States[stateIndex].Preview)
			}
		}
	}
	return screenMapPayload{
		Screens: screens, Transitions: model.Knowledge.Transitions,
		Flows: model.Knowledge.PlayableFlows, Hotspots: model.Knowledge.Hotspots,
		Modules: modules, ScreenURLs: screenURLs, FlowURLs: flowURLs,
	}
}

func jsonScript(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func screenModuleOptions(model *Model) string {
	var options strings.Builder
	seen := map[string]bool{}
	for _, screen := range model.Knowledge.Screens {
		if seen[screen.ModuleID] {
			continue
		}
		seen[screen.ModuleID] = true
		label := screen.ModuleID
		for _, module := range model.Knowledge.Modules {
			if module.ID == screen.ModuleID && module.Title != "" {
				label += " · " + module.Title
				break
			}
		}
		writeStrings(&options, `<option value="`, escapeAttr(screen.ModuleID), `">`, escapeHTML(label), `</option>`)
	}
	return options.String()
}

func screenStatusOptions(model *Model) string {
	ui := portalUI(model)
	seen := map[string]string{}
	var options strings.Builder
	for _, screen := range model.Knowledge.Screens {
		if _, exists := seen[screen.Status.Kind]; exists {
			continue
		}
		label := ui.Text("status." + screen.Status.Kind)
		if strings.HasPrefix(label, "status.") {
			label = screen.Status.Label
		}
		seen[screen.Status.Kind] = label
		writeStrings(&options, `<option value="`, escapeAttr(screen.Status.Kind), `">`, escapeHTML(label), `</option>`)
	}
	return options.String()
}

func screenUseCaseOptions(model *Model, selectedValues ...string) string {
	selected := ""
	if len(selectedValues) > 0 {
		selected = selectedValues[0]
	}
	var options strings.Builder
	for _, flow := range model.Knowledge.PlayableFlows {
		title := flow.UseCaseID
		for _, useCase := range model.Knowledge.UseCases {
			if useCase.ID == flow.UseCaseID {
				title += " · " + screenTitleForUseCase(useCase)
				break
			}
		}
		selectedAttribute := ""
		if flow.UseCaseID == selected {
			selectedAttribute = ` selected`
		}
		writeStrings(&options, `<option value="`, escapeAttr(flow.UseCaseID), `"`, selectedAttribute, `>`, escapeHTML(title), `</option>`)
	}
	return options.String()
}

func screenTitleForUseCase(useCase KnowledgeUseCase) string {
	for _, separator := range []string{":", "—"} {
		if strings.HasPrefix(useCase.Title, useCase.ID+separator) {
			return strings.TrimSpace(strings.TrimPrefix(useCase.Title, useCase.ID+separator))
		}
	}
	return useCase.Title
}

func screenMapTarget(model *Model) string {
	if model.ScreenMapEnabled {
		return "screens/index.html"
	}
	return "screens/catalog.html"
}

func previewHTML(model *Model, current string, screen KnowledgeScreen, className string) string {
	ui := portalUI(model)
	if screen.Preview == "" {
		return `<div class="screen-preview-placeholder ` + className + `"><strong>` + escapeHTML(screen.ID) + `</strong><span>` + escapeHTML(ui.Text("screen.noPreview")) + `</span></div>`
	}
	return `<img class="` + className + `" src="` + escapeAttr(relativeURL(current, screen.Preview)) + `" alt="` + escapeAttr(ui.Text("screen.previewAlt", screen.Title)) + `" loading="lazy">`
}

func screenCardHTML(model *Model, current string, screen KnowledgeScreen) string {
	ui := portalUI(model)
	route := ui.Text("screen.noRoute")
	if screen.Route != "" {
		route = screen.Route
	}
	incoming := len(screen.IncomingTransitionIDs)
	outgoing := len(screen.OutgoingTransitionIDs)
	return `<article class="screen-node" data-screen-node="` + escapeAttr(screen.ID) + `" data-module="` + escapeAttr(screen.ModuleID) +
		`" data-status="` + escapeAttr(screen.Status.Kind) + `" tabindex="0" role="button" aria-label="` +
		escapeAttr(ui.Text("screen.cardAria", screen.ID, screen.Title, incoming, outgoing)) + `">` + previewHTML(model, current, screen, "screen-node-preview") +
		`<div class="screen-node-copy"><strong>` + escapeHTML(screen.ID) + `</strong><span>` + escapeHTML(screen.Title) +
		`</span><small>` + escapeHTML(route) + `</small><div class="screen-node-meta">` + renderStatusChip(model, screen.Status) +
		`<span class="screen-module-label">` + escapeHTML(screen.ModuleID) + `</span></div>` +
		`<div class="screen-node-transition-counts" aria-label="` + escapeAttr(ui.Text("screen.transitionAria", incoming, outgoing)) + `">` +
		`<span title="` + escapeAttr(ui.Text("screen.incomingTransitions")) + `">` + fmt.Sprint(incoming) + ` ` + escapeHTML(ui.Text("screen.incomingShort")) + `</span>` +
		`<span title="` + escapeAttr(ui.Text("screen.outgoingTransitions")) + `">` + fmt.Sprint(outgoing) + ` ` + escapeHTML(ui.Text("screen.outgoingShort")) + `</span></div></div></article>`
}

func blockingScreenMapIssues(model *Model) []Issue {
	result := []Issue{}
	for _, issue := range model.Issues {
		if issue.Severity == "error" && strings.HasPrefix(issue.DocumentPath, "screens/") && issue.DocumentPath != "screens/hotspots.json" {
			result = append(result, issue)
		}
	}
	return result
}

func renderScreenMapWorkspace(model *Model, current, initialUseCase string, embedded bool) string {
	ui := portalUI(model)
	var cards strings.Builder
	for _, screen := range model.Knowledge.Screens {
		cards.WriteString(screenCardHTML(model, current, screen))
	}
	workspaceClass := "screen-map-workspace"
	if embedded {
		workspaceClass += " is-embedded"
	}
	initialAttribute := ""
	modeControls := `<div class="screen-map-modes" role="group" aria-label="` + escapeAttr(ui.Text("screen.mapMode")) + `">` +
		`<button type="button" class="toolbar-button is-active" data-map-mode="all" aria-pressed="true">` + escapeHTML(ui.Text("screen.all")) + `</button>` +
		`<button type="button" class="toolbar-button" data-map-mode="module" aria-pressed="false">` + escapeHTML(ui.Text("screen.module")) + `</button>` +
		`<button type="button" class="toolbar-button" data-map-mode="usecase" aria-pressed="false">` + escapeHTML(ui.Text("screen.useCase")) + `</button>` +
		`<button type="button" class="toolbar-button" data-map-mode="unfinished" aria-pressed="false">` + escapeHTML(ui.Text("screen.unfinished")) + `</button>` +
		`<button type="button" class="toolbar-button" data-map-mode="sitemap" aria-pressed="false">` + escapeHTML(ui.Text("screen.sitemap")) + `</button></div>`
	if initialUseCase != "" {
		initialAttribute = ` data-map-initial-usecase="` + escapeAttr(initialUseCase) + `"`
		modeControls = `<div class="screen-map-modes"><span class="screen-map-fixed-mode">` + escapeHTML(ui.Text("screen.useCase")) + ` <code>` + escapeHTML(initialUseCase) + `</code></span></div>`
	}
	changesControl := ""
	if model.serveRevision != "" {
		changesControl = `<button type="button" class="toolbar-button" data-map-changes aria-pressed="false">` + escapeHTML(ui.Text("screen.showChanges")) + `</button><select data-map-change-status aria-label="` + escapeAttr(ui.Text("screen.changeStatus")) + `" hidden><option value="">` + escapeHTML(ui.Text("screen.allChanges")) + `</option><option value="added">` + escapeHTML(ui.Text("screen.added")) + `</option><option value="modified">` + escapeHTML(ui.Text("screen.modified")) + `</option><option value="removed">` + escapeHTML(ui.Text("screen.removed")) + `</option></select>`
	}
	return `<section class="` + workspaceClass + `" data-screen-map` + initialAttribute + `>` +
		`<div class="screen-map-toolbar">` + modeControls +
		`<div class="screen-map-filters"><input type="search" data-map-search placeholder="` + escapeAttr(ui.Text("screen.find")) + `" aria-label="` + escapeAttr(ui.Text("screen.find")) + `">` +
		`<select data-map-status aria-label="` + escapeAttr(ui.Text("screen.status")) + `"><option value="">` + escapeHTML(ui.Text("screen.allStatuses")) + `</option>` + screenStatusOptions(model) + `</select>` +
		`<select data-map-module aria-label="` + escapeAttr(ui.Text("screen.module")) + `" hidden><option value="">` + escapeHTML(ui.Text("screen.chooseModule")) + `</option>` + screenModuleOptions(model) + `</select>` +
		`<select data-map-usecase aria-label="` + escapeAttr(ui.Text("screen.useCase")) + `" hidden><option value="">` + escapeHTML(ui.Text("screen.chooseUseCase")) + `</option>` + screenUseCaseOptions(model, initialUseCase) + `</select>` + changesControl + `</div>` +
		`<div class="screen-map-zoom" role="group" aria-label="` + escapeAttr(ui.Text("screen.zoom")) + `"><button type="button" data-map-zoom-out aria-label="` + escapeAttr(ui.Text("screen.zoomOut")) + `">−</button>` +
		`<button type="button" data-map-fit>` + escapeHTML(ui.Text("screen.fit")) + `</button><button type="button" data-map-reset>` + escapeHTML(ui.Text("screen.reset")) + `</button>` +
		`<button type="button" data-map-zoom-in aria-label="` + escapeAttr(ui.Text("screen.zoomIn")) + `">+</button><button type="button" data-map-fullscreen>` + escapeHTML(ui.Text("screen.fullscreen")) + `</button></div></div>` +
		`<div class="screen-map-shell"><div class="screen-map-stage" data-map-stage tabindex="0" aria-label="` + escapeAttr(ui.Text("screen.interactiveMap")) + `">` +
		`<div class="screen-map-viewport" data-map-viewport><div class="screen-map-groups" data-map-groups></div><svg class="screen-map-edges" data-map-edges aria-hidden="true"></svg>` +
		`<div class="screen-map-nodes" data-map-nodes>` + cards.String() + `</div><div class="screen-map-labels" data-map-labels></div></div><p class="screen-map-empty" data-map-empty hidden>` + escapeHTML(ui.Text("screen.noneForMode")) + `</p></div>` +
		`<aside class="screen-inspector" data-map-inspector aria-live="polite"><div class="screen-inspector-empty"><strong>` + escapeHTML(ui.Text("screen.choose")) + `</strong><span>` + escapeHTML(ui.Text("screen.chooseHelp")) + `</span></div></aside></div>` +
		`<p class="screen-map-status" data-map-summary aria-live="polite"></p><script type="application/json" data-screen-map-data>` +
		jsonScript(screenMapData(model, current)) + `</script></section>`
}

func renderScreenMapPage(model *Model, current string) string {
	ui := portalUI(model)
	if issues := blockingScreenMapIssues(model); len(issues) > 0 {
		var reasons strings.Builder
		for _, issue := range issues {
			location := issue.DocumentPath
			if issue.Line > 0 {
				location += fmt.Sprintf(":%d", issue.Line)
			}
			writeStrings(&reasons, `<li><code>`, escapeHTML(location), `</code> — `, escapeHTML(issue.Message), `</li>`)
		}
		content := breadcrumbs(model, current, ui.Text("screen.map")) +
			`<header class="page-header"><h1>` + escapeHTML(ui.Text("screen.mapUnavailable")) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("screen.mapUnavailableHelp")) + `</p></header>` +
			`<section class="dashboard-section"><h2>` + escapeHTML(ui.Text("screen.reasons")) + `</h2><ul>` + reasons.String() + `</ul><p><a class="primary-link" href="` +
			escapeAttr(relativeURL(current, "screens/catalog.html")) + `">` + escapeHTML(ui.Text("screen.openCatalog")) + `</a></p></section>`
		return pageShell(model, current, ui.Text("screen.mapUnavailable"), ui.Text("screen.mapErrors"), content, "")
	}
	content := breadcrumbs(model, current, ui.Text("screen.map")) +
		`<header class="page-header screen-map-header"><div class="page-kicker"><span class="badge">` + escapeHTML(ui.Text("screen.map")) + `</span><a class="badge" href="` +
		escapeAttr(relativeURL(current, "screens/catalog.html")) + `">` + escapeHTML(ui.Text("screen.catalog")) + `</a><a class="badge" href="` +
		escapeAttr(relativeURL(current, sectionCatalogOutput(SectionFlows))) + `">` + escapeHTML(modelDirectoryLabel(model, "flows")) + `</a></div><h1>` + escapeHTML(ui.Text("screen.map")) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("screen.mapDescription")) + `</p></header>` +
		renderScreenMapWorkspace(model, current, "", false)
	return pageShell(model, current, ui.Text("screen.map"), ui.Text("screen.mapInteractiveDescription"), content, "")
}

func screenErrorIDs(model *Model, screenID string) []string {
	values := []string{}
	for _, transition := range model.Knowledge.Transitions {
		if transition.FromID == screenID && transition.ErrorID != "" {
			values = append(values, transition.ErrorID)
		}
	}
	return uniqueStrings(values)
}

func screenKindLabel(model *Model, kind string) string {
	if label := portalUI(model).Text("screen.kind." + kind); !strings.HasPrefix(label, "screen.kind.") {
		return label
	}
	return kind
}

func screenModuleDetails(model *Model, moduleID string) string {
	title := ""
	for _, module := range model.Knowledge.Modules {
		if module.ID == moduleID {
			title = module.Title
			break
		}
	}
	result := `<code>` + escapeHTML(moduleID) + `</code>`
	if title != "" {
		result += `<span>` + escapeHTML(title) + `</span>`
	}
	return `<div class="screen-catalog-module">` + result + `</div>`
}

func screenCatalogUseCases(model *Model, current string, ids []string) string {
	if len(ids) == 0 {
		return `<span class="screen-catalog-empty-value">` + escapeHTML(portalUI(model).Text("screen.unused")) + `</span>`
	}
	var values strings.Builder
	for _, id := range ids {
		label := `<span class="screen-usecase-chip">` + escapeHTML(id) + `</span>`
		for _, useCase := range model.Knowledge.UseCases {
			if useCase.ID != id {
				continue
			}
			if document := model.DocByPath[useCase.Document]; document != nil {
				label = `<a class="screen-usecase-chip" href="` + escapeAttr(relativeURL(current, document.OutputPath)) +
					`" title="` + escapeAttr(screenTitleForUseCase(useCase)) + `">` + escapeHTML(id) + `</a>`
			}
			break
		}
		values.WriteString(label)
	}
	return `<div class="screen-catalog-usecases">` + values.String() + `</div>`
}

func screenCatalogErrors(model *Model, errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	var values strings.Builder
	for _, id := range errors {
		writeStrings(&values, `<span class="screen-error-id">`, escapeHTML(id), `</span>`)
	}
	return `<div class="screen-catalog-errors" aria-label="` + escapeAttr(portalUI(model).Text("screen.errorsAria")) + `">` + values.String() + `</div>`
}

func screenCatalogTransitionCounts(model *Model, incoming, outgoing int) string {
	ui := portalUI(model)
	return `<div class="screen-transition-summary" aria-label="` + escapeAttr(ui.Text("screen.transitionAria", incoming, outgoing)) + `">` +
		`<span><strong>` + fmt.Sprint(incoming) + `</strong><small>` + escapeHTML(ui.Text("screen.incoming")) + `</small></span>` +
		`<span><strong>` + fmt.Sprint(outgoing) + `</strong><small>` + escapeHTML(ui.Text("screen.outgoing")) + `</small></span></div>`
}

func screenCatalogRows(model *Model, current string) string {
	var rows strings.Builder
	for _, screen := range model.Knowledge.Screens {
		document := model.DocByPath[screen.Document]
		title := `<strong class="screen-catalog-title">` + escapeHTML(screen.Title) + `</strong>`
		if document != nil {
			title = `<a class="screen-catalog-title" href="` + escapeAttr(relativeURL(current, document.OutputPath)) + `">` + escapeHTML(screen.Title) + `</a>`
		}
		route := "—"
		if screen.Route != "" {
			route = `<code class="screen-catalog-route" title="` + escapeAttr(screen.Route) + `">` + escapeHTML(screen.Route) + `</code>`
		}
		errors := screenErrorIDs(model, screen.ID)
		search := strings.Join([]string{screen.ID, screen.Title, screen.ModuleID, screen.Route, strings.Join(errors, " "), strings.Join(screen.UseCaseIDs, " ")}, " ")
		_, _ = fmt.Fprintf(&rows, `<tr data-filter-item data-search="%s" data-module="%s" data-status="%s" data-usecase="%s"><td><div class="screen-catalog-screen">%s<div class="screen-catalog-identity">%s<code>%s</code><span>%s</span>%s</div></div></td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			escapeAttr(search), escapeAttr(screen.ModuleID), escapeAttr(screen.Status.Kind), escapeAttr(strings.Join(screen.UseCaseIDs, "|")),
			previewHTML(model, current, screen, "screen-catalog-preview"), title, escapeHTML(screen.ID), escapeHTML(screenKindLabel(model, screen.Kind)), screenCatalogErrors(model, errors),
			screenModuleDetails(model, screen.ModuleID), route, screenCatalogUseCases(model, current, screen.UseCaseIDs), renderStatusChip(model, screen.Status),
			screenCatalogTransitionCounts(model, len(screen.IncomingTransitionIDs), len(screen.OutgoingTransitionIDs)))
	}
	return rows.String()
}

func renderScreenCatalogPage(model *Model, current string) string {
	ui := portalUI(model)
	mapBadge := ""
	if model.ScreenMapEnabled {
		mapBadge = `<div class="page-kicker"><a class="badge" href="` + escapeAttr(relativeURL(current, "screens/index.html")) + `">` + escapeHTML(ui.Text("process.map")) + `</a></div>`
	}
	content := breadcrumbs(model, current, ui.Text("screen.catalogTitle")) +
		`<header class="page-header">` + mapBadge + `<h1>` + escapeHTML(ui.Text("screen.catalogTitle")) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("screen.catalogDescription")) + `</p></header>` +
		`<section class="dashboard-section screen-catalog" data-filter-scope aria-labelledby="screen-catalog-filters-title">` +
		`<h2 class="screen-reader-only" id="screen-catalog-filters-title">` + escapeHTML(ui.Text("screen.catalogFilters")) + `</h2>` +
		`<div class="screen-catalog-filterbar">` +
		`<label class="screen-filter-field screen-filter-search"><span>` + escapeHTML(ui.Text("process.search")) + `</span><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("screen.catalogSearchPlaceholder")) + `"></label>` +
		`<label class="screen-filter-field"><span>` + escapeHTML(ui.Text("screen.module")) + `</span><select data-filter-control="module"><option value="all">` + escapeHTML(ui.Text("screen.allModules")) + `</option>` + screenModuleOptions(model) + `</select></label>` +
		`<label class="screen-filter-field"><span>` + escapeHTML(ui.Text("screen.status")) + `</span><select data-filter-control="status"><option value="all">` + escapeHTML(ui.Text("screen.allStatuses")) + `</option>` + screenStatusOptions(model) + `</select></label>` +
		`<label class="screen-filter-field"><span>` + escapeHTML(ui.Text("screen.useCase")) + `</span><select data-filter-control="usecase"><option value="all">` + escapeHTML(ui.Text("screen.allUseCases")) + `</option>` + screenUseCaseOptions(model) + `</select></label>` +
		`<button class="toolbar-button screen-filter-reset" type="button" data-filter-reset>` + escapeHTML(ui.Text("process.reset")) + `</button></div>` +
		`<div class="screen-catalog-summary" aria-live="polite"><span>` + escapeHTML(ui.Text("screen.found")) + ` <strong data-filter-count></strong></span><span>` + escapeHTML(ui.Text("screen.filtersImmediate")) + `</span></div>` +
		`<div class="screen-catalog-table"><table><caption class="screen-reader-only">` + escapeHTML(ui.Text("screen.catalogCaption")) + `</caption>` +
		`<thead><tr><th scope="col">` + escapeHTML(ui.Text("screen.kind.screen")) + `</th><th scope="col">` + escapeHTML(ui.Text("screen.module")) + `</th><th scope="col">` + escapeHTML(ui.Text("screen.route")) + `</th><th scope="col">` + escapeHTML(ui.Text("screen.useCases")) + `</th><th scope="col">` + escapeHTML(ui.Text("screen.status")) + `</th><th scope="col">` + escapeHTML(ui.Text("screen.transitions")) + `</th></tr></thead><tbody>` +
		screenCatalogRows(model, current) + `</tbody></table></div><div class="empty-state screen-catalog-empty" data-filter-empty hidden><strong>` + escapeHTML(ui.Text("screen.noneFound")) + `</strong><span>` + escapeHTML(ui.Text("screen.noneFoundHelp")) + `</span><button class="toolbar-button" type="button" data-filter-reset>` + escapeHTML(ui.Text("screen.resetFilters")) + `</button></div></section>`
	return pageShell(model, current, ui.Text("screen.catalogTitle"), ui.Text("screen.catalogProject"), content, "")
}

func findUseCase(model *Model, id string) *KnowledgeUseCase {
	for index := range model.Knowledge.UseCases {
		if model.Knowledge.UseCases[index].ID == id {
			return &model.Knowledge.UseCases[index]
		}
	}
	return nil
}

func renderPlayableFlowComponent(model *Model, flow PlayableFlow, current string, embedded bool) string {
	ui := portalUI(model)
	useCase := findUseCase(model, flow.UseCaseID)
	title := flow.UseCaseID
	if useCase != nil {
		title += " · " + screenTitleForUseCase(*useCase)
	}
	if !flow.Valid {
		var issues strings.Builder
		for _, code := range flow.IssueCodes {
			message := code
			for _, issue := range model.Issues {
				if issue.Code == code {
					message = issue.Message
					break
				}
			}
			writeStrings(&issues, `<li><code>`, escapeHTML(code), `</code> — `, escapeHTML(message), `</li>`)
		}
		return `<section class="playable-unavailable"><h2>` + escapeHTML(ui.Text("play.unavailable")) + `</h2><p>` + escapeHTML(ui.Text("play.fixModel")) + `</p><ul>` +
			issues.String() + `</ul><a href="` + escapeAttr(relativeURL(current, model.HealthOutputPath)) + `">` + escapeHTML(ui.Text("play.openDiagnostics")) + `</a></section>`
	}
	payload := screenMapData(model, current)
	useCaseLink := ""
	mapLink := relativeURL(current, screenMapTarget(model)) + "#usecase=" + flow.UseCaseID
	if useCase != nil {
		if document := model.DocByPath[useCase.Document]; document != nil {
			useCaseTarget := relativeURL(current, document.OutputPath)
			useCaseLink = `<a class="primary-button" href="` + escapeAttr(useCaseTarget+"#overview") + `">` + escapeHTML(ui.Text("play.openUseCase")) + `</a>`
			if embedded {
				mapLink = "#map"
			}
		}
	}
	return `<section class="playable-flow" data-playable-flow><header class="playable-header"><div><span class="page-kicker">` + escapeHTML(ui.Text("play.badge")) + `</span><h2>` +
		escapeHTML(title) + `</h2><p>` + escapeHTML(ui.Text("play.step")) + ` <strong data-flow-step>1</strong> · <span data-flow-history-label>` + escapeHTML(ui.Text("play.start")) + `</span></p></div>` +
		`<a class="toolbar-button" href="` + escapeAttr(mapLink) + `">` + escapeHTML(ui.Text("play.showScreens")) + `</a></header>` +
		`<div class="playable-stage"><div class="playable-preview-wrap" data-flow-preview></div><div class="playable-copy"><div data-flow-alert></div>` +
		`<p class="screen-eyebrow" data-flow-screen-id></p><h2 data-flow-screen-title></h2><p data-flow-state></p><div class="playable-actions" data-flow-actions></div>` +
		`<div class="playable-complete" data-flow-complete hidden><strong>` + escapeHTML(ui.Text("play.complete")) + `</strong><p>` + escapeHTML(flow.Result) +
		`</p><div class="playable-complete-actions"><button type="button" class="primary-button" data-flow-reset>` + escapeHTML(ui.Text("play.restart")) + `</button>` +
		`<a class="primary-button" href="` + escapeAttr(mapLink) + `">` + escapeHTML(ui.Text("play.showMap")) + `</a>` + useCaseLink + `</div></div></div></div>` +
		`<footer class="playable-footer"><button type="button" data-flow-back disabled>` + escapeHTML(ui.Text("play.back")) + `</button><button type="button" data-flow-reset>` + escapeHTML(ui.Text("play.fromStart")) + `</button>` +
		`<label><input type="checkbox" data-flow-show-hotspots> ` + escapeHTML(ui.Text("play.showHotspots")) + `</label></footer>` +
		`<script type="application/json" data-playable-data>` + jsonScript(struct {
		Model screenMapPayload `json:"model"`
		Flow  PlayableFlow     `json:"flow"`
	}{Model: payload, Flow: flow}) + `</script></section>`
}

func renderTraceabilityPage(model *Model, current string) string {
	ui := portalUI(model)
	var rows strings.Builder
	for _, row := range model.Knowledge.Traceability {
		search := strings.Join([]string{row.UseCaseID, row.ScreenID, row.TransitionID, row.TaskID, row.CriterionID, row.Verification}, " ")
		_, _ = fmt.Fprintf(&rows, `<tr data-filter-item data-search="%s"><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><code>%s</code></td></tr>`,
			escapeAttr(search), escapeHTML(fallbackDash(row.UseCaseID)), escapeHTML(row.ScreenID), escapeHTML(row.TransitionID),
			escapeHTML(row.TaskID), escapeHTML(row.CriterionID), escapeHTML(row.Verification))
	}
	content := breadcrumbs(model, current, ui.Text("nav.traceability")) +
		`<header class="page-header"><h1>` + escapeHTML(ui.Text("trace.title")) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("trace.description")) + `</p></header>` +
		`<section class="dashboard-section" data-filter-scope><div class="collection-controls"><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("trace.placeholder")) + `" aria-label="` + escapeAttr(ui.Text("trace.search")) + `"></div>` +
		`<div class="collection-summary">` + escapeHTML(ui.Text("collection.shown")) + ` <strong data-filter-count></strong></div><div class="screen-catalog-table"><table><thead><tr><th>` + escapeHTML(ui.Text("process.useCase")) + `</th><th>` + escapeHTML(ui.Text("screen.kind.screen")) + `</th><th>` + escapeHTML(ui.Text("process.transition")) + `</th><th>` + escapeHTML(ui.Text("process.task")) + `</th><th>` + escapeHTML(ui.Text("process.criterion")) + `</th><th>` + escapeHTML(ui.Text("process.verification")) + `</th></tr></thead><tbody>` +
		rows.String() + `</tbody></table></div><div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("trace.none")) + `</div></section>`
	return pageShell(model, current, ui.Text("nav.traceability"), ui.Text("trace.title"), content, "")
}

func renderScreenConnections(model *Model, document *Document) string {
	ui := portalUI(model)
	id := document.Metadata["id"]
	var incoming, outgoing strings.Builder
	linkScreen := func(screenID string) string {
		for _, screen := range model.Knowledge.Screens {
			if screen.ID == screenID {
				if target := model.DocByPath[screen.Document]; target != nil {
					return `<a href="` + escapeAttr(relativeURL(document.OutputPath, target.OutputPath)) + `">` + escapeHTML(screen.ID+" · "+screen.Title) + `</a>`
				}
			}
		}
		return escapeHTML(screenID)
	}
	for _, transition := range model.Knowledge.Transitions {
		label := `<code>` + escapeHTML(transition.ID) + `</code> · ` + escapeHTML(transition.Action+" · "+transition.Condition)
		if transition.ToID == id {
			writeStrings(&incoming, `<tr><td>`, linkScreen(transition.FromID), `</td><td>`, label, `</td><td>`, escapeHTML(transition.Kind), `</td></tr>`)
		}
		if transition.FromID == id {
			writeStrings(&outgoing, `<tr><td>`, label, `</td><td>`, linkScreen(transition.ToID), `</td><td>`, escapeHTML(transition.Kind), `</td></tr>`)
		}
	}
	table := func(headers, body string) string {
		if body == "" {
			return `<p class="empty-state">` + escapeHTML(ui.Text("screen.noTransitions")) + `</p>`
		}
		return `<div class="data-table"><table><thead><tr>` + headers + `</tr></thead><tbody>` + body + `</tbody></table></div>`
	}
	return `<section class="dashboard-section dashboard-support-panel screen-connections"><div class="section-heading"><div><h2>` + escapeHTML(ui.Text("screen.computedRelations")) + `</h2><p>` + escapeHTML(ui.Text("screen.computedHelp")) + `</p></div>` +
		`<a href="` + escapeAttr(relativeURL(document.OutputPath, screenMapTarget(model))+"#screen="+id) + `">` + escapeHTML(ui.Text("screen.openInCatalog")) + `</a></div>` +
		`<h3>` + escapeHTML(ui.Text("screen.incoming")) + `</h3>` + table(`<th>`+escapeHTML(ui.Text("screen.from"))+`</th><th>`+escapeHTML(ui.Text("process.transition"))+`</th><th>`+escapeHTML(ui.Text("screen.type"))+`</th>`, incoming.String()) +
		`<h3>` + escapeHTML(ui.Text("screen.outgoing")) + `</h3>` + table(`<th>`+escapeHTML(ui.Text("process.transition"))+`</th><th>`+escapeHTML(ui.Text("screen.to"))+`</th><th>`+escapeHTML(ui.Text("screen.type"))+`</th>`, outgoing.String()) + `</section>`
}
