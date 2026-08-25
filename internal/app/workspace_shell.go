package toudocu

import (
	"html/template"
	"strings"
	frontend "toudocu/internal/site"
)

type workspaceSurface string

const (
	workspacePortal  workspaceSurface = "portal"
	workspaceEditor  workspaceSurface = "editor"
	workspaceChanges workspaceSurface = "changes"
)

func workspaceModel(model *Model) *Model {
	defaults := defaultSiteConfig()
	if model == nil {
		return &Model{Project: ProjectInfo{Title: "Toudocu"}, SiteConfig: defaults}
	}
	copy := *model
	if copy.Project.Title == "" {
		copy.Project.Title = "Toudocu"
	}
	if copy.SiteConfig.Theme == "" {
		copy.SiteConfig.Theme = defaults.Theme
	}
	if copy.SiteConfig.ColorScheme == "" {
		copy.SiteConfig.ColorScheme = defaults.ColorScheme
	}
	if copy.SiteConfig.Accent == "" {
		copy.SiteConfig.Accent = defaults.Accent
	}
	if copy.SiteConfig.Density == "" {
		copy.SiteConfig.Density = defaults.Density
	}
	if copy.SiteConfig.ContentWidth == "" {
		copy.SiteConfig.ContentWidth = defaults.ContentWidth
	}
	return &copy
}

func appearanceAttributes(config SiteConfig) string {
	initial := config.ColorScheme
	if initial == "system" {
		initial = "light"
	}
	return ` data-theme="` + escapeAttr(initial) + `" data-site-theme="` + escapeAttr(config.Theme) +
		`" data-color-scheme="` + escapeAttr(config.ColorScheme) + `" data-accent="` + escapeAttr(config.Accent) +
		`" data-density="` + escapeAttr(config.Density) + `" data-content-width="` + escapeAttr(config.ContentWidth) + `"`
}

func workspaceBrand(model *Model, href string) string {
	mark := `<span class="brand-mark" aria-hidden="true">` + escapeHTML(projectBrandMark(model.Project.Title)) + `</span>`
	if logo := brandingOutput(model, "logo"); logo != "" {
		mark = `<img class="brand-logo" src="/` + escapeAttr(logo) + `" alt="">`
	}
	return `<a class="workspace-brand brand" href="` + escapeAttr(href) + `">` + mark + `<span class="brand-text">` + escapeHTML(model.Project.Title) + `</span></a>`
}

func workspaceNavigation(ui frontend.UI, active workspaceSurface) string {
	items := []struct {
		surface workspaceSurface
		href    string
		label   string
	}{
		{workspacePortal, "/", ui.Text("nav.portal")},
		{workspaceEditor, "/_toudocu/editor/", ui.Text("nav.editor")},
		{workspaceChanges, "/changes/", ui.Text("nav.changes")},
	}
	var b strings.Builder
	b.WriteString(`<nav class="workspace-nav" aria-label="` + escapeAttr(ui.Text("nav.workspaces")) + `">`)
	for _, item := range items {
		current := ""
		if item.surface == active {
			current = ` aria-current="page"`
		}
		b.WriteString(`<a class="workspace-nav-link" href="` + item.href + `" aria-label="` + escapeAttr(ui.Text("nav.open", strings.ToLower(item.label))) + `"` + current + ` data-workspace="` + string(item.surface) + `"><span class="workspace-nav-label">` + escapeHTML(item.label) + `</span></a>`)
	}
	b.WriteString(`</nav>`)
	return b.String()
}

func workspaceAppearanceControls(ui frontend.UI, config SiteConfig) string {
	themeLabel, themeIndicator := siteThemePresentation(ui, config.Theme)
	return `<div class="workspace-appearance" aria-label="` + escapeAttr(ui.Text("header.appearance")) + `">` +
		`<label class="header-select site-theme-select"><span class="header-select-visual" aria-hidden="true"><span class="site-theme-indicator" data-site-theme-indicator>` + escapeHTML(themeIndicator) + `</span><span data-site-theme-label>` + escapeHTML(themeLabel) + `</span></span><select data-site-theme-select aria-label="` + escapeAttr(ui.Text("header.theme")) + `">` + selectOptions(config.Theme, []selectOption{{"classic", ui.Text("theme.classic")}, {"paper", ui.Text("theme.paper")}, {"terminal", ui.Text("theme.terminal")}}) + `</select></label>` +
		`<label class="header-select scheme-select"><span class="header-select-visual" aria-hidden="true"><span class="scheme-toggle-indicator"></span><span data-theme-label>` + escapeHTML(colorSchemeLabel(ui, config.ColorScheme)) + `</span></span><select data-color-scheme-select aria-label="` + escapeAttr(ui.Text("header.scheme")) + `">` + selectOptions(config.ColorScheme, []selectOption{{"system", ui.Text("scheme.system")}, {"light", ui.Text("scheme.light")}, {"dark", ui.Text("scheme.dark")}}) + `</select></label></div>`
}

func discussionToggle(ui frontend.UI) string {
	return `<button class="header-review-toggle" type="button" data-discussions-toggle aria-expanded="false" aria-controls="project-discussions-panel">` +
		escapeHTML(ui.Text("core.portal.054")) + ` · <span data-open-discussion-count>0</span></button>`
}

func workspaceHeader(model *Model, active workspaceSurface) string {
	ui := portalUI(model)
	review := ""
	if active == workspaceChanges && model.translationLocale == "" {
		review = discussionToggle(ui)
	}
	return `<header class="workspace-header">` + workspaceBrand(model, "/") + workspaceNavigation(ui, active) +
		`<div class="workspace-header-actions">` + review + workspaceAppearanceControls(ui, model.SiteConfig) + `</div></header>`
}

func workspaceFavicon(model *Model) string {
	if favicon := brandingOutput(model, "favicon"); favicon != "" {
		return "/" + favicon
	}
	return "/assets/" + mustFrontendAsset("favicon.svg")
}

func workspacePageBootstrap(model *Model, pagePath, assetBase string, capabilities frontend.Capabilities) template.JS {
	locale := model.SiteConfig.Project.Locale
	if locale == "" {
		locale = "en"
	}
	endpoints := &frontend.Endpoints{
		Editor: editorAPIBase, EditorWorkspace: editorUIPath, Changes: changesAPIBase, Rebuild: rebuildEndpoint,
	}
	if capabilities.Review {
		endpoints.Review = reviewAPIBase
	}
	bootstrap, err := frontend.MarshalBootstrap(frontend.PageBootstrap{
		SchemaVersion: 1,
		Runtime:       frontend.RuntimeServe,
		Page:          frontend.PageReference{Kind: "document", Path: pagePath},
		Portal:        frontend.PortalReference{AssetBase: assetBase, DataBase: strings.Replace(assetBase, "assets/", "data/", 1)},
		UI: frontend.UISettings{
			Locale: locale, Theme: model.SiteConfig.Theme, ColorScheme: model.SiteConfig.ColorScheme,
			Accent: model.SiteConfig.Accent, Density: model.SiteConfig.Density, ContentWidth: model.SiteConfig.ContentWidth,
		},
		Capabilities: capabilities,
		Endpoints:    endpoints,
	})
	if err != nil {
		panic(err)
	}
	return bootstrap
}
