package toudocu

import (
	"html/template"
	"io"
	"net/http"
	frontend "toudocu/internal/site"
)

func (s *documentationServer) serveChangesUI(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request.Method == http.MethodHead {
		return
	}
	uiModel := workspaceModel(s.model)
	ui := portalUI(uiModel)
	locale := uiModel.SiteConfig.Project.Locale
	if locale == "" {
		locale = "en"
	}
	html, err := frontend.RenderChanges(frontend.WorkspaceView{
		UI: ui, Lang: locale, HTMLAttributes: template.HTMLAttr(appearanceAttributes(uiModel.SiteConfig)),
		Title: ui.Text("changes.title") + " — " + uiModel.Project.Title, Favicon: workspaceFavicon(uiModel),
		AppearanceJS: "/assets/" + mustFrontendAsset("appearance.js"),
		Styles: []string{
			"/assets/" + mustFrontendAsset("portal.css"),
			"/assets/" + mustFrontendAsset("serve.css"),
			"/assets/" + mustFrontendAsset("changes.css"),
		},
		Scripts: []frontend.ScriptAsset{
			{URL: "/assets/" + mustFrontendAsset("mermaid.tiny.js")},
			{URL: "/assets/" + mustFrontendAsset("codemirror.js"), Module: true},
			{URL: "/assets/" + mustFrontendAsset("serve.js"), Module: true},
			{URL: "/assets/" + mustFrontendAsset("changes.js"), Module: true},
		},
		Bootstrap: workspacePageBootstrap(uiModel, "changes/index.html", "../assets/", frontend.Capabilities{Changes: true, Review: !s.translationReadOnly, AgentConsole: uiModel.agentConsoleEnabled && !s.translationReadOnly}),
		Header:    template.HTML(workspaceHeader(uiModel, workspaceChanges)),
	})
	if err != nil {
		http.Error(w, "Could not render changes viewer", http.StatusInternalServerError)
		return
	}
	_, _ = io.WriteString(w, html)
}
