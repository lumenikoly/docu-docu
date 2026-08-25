package site

import (
	"html/template"
	"strings"
	"testing"
)

func testBootstrap() PageBootstrap {
	return PageBootstrap{
		SchemaVersion: 1,
		Runtime:       RuntimeStatic,
		Page:          PageReference{Kind: "document", Path: "guides/start.html"},
		Portal:        PortalReference{AssetBase: "../assets/", DataBase: "../data/"},
		UI:            UISettings{Locale: "ru", Theme: "classic", ColorScheme: "system", Accent: "indigo", Density: "comfortable", ContentWidth: "standard"},
		Capabilities:  Capabilities{Search: true, Diagrams: true},
	}
}

func TestWorkspaceTemplatesRenderSemanticContent(t *testing.T) {
	view := WorkspaceView{Lang: "ru", Title: "Project", Favicon: "/favicon.svg", AppearanceJS: "/appearance.js", Styles: []string{"/portal.css"}, Bootstrap: template.JS(`{"schemaVersion":1}`)}
	changes, err := RenderChanges(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(changes, "data-changes-root") {
		t.Fatalf("changes body missing: %s", changes)
	}
	for _, removed := range []string{"changes-toolbar", "changes-metrics", "data-classification", "data-git-state", "data-review-type"} {
		if strings.Contains(changes, removed) {
			t.Fatalf("changes includes removed %s: %s", removed, changes)
		}
	}
	editor, err := RenderEditor(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(editor, "data-editor-root") {
		t.Fatalf("editor body missing: %s", editor)
	}
	for name, rendered := range map[string]string{"editor": editor, "changes": changes} {
		appearance := strings.Index(rendered, `<script src="/appearance.js"></script>`)
		stylesheet := strings.Index(rendered, `<link rel="stylesheet" href="/portal.css">`)
		if appearance < 0 || stylesheet < 0 || appearance > stylesheet {
			t.Fatalf("%s appearance bootstrap must precede stylesheets: %s", name, rendered)
		}
	}
	apiDocs, err := RenderAPIDocs(view)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(apiDocs, "swagger-ui") {
		t.Fatalf("api docs body missing: %s", apiDocs)
	}
}

func TestPortalTemplateLoadsAppearanceBeforeStylesheet(t *testing.T) {
	rendered, err := RenderShell(ShellView{
		Lang: "ru", Title: "Project", Favicon: "/favicon.svg", AppearanceJS: "/appearance.js",
		PortalCSS: "/portal.css", Bootstrap: template.JS(`{"schemaVersion":1}`), PortalJS: "/portal.js",
	})
	if err != nil {
		t.Fatal(err)
	}
	appearance := strings.Index(rendered, `<script src="/appearance.js"></script>`)
	stylesheet := strings.Index(rendered, `<link rel="stylesheet" href="/portal.css">`)
	if appearance < 0 || stylesheet < 0 || appearance > stylesheet {
		t.Fatalf("portal appearance bootstrap must precede stylesheet: %s", rendered)
	}
}

func TestBootstrapSerializationEscapesHTMLTermination(t *testing.T) {
	value := testBootstrap()
	value.Page.ID = "</script><script>alert(1)</script>"
	encoded, err := MarshalBootstrap(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "</script") || !strings.Contains(string(encoded), `\u003c/script\u003e`) {
		t.Fatalf("unsafe bootstrap JSON: %s", encoded)
	}
}

func TestStaticBootstrapRejectsEndpointsAndAbsoluteBases(t *testing.T) {
	value := testBootstrap()
	value.Endpoints = &Endpoints{Editor: "/_toudocu/api/editor"}
	if _, err := MarshalBootstrap(value); err == nil {
		t.Fatal("static runtime accepted server endpoints")
	}
	value.Endpoints = nil
	value.Portal.AssetBase = "/assets/"
	if _, err := MarshalBootstrap(value); err == nil {
		t.Fatal("absolute asset base accepted")
	}
}

func TestGeneratedManifest(t *testing.T) {
	manifest, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"appearance.js", "portal.css", "portal.js", "serve.js", "editor.js", "changes.js"} {
		if _, ok := manifest.Assets[name]; !ok {
			t.Fatalf("missing generated asset %s", name)
		}
	}
}
