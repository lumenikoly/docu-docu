package toudocu

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	frontend "toudocu/internal/site"
)

const (
	editorAPIBase   = "/_toudocu/api/editor"
	editorUIPath    = "/_toudocu/editor/"
	editorBodyLimit = 3 << 20
)

type apiRoute struct {
	Path    string
	Methods []string
	Handler func(*documentationServer, http.ResponseWriter, *http.Request)
}

var editorRouteRegistry = []apiRoute{
	{Path: editorAPIBase + "/files", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveEditorFiles},
	{Path: editorAPIBase + "/file", Methods: []string{http.MethodGet, http.MethodPut}, Handler: (*documentationServer).serveEditorFile},
	{Path: editorAPIBase + "/roadmap", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveEditorRoadmap},
	{Path: editorAPIBase + "/roadmap/items", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveEditorRoadmapItems},
	{Path: editorAPIBase + "/preview", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveEditorPreview},
	{Path: editorAPIBase + "/validate", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveEditorValidate},
	{Path: editorAPIBase + "/create", Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveEditorCreate},
}

var editorServiceRouteRegistry = []apiRoute{
	{Path: rebuildEndpoint, Methods: []string{http.MethodPost}, Handler: (*documentationServer).serveRebuild},
	{Path: versionEndpoint, Methods: []string{http.MethodGet, http.MethodHead}, Handler: (*documentationServer).serveVersion},
}

type editorErrorEnvelope struct {
	SchemaVersion int               `json:"schemaVersion"`
	Error         editorErrorDetail `json:"error"`
}

type editorErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type editorRebuild struct {
	Documents int `json:"documents"`
	Pages     int `json:"pages"`
	Warnings  int `json:"warnings"`
	Errors    int `json:"errors"`
}

func (s *documentationServer) rebuildPayload() editorRebuild {
	return editorRebuild{Documents: s.model.Stats.Documents, Pages: s.result.Pages, Warnings: s.model.Stats.Warnings, Errors: s.model.Stats.Errors}
}

func writeEditorJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeEditorError(w http.ResponseWriter, status int, code, message string, details any) {
	writeEditorJSON(w, status, editorErrorEnvelope{SchemaVersion: 1, Error: editorErrorDetail{Code: code, Message: message, Details: details}})
}

func allowEditorMethods(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeEditorError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	return false
}

func editorOriginAllowed(r *http.Request) bool {
	fetchSite := strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))
	if fetchSite != "" && fetchSite != "same-origin" {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return fetchSite == "same-origin"
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.User != nil || parsed.Host != r.Host {
		return false
	}
	expectedScheme := "http"
	if r.TLS != nil {
		expectedScheme = "https"
	}
	return parsed.Scheme == expectedScheme
}

func requireEditorJSONAction(w http.ResponseWriter, r *http.Request, action string) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeEditorError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Content-Type application/json is required", nil)
		return false
	}
	if r.Header.Get("X-Toudocu-Action") != action {
		writeEditorError(w, http.StatusForbidden, "action_forbidden", "Invalid X-Toudocu-Action", nil)
		return false
	}
	if !editorOriginAllowed(r) {
		writeEditorError(w, http.StatusForbidden, "origin_forbidden", "Request must be same-origin", nil)
		return false
	}
	return true
}

func decodeEditorJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, editorBodyLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeEditorError(w, http.StatusRequestEntityTooLarge, "request_too_large", "JSON body exceeds 3 MiB", nil)
		} else {
			writeEditorError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON: "+err.Error(), nil)
		}
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeEditorError(w, http.StatusBadRequest, "invalid_json", "Unexpected data after the JSON object", nil)
		return false
	}
	return true
}

func editorStatusForError(err error) (int, string) {
	switch workspaceErrorCode(err) {
	case "invalid_path":
		return http.StatusBadRequest, "invalid_path"
	case "unsupported_extension":
		return http.StatusUnsupportedMediaType, "unsupported_extension"
	case "path_forbidden":
		return http.StatusForbidden, "path_forbidden"
	case "file_not_found":
		return http.StatusNotFound, "file_not_found"
	case "content_too_large":
		return http.StatusRequestEntityTooLarge, "content_too_large"
	default:
		return http.StatusInternalServerError, "workspace_error"
	}
}

func (s *documentationServer) serveEditorAPI(w http.ResponseWriter, r *http.Request) {
	for _, route := range editorRouteRegistry {
		if r.URL.Path == route.Path {
			if !allowEditorMethods(w, r, route.Methods...) {
				return
			}
			route.Handler(s, w, r)
			return
		}
	}
	writeEditorError(w, http.StatusNotFound, "route_not_found", "Editor API route not found", nil)
}

func (s *documentationServer) serveEditorFiles(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodGet) {
		return
	}
	revision := s.revision
	if revision != "" {
		etag := `"` + revision + `"`
		w.Header().Set("ETag", etag)
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	files, scannedRevision, err := s.workspace.scan(s.model)
	if err != nil {
		writeEditorError(w, http.StatusInternalServerError, "workspace_error", err.Error(), nil)
		return
	}
	if revision == "" {
		revision = scannedRevision
	}
	etag := `"` + revision + `"`
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeEditorJSON(w, http.StatusOK, struct {
		SchemaVersion int                 `json:"schemaVersion"`
		Revision      string              `json:"revision"`
		Files         []editorFileSummary `json:"files"`
		Templates     []editorTemplate    `json:"templates"`
	}{1, revision, editorFileSummaries(files), editorTemplates(s.model)})
}

func editorFileSummaries(files []editorFile) []editorFileSummary {
	summaries := make([]editorFileSummary, 0, len(files))
	for _, file := range files {
		summaries = append(summaries, file.summary())
	}
	return summaries
}

func (s *documentationServer) serveEditorFile(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodGet, http.MethodPut) {
		return
	}
	if r.Method == http.MethodGet {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		filePath := r.URL.Query().Get("path")
		item, err := s.workspace.read(filePath, s.model, r.URL.Query().Get("raw") != "1")
		if err != nil {
			status, code := editorStatusForError(err)
			writeEditorError(w, status, code, err.Error(), nil)
			return
		}
		if r.URL.Query().Get("raw") == "1" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, item.Content)
			return
		}
		_, revision, scanErr := s.workspace.scan(s.model)
		if scanErr != nil {
			writeEditorError(w, http.StatusInternalServerError, "workspace_error", scanErr.Error(), nil)
			return
		}
		writeEditorJSON(w, http.StatusOK, struct {
			SchemaVersion int        `json:"schemaVersion"`
			Revision      string     `json:"revision"`
			File          editorFile `json:"file"`
		}{1, revision, item})
		return
	}
	if !requireEditorJSONAction(w, r, "save") {
		return
	}
	var request struct {
		Path             string `json:"path"`
		Content          string `json:"content"`
		ExpectedDigest   string `json:"expectedDigest"`
		ConfirmOverwrite bool   `json:"confirmOverwrite"`
	}
	if !decodeEditorJSON(w, r, &request) {
		return
	}
	if len(request.Content) > editorContentLimit {
		writeEditorError(w, http.StatusRequestEntityTooLarge, "content_too_large", "Content exceeds 2 MiB", nil)
		return
	}
	if pendingDigest, pending := s.overwrites[request.Path]; pending {
		if !request.ConfirmOverwrite || request.ExpectedDigest != pendingDigest {
			current, currentErr := s.workspace.read(request.Path, s.model, false)
			if currentErr != nil {
				status, code := editorStatusForError(currentErr)
				writeEditorError(w, status, code, currentErr.Error(), nil)
				return
			}
			s.writeEditorStale(w, request.Path, current)
			return
		}
	} else if request.ConfirmOverwrite {
		current, currentErr := s.workspace.read(request.Path, s.model, false)
		if currentErr != nil {
			status, code := editorStatusForError(currentErr)
			writeEditorError(w, status, code, currentErr.Error(), nil)
			return
		}
		s.writeEditorStale(w, request.Path, current)
		return
	}
	_, err := s.workspace.save(request.Path, []byte(request.Content), request.ExpectedDigest)
	if err != nil {
		var stale *staleFileError
		if errors.As(err, &stale) {
			s.writeEditorStale(w, request.Path, stale.file)
			return
		}
		status, code := editorStatusForError(err)
		writeEditorError(w, status, code, err.Error(), nil)
		return
	}
	delete(s.overwrites, request.Path)
	if _, _, err = s.rebuild(); err != nil {
		writeEditorError(w, http.StatusInternalServerError, "rebuild_failed", err.Error(), nil)
		return
	}
	item, err := s.workspace.read(request.Path, s.model, true)
	if err != nil {
		writeEditorError(w, http.StatusInternalServerError, "workspace_error", err.Error(), nil)
		return
	}
	writeEditorJSON(w, http.StatusOK, struct {
		SchemaVersion int           `json:"schemaVersion"`
		Revision      string        `json:"revision"`
		File          editorFile    `json:"file"`
		Rebuild       editorRebuild `json:"rebuild"`
	}{1, s.revision, item, s.rebuildPayload()})
}

func (s *documentationServer) writeEditorStale(w http.ResponseWriter, filePath string, current editorFile) {
	_, revision, err := s.workspace.scan(s.model)
	if err != nil {
		revision = s.revision
	}
	s.overwrites[filePath] = current.Digest
	writeEditorError(w, http.StatusConflict, "stale_digest", "file changed by another process", map[string]any{
		"digest": current.Digest, "content": current.Content, "revision": revision,
	})
}

type editorContentRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (s *documentationServer) decodeContentRequest(w http.ResponseWriter, r *http.Request, action string) (editorContentRequest, bool) {
	if !requireEditorJSONAction(w, r, action) {
		return editorContentRequest{}, false
	}
	var request editorContentRequest
	if !decodeEditorJSON(w, r, &request) {
		return editorContentRequest{}, false
	}
	if len(request.Content) > editorContentLimit {
		writeEditorError(w, http.StatusRequestEntityTooLarge, "content_too_large", "Content exceeds 2 MiB", nil)
		return editorContentRequest{}, false
	}
	if _, _, err := s.workspace.resolve(request.Path, false); err != nil {
		status, code := editorStatusForError(err)
		writeEditorError(w, status, code, err.Error(), nil)
		return editorContentRequest{}, false
	}
	return request, true
}

func (s *documentationServer) serveEditorPreview(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodPost) {
		return
	}
	request, ok := s.decodeContentRequest(w, r, "preview")
	if !ok {
		return
	}
	if language, _ := editorLanguage(request.Path); language != "markdown" {
		writeEditorError(w, http.StatusUnsupportedMediaType, "preview_not_supported", "Preview is available for Markdown only", nil)
		return
	}
	model, err := buildDocumentationModel(s.options, map[string][]byte{request.Path: []byte(request.Content)})
	if err != nil {
		writeEditorError(w, http.StatusInternalServerError, "preview_failed", err.Error(), nil)
		return
	}
	document := model.DocByPath[request.Path]
	if document == nil {
		writeEditorError(w, http.StatusNotFound, "file_not_found", "Markdown document not found", nil)
		return
	}
	html := renderDocumentMarkdown(model, document, linkResolverFor(model, document), nil)
	diagnostics := issueDiagnostics(model.Issues)
	writeEditorJSON(w, http.StatusOK, struct {
		SchemaVersion int                `json:"schemaVersion"`
		Path          string             `json:"path"`
		HTML          string             `json:"html"`
		Diagnostics   []editorDiagnostic `json:"diagnostics"`
	}{1, request.Path, html, diagnostics})
}

func (s *documentationServer) serveEditorValidate(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodPost) {
		return
	}
	request, ok := s.decodeContentRequest(w, r, "validate")
	if !ok {
		return
	}
	diagnostics, err := s.workspace.diagnostics(request.Path, []byte(request.Content))
	if err != nil {
		writeEditorError(w, http.StatusInternalServerError, "validation_failed", err.Error(), nil)
		return
	}
	writeEditorJSON(w, http.StatusOK, struct {
		SchemaVersion int                `json:"schemaVersion"`
		Path          string             `json:"path"`
		Diagnostics   []editorDiagnostic `json:"diagnostics"`
	}{1, request.Path, diagnostics})
}

func (s *documentationServer) serveEditorCreate(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodPost) || !requireEditorJSONAction(w, r, "create") {
		return
	}
	var request struct {
		Template string            `json:"template"`
		Language string            `json:"language"`
		Fields   map[string]string `json:"fields"`
	}
	if !decodeEditorJSON(w, r, &request) {
		return
	}
	template, templateExists := scaffoldTemplate(request.Template)
	if !templateExists {
		writeEditorError(w, http.StatusBadRequest, "invalid_template", "Unknown template", nil)
		return
	}
	createDirectory := template.spec.directory
	if request.Template == "task-init" {
		createDirectory = "work"
	}
	if err := s.workspace.validateCreateDirectory(createDirectory); err != nil {
		status, code := editorStatusForError(err)
		writeEditorError(w, status, code, err.Error(), nil)
		return
	}
	createdPath, err := createFromEditorTemplate(s.options, request.Template, request.Language, request.Fields)
	if err != nil {
		code, status := "invalid_template", http.StatusBadRequest
		if strings.Contains(err.Error(), "already exists") {
			code, status = "file_exists", http.StatusConflict
		}
		writeEditorError(w, status, code, err.Error(), nil)
		return
	}
	if _, _, err = s.rebuild(); err != nil {
		writeEditorError(w, http.StatusInternalServerError, "rebuild_failed", err.Error(), nil)
		return
	}
	item, err := s.workspace.read(createdPath, s.model, true)
	if err != nil {
		writeEditorError(w, http.StatusInternalServerError, "workspace_error", err.Error(), nil)
		return
	}
	writeEditorJSON(w, http.StatusCreated, struct {
		SchemaVersion int           `json:"schemaVersion"`
		Revision      string        `json:"revision"`
		File          editorFile    `json:"file"`
		Rebuild       editorRebuild `json:"rebuild"`
	}{1, s.revision, item, s.rebuildPayload()})
}

func (s *documentationServer) serveEditorUI(w http.ResponseWriter, r *http.Request) {
	if !allowEditorMethods(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		return
	}
	uiModel := workspaceModel(s.model)
	ui := portalUI(uiModel)
	locale := uiModel.SiteConfig.Project.Locale
	if locale == "" {
		locale = "en"
	}
	html, err := frontend.RenderEditor(frontend.WorkspaceView{
		UI: ui, Lang: locale, HTMLAttributes: template.HTMLAttr(appearanceAttributes(uiModel.SiteConfig)),
		Title: ui.Text("nav.editor") + " — " + uiModel.Project.Title, Favicon: workspaceFavicon(uiModel),
		AppearanceJS: "/assets/" + mustFrontendAsset("appearance.js"),
		Styles:       []string{"/assets/" + mustFrontendAsset("portal.css"), "/assets/" + mustFrontendAsset("serve.css"), "/assets/" + mustFrontendAsset("editor.css")},
		Scripts: []frontend.ScriptAsset{
			{URL: "/assets/" + mustFrontendAsset("codemirror.js"), Module: true},
			{URL: "/assets/" + mustFrontendAsset("serve.js"), Module: true},
			{URL: "/assets/" + mustFrontendAsset("editor.js"), Module: true},
		},
		Bootstrap: workspacePageBootstrap(uiModel, "_toudocu/editor/index.html", "../../assets/", frontend.Capabilities{Editor: true, Rebuild: true, AgentConsole: uiModel.agentConsoleEnabled}),
		Header:    template.HTML(workspaceHeader(uiModel, workspaceEditor)),
	})
	if err != nil {
		http.Error(w, "Could not render editor", http.StatusInternalServerError)
		return
	}
	_, _ = io.WriteString(w, html)
}
