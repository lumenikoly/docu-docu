package toudocu

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	changesAPIBase = "/_toudocu/api/changes"
	changesUIPath  = "/changes/"
)

var changesRouteRegistry = []apiRoute{
	{Path: changesAPIBase, Methods: []string{http.MethodGet, http.MethodHead}, Handler: (*documentationServer).serveChangesSummary},
	{Path: changesAPIBase + "/file", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveChangesFile},
	{Path: changesAPIBase + "/task", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveChangesTask},
	{Path: changesAPIBase + "/content", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveChangesContent},
	{Path: changesAPIBase + "/render", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveChangesRenderedContent},
	{Path: changesAPIBase + "/screen-map", Methods: []string{http.MethodGet}, Handler: (*documentationServer).serveChangesScreenMap},
}

func allChangesRouteRegistry() []apiRoute {
	routes := append([]apiRoute{}, changesRouteRegistry...)
	return append(routes, reviewRepositoryRouteRegistry...)
}

func changesOptionsFromRequest(base Options, request *http.Request) Options {
	options := base
	query := request.URL.Query()
	options.ChangeBase = query.Get("base")
	options.ChangeTarget = query.Get("target")
	options.ChangeStatus = query.Get("status")
	options.ChangeEntityType = query.Get("type")
	options.ChangeModule = query.Get("module")
	options.ChangeTaskID = query.Get("task")
	options.ChangeBranchBase = query.Get("branchBase")
	if options.ChangeTaskID == "" {
		options.ChangeTaskID = query.Get("id")
	}
	return options
}

func (s *documentationServer) buildChanges(request *http.Request) (*ChangeSetReport, error) {
	options := changesOptionsFromRequest(s.options, request)
	endpoint := strings.TrimPrefix(request.URL.Path, changesAPIBase)
	options.ChangeOmitSourceDiff = endpoint != "/file"
	if endpoint == "/file" || endpoint == "/content" || endpoint == "/render" {
		options.ChangeFile = filepath.ToSlash(request.URL.Query().Get("path"))
	}
	cacheKey := ""
	if fingerprint, err := s.changesFingerprint(options); err == nil {
		cacheKey = fmt.Sprintf("%x:%s:%s", sha256.Sum256([]byte(fingerprint)), endpoint, request.URL.RawQuery)
		if cached := s.changesCache[cacheKey]; cached != nil {
			return cached, nil
		}
	}
	report, err := BuildDocumentationChanges(options)
	if err != nil {
		return nil, err
	}
	filterDocumentationChanges(report, options)
	if cacheKey != "" {
		if s.changesCache == nil {
			s.changesCache = map[string]*ChangeSetReport{}
		}
		if len(s.changesCache) >= 16 {
			s.changesCache = map[string]*ChangeSetReport{}
		}
		s.changesCache[cacheKey] = report
	}
	return report, nil
}

func (s *documentationServer) changesFingerprint(options Options) (string, error) {
	g, err := openGitChangeSource(s.options.InputDirectory, s.options.ChangeRenameSimilarity)
	if err != nil {
		return "", err
	}
	head, err := g.resolveCommit("HEAD")
	if err != nil {
		return "", err
	}
	status, err := g.run("status", "--porcelain=v2", "-z", "--untracked-files=all", "--", g.docsRel)
	if err != nil {
		return "", err
	}
	resolved := []string{s.revision, head, string(status)}
	for _, revision := range []string{options.ChangeBase, options.ChangeTarget, options.ChangeBranchBase} {
		if revision == "" || revision == "HEAD" || revision == "index" || revision == "working-tree" {
			continue
		}
		commit, resolveErr := g.resolveCommit(revision)
		if resolveErr != nil {
			return "", resolveErr
		}
		resolved = append(resolved, revision, commit)
	}
	return strings.Join(resolved, "\x00"), nil
}

func writeChangesJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeChangesAPIError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "git-command-failed"
	if failure, ok := err.(*changeFailure); ok {
		code = failure.IssueCode
		switch failure.Code {
		case 2:
			status = http.StatusBadRequest
		case 3:
			status = http.StatusServiceUnavailable
		}
	}
	writeChangesJSON(w, status, map[string]any{"schemaVersion": 1, "diagnostics": []Issue{{Severity: "error", Code: code, Message: err.Error()}}})
}

func writeChangesDiagnostic(w http.ResponseWriter, status int, code, message string) {
	writeChangesJSON(w, status, map[string]any{"schemaVersion": 1, "diagnostics": []Issue{{Severity: "error", Code: code, Message: message}}})
}

func matchAPIRoute(registry []apiRoute, request *http.Request) (*apiRoute, bool) {
	for index := range registry {
		route := &registry[index]
		if request.URL.Path != route.Path && (route.Path != changesAPIBase || request.URL.Path != changesAPIBase+"/") {
			continue
		}
		for _, method := range route.Methods {
			if request.Method == method {
				return route, true
			}
		}
		return route, false
	}
	return nil, false
}

func (s *documentationServer) serveChangesAPI(w http.ResponseWriter, request *http.Request) {
	if request.URL.Path == reviewRepositoryAPIBase || strings.HasPrefix(request.URL.Path, reviewRepositoryAPIBase+"/") {
		s.serveReviewAPI(w, request)
		return
	}
	route, methodAllowed := matchAPIRoute(changesRouteRegistry, request)
	if route == nil {
		writeChangesDiagnostic(w, http.StatusNotFound, "route_not_found", "Changes API route not found")
		return
	}
	if !methodAllowed {
		w.Header().Set("Allow", strings.Join(route.Methods, ", "))
		writeChangesDiagnostic(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	route.Handler(s, w, request)
}

func (s *documentationServer) changesReport(w http.ResponseWriter, request *http.Request) (*ChangeSetReport, bool) {
	report, err := s.buildChanges(request)
	if err != nil {
		writeChangesAPIError(w, err)
		return nil, false
	}
	return report, true

}

func (s *documentationServer) serveChangesSummary(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if !ok {
		return
	}
	etag := `"` + report.ChangeSetDigest + `"`
	w.Header().Set("ETag", etag)
	if request.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if request.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	writeChangesJSON(w, http.StatusOK, report)
}

func (s *documentationServer) serveChangesFile(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if !ok {
		return
	}
	change := findRequestedChange(report, request.URL.Query().Get("path"))
	if change == nil {
		writeChangesDiagnostic(w, http.StatusNotFound, "change-new-version-missing", "File change not found.")
		return
	}
	writeChangesJSON(w, http.StatusOK, change)
}

func (s *documentationServer) serveChangesTask(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if !ok {
		return
	}
	if report.TaskImpact == nil {
		writeChangesDiagnostic(w, http.StatusBadRequest, "task-not-found", "The task or id parameter is required.")
		return
	}
	writeChangesJSON(w, http.StatusOK, report.TaskImpact)
}

func (s *documentationServer) serveChangesContent(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if ok {
		s.serveChangedContent(w, request, report, false)
	}
}

func (s *documentationServer) serveChangesRenderedContent(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if ok {
		s.serveChangedContent(w, request, report, true)
	}
}

func (s *documentationServer) serveChangesScreenMap(w http.ResponseWriter, request *http.Request) {
	report, ok := s.changesReport(w, request)
	if ok {
		writeChangesJSON(w, http.StatusOK, buildScreenMapChanges(report))
	}
}

func findRequestedChange(report *ChangeSetReport, requested string) *DocumentationChange {
	requested = filepath.ToSlash(requested)
	if filepath.IsAbs(requested) || requested == ".." || strings.HasPrefix(requested, "../") || strings.Contains(requested, "/../") || strings.Contains(requested, "\\") {
		return nil
	}
	for i := range report.Changes {
		if report.Changes[i].Path == requested || report.Changes[i].OldPath == requested {
			return &report.Changes[i]
		}
	}
	return nil
}

func (s *documentationServer) serveChangedContent(w http.ResponseWriter, request *http.Request, report *ChangeSetReport, render bool) {
	change := findRequestedChange(report, request.URL.Query().Get("path"))
	if change == nil {
		writeChangesDiagnostic(w, http.StatusNotFound, "change-new-version-missing", "File change not found.")
		return
	}
	sideName := request.URL.Query().Get("side")
	side := report.Comparison.Target
	path := change.Path
	if sideName == "before" {
		side, path = report.Comparison.Base, change.Path
		if change.OldPath != "" {
			path = change.OldPath
		}
	} else if sideName != "after" {
		writeChangesDiagnostic(w, http.StatusBadRequest, "invalid-change-side", "side must be before or after")
		return
	}
	if sideName == "before" && (change.Status == "added" || change.Status == "untracked") || sideName == "after" && change.Status == "deleted" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	g, err := openGitChangeSource(s.options.InputDirectory, s.options.ChangeRenameSimilarity)
	if err != nil {
		writeChangesAPIError(w, err)
		return
	}
	content, err := g.content(side, path)
	if err != nil {
		writeChangesAPIError(w, err)
		return
	}
	limit := s.options.ChangeMaxSourceDiffBytes
	if limit <= 0 {
		limit = 2 * 1024 * 1024
	}
	if len(content) > limit {
		writeChangesJSON(w, http.StatusRequestEntityTooLarge, map[string]any{"schemaVersion": 1, "diagnostics": []Issue{{Severity: "warning", Code: "change-file-too-large", Message: "Content exceeds the Changes API limit.", DocumentPath: path}}})
		return
	}
	if render {
		if strings.ToLower(filepath.Ext(path)) != ".md" {
			writeChangesDiagnostic(w, http.StatusUnsupportedMediaType, "render-not-supported", "Rendered diff supports Markdown only")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, s.renderChangesMarkdown(string(content)))
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'none'; sandbox")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	contentType := http.DetectContentType(content)
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		contentType = "image/svg+xml"
	}
	if strings.HasPrefix(contentType, "text/") {
		contentType += "; charset=utf-8"
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

func (s *documentationServer) renderChangesMarkdown(content string) string {
	locale := ""
	if s.model != nil {
		locale = s.model.SiteConfig.Project.Locale
	}
	return renderMarkdown(analyzeMarkdown(content), renderContext{}, renderOptions{SkipH1: false, SuppressMetadata: false, InteractiveMermaid: true, Locale: locale})
}

func buildScreenMapChanges(report *ChangeSetReport) map[string]any {
	nodes := []map[string]any{}
	edges := []map[string]any{}
	for _, change := range report.Changes {
		if change.Screen == nil {
			continue
		}
		nodes = append(nodes, map[string]any{"id": firstScreenID(change.Screen), "status": change.Status, "path": change.Path, "before": change.Screen.Before, "after": change.Screen.After, "ghost": change.Screen.After == nil})
		for _, transition := range change.Screen.Transitions {
			edges = append(edges, map[string]any{"id": transition.ID, "status": transition.Status, "before": transition.Before, "after": transition.After, "ghost": transition.After == nil})
		}
	}
	return map[string]any{"schemaVersion": 1, "changeSetDigest": report.ChangeSetDigest, "nodes": nodes, "edges": edges}
}

func firstScreenID(screen *ScreenDiffMetadata) string {
	if screen.After != nil {
		return screen.After.ID
	}
	if screen.Before != nil {
		return screen.Before.ID
	}
	return ""
}
