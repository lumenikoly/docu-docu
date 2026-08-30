package toudocu

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	frontend "toudocu/internal/site"
	"unicode"
)

func writeStrings(b *strings.Builder, values ...string) {
	for _, value := range values {
		b.WriteString(value)
	}
}

const Version = "0.0.7"

var fieldOrder = []string{"id", "status", "taskType", "screenKind", "author", "priority", "severity", "reproducibility", "regression", "module", "useCase", "flow", "screens", "transitions", "standards", "runbooks", "parentTask", "startScreen", "terminalScreens", "allowCycle", "route", "preview", "parentScreen", "component", "environment", "risk", "lastVerified", "supersededBy", "dependsOn", "date", "plannedDate", "updated", "probability", "impact", "scope"}

var documentIcons = map[string]string{
	"overview": "home", "status": "activity", "roadmap": "milestone", "risks": "triangle", "ideas": "lightbulb", "notes": "stickyNote", "changelog": "history",
	"use-case": "userFlow", "module": "box", "architecture": "network", "contract": "code", "decision": "shield", "flow": "workflow", "screen-map": "map",
	"screen-index": "map", "screen": "monitor", "guide": "compass", "work": "clipboard", "draft": "fileEdit", "reference": "layers", "standard": "checkCircle",
	"quality-index": "checkCircle", "runbook": "book", "runbook-index": "book", "document": "file",
}

func portalUI(model *Model) frontend.UI {
	locale := ""
	if model != nil {
		locale = model.SiteConfig.Project.Locale
	}
	return frontend.NewUI(locale)
}

func localizedTypeLabel(model *Model, documentType string) string {
	return portalUI(model).Text("type." + documentType)
}

func navigationDocumentIcon(document *Document) (icon, statusClass, statusLabel string) {
	icon = documentIcons[document.Type]
	if icon == "" {
		icon = documentIcons["document"]
	}
	if strings.TrimSpace(document.Metadata["status"]) == "" {
		return icon, "", ""
	}
	statusLabel = document.Status.Label
	if document.Status.Recognized && document.Status.Kind != "neutral" {
		statusClass = " status-" + document.Status.Kind
	}
	return icon, statusClass, statusLabel
}

func renderStatusChip(model *Model, status StatusInfo) string {
	ui := portalUI(model)
	label := ""
	if status.Label != "" {
		key := "status." + status.Label
		if localized := ui.Text(key); localized != key {
			label = localized
		}
	}
	if label == "" {
		label = ui.Text("status." + status.Kind)
	}
	if strings.HasPrefix(label, "status.") {
		label = status.Label
	}
	return fmt.Sprintf(`<span class="status-chip status-%s" title="%s"><span>%s</span></span>`, escapeAttr(status.Kind), escapeAttr(label), escapeHTML(label))
}

func localizedSemanticValue(ui frontend.UI, key, value string) string {
	if value == "" {
		return value
	}
	translationKey := "value." + value
	if key == "status" {
		translationKey = "status." + value
	}
	if localized := ui.Text(translationKey); localized != translationKey {
		return localized
	}
	return value
}

func renderProgress(ui frontend.UI, stats TaskStats, label string) string {
	if stats.Total == 0 {
		return ""
	}
	percent := percentOrZero(stats.Percent)
	complete := ""
	if percent == 100 {
		complete = " is-complete"
	}
	return fmt.Sprintf(`<div class="progress-block"><div class="progress-header"><span class="progress-label">%s · %s</span><span class="progress-value">%d%%</span></div><div class="progress-track%s" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow="%d"><div class="progress-fill" style="width:%d%%"></div></div></div>`, escapeHTML(label), escapeHTML(ui.Text("label.countOf", stats.Completed, stats.Total)), percent, complete, percent, percent)
}

func documentContextPath(model *Model, document *Document) string {
	if ensureInside(model.RepositoryRoot, document.AbsolutePath) {
		return toPosixRelative(model.RepositoryRoot, document.AbsolutePath)
	}
	return document.SourcePath
}

func renderDocumentContextButton(model *Model, document *Document) string {
	if document == nil || document == model.ProjectChangelog {
		return ""
	}
	ui := portalUI(model)
	copyButton := `<button class="document-context-button" type="button" data-copy-document-context data-document-context-title="` +
		escapeAttr(document.Title) + `" data-document-context-path="` + escapeAttr(documentContextPath(model, document)) +
		`"><span class="document-context-icon" aria-hidden="true">⧉</span><span data-copy-document-context-label aria-live="polite">` + escapeHTML(ui.Text("action.copyContext")) + `</span></button>`
	if !model.serveMode {
		return copyButton
	}
	encoded := url.QueryEscape(document.SourcePath)
	return `<div class="document-context-actions">` + copyButton +
		`<a class="document-context-button" href="/_toudocu/editor/?path=` + escapeAttr(encoded) + `">` + escapeHTML(ui.Text("action.edit")) + `</a>` +
		`<a class="document-context-button" href="/_toudocu/api/editor/file?raw=1&amp;path=` + escapeAttr(encoded) + `" target="_blank" rel="noopener">` + escapeHTML(ui.Text("action.openSource")) + `</a></div>`
}

func renderOpenAPIContractButton(model *Model, document *Document) string {
	if !model.serveMode || document == nil {
		return ""
	}
	for _, link := range document.Links {
		destination, _, _ := splitLinkDestination(link.Destination)
		if destination == "" || isExternalDestination(destination) {
			continue
		}
		candidate := path.Clean(path.Join(path.Dir(document.SourcePath), destination))
		for _, contract := range model.openAPIContracts {
			if candidate == contract.Path {
				return `<a class="document-context-button" href="` + apiDocsUIPath + `?spec=` + escapeAttr(url.QueryEscape(contract.Path)) + `">` + escapeHTML(portalUI(model).Text("action.openSwagger")) + `</a>`
			}
		}
	}
	return ""
}

func renderRoadmapAddButton(model *Model, document *Document) string {
	if !model.serveMode || document == nil || document.Type != "roadmap" {
		return ""
	}
	return `<button class="document-context-button roadmap-add-button" type="button" data-roadmap-add>` + escapeHTML(portalUI(model).Text("action.addOutcome")) + `</button><span data-td-island="roadmap" data-td-island-instance="roadmap-add"><span class="visually-hidden" data-td-island-error hidden role="alert"></span></span>`
}

func metricCard(label string, value any, detail string) string {
	out := fmt.Sprintf(`<div class="metric-card"><div class="metric-label">%s</div><div class="metric-value">%s</div>`, escapeHTML(label), escapeHTML(value))
	if detail != "" {
		out += `<div class="metric-detail">` + escapeHTML(detail) + `</div>`
	}
	return out + `</div>`
}

func outputForDirectory(model *Model, directory string) string {
	section := sectionTypeForPath(directory)
	if section == SectionDrafts {
		return sectionCatalogOutput(section)
	}
	if sectionRoute(section) != "" && sectionRoute(section) != directory {
		return sectionCatalogOutput(section)
	}
	if section == SectionScreens && len(model.Knowledge.Screens) > 0 {
		return "screens/catalog.html"
	}
	if section == SectionQuality || section == SectionRunbooks {
		return path.Join(directory, "index.html")
	}
	if document := model.DocByPath[path.Join(directory, "index.md")]; document != nil {
		return document.OutputPath
	}
	return path.Join(directory, "index.html")
}

func modelDirectoryLabel(model *Model, directory string) string {
	if section := sectionTypeForPath(directory); section != "" {
		if model.SiteConfig.Project.Locale != "" && len(model.SiteConfig.Project.Sections) == len(BuiltinSections) && strings.TrimSpace(model.SiteConfig.Project.Sections[section]) != "" {
			title := model.SiteConfig.Project.Sections[section]
			return title
		}
		if spec, ok := sectionSpec(section); ok {
			return spec.EnglishTitle
		}
	}
	if manifest := model.DocByPath[path.Join(directory, "index.md")]; manifest != nil && manifest.Title != "" {
		return manifest.Title
	}
	return directoryLabel(model, directory)
}

func renderNavigation(model *Model, current string) string {
	var b strings.Builder
	ui := portalUI(model)
	writeStrings(&b, `<nav aria-label="`, escapeAttr(ui.Text("nav.documentation")), `"><div class="nav-title">`, escapeHTML(ui.Text("nav.project")), `</div><ul class="nav-tree">`)
	rootDocs := []*Document{}
	sectionGroups := map[SectionType][]*Document{}
	customGroups := map[string][]*Document{}
	for _, document := range model.Documents {
		first := strings.Split(document.SourcePath, "/")[0]
		if !strings.Contains(document.SourcePath, "/") {
			rootDocs = append(rootDocs, document)
		} else if document.SectionType != "" {
			sectionGroups[document.SectionType] = append(sectionGroups[document.SectionType], document)
		} else {
			customGroups[first] = append(customGroups[first], document)
		}
	}
	if len(model.Knowledge.UseCases) > 0 {
		if _, exists := sectionGroups[SectionUseCases]; !exists {
			sectionGroups[SectionUseCases] = []*Document{}
		}
	}
	if len(model.Knowledge.UseCases)+len(model.Knowledge.Flows) > 0 {
		if _, exists := sectionGroups[SectionFlows]; !exists {
			sectionGroups[SectionFlows] = []*Document{}
		}
	}
	if len(model.Knowledge.Screens) > 0 {
		if _, exists := sectionGroups[SectionScreens]; !exists {
			sectionGroups[SectionScreens] = []*Document{}
		}
	}
	documentLink := func(document *Document, label string) string {
		active := ""
		aria := ""
		if document.OutputPath == current {
			active = " is-active"
			aria = ` aria-current="page"`
		}
		icon, statusClass, statusLabel := navigationDocumentIcon(document)
		if statusLabel != "" {
			if localized := ui.Text("status." + document.Status.Kind); !strings.HasPrefix(localized, "status.") {
				statusLabel = localized
			}
		}
		statusTitle := ""
		accessibleStatus := ""
		if statusLabel != "" {
			statusTitle = ` title="` + escapeAttr(ui.Text("label.statusValue", statusLabel)) + `"`
			accessibleStatus = `<span class="visually-hidden"> · ` + escapeHTML(ui.Text("label.statusValue", statusLabel)) + `</span>`
		}
		if label == "" {
			label = document.Title
		}
		return fmt.Sprintf(`<a class="nav-link%s" href="%s"%s><span class="nav-icon%s" aria-hidden="true"%s>%s</span><span>%s</span>%s</a>`, active, escapeAttr(relativeURL(current, document.OutputPath)), aria, escapeAttr(statusClass), statusTitle, renderIcon(icon, ""), escapeHTML(label), accessibleStatus)
	}
	writeDoc := func(document *Document, label string) {
		writeStrings(&b, `<li class="nav-item">`, documentLink(document, label), `</li>`)
	}
	for _, doc := range rootDocs {
		writeDoc(doc, "")
	}
	if model.ProjectChangelog != nil {
		writeDoc(model.ProjectChangelog, ui.Text("type.changelog"))
	}
	writeGroup := func(section SectionType, directory, groupKey string, docs []*Document) {
		target := outputForDirectory(model, directory)
		active := ""
		if target == current || strings.HasPrefix(current, directory+"/") || sectionRoute(section) != "" && strings.HasPrefix(current, sectionRoute(section)+"/") {
			active = " is-active"
		}
		if section == SectionUseCases && strings.HasPrefix(current, "use-cases/") {
			active = " is-active"
		}
		if section == SectionFlows && (strings.HasPrefix(current, sectionRoute(section)+"/") || strings.HasPrefix(current, "flows/")) {
			active = " is-active"
		}
		if section == SectionScreens && strings.HasPrefix(current, "screens/") {
			active = " is-active"
		}
		label := modelDirectoryLabel(model, directory)
		groupID := "nav-group-" + slugify(groupKey)
		sort.SliceStable(docs, func(i, j int) bool { return documentLess(docs[i], docs[j]) })
		if section == SectionFlows {
			hasFlowDocuments := false
			for _, doc := range docs {
				if doc.Type == "flow" && !strings.EqualFold(doc.FileName, "index.md") {
					hasFlowDocuments = true
					break
				}
			}
			if !hasFlowDocuments {
				_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link%s" href="%s"><span class="nav-icon" aria-hidden="true">%s</span><span>%s</span></a></li>`,
					active, escapeAttr(relativeURL(current, target)), renderIcon("route", ""), escapeHTML(label))
				return
			}
		}
		_, _ = fmt.Fprintf(&b, `<li class="nav-item nav-folder" data-nav-folder="%s"><div class="nav-folder-row"><button class="nav-folder-toggle" type="button" data-nav-folder-toggle aria-expanded="true" aria-controls="%s" aria-label="%s"><span aria-hidden="true"></span></button><a class="nav-folder-link%s" href="%s"><span>%s</span></a></div><ul id="%s">`, escapeAttr(groupKey), escapeAttr(groupID), escapeAttr(ui.Text("nav.collapseSection", label)), active, escapeAttr(relativeURL(current, target)), escapeHTML(label), escapeAttr(groupID))
		if section == SectionScreens && model.ScreenMapEnabled {
			activeClass := ""
			aria := ""
			if current == "screens/index.html" {
				activeClass = " is-active"
				aria = ` aria-current="page"`
			}
			_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link%s" href="%s"%s><span class="nav-icon" aria-hidden="true">%s</span><span>%s</span></a></li>`,
				activeClass, escapeAttr(relativeURL(current, "screens/index.html")), aria, renderIcon("map", ""), escapeHTML(ui.Text("nav.screenMap")))
		}
		if section == SectionFlows {
			for _, doc := range docs {
				if doc.Type == "flow" && !strings.EqualFold(doc.FileName, "index.md") {
					writeDoc(doc, "")
				}
			}
			b.WriteString(`</ul></li>`)
			return
		}
		if section == SectionWork {
			workDocs := map[string]*Document{}
			workItems := map[string]*WorkItem{}
			for _, doc := range docs {
				if strings.EqualFold(doc.FileName, "index.md") {
					continue
				}
				archived, _, _ := taskArchivePathInfo(doc.SourcePath)
				if archived {
					continue
				}
				id := stableDocumentID(model, doc.SourcePath)
				workDocs[id] = doc
			}
			for index := range model.Knowledge.WorkItems {
				item := &model.Knowledge.WorkItems[index]
				if workDocs[item.ID] != nil {
					workItems[item.ID] = item
				}
			}
			seen := map[string]bool{}
			var writeTask func(*WorkItem)
			writeTask = func(item *WorkItem) {
				if item == nil || seen[item.ID] {
					return
				}
				seen[item.ID] = true
				doc := workDocs[item.ID]
				children := []*WorkItem{}
				for _, childID := range item.ChildIDs {
					if child := workItems[childID]; child != nil {
						children = append(children, child)
					}
				}
				if len(children) == 0 {
					writeStrings(&b, `<li class="nav-item nav-task-leaf"><div class="nav-folder-row"><span class="nav-folder-spacer" aria-hidden="true"></span>`, documentLink(doc, ""), `</div></li>`)
					return
				}
				folderID := "nav-task-" + slugify(item.ID)
				folderKey := "task-" + slugify(item.ID)
				_, _ = fmt.Fprintf(&b, `<li class="nav-item nav-folder nav-task-folder" data-nav-folder="%s"><div class="nav-folder-row"><button class="nav-folder-toggle" type="button" data-nav-folder-toggle aria-expanded="true" aria-controls="%s" aria-label="%s"><span aria-hidden="true"></span></button>%s</div><ul id="%s">`, escapeAttr(folderKey), escapeAttr(folderID), escapeAttr(ui.Text("nav.collapseTask", item.Title)), documentLink(doc, ""), escapeAttr(folderID))
				for _, child := range children {
					writeTask(child)
				}
				b.WriteString(`</ul></li>`)
			}
			rootCount := 0
			for _, doc := range docs {
				id := stableDocumentID(model, doc.SourcePath)
				item := workItems[id]
				if item == nil {
					if workDocs[id] != nil {
						writeDoc(doc, "")
					}
					continue
				}
				if parent := workItems[taskParentID(item)]; parent != nil {
					continue
				}
				rootCount++
				writeTask(item)
			}
			if rootCount == 0 {
				for _, doc := range docs {
					writeTask(workItems[stableDocumentID(model, doc.SourcePath)])
				}
			}
			b.WriteString(`</ul></li>`)
			return
		}
		for _, doc := range docs {
			if strings.EqualFold(doc.FileName, "index.md") || doc.Type == "screen-map" || doc.OutputPath == target {
				continue
			}
			if section == SectionArchitecture && strings.EqualFold(doc.FileName, "overview.md") {
				writeDoc(doc, ui.Text("nav.architectureOverview"))
				continue
			}
			if section == SectionWork {
				archived, _, _ := taskArchivePathInfo(doc.SourcePath)
				if archived {
					continue
				}
			}
			writeDoc(doc, "")
		}
		b.WriteString(`</ul></li>`)
	}
	for _, spec := range BuiltinSections {
		docs, exists := sectionGroups[spec.Type]
		if !exists {
			continue
		}
		writeGroup(spec.Type, spec.SourceDir, spec.Route, docs)
	}
	keys := make([]string, 0, len(customGroups))
	for key := range customGroups {
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(i, j int) bool { return naturalCompare(keys[i], keys[j]) < 0 })
	for _, key := range keys {
		writeGroup("", key, key, customGroups[key])
	}
	writeStrings(&b, `</ul><div class="nav-title">`, escapeHTML(ui.Text("nav.control")), `</div><ul class="nav-tree">`)
	active := ""
	if current == model.HealthOutputPath {
		active = " is-active"
	}
	_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link%s" href="%s"><span>%s</span></a></li>`, active, escapeAttr(relativeURL(current, model.HealthOutputPath)), escapeHTML(ui.Text("nav.quality")))
	if model.serveMode {
		_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link" href="/changes/"><span>%s</span></a></li>`, escapeHTML(ui.Text("nav.changes")))
		if len(model.openAPIContracts) > 0 {
			_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link" href="/_toudocu/api-docs/"><span>HTTP API</span></a></li>`)
		}
	}
	if len(model.Knowledge.Screens) > 0 {
		active = ""
		if current == "traceability.html" {
			active = " is-active"
		}
		_, _ = fmt.Fprintf(&b, `<li class="nav-item"><a class="nav-link%s" href="%s"><span>%s</span></a></li>`, active, escapeAttr(relativeURL(current, "traceability.html")), escapeHTML(ui.Text("nav.traceability")))
	}
	b.WriteString(`</ul></nav>`)
	return b.String()
}

func projectChangelogSearchItem(model *Model, document *Document) SearchItem {
	description := document.Description
	if description == "" {
		description = document.PlainText
	}
	return SearchItem{
		Title: document.Title, Path: projectChangelogFile, URL: document.OutputPath,
		Type: document.Type, TypeLabel: localizedTypeLabel(model, document.Type), Description: truncate(description, 220),
		Text: canonicalText(strings.Join([]string{document.Title, projectChangelogFile, document.PlainText}, " ")),
	}
}

func mustFrontendAsset(logical string) string {
	asset, err := frontend.AssetName(logical)
	if err != nil {
		panic(err)
	}
	return asset
}

func pageReference(model *Model, current string) (kind, id string) {
	kind = "document"
	var document *Document
	for _, candidate := range model.Documents {
		if candidate.OutputPath == current {
			document = candidate
			break
		}
	}
	if document == nil && model.ProjectChangelog != nil && model.ProjectChangelog.OutputPath == current {
		document = model.ProjectChangelog
	}
	if document == nil {
		if strings.HasPrefix(current, "screens/") {
			return "screen", ""
		}
		return kind, ""
	}
	id = stableDocumentID(model, document.SourcePath)
	switch document.Type {
	case "architecture", "module", "use-case", "flow", "screen", "standard", "runbook":
		kind = document.Type
	case "work":
		kind = "task"
	case "screen-map", "screen-index":
		kind = "screen"
	case "quality-index":
		kind = "standard"
	}
	return kind, id
}

func stableDocumentID(model *Model, sourcePath string) string {
	for _, module := range model.Knowledge.Modules {
		if module.Document == sourcePath {
			return module.ID
		}
	}
	for _, useCase := range model.Knowledge.UseCases {
		if useCase.Document == sourcePath {
			return useCase.ID
		}
	}
	for _, flow := range model.Knowledge.Flows {
		if flow.Document == sourcePath {
			return flow.ID
		}
	}
	for _, screen := range model.Knowledge.Screens {
		if screen.Document == sourcePath {
			return screen.ID
		}
	}
	for _, standard := range model.Knowledge.Standards {
		if standard.Document == sourcePath {
			return standard.ID
		}
	}
	for _, runbook := range model.Knowledge.Runbooks {
		if runbook.Document == sourcePath {
			return runbook.ID
		}
	}
	for _, item := range model.Knowledge.WorkItems {
		if item.Document == sourcePath {
			return item.ID
		}
	}
	return ""
}

func pageShell(model *Model, current, title, description, content, toc string) string {
	prefix := rootPrefix(current)
	config := model.SiteConfig
	ui := portalUI(model)
	fullTitle := title + " — " + model.Project.Title
	if current == "index.html" {
		fullTitle = model.Project.Title
	}
	tocHTML := ""
	gridClass := " no-toc"
	if toc != "" {
		tocHTML = `<aside class="page-toc" aria-label="` + escapeAttr(ui.Text("toc.label")) + `"><div class="page-toc-title">` + escapeHTML(ui.Text("toc.title")) + `</div>` + toc + `</aside>`
		gridClass = ""
	}
	if current == "work/index.html" {
		gridClass += " task-workspace-page"
	}
	extraStyles := ""
	if strings.Contains(content, `data-screen-map`) {
		extraStyles += `<link rel="stylesheet" data-page-style="screen-map" href="` + escapeAttr(prefix+"assets/"+mustFrontendAsset("screen-map.css")) + `">`
	}
	if strings.Contains(content, `data-playable-flow`) {
		extraStyles += `<link rel="stylesheet" data-page-style="playable-flow" href="` + escapeAttr(prefix+"assets/"+mustFrontendAsset("playable-flow.css")) + `">`
	}
	favicon := "assets/" + mustFrontendAsset("favicon.svg")
	if custom := brandingOutput(model, "favicon"); custom != "" {
		favicon = custom
	}
	attributes := appearanceAttributes(config)
	brandMark := `<span class="brand-mark" aria-hidden="true">` + escapeHTML(projectBrandMark(model.Project.Title)) + `</span>`
	if logo := brandingOutput(model, "logo"); logo != "" {
		brandMark = `<img class="brand-logo" src="` + escapeAttr(relativeURL(current, logo)) + `" alt="">`
	}
	footer := renderFooter(ui, config.Footer)
	themeLabel, _ := siteThemePresentation(ui, config.Theme)
	schemeLabel := colorSchemeLabel(ui, config.ColorScheme)
	themeSelect := `<label class="header-select site-theme-select" title="` + escapeAttr(ui.Text("header.theme")) + `"><span class="header-select-visual" aria-hidden="true">` + renderIcon("brush", "header-select-icon") + `<span data-site-theme-label>` + escapeHTML(themeLabel) + `</span></span><select data-site-theme-select aria-label="` + escapeAttr(ui.Text("header.theme")) + `">` + selectOptions(config.Theme, []selectOption{{"classic", ui.Text("theme.classic")}, {"paper", ui.Text("theme.paper")}, {"terminal", ui.Text("theme.terminal")}}) + `</select></label>`
	schemeSelect := `<label class="header-select scheme-select" title="` + escapeAttr(ui.Text("header.scheme")) + `"><span class="header-select-visual" aria-hidden="true"><span class="scheme-toggle-indicator"></span><span data-theme-label>` + escapeHTML(schemeLabel) + `</span></span><select data-color-scheme-select aria-label="` + escapeAttr(ui.Text("header.scheme")) + `">` + selectOptions(config.ColorScheme, []selectOption{{"system", ui.Text("scheme.system")}, {"light", ui.Text("scheme.light")}, {"dark", ui.Text("scheme.dark")}}) + `</select></label>`
	languageSelect := renderLanguageSelect(ui, model.languageTargets[current])
	serveControls, serveCSS, serveJS, serveRevision := "", "", "", ""
	if model.serveMode {
		review := ""
		review = discussionToggle(ui)
		agent := ""
		if model.agentConsoleEnabled {
			agent = agentConsoleToggle(ui) + agentTerminalToggle(ui)
		}
		serveControls = `<div class="header-action-group header-workspace-actions">` + workspaceNavigation(ui, workspacePortal) + `</div><div class="header-action-group header-server-actions">` + agent + review + `<button class="icon-button server-rebuild" type="button" data-server-rebuild aria-label="` + escapeAttr(ui.Text("header.rebuild")) + `" title="` + escapeAttr(ui.Text("header.rebuild")) + `"><svg class="server-rebuild-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M20 11a8 8 0 1 0-2.34 5.66M20 5v6h-6"/></svg></button><span class="visually-hidden" data-server-rebuild-status role="status" aria-live="polite"></span></div>`
		serveCSS = prefix + "assets/" + mustFrontendAsset("serve.css")
		serveJS = prefix + "assets/" + mustFrontendAsset("serve.js")
		serveRevision = model.serveRevision
	}
	locale := model.SiteConfig.Project.Locale
	if locale == "" {
		locale = "en"
	}
	kind, id := pageReference(model, current)
	runtime := frontend.RuntimeStatic
	capabilities := frontend.Capabilities{Search: true, Diagrams: true}
	var endpoints *frontend.Endpoints
	if model.serveMode {
		runtime = frontend.RuntimeServe
		capabilities.Editor, capabilities.Changes, capabilities.Rebuild, capabilities.TaskWorkspace = true, true, true, true
		capabilities.Review = true
		capabilities.UpdateCheck = model.updateCheckEnabled
		capabilities.AgentConsole = model.agentConsoleEnabled
		capabilities.TaskActions = model.taskActionsEnabled
		endpoints = &frontend.Endpoints{Editor: serveEndpoint(model, editorAPIBase), Changes: serveEndpoint(model, changesAPIBase), Rebuild: serveEndpoint(model, rebuildEndpoint)}
		if capabilities.TaskActions {
			endpoints.TaskActions = serveEndpoint(model, "/_toudocu/api/tasks")
		}
		if capabilities.AgentConsole {
			endpoints.AgentConsole = serveEndpoint(model, agentConsoleAPIBase)
		}
		if capabilities.Review {
			endpoints.Review = serveEndpoint(model, reviewAPIBase)
		}
		if capabilities.UpdateCheck {
			endpoints.Version = versionEndpoint
		}
	}
	bootstrap, err := frontend.MarshalBootstrap(frontend.PageBootstrap{
		SchemaVersion: 1,
		Runtime:       runtime,
		Page:          frontend.PageReference{Kind: kind, ID: id, Path: current},
		Portal:        frontend.PortalReference{AssetBase: prefix + "assets/", DataBase: prefix + "data/"},
		UI: frontend.UISettings{
			Locale: locale, Theme: config.Theme, ColorScheme: config.ColorScheme,
			Accent: config.Accent, Density: config.Density, ContentWidth: config.ContentWidth,
		},
		Capabilities: capabilities,
		Endpoints:    endpoints,
	})
	if err != nil {
		panic(err)
	}
	header := `<header class="site-header"><div class="brand-area"><button class="icon-button sidebar-toggle" type="button" data-sidebar-toggle aria-label="` + escapeAttr(ui.Text("nav.openNavigation")) + `" title="` + escapeAttr(ui.Text("nav.openNavigation")) + `">` + renderIcon("menu", "") + `</button><a class="brand" href="` + escapeAttr(relativeURL(current, "index.html")) + `" title="` + escapeAttr(model.Project.Title) + `">` + brandMark + `<span class="brand-text">` + escapeHTML(model.Project.Title) + `</span></a></div><div class="global-search" role="search"><div class="search-input-wrap"><input type="search" data-global-search placeholder="` + escapeAttr(ui.Text("header.search")) + `" aria-label="` + escapeAttr(ui.Text("header.search")) + `" aria-expanded="false" aria-controls="global-search-results"><span class="search-shortcut">/</span></div><div class="search-results" id="global-search-results" data-search-results role="listbox" hidden></div></div><div class="header-actions"><div class="header-action-group header-print-actions"><button class="icon-button print-button" type="button" data-print aria-label="` + escapeAttr(ui.Text("header.print")) + `" title="` + escapeAttr(ui.Text("header.print")) + `">` + renderIcon("print", "") + `</button></div>` + serveControls + `<div class="header-action-group header-appearance">` + languageSelect + themeSelect + schemeSelect + `</div></div></header>`
	rendered, err := frontend.RenderShell(frontend.ShellView{
		UI: ui, Lang: locale, HTMLAttributes: template.HTMLAttr(attributes), Revision: serveRevision,
		Description: description, Title: fullTitle, Favicon: relativeURL(current, favicon),
		AppearanceJS: prefix + "assets/" + mustFrontendAsset("appearance.js"),
		PortalCSS:    prefix + "assets/" + mustFrontendAsset("portal.css"), ServeCSS: serveCSS,
		ExtraStyles: template.HTML(extraStyles), Bootstrap: bootstrap,
		PortalJS: prefix + "assets/" + mustFrontendAsset("portal.js"), ServeJS: serveJS,
		RootPrefix: prefix, Header: template.HTML(header), Navigation: template.HTML(renderNavigation(model, current)),
		MainClass: gridClass, Content: template.HTML(content), TOC: template.HTML(tocHTML), Footer: template.HTML(footer),
	})
	if err != nil {
		panic(err)
	}
	return rendered
}

func serveEndpoint(model *Model, endpoint string) string {
	if model == nil || model.serveBaseURL == "" {
		return endpoint
	}
	return model.serveBaseURL + endpoint
}

func agentConsoleToggle(ui frontend.UI) string {
	label := ui.Text("core.agent.001")
	return `<button class="icon-button header-agent-toggle" type="button" data-agent-console-toggle aria-label="` + escapeAttr(label) + `" title="` + escapeAttr(label) + `" aria-expanded="false" aria-controls="agent-console-panel">` +
		renderIcon("activity", "") + `<span class="visually-hidden" data-agent-console-summary></span><span class="header-agent-attention" data-agent-console-attention hidden aria-hidden="true">!</span></button>`
}

func agentTerminalToggle(ui frontend.UI) string {
	label := ui.Text("core.agent.064")
	return `<button class="icon-button header-terminal-toggle" type="button" data-agent-terminal-toggle aria-label="` + escapeAttr(label) + `" title="` + escapeAttr(label) + `" aria-expanded="false" aria-controls="agent-console-panel">` + renderIcon("terminal", "") + `</button>`
}

func renderFooter(ui frontend.UI, config FooterConfig) string {
	if config.defaultText {
		brand := "Toudocu"
		if config.URL != "" {
			brand = `<a href="` + escapeAttr(config.URL) + `" rel="noopener noreferrer">Toudocu</a>`
		}
		return escapeHTML(ui.Text("footer.generated")) + " " + brand + " " + escapeHTML(Version)
	}
	footer := escapeHTML(config.Text)
	if config.URL != "" {
		footer = `<a href="` + escapeAttr(config.URL) + `" rel="noopener noreferrer">` + footer + `</a>`
	}
	return footer
}

func renderLanguageSelect(ui frontend.UI, targets []LanguageTarget) string {
	if len(targets) < 2 {
		return ""
	}
	active := targets[0].Locale
	for _, target := range targets {
		if target.Active {
			active = target.Locale
			break
		}
	}
	var b strings.Builder
	writeStrings(&b, `<label class="header-select language-select" title="`, escapeAttr(ui.Text("header.language")), `"><span class="header-select-visual language-indicator" aria-hidden="true">`, escapeHTML(strings.ToUpper(active)), `</span><select aria-label="`, escapeAttr(ui.Text("header.language")), `" onchange="location.href=this.value">`)
	for _, target := range targets {
		selected := ""
		if target.Active {
			selected = " selected"
		}
		writeStrings(&b, `<option value="`, escapeAttr(target.URL), `"`, selected, `>`, escapeHTML(target.Locale), `</option>`)
	}
	b.WriteString(`</select></label>`)
	return b.String()
}

type selectOption struct {
	Value string
	Label string
}

func selectOptions(current string, options []selectOption) string {
	var b strings.Builder
	for _, option := range options {
		selected := ""
		if option.Value == current {
			selected = " selected"
		}
		writeStrings(&b, `<option value="`, escapeAttr(option.Value), `"`, selected, `>`, escapeHTML(option.Label), `</option>`)
	}
	return b.String()
}

func siteThemePresentation(ui frontend.UI, theme string) (string, string) {
	switch theme {
	case "paper":
		return ui.Text("theme.paper"), "P"
	case "terminal":
		return ui.Text("theme.terminal"), "T"
	default:
		return ui.Text("theme.classic"), "C"
	}
}

func colorSchemeLabel(ui frontend.UI, scheme string) string {
	switch scheme {
	case "light":
		return ui.Text("scheme.light")
	case "dark":
		return ui.Text("scheme.dark")
	default:
		return ui.Text("scheme.system")
	}
}

func brandingOutput(model *Model, kind string) string {
	prefix := "assets/branding/" + kind + "."
	for output := range model.BrandingAssets {
		if strings.HasPrefix(output, prefix) {
			return output
		}
	}
	return ""
}

func projectBrandMark(title string) string {
	mark := make([]rune, 0, 2)
	words := 0
	inWord := false
	for _, r := range title {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			inWord = false
			continue
		}
		if !inWord {
			words++
			inWord = true
			if words == 2 {
				if len(mark) == 1 {
					mark = append(mark, r)
				} else {
					mark[1] = r
				}
				return strings.ToUpper(string(mark))
			}
		}
		if words == 1 && len(mark) < 2 {
			mark = append(mark, r)
		}
	}
	if len(mark) == 0 {
		return "T"
	}
	return strings.ToUpper(string(mark))
}

func breadcrumbs(model *Model, current, title string) string {
	return `<nav class="breadcrumbs" aria-label="` + escapeAttr(portalUI(model).Text("breadcrumbs")) + `"><a href="` + escapeAttr(relativeURL(current, "index.html")) + `">` + escapeHTML(model.Project.Title) + `</a><span class="breadcrumb-separator" aria-hidden="true"></span><span>` + escapeHTML(title) + `</span></nav>`
}

func renderMetadata(model *Model, document *Document) string {
	if len(document.Metadata) == 0 && len(document.MetadataExtras) == 0 {
		return ""
	}
	var b strings.Builder
	ui := portalUI(model)
	b.WriteString(`<dl class="metadata-grid">`)
	seen := map[string]bool{}
	for _, key := range fieldOrder {
		if value := document.Metadata[key]; value != "" {
			seen[key] = true
			label := ui.Text("field." + key)
			if strings.HasPrefix(label, "field.") {
				label = key
			}
			writeStrings(&b, `<div><dt>`, escapeHTML(label), `</dt><dd>`, escapeHTML(localizedSemanticValue(ui, key, value)), `</dd></div>`)
		}
	}
	keys := []string{}
	for key := range document.Metadata {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeStrings(&b, `<div><dt>`, escapeHTML(key), `</dt><dd>`, escapeHTML(localizedSemanticValue(ui, key, document.Metadata[key])), `</dd></div>`)
	}
	for _, extra := range document.MetadataExtras {
		writeStrings(&b, `<div><dt>`, escapeHTML(extra.Key), `</dt><dd>`, escapeHTML(extra.Value), `</dd></div>`)
	}
	return b.String() + `</dl>`
}

func renderTOC(document *Document) string {
	var b strings.Builder
	b.WriteString(`<ul>`)
	for _, h := range document.Headings {
		if h.Level < 2 || h.Level > 3 {
			continue
		}
		writeStrings(&b, `<li class="toc-level-`, fmt.Sprint(h.Level), `"><a href="#`, escapeAttr(h.ID), `">`, escapeHTML(h.Title), `</a></li>`)
	}
	return b.String() + `</ul>`
}

func renderRelated(model *Model, document *Document) string {
	items := append([]*Document{}, document.RelatedDocuments...)
	items = append(items, document.Backlinks...)
	items = append(items, document.LinkedUseCases...)
	items = append(items, document.LinkedModules...)
	seen := map[string]bool{}
	var b strings.Builder
	for _, item := range items {
		if item == nil || item == document || seen[item.SourcePath] {
			continue
		}
		seen[item.SourcePath] = true
		writeStrings(&b, `<li><a href="`, escapeAttr(relativeURL(document.OutputPath, item.OutputPath)), `">`, escapeHTML(item.Title), `</a><span class="table-subtext">`, escapeHTML(localizedTypeLabel(model, item.Type)), `</span></li>`)
	}
	if b.Len() == 0 {
		return ""
	}
	return `<section class="dashboard-section dashboard-support-panel"><h2>` + escapeHTML(portalUI(model).Text("related.title")) + `</h2><ul class="related-list">` + b.String() + `</ul></section>`
}

func renderRiskStatus(model *Model, document *Document) string {
	if document == nil || document.Type != "risks" {
		return ""
	}
	ui := portalUI(model)
	counts := map[string]int{}
	labels := []string{}
	total := 0
	for _, risk := range model.Risks {
		if risk.Document != document {
			continue
		}
		label := ui.Text("status." + risk.Status.Kind)
		switch risk.Status.Kind {
		case "open":
			label = ui.Text("risk.open")
		case "in-progress":
			label = ui.Text("risk.reducing")
		case "risk-accepted":
			label = ui.Text("risk.accepted")
		}
		if counts[label] == 0 {
			labels = append(labels, label)
		}
		counts[label]++
		total++
	}
	if total == 0 {
		return ""
	}
	statusCounts := make([]string, 0, len(labels))
	for _, label := range labels {
		statusCounts = append(statusCounts, fmt.Sprintf(`<span>%d %s</span>`, counts[label], escapeHTML(strings.ToLower(label))))
	}
	return `<section class="risk-status" aria-labelledby="risk-status-title"><div><h2 id="risk-status-title">` + escapeHTML(ui.Text("risk.title")) + `</h2><p class="risk-status-total">` + escapeHTML(ui.Text("risk.openCount", model.Stats.OpenRisks, total)) + `</p><p class="risk-status-counts">` + strings.Join(statusCounts, ` <span aria-hidden="true">·</span> `) + `</p></div><ul class="risk-status-explanations"><li><strong>` + escapeHTML(ui.Text("risk.open")) + `</strong> — ` + escapeHTML(ui.Text("risk.openHelp")) + `</li><li><strong>` + escapeHTML(ui.Text("risk.reducing")) + `</strong> — ` + escapeHTML(ui.Text("risk.reducingHelp")) + `</li><li><strong>` + escapeHTML(ui.Text("risk.accepted")) + `</strong> — ` + escapeHTML(ui.Text("risk.acceptedHelp")) + `</li></ul></section>`
}

func renderDocumentPage(model *Model, document *Document) string {
	ui := portalUI(model)
	resolver := linkResolverFor(model, document)
	taskCompletionByLine := map[int]bool{}
	if document.Type == "roadmap" {
		for _, stage := range model.RoadmapStages {
			if stage.Document != document {
				continue
			}
			for _, item := range stage.Items {
				taskCompletionByLine[item.Line-1] = item.EffectiveCompleted
			}
		}
	}
	body := renderDocumentBody(model, document, resolver, taskCompletionByLine)
	issues := ""
	if len(document.Warnings)+len(document.Errors) > 0 {
		issues = fmt.Sprintf(`<a class="badge" href="%s">%s</a>`, escapeAttr(relativeURL(document.OutputPath, model.HealthOutputPath)), escapeHTML(ui.Text("issues.count", len(document.Warnings)+len(document.Errors))))
	}
	computedStatus := ""
	if document.Type == "status" {
		computedStatus = renderComputedStatus(model, document.OutputPath)
	}
	screenConnections := ""
	flowConnections := ""
	displayStatus := document.Status
	if document.Type == "screen" {
		screenConnections = renderScreenConnections(model, document)
		for _, screen := range model.Knowledge.Screens {
			if screen.ID == document.Metadata["id"] {
				displayStatus = screen.Status
				break
			}
		}
	}
	if document.Type == "flow" {
		flowConnections = renderFlowConnections(model, document)
	}
	if document.Type == "work" {
		computedStatus += renderTaskHierarchy(model, document)
	}
	if model.serveMode && document.Type == "work" {
		id := document.Metadata["id"]
		computedStatus = `<nav class="task-page-tabs" aria-label="` + escapeAttr(ui.Text("task.views")) + `"><span aria-current="page">` + escapeHTML(ui.Text("task.contract")) + `</span><a href="/changes/?task=` + escapeAttr(url.QueryEscape(id)) + `&amp;path=` + escapeAttr(url.QueryEscape(documentContextPath(model, document))) + `">` + escapeHTML(ui.Text("nav.changes")) + `</a></nav>` + computedStatus
	}
	statusChip := ""
	if strings.TrimSpace(document.Metadata["status"]) != "" {
		statusChip = renderStatusChip(model, displayStatus)
	}
	progressLabel := ui.Text("progress.document")
	if document.Type == "risks" {
		progressLabel = ui.Text("progress.mitigation")
	}
	content := breadcrumbs(model, document.OutputPath, document.Title) + `<header class="page-header"><div class="page-kicker">` + statusChip + `<span class="badge">` + escapeHTML(localizedTypeLabel(model, document.Type)) + `</span>` + issues + `</div><h1>` + escapeHTML(document.Title) + `</h1><p class="page-lead">` + escapeHTML(document.Description) + `</p>` + renderMetadata(model, document) + renderTaskPageActions(model, document) + renderRiskStatus(model, document) + renderProgress(ui, document.TaskStats, progressLabel) + `<div class="page-actions">` + renderRoadmapAddButton(model, document) + renderDocumentContextButton(model, document) + renderOpenAPIContractButton(model, document) + `<button class="collapse-all-button" type="button" data-collapse-all data-collapse-state="expanded" aria-expanded="true"><span class="collapse-all-icon" aria-hidden="true"><span class="collapse-icon collapse-icon-up"></span><span class="collapse-icon collapse-icon-down"></span></span><span data-collapse-label>` + escapeHTML(ui.Text("action.collapseSections")) + `</span></button></div></header>` + computedStatus + `<article class="doc-content">` + body + `</article>` + screenConnections + renderRelated(model, document)
	content += flowConnections
	return pageShell(model, document.OutputPath, document.Title, document.Description, content, renderTOC(document))
}

func renderTaskHierarchy(model *Model, document *Document) string {
	item, err := findWorkItem(model, stableDocumentID(model, document.SourcePath))
	if err != nil || item.ParentID == nil && len(item.ChildIDs) == 0 {
		return ""
	}
	ui := portalUI(model)
	byID := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		candidate := &model.Knowledge.WorkItems[index]
		byID[candidate.ID] = candidate
	}
	link := func(candidate *WorkItem) string {
		target := model.DocByPath[candidate.Document]
		if target == nil {
			return `<code>` + escapeHTML(candidate.ID) + `</code> · ` + escapeHTML(candidate.Title)
		}
		return `<a href="` + escapeAttr(relativeURL(document.OutputPath, target.OutputPath)) + `"><code>` + escapeHTML(candidate.ID) + `</code> · ` + escapeHTML(candidate.Title) + `</a>`
	}
	ancestors := taskHierarchy(model, item).Ancestors
	var trail, children strings.Builder
	for index, ancestor := range ancestors {
		if index > 0 {
			trail.WriteString(`<span aria-hidden="true">/</span>`)
		}
		trail.WriteString(link(byID[ancestor.ID]))
	}
	if trail.Len() > 0 {
		writeStrings(&trail, `<span aria-hidden="true">/</span><strong><code>`, escapeHTML(item.ID), `</code></strong>`)
	}
	var renderNode func(TaskTreeNode, bool)
	renderNode = func(node TaskTreeNode, current bool) {
		candidate := byID[node.ID]
		if candidate == nil {
			return
		}
		children.WriteString(`<li class="task-tree-item"><div class="task-tree-node`)
		if current {
			children.WriteString(` is-current`)
		}
		children.WriteString(`">`)
		if current {
			writeStrings(&children, `<span class="task-tree-link" aria-current="page"><code>`, escapeHTML(candidate.ID), `</code><span>`, escapeHTML(candidate.Title), `</span></span>`)
		} else if target := model.DocByPath[candidate.Document]; target != nil {
			writeStrings(&children, `<a class="task-tree-link" href="`, escapeAttr(relativeURL(document.OutputPath, target.OutputPath)), `"><code>`, escapeHTML(candidate.ID), `</code><span>`, escapeHTML(candidate.Title), `</span></a>`)
		} else {
			writeStrings(&children, `<span class="task-tree-link"><code>`, escapeHTML(candidate.ID), `</code><span>`, escapeHTML(candidate.Title), `</span></span>`)
		}
		state := strings.ReplaceAll(string(node.WorkState), "_", "-")
		children.WriteString(`<span class="task-tree-meta">`)
		writeStrings(&children, `<span class="status-chip status-`, escapeAttr(state), `">`, escapeHTML(taskWorkspaceStateLabel(ui, state)), `</span>`)
		if node.Descendants != nil {
			progressState := ""
			if node.Descendants.Complete {
				progressState = " is-complete"
			} else if node.Descendants.Started {
				progressState = " is-started"
			}
			writeStrings(&children, `<span class="task-tree-progress`, progressState, `">`, escapeHTML(ui.Text("work.workspace.descendants", node.Descendants.Counts.Done, node.Descendants.Total)), `</span>`)
		}
		children.WriteString(`</span></div>`)
		if len(node.Children) > 0 {
			children.WriteString(`<ul role="list">`)
			for _, child := range node.Children {
				renderNode(child, false)
			}
			children.WriteString(`</ul>`)
		}
		children.WriteString(`</li>`)
	}
	if len(item.ChildIDs) > 0 {
		renderNode(taskTreeNode(model, item), true)
	}
	parent := ""
	if candidate := byID[taskParentID(item)]; candidate != nil {
		parent = `<p><strong>` + escapeHTML(ui.Text("task.parent")) + `:</strong> ` + link(candidate) + `</p>`
	}
	childList := ""
	if children.Len() > 0 {
		childList = `<h3>` + escapeHTML(ui.Text("task.hierarchy")) + `</h3><ul class="task-tree" role="list">` + children.String() + `</ul>`
	}
	breadcrumb := ""
	if trail.Len() > 0 {
		breadcrumb = `<nav class="task-hierarchy-breadcrumb" aria-label="` + escapeAttr(ui.Text("task.hierarchy")) + `">` + trail.String() + `</nav>`
	}
	return `<section class="dashboard-section task-decomposition"><h2>` + escapeHTML(ui.Text("task.decomposition")) + `</h2>` + breadcrumb + parent + childList + `</section>`
}

func docCard(model *Model, current string, document *Document) string {
	ui := portalUI(model)
	archived, archiveYear, _ := taskArchivePathInfo(document.SourcePath)
	archiveState := "active"
	archiveBadge := ""
	if archived {
		archiveState = "archived"
		archiveBadge = `<span class="badge">` + escapeHTML(ui.Text("archive.year", archiveYear)) + `</span>`
	}
	workType := ""
	workDetails := ""
	if document.Type == "work" {
		if normalized, ok := taskType(document.Metadata["taskType"]); ok {
			workType = string(normalized)
		}
		if workType == "bug" {
			workDetails = `<p class="table-subtext">` + escapeHTML(ui.Text("work.severity")) + `: ` + escapeHTML(fallbackDash(localizedSemanticValue(ui, "severity", document.Metadata["severity"]))) +
				` · ` + escapeHTML(ui.Text("work.priority")) + `: ` + escapeHTML(fallbackDash(localizedSemanticValue(ui, "priority", document.Metadata["priority"]))) +
				` · ` + escapeHTML(ui.Text("work.reproducibility")) + `: ` + escapeHTML(fallbackDash(localizedSemanticValue(ui, "reproducibility", document.Metadata["reproducibility"]))) +
				` · ` + escapeHTML(ui.Text("work.regression")) + `: ` + escapeHTML(fallbackDash(localizedSemanticValue(ui, "regression", document.Metadata["regression"]))) +
				` · ` + escapeHTML(ui.Text("work.module")) + `: ` + escapeHTML(fallbackDash(document.Metadata["module"])) + `</p>`
		}
	}
	searchText := strings.Join([]string{documentStableID(document), document.Title, document.Description, document.SourcePath}, " ")
	severity := document.Metadata["severity"]
	regression := document.Metadata["regression"]
	reproducibility := document.Metadata["reproducibility"]
	if reproducibility == "not-reproduced" || reproducibility == "unknown" {
		reproducibility = "missing"
	}
	causeState := ""
	regressionTestState := ""
	if workType == "bug" {
		causeState = "missing"
		regressionTestState = "missing"
		if items := parseWorkItems(document); len(items) == 1 {
			cause, found := workSection(items[0], SectionKindCause)
			if found && strings.TrimSpace(cause.Text) != "" {
				causeState = "established"
			}
			criteria, _ := workSection(items[0], SectionKindAcceptanceCriteria)
			if bugHasRegressionCoverage(items[0], criteria.Tasks) {
				regressionTestState = "present"
			}
		}
	}
	statusChip := ""
	if strings.TrimSpace(document.Metadata["status"]) != "" {
		statusChip = renderStatusChip(model, document.Status)
	}
	return `<article class="document-card" data-filter-item data-search="` + escapeAttr(searchText) + `" data-status="` + escapeAttr(document.Status.Kind) + `" data-type="` + escapeAttr(document.Type) + `" data-work-type="` + escapeAttr(workType) + `" data-severity="` + escapeAttr(severity) + `" data-regression="` + escapeAttr(regression) + `" data-reproducibility="` + escapeAttr(reproducibility) + `" data-cause="` + escapeAttr(causeState) + `" data-regression-test="` + escapeAttr(regressionTestState) + `" data-archive="` + archiveState + `"><div class="card-kicker">` + statusChip + `<span class="badge">` + escapeHTML(localizedTypeLabel(model, document.Type)) + `</span>` + archiveBadge + `</div><h3><a href="` + escapeAttr(relativeURL(current, document.OutputPath)) + `">` + escapeHTML(document.Title) + `</a></h3><p>` + escapeHTML(truncate(document.Description, 180)) + `</p>` + workDetails + renderProgress(ui, document.TaskStats, ui.Text("progress.tasks")) + `<div class="card-path">` + escapeHTML(document.SourcePath) + `</div></article>`
}

func filterControls(ui frontend.UI, includeStatus, includeType bool) string {
	statusControl := ""
	if includeStatus {
		statusControl = `<select data-filter-control="status"><option value="all">` + escapeHTML(ui.Text("filter.allStatuses")) + `</option><option value="not-started">` + escapeHTML(ui.Text("filter.draft")) + `</option><option value="done">` + escapeHTML(ui.Text("filter.done")) + `</option><option value="in-progress">` + escapeHTML(ui.Text("filter.inProgress")) + `</option><option value="planned">` + escapeHTML(ui.Text("filter.planned")) + `</option><option value="blocked">` + escapeHTML(ui.Text("filter.blocked")) + `</option><option value="cancelled">` + escapeHTML(ui.Text("filter.cancelled")) + `</option></select>`
	}
	typeControl := ""
	if includeType {
		typeControl = `<select data-filter-control="type"><option value="all">` + escapeHTML(ui.Text("filter.allTypes")) + `</option><option value="module">` + escapeHTML(ui.Text("filter.modules")) + `</option><option value="use-case">` + escapeHTML(ui.Text("filter.useCases")) + `</option><option value="screen-map">` + escapeHTML(ui.Text("filter.screenMaps")) + `</option><option value="screen">` + escapeHTML(ui.Text("filter.screens")) + `</option><option value="flow">` + escapeHTML(ui.Text("filter.flows")) + `</option><option value="architecture">` + escapeHTML(ui.Text("filter.architecture")) + `</option><option value="decision">` + escapeHTML(ui.Text("filter.decisions")) + `</option><option value="work">` + escapeHTML(ui.Text("filter.work")) + `</option></select>`
	}
	return `<div class="collection-controls"><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("filter.placeholder")) + `" aria-label="` + escapeAttr(ui.Text("filter.placeholder")) + `">` + statusControl + typeControl + `</div>`
}

func workFilterControls(ui frontend.UI, includeStatus bool) string {
	statusControl := ""
	if includeStatus {
		statusControl = `<select data-filter-control="status"><option value="all">` + escapeHTML(ui.Text("filter.allStatuses")) + `</option><option value="not-started">` + escapeHTML(ui.Text("filter.draft")) + `</option><option value="done">` + escapeHTML(ui.Text("filter.done")) + `</option><option value="in-progress">` + escapeHTML(ui.Text("filter.inProgress")) + `</option><option value="planned">` + escapeHTML(ui.Text("filter.planned")) + `</option><option value="blocked">` + escapeHTML(ui.Text("filter.blocked")) + `</option><option value="cancelled">` + escapeHTML(ui.Text("filter.cancelled")) + `</option></select>`
	}
	return `<div class="collection-controls"><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("filter.placeholder")) + `" aria-label="` + escapeAttr(ui.Text("filter.placeholder")) + `">` +
		statusControl +
		`<select data-filter-control="workType"><option value="all">` + escapeHTML(ui.Text("filter.all")) + `</option><option value="feature">` + escapeHTML(ui.Text("filter.feature")) + `</option><option value="bug">` + escapeHTML(ui.Text("filter.bug")) + `</option><option value="maintenance">` + escapeHTML(ui.Text("filter.maintenance")) + `</option><option value="documentation">` + escapeHTML(ui.Text("filter.documentation")) + `</option><option value="research">` + escapeHTML(ui.Text("filter.research")) + `</option></select>` +
		`<select data-filter-control="severity"><option value="all">` + escapeHTML(ui.Text("filter.anySeverity")) + `</option><option value="critical">` + escapeHTML(ui.Text("filter.critical")) + `</option><option value="high">` + escapeHTML(ui.Text("filter.high")) + `</option></select>` +
		`<select data-filter-control="regression"><option value="all">` + escapeHTML(ui.Text("filter.anyRegression")) + `</option><option value="true">` + escapeHTML(ui.Text("filter.regressions")) + `</option></select>` +
		`<select data-filter-control="reproducibility"><option value="all">` + escapeHTML(ui.Text("filter.anyReproducibility")) + `</option><option value="always">` + escapeHTML(ui.Text("filter.alwaysReproduced")) + `</option><option value="missing">` + escapeHTML(ui.Text("filter.notReproduced")) + `</option></select>` +
		`<select data-filter-control="cause"><option value="all">` + escapeHTML(ui.Text("filter.anyCause")) + `</option><option value="missing">` + escapeHTML(ui.Text("filter.noCause")) + `</option></select>` +
		`<select data-filter-control="regressionTest"><option value="all">` + escapeHTML(ui.Text("filter.anyRegressionTest")) + `</option><option value="missing">` + escapeHTML(ui.Text("filter.noRegressionTest")) + `</option></select>` +
		`<select data-filter-control="archive" data-filter-default="active"><option value="active" selected>` + escapeHTML(ui.Text("filter.active")) + `</option><option value="archived">` + escapeHTML(ui.Text("filter.archive")) + `</option><option value="all">` + escapeHTML(ui.Text("filter.all")) + `</option></select></div>`
}

func documentsHaveDifferentStatuses(documents []*Document) bool {
	if len(documents) < 2 {
		return false
	}
	first := documents[0].Status.Kind
	for _, document := range documents[1:] {
		if document.Status.Kind != first {
			return true
		}
	}
	return false
}

func renderComputedStatus(model *Model, current string) string {
	ui := portalUI(model)
	var active strings.Builder
	for _, item := range model.CurrentStatus.ActiveWork {
		href := "#"
		if document := model.DocByPath[item.Document]; document != nil {
			href = relativeURL(current, document.OutputPath) + "#" + item.Anchor
		}
		writeStrings(&active, `<tr><td><a href="`, escapeAttr(href), `">`, escapeHTML(item.ID), `</a></td><td>`, escapeHTML(item.Title), `</td><td>`, renderStatusChip(model, item.Status), `</td><td>`, escapeHTML(item.ModuleID), `</td></tr>`)
	}
	activeHTML := `<p class="empty-state">` + escapeHTML(ui.Text("status.noActive")) + `</p>`
	if active.Len() > 0 {
		activeHTML = `<div class="data-table"><table><thead><tr><th>ID</th><th>` + escapeHTML(ui.Text("status.task")) + `</th><th>` + escapeHTML(ui.Text("field.status")) + `</th><th>` + escapeHTML(ui.Text("field.module")) + `</th></tr></thead><tbody>` + active.String() + `</tbody></table></div>`
	}

	var blockers strings.Builder
	for _, blocker := range model.CurrentStatus.Blockers {
		href := "#"
		if document := model.DocByPath[blocker.Document]; document != nil {
			href = relativeURL(current, document.OutputPath) + "#" + blocker.Anchor
		}
		text := blocker.Text
		if text == "" {
			text = ui.Text("status.noBlockerReason")
		}
		writeStrings(&blockers, `<li><a href="`, escapeAttr(href), `">`, escapeHTML(blocker.TaskID), `</a> — `, escapeHTML(text), `</li>`)
	}
	blockersHTML := `<p>` + escapeHTML(ui.Text("status.noBlockers")) + `</p>`
	if blockers.Len() > 0 {
		blockersHTML = `<ul class="related-list">` + blockers.String() + `</ul>`
	}

	nextHTML := `<p>` + escapeHTML(ui.Text("status.roadmapComplete")) + `</p>`
	if next := model.CurrentStatus.NextResult; next != nil {
		target := next.TargetDocument
		if target == "" {
			target = next.Document
		}
		href := "#"
		if document := model.DocByPath[target]; document != nil {
			href = relativeURL(current, document.OutputPath)
		}
		status := StatusFor("planned")
		if next.CompletionSource == "use-case-status" && next.TargetStatus != nil {
			status = *next.TargetStatus
		}
		text := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(next.Text, next.ID), " :—-"))
		nextHTML = `<div class="card-kicker">` + renderStatusChip(model, status) + `</div><p><a href="` + escapeAttr(href) + `"><strong>` + escapeHTML(next.ID) + `</strong></a> — ` + escapeHTML(text) + `</p>`
	}

	return `<section class="dashboard-section dashboard-support-panel"><div class="section-heading"><div><h2>` + escapeHTML(ui.Text("status.computed")) + `</h2><p>` + escapeHTML(ui.Text("status.computedHelp")) + `</p></div></div><h3>` + escapeHTML(ui.Text("status.active")) + `</h3>` + activeHTML + `<h3>` + escapeHTML(ui.Text("status.blockers")) + `</h3>` + blockersHTML + `<h3>` + escapeHTML(ui.Text("status.next")) + `</h3>` + nextHTML + `</section>`
}

func renderRecommendedEntries(model *Model) string {
	ui := portalUI(model)
	type entry struct{ title, description, href string }
	entries := []entry{}
	seen := map[string]bool{}
	add := func(title, description, href string) {
		if href == "" || seen[href] || len(entries) >= 5 {
			return
		}
		seen[href] = true
		entries = append(entries, entry{title: title, description: description, href: href})
	}
	if overview := model.DocByPath["architecture/overview.md"]; overview != nil {
		add(ui.Text("start.architecture"), ui.Text("start.architectureHelp"), overview.OutputPath)
	}
	if len(model.Knowledge.UseCases) > 0 {
		add(ui.Text("start.useCases"), ui.Text("start.useCasesHelp"), "use-cases/index.html")
	}
	hasGuides := false
	for _, document := range model.Documents {
		if document.Directory == "guides" {
			hasGuides = true
			break
		}
	}
	if hasGuides {
		add(ui.Text("start.guides"), ui.Text("start.guidesHelp"), outputForDirectory(model, "guides"))
	}
	if len(model.Knowledge.WorkItems) > 0 {
		add(ui.Text("start.work"), ui.Text("start.workHelp"), "work/index.html")
	}
	add(ui.Text("start.quality"), ui.Text("start.qualityHelp"), model.HealthOutputPath)
	for _, document := range model.Documents {
		if len(entries) >= 3 {
			break
		}
		if document.SourcePath != "index.md" {
			add(document.Title, document.Description, document.OutputPath)
		}
	}
	var cards strings.Builder
	for _, item := range entries {
		writeStrings(&cards, `<a class="recommended-entry" href="`, escapeAttr(item.href), `"><strong>`, escapeHTML(item.title), `</strong><span>`, escapeHTML(item.description), `</span></a>`)
	}
	return `<section class="dashboard-section recommended-entries"><div class="section-heading"><div><h2>` + escapeHTML(ui.Text("start.title")) + `</h2><p>` + escapeHTML(ui.Text("start.help")) + `</p></div></div><div class="recommended-entry-grid">` + cards.String() + `</div></section>`
}

func renderDashboardFocus(model *Model) string {
	ui := portalUI(model)
	roadmap := model.DocByPath["roadmap.md"]
	risksDocument := model.DocByPath["risks.md"]
	statusDocument := model.Project.StatusDocument
	if statusDocument == nil && roadmap == nil && len(model.Knowledge.WorkItems) == 0 && risksDocument == nil {
		return ""
	}

	nextLabel := ui.Text("focus.noNext")
	nextHref := ""
	if next := model.CurrentStatus.NextResult; next != nil {
		target := next.TargetDocument
		if target == "" {
			target = next.Document
		}
		href := "#"
		if document := model.DocByPath[target]; document != nil {
			href = relativeURL("index.html", document.OutputPath)
		}
		text := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(next.Text, next.ID), " :—-"))
		if text == "" {
			text = next.ID
		}
		nextLabel = strings.TrimSpace(next.ID + " · " + text)
		if href != "#" {
			nextHref = href
		}
	}

	workTarget := ""
	if len(model.Knowledge.WorkItems) > 0 {
		workTarget = "work/index.html"
	}
	workHref := ""
	if workTarget != "" {
		workHref = relativeURL("index.html", workTarget)
	}
	riskHref := ""
	if risksDocument != nil {
		riskHref = relativeURL("index.html", risksDocument.OutputPath)
	}
	target := ""
	if statusDocument != nil {
		target = relativeURL("index.html", statusDocument.OutputPath)
	} else if nextHref != "" {
		target = nextHref
	} else if workHref != "" {
		target = workHref
	} else {
		target = riskHref
	}
	statusLabel := strings.TrimSpace(model.Project.Status.Label)
	if model.Project.Status.Kind == "neutral" && model.Project.Status.Recognized {
		statusLabel = ui.Text("status.neutral")
	} else if statusLabel == "" {
		statusLabel = ui.Text("focus.projectStatus")
	}
	content := `<span class="focus-status">` + escapeHTML(statusLabel) + `</span><span class="focus-result"><span>` + escapeHTML(ui.Text("focus.next")) + `</span><strong>` + escapeHTML(nextLabel) + `</strong></span><span class="focus-arrow" aria-hidden="true"></span>`
	if target == "" {
		return `<div class="dashboard-section dashboard-focus" aria-label="` + escapeAttr(ui.Text("focus.current")) + `">` + content + `</div>`
	}
	return `<a class="dashboard-section dashboard-focus" aria-label="` + escapeAttr(ui.Text("focus.currentValue", nextLabel)) + `" href="` + escapeAttr(target) + `">` + content + `</a>`
}

func renderDashboard(model *Model) string {
	ui := portalUI(model)
	meta := ""
	if values := nonEmpty([]string{model.Project.Stage, model.Project.Updated}); len(values) > 0 {
		meta = `<div class="hero-meta">` + escapeHTML(strings.Join(values, " · ")) + `</div>`
	}
	overview := ""
	if document := model.Project.OverviewDocument; document != nil {
		body := renderDocumentBody(model, document, linkResolverFor(model, document), nil)
		if body != "" {
			overview = `<section class="dashboard-section dashboard-overview" data-dashboard-overview aria-labelledby="dashboard-overview-title"><div class="dashboard-overview-heading"><div><h2 id="dashboard-overview-title">` + escapeHTML(ui.Text("dashboard.overview")) + `</h2><small>` + escapeHTML(ui.Text("dashboard.fullIndex")) + `</small></div><div class="page-actions dashboard-page-actions">` + renderDocumentContextButton(model, document) + `</div></div><div class="dashboard-overview-body"><article class="doc-content">` + body + `</article></div></section>`
		}
	}
	title := `<div class="dashboard-title"><h1>` + escapeHTML(model.Project.Title) + `</h1><span class="beta-badge">Beta</span></div>`
	hero := `<header class="page-header dashboard-about">` + title + `<p class="page-lead">` + escapeHTML(model.Project.Description) + `</p>` + meta + `</header>`
	if model.SiteConfig.Hero.Enabled {
		heroLogo := ""
		if logo := brandingOutput(model, "logo"); logo != "" {
			heroLogo = `<img class="hero-logo" src="` + escapeAttr(logo) + `" alt="">`
		}
		heroImage := ""
		heroClass := "hero"
		if image := brandingOutput(model, "hero"); image != "" {
			heroClass += " has-image"
			heroImage = `<div class="hero-media"><img src="` + escapeAttr(image) + `" alt=""></div>`
		}
		hero = `<header class="` + heroClass + ` dashboard-about"><div class="hero-copy">` + heroLogo + title + `<p class="hero-description">` + escapeHTML(model.Project.Description) + `</p>` + meta + `</div>` + heroImage + `</header>`
	}
	content := hero + renderDashboardFocus(model) + renderRecommendedEntries(model) + overview
	return pageShell(model, "index.html", model.Project.Title, model.Project.Description, content, "")
}

func renderNotFoundPage(model *Model, current string) string {
	ui := portalUI(model)
	content := `<section class="not-found" aria-labelledby="not-found-title"><div class="not-found-code" aria-hidden="true">404</div><div class="not-found-copy"><h1 id="not-found-title">` + escapeHTML(ui.Text("notFound.title")) + `</h1><p>` + escapeHTML(ui.Text("notFound.description")) + `</p><a class="button primary" href="` + escapeAttr(relativeURL(current, "index.html")) + `">` + escapeHTML(ui.Text("notFound.action")) + `</a></div></section>`
	return pageShell(model, current, ui.Text("notFound.title"), ui.Text("notFound.description"), content, "")
}

func nonEmpty(values []string) []string {
	out := []string{}
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func renderDirectoryPage(model *Model, directory string) string {
	if sectionTypeForPath(directory) == SectionWork {
		return renderTaskWorkspacePage(model)
	}
	ui := portalUI(model)
	current := path.Join(directory, "index.html")
	docs := []*Document{}
	prefix := directory + "/"
	for _, doc := range model.Documents {
		if strings.HasPrefix(doc.SourcePath, prefix) && !strings.EqualFold(doc.FileName, "index.md") {
			docs = append(docs, doc)
		}
	}
	var cards strings.Builder
	for _, doc := range docs {
		cards.WriteString(docCard(model, current, doc))
	}
	collection := `<div class="card-grid">` + cards.String() + `</div>`
	if len(docs) > 1 || directory == "work" && len(docs) > 0 {
		controls := filterControls(ui, documentsHaveDifferentStatuses(docs), false)
		if directory == "work" {
			controls = workFilterControls(ui, documentsHaveDifferentStatuses(docs))
		}
		collection = `<section data-filter-scope>` + controls + `<div class="collection-summary">` + escapeHTML(ui.Text("collection.shown")) + ` <strong data-filter-count></strong></div>` + collection + `<div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("collection.empty")) + `</div></section>`
	}
	title := modelDirectoryLabel(model, directory)
	content := breadcrumbs(model, current, title) + `<header class="page-header"><h1>` + escapeHTML(title) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("directory.count", len(docs))) + `</p></header>` + collection
	return pageShell(model, current, title, title, content, "")
}

func renderKnowledgeCatalogPage(model *Model, kind string) string {
	ui := portalUI(model)
	current := path.Join(kind, "index.html")
	title := modelDirectoryLabel(model, kind)
	var cards strings.Builder
	if kind == "quality" {
		for _, standard := range model.Knowledge.Standards {
			document := model.DocByPath[standard.Document]
			if document != nil {
				cards.WriteString(docCard(model, current, document))
			}
		}
		content := breadcrumbs(model, current, title) +
			`<header class="page-header"><h1>` + escapeHTML(title) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("quality.description")) + `</p></header>` +
			`<section data-filter-scope>` + filterControls(ui, true, false) + `<div class="collection-summary">` + escapeHTML(ui.Text("collection.shown")) + ` <strong data-filter-count></strong></div><div class="card-grid">` +
			cards.String() + `</div><div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("quality.none")) + `</div></section>`
		return pageShell(model, current, title, title, content, "")
	}
	for _, runbook := range model.Knowledge.Runbooks {
		document := model.DocByPath[runbook.Document]
		if document == nil {
			continue
		}
		searchText := strings.Join([]string{runbook.ID, runbook.Title, runbook.Environment, runbook.Risk}, " ")
		writeStrings(&cards,
			`<article class="document-card" data-filter-item data-search="`, escapeAttr(searchText),
			`" data-status="`, escapeAttr(document.Status.Kind), `" data-freshness="`, escapeAttr(runbook.Freshness),
			`"><div class="card-kicker">`, renderStatusChip(model, document.Status), `<span class="badge">`, escapeHTML(runbook.Freshness),
			`</span></div><h3><a href="`, escapeAttr(relativeURL(current, document.OutputPath)), `">`, escapeHTML(runbook.Title),
			`</a></h3><p>`, escapeHTML(truncate(document.Description, 180)), `</p><p class="table-subtext">`, escapeHTML(ui.Text("runbooks.environment")), `: `,
			escapeHTML(fallbackDash(runbook.Environment)), ` · `, escapeHTML(ui.Text("runbooks.risk")), `: `, escapeHTML(fallbackDash(localizedSemanticValue(ui, "risk", runbook.Risk))),
			` · `, escapeHTML(ui.Text("runbooks.lastVerified")), `: `, escapeHTML(fallbackDash(runbook.LastVerified)), `</p><div class="card-path">`,
			escapeHTML(runbook.Document), `</div></article>`)
	}
	controls := `<div class="collection-controls"><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("filter.placeholder")) + `" aria-label="` + escapeAttr(ui.Text("filter.placeholder")) + `">` +
		`<select data-filter-control="freshness"><option value="all">` + escapeHTML(ui.Text("runbooks.anyFreshness")) + `</option><option value="recent">` + escapeHTML(ui.Text("runbooks.recent")) + `</option><option value="review-required">` + escapeHTML(ui.Text("runbooks.reviewRequired")) + `</option><option value="overdue">` + escapeHTML(ui.Text("runbooks.overdue")) + `</option></select></div>`
	content := breadcrumbs(model, current, title) +
		`<header class="page-header"><h1>` + escapeHTML(title) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("runbooks.description")) + `</p></header>` +
		`<section class="metric-grid">` + metricCard(ui.Text("metric.total"), model.Stats.RunbooksTotal, "") + metricCard(ui.Text("runbooks.recent"), model.Stats.RunbooksRecent, "") +
		metricCard(ui.Text("runbooks.reviewRequired"), model.Stats.RunbooksReviewRequired, "") + metricCard(ui.Text("runbooks.overdue"), model.Stats.RunbooksOverdue, "") + `</section>` +
		`<section data-filter-scope>` + controls + `<div class="collection-summary">` + escapeHTML(ui.Text("collection.shown")) + ` <strong data-filter-count></strong></div><div class="card-grid">` +
		cards.String() + `</div><div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("runbooks.none")) + `</div></section>`
	return pageShell(model, current, title, title, content, "")
}

func renderHealthPage(model *Model) string {
	ui := portalUI(model)
	current := model.HealthOutputPath
	var rows strings.Builder
	for _, issue := range model.Issues {
		location := escapeHTML(issue.DocumentPath)
		if doc := model.DocByPath[issue.DocumentPath]; doc != nil {
			location = `<a href="` + escapeAttr(relativeURL(current, doc.OutputPath)) + `">` + escapeHTML(issue.DocumentPath) + `</a>`
		}
		_, _ = fmt.Fprintf(&rows, `<div class="issue-row" data-filter-item data-search="%s" data-severity="%s" data-code="%s"><div class="issue-severity">%s</div><div><strong>%s</strong><span class="table-subtext">%s</span></div><div class="issue-location">%s%s</div></div>`, escapeAttr(issue.Message+" "+issue.Code+" "+issue.DocumentPath), escapeAttr(issue.Severity), escapeAttr(issue.Code), ui.Text(map[bool]string{true: "health.error", false: "health.warning"}[issue.Severity == "error"]), escapeHTML(issue.Message), escapeHTML(issue.Code), location, func() string {
			if issue.Line > 0 {
				return " · " + escapeHTML(ui.Text("health.line", issue.Line))
			}
			return ""
		}())
	}
	content := breadcrumbs(model, current, ui.Text("health.title")) + `<header class="page-header"><h1>` + escapeHTML(ui.Text("health.title")) + `</h1><p class="page-lead">` + escapeHTML(ui.Text("health.description")) + `</p></header><section class="metric-grid">` + metricCard(ui.Text("metric.documents"), model.Stats.Documents, "") + metricCard(ui.Text("metric.warnings"), model.Stats.Warnings, "") + metricCard(ui.Text("metric.errors"), model.Stats.Errors, "") + metricCard(ui.Text("metric.brokenLinks"), model.Stats.BrokenLinks, "") + `</section><section class="dashboard-section"><div class="section-heading"><h2>` + escapeHTML(ui.Text("health.issues")) + `</h2><a href="` + escapeAttr(relativeURL(current, model.ReportOutputPath)) + `">report.json</a></div><div data-filter-scope><div class="collection-controls"><input type="search" data-filter-control="search" placeholder="` + escapeAttr(ui.Text("health.search")) + `"><select data-filter-control="severity"><option value="all">` + escapeHTML(ui.Text("health.allLevels")) + `</option><option value="warning">` + escapeHTML(ui.Text("metric.warnings")) + `</option><option value="error">` + escapeHTML(ui.Text("metric.errors")) + `</option></select></div><div class="collection-summary">` + escapeHTML(ui.Text("collection.shown")) + ` <strong data-filter-count></strong></div><div class="issue-list">` + rows.String() + `</div><div class="empty-state" data-filter-empty hidden>` + escapeHTML(ui.Text("health.none")) + `</div></div></section>`
	return pageShell(model, current, ui.Text("health.title"), ui.Text("health.description"), content, "")
}

func BuildReport(model *Model) ProjectReport {
	documents := []ReportDocument{}
	for _, doc := range model.Documents {
		links := []ReportLink{}
		for _, link := range doc.ResolvedLinks {
			target := ""
			kind := ""
			if link.TargetDocument != nil {
				target = link.TargetDocument.SourcePath
				kind = "document"
			} else if link.RepositoryPath != "" {
				target = link.RepositoryPath
				kind = "repository"
			} else if link.AssetPath != "" {
				target = link.AssetPath
				kind = "asset"
			} else if link.GeneratedTarget != "" {
				target = link.GeneratedTarget
				kind = "directory"
			} else if link.External {
				kind = "external"
			}
			links = append(links, ReportLink{
				Destination: link.Destination, Broken: link.Broken, Blocked: link.Blocked,
				TargetKind: kind, Target: target, Href: link.Href,
			})
		}
		backlinks := []string{}
		for _, item := range doc.Backlinks {
			backlinks = append(backlinks, item.SourcePath)
		}
		related := []string{}
		for _, item := range doc.RelatedDocuments {
			related = append(related, item.SourcePath)
		}
		documents = append(documents, ReportDocument{
			ID: doc.Metadata["id"], SourcePath: doc.SourcePath, OutputPath: doc.OutputPath,
			Type: doc.Type, SectionType: doc.SectionType, Title: doc.Title, Description: doc.Description, Metadata: doc.Metadata,
			Status: doc.Status, TaskStats: doc.TaskStats, UpdatedAt: doc.UpdatedAt, Stale: doc.Stale,
			Warnings: len(doc.Warnings), Errors: len(doc.Errors), Links: links,
			Backlinks: backlinks, RelatedDocuments: related,
		})
	}
	roadmap := []ReportRoadmapStage{}
	for _, stage := range model.RoadmapStages {
		roadmap = append(roadmap, ReportRoadmapStage{
			Title: stage.Title, Status: stage.Status, PlannedDate: stage.PlannedDate,
			TaskStats: stage.TaskStats, Items: append([]RoadmapItem{}, stage.Items...),
			Document: stage.Document.SourcePath, Anchor: stage.Anchor,
		})
	}
	risks := []ReportRisk{}
	for _, risk := range model.Risks {
		risks = append(risks, ReportRisk{
			ID: risk.ID, Title: risk.Title, Status: risk.Status, Probability: risk.Probability,
			Impact: risk.Impact, TaskStats: risk.TaskStats,
			Document: risk.Document.SourcePath, Anchor: risk.Anchor,
		})
	}
	project := ReportProject{
		Title: model.Project.Title, Description: model.Project.Description, Status: model.Project.Status,
		Stage: model.Project.Stage, Updated: model.Project.Updated, Summary: model.Project.Summary,
	}
	screens := make([]ReportScreen, 0, len(model.Knowledge.Screens))
	for _, screen := range model.Knowledge.Screens {
		status := screen.Status.Kind
		if status == "done" {
			status = "implemented"
		}
		screens = append(screens, ReportScreen{
			ID: screen.ID, Title: screen.Title, Description: screen.Description,
			Module: screen.ModuleID, Type: screen.Kind, Status: status,
			Route: screen.Route, Preview: screen.Preview, Component: screen.Component,
			Updated: screen.Updated, Parent: screen.ParentID, States: append([]ScreenState{}, screen.States...),
			IncomingTransitions: append([]string{}, screen.IncomingTransitionIDs...),
			OutgoingTransitions: append([]string{}, screen.OutgoingTransitionIDs...),
			UseCases:            append([]string{}, screen.UseCaseIDs...), WorkItems: append([]string{}, screen.WorkItemIDs...),
			Contracts: append([]string{}, screen.ContractDocuments...), Document: screen.Document,
		})
	}
	return ProjectReport{
		SchemaVersion: 1, Generator: GeneratorInfo{Name: "Toudocu", Version: Version},
		GeneratedAt: model.GeneratedAt, SourceDirectory: pathBase(model.RootDirectory),
		StaleDays: model.StaleDays, Project: project, CurrentStatus: model.CurrentStatus,
		Stats: model.Stats, Documents: documents, Roadmap: roadmap, Risks: risks,
		Knowledge: ReportKnowledge{
			Modules: model.Knowledge.Modules, UseCases: model.Knowledge.UseCases,
			Flows: model.Knowledge.Flows, Standards: model.Knowledge.Standards, Runbooks: model.Knowledge.Runbooks,
			BusinessRules: model.Knowledge.BusinessRules, WorkItems: model.Knowledge.WorkItems,
		},
		Screens: screens, Transitions: model.Knowledge.Transitions,
		PlayableFlows: model.Knowledge.PlayableFlows, Hotspots: model.Knowledge.Hotspots,
		ErrorDefinitions: model.Knowledge.Errors, Traceability: model.Knowledge.Traceability,
		Issues: append([]Issue{}, model.Issues...),
	}
}

func ensureOutputSafety(inputDirectory, outputDirectory string) error {
	input, _ := filepath.Abs(inputDirectory)
	output, _ := filepath.Abs(outputDirectory)
	resolvedInput, err := resolvePathForSafety(input)
	if err != nil {
		return err
	}
	resolvedOutput, err := resolvePathForSafety(output)
	if err != nil {
		return err
	}
	if samePath(resolvedInput, resolvedOutput) {
		return fmt.Errorf("output directory cannot match the source documentation directory")
	}
	volume := filepath.VolumeName(resolvedOutput)
	root := string(filepath.Separator)
	if volume != "" {
		root = volume + string(filepath.Separator)
	}
	if samePath(resolvedOutput, root) {
		return fmt.Errorf("filesystem root cannot be used as the output directory")
	}
	if ensureInside(resolvedOutput, resolvedInput) {
		return fmt.Errorf("output directory cannot be a parent of the source documentation directory")
	}
	return nil
}

// GenerateSite writes a backend-independent portal for HTTP(S) static hosting.
func GenerateSite(model *Model, options Options) (GenerateResult, error) {
	return generateSite(model, options, false)
}

func generateServeSite(model *Model, options Options) (GenerateResult, error) {
	return generateSite(model, options, true)
}

func generateSite(model *Model, options Options, serve bool) (GenerateResult, error) {
	if issue := documentationVersionIssue(model.SiteConfig); issue != nil {
		return GenerateResult{}, documentationVersionError(issue)
	}
	model.serveMode = serve
	defer func() { model.serveMode = false }()
	model.ScreenMapEnabled = !options.NoScreenMap
	output, err := filepath.Abs(options.OutputDirectory)
	if err != nil {
		return GenerateResult{}, err
	}
	if err = ensureOutputSafety(model.RootDirectory, output); err != nil {
		return GenerateResult{}, err
	}
	if options.Clean {
		if _, statErr := os.Stat(output); statErr == nil {
			if err = safeRemoveDirectory(output, model.RootDirectory); err != nil {
				return GenerateResult{}, err
			}
		}
	}
	if err = mkdirp(output); err != nil {
		return GenerateResult{}, err
	}
	if err = rejectSymlinks(output); err != nil {
		return GenerateResult{}, err
	}
	runtime := "static"
	if serve {
		runtime = "serve"
	}
	assets, err := frontend.RuntimeAssets(runtime)
	if err != nil {
		return GenerateResult{}, err
	}
	generated, err := frontend.GeneratedFS()
	if err != nil {
		return GenerateResult{}, err
	}
	if !serve {
		serveAssets, manifestErr := frontend.RuntimeAssets("serve")
		if manifestErr != nil {
			return GenerateResult{}, manifestErr
		}
		staticSet := map[string]struct{}{}
		for _, asset := range assets {
			staticSet[asset] = struct{}{}
		}
		for _, asset := range serveAssets {
			if _, keep := staticSet[asset]; keep {
				continue
			}
			if removeErr := os.Remove(filepath.Join(output, "assets", filepath.FromSlash(asset))); removeErr != nil && !os.IsNotExist(removeErr) {
				return GenerateResult{}, removeErr
			}
		}
	}
	for _, asset := range assets {
		if err = copyFSFile(generated, asset, filepath.Join(output, "assets", filepath.FromSlash(asset))); err != nil {
			return GenerateResult{}, err
		}
	}
	searchIndex := append([]SearchItem{}, model.SearchIndex...)
	if model.ProjectChangelog != nil {
		searchIndex = append(searchIndex, projectChangelogSearchItem(model, model.ProjectChangelog))
	}
	if err = writeStaticData(output, model, searchIndex); err != nil {
		return GenerateResult{}, err
	}
	for outputPath, sourcePath := range model.Assets {
		if err = copyFileEnsured(sourcePath, filepath.Join(output, filepath.FromSlash(outputPath))); err != nil {
			return GenerateResult{}, err
		}
	}
	for outputPath, sourcePath := range model.BrandingAssets {
		if err = copyFileEnsured(sourcePath, filepath.Join(output, filepath.FromSlash(outputPath))); err != nil {
			return GenerateResult{}, err
		}
	}
	for _, contract := range model.openAPIContracts {
		if err = copyFileEnsured(filepath.Join(model.RootDirectory, filepath.FromSlash(contract.Path)), filepath.Join(output, filepath.FromSlash(contract.Path))); err != nil {
			return GenerateResult{}, err
		}
	}
	if err = writeFileEnsured(filepath.Join(output, "index.html"), []byte(renderDashboard(model))); err != nil {
		return GenerateResult{}, err
	}
	if err = writeFileEnsured(filepath.Join(output, "404.html"), []byte(renderNotFoundPage(model, "404.html"))); err != nil {
		return GenerateResult{}, err
	}
	pages := 2
	for _, document := range model.Documents {
		directory := strings.Split(document.SourcePath, "/")[0]
		typedCatalogIndex := strings.EqualFold(document.FileName, "index.md") && (directory == "use-cases" || directory == "flows" || directory == "quality" || directory == "runbooks" || directory == "drafts")
		if document.SourcePath == "index.md" || document.Type == "screen-index" || typedCatalogIndex {
			continue
		}
		pageHTML := renderDocumentPage(model, document)
		if document.Type == "use-case" {
			pageHTML = renderUseCasePage(model, document)
		}
		if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(document.OutputPath)), []byte(pageHTML)); err != nil {
			return GenerateResult{}, err
		}
		pages++
	}
	if model.ProjectChangelog != nil {
		if err = writeFileEnsured(filepath.Join(output, projectChangelogOutput), []byte(renderDocumentPage(model, model.ProjectChangelog))); err != nil {
			return GenerateResult{}, err
		}
		pages++
	} else if removeErr := os.Remove(filepath.Join(output, projectChangelogOutput)); removeErr != nil && !os.IsNotExist(removeErr) {
		return GenerateResult{}, removeErr
	}
	if len(model.Knowledge.UseCases)+len(model.Knowledge.Flows) > 0 {
		processPages := map[string]string{
			sectionCatalogOutput(SectionFlows):    renderProcessCatalogPage(model, sectionCatalogOutput(SectionFlows), "flow"),
			sectionCatalogOutput(SectionUseCases): renderProcessCatalogPage(model, sectionCatalogOutput(SectionUseCases), "use-case"),
		}
		for target, pageHTML := range processPages {
			if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(target)), []byte(pageHTML)); err != nil {
				return GenerateResult{}, err
			}
			pages++
		}
	}
	if len(model.Knowledge.Screens) > 0 {
		generatedPages := map[string]string{
			"screens/catalog.html": renderScreenCatalogPage(model, "screens/catalog.html"),
			"traceability.html":    renderTraceabilityPage(model, "traceability.html"),
		}
		if model.ScreenMapEnabled {
			generatedPages["screens/index.html"] = renderScreenMapPage(model, "screens/index.html")
		}
		for target, pageHTML := range generatedPages {
			if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(target)), []byte(pageHTML)); err != nil {
				return GenerateResult{}, err
			}
			pages++
		}
	}
	for _, kind := range []string{"quality", "runbooks"} {
		if _, exists := model.Directories[kind]; !exists {
			continue
		}
		target := path.Join(kind, "index.html")
		if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(target)), []byte(renderKnowledgeCatalogPage(model, kind))); err != nil {
			return GenerateResult{}, err
		}
		pages++
	}
	if _, exists := model.Directories["drafts"]; exists {
		target := sectionCatalogOutput(SectionDrafts)
		if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(target)), []byte(renderDirectoryPage(model, "drafts"))); err != nil {
			return GenerateResult{}, err
		}
		pages++
	}
	directories := make([]string, 0, len(model.Directories))
	for d := range model.Directories {
		directories = append(directories, d)
	}
	sort.SliceStable(directories, func(i, j int) bool { return naturalCompare(directories[i], directories[j]) < 0 })
	for _, directory := range directories {
		if directory == "use-cases" || directory == "flows" || directory == "quality" || directory == "runbooks" || directory == "drafts" {
			continue
		}
		if directoryHasSourceIndex(model, directory) {
			continue
		}
		target := path.Join(directory, "index.html")
		if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(target)), []byte(renderDirectoryPage(model, directory))); err != nil {
			return GenerateResult{}, err
		}
		pages++
	}
	if err = writeFileEnsured(filepath.Join(output, filepath.FromSlash(model.HealthOutputPath)), []byte(renderHealthPage(model))); err != nil {
		return GenerateResult{}, err
	}
	pages++
	report, err := json.MarshalIndent(BuildReport(model), "", "  ")
	if err != nil {
		return GenerateResult{}, err
	}
	report = append(report, '\n')
	if err = writeFileEnsured(filepath.Join(output, model.ReportOutputPath), report); err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{OutputDirectory: output, Pages: pages, Assets: len(model.Assets) + len(model.BrandingAssets) + len(assets) + 1}, nil
}
