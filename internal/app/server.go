package toudocu

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	frontend "toudocu/internal/site"
)

const rebuildEndpoint = "/__toudocu/rebuild"
const localeMountBase = "/_toudocu/locales/"

// LanguageTarget is a server-computed navigation target for one portal locale.
type LanguageTarget struct {
	Locale    string
	URL       string
	Active    bool
	Available bool
}

type PortalStatus string

const (
	portalReady       PortalStatus = "ready"
	portalUnavailable PortalStatus = "unavailable"
	portalRebuilding  PortalStatus = "rebuilding"
)

// GeneratedPortal identifies the last successful generated portal snapshot.
type GeneratedPortal struct{ OutputDirectory string }

// ServePortalState is runtime-only state; it is intentionally separate from
// ProjectModel, reports, task context and knowledge collections.
type ServePortalState struct {
	Locale  string
	BaseURL string
	Root    string
	Portal  GeneratedPortal
	Status  PortalStatus
	PageMap map[string]string

	options  Options
	model    *Model
	revision string
}

type documentationServer struct {
	options             Options
	stderr              io.Writer
	mu                  sync.Mutex
	workspace           *editorWorkspace
	model               *Model // canonical only: editor and changes APIs never cross this boundary.
	result              GenerateResult
	revision            string
	overwrites          map[string]string
	changesCache        map[string]*ChangeSetReport
	portals             map[string]*ServePortalState
	configDigest        string
	translationReadOnly bool
	updateChecker       *updateChecker
	agentConsole        *agentConsole
	taskRunner          commandRunner
}

func newDocumentationServer(options Options, stderr io.Writer) (*documentationServer, *Model, GenerateResult, error) {
	workspace, err := newEditorWorkspace(options)
	if err != nil {
		return nil, nil, GenerateResult{}, err
	}
	s := &documentationServer{options: options, stderr: stderr, workspace: workspace, overwrites: map[string]string{}, changesCache: map[string]*ChangeSetReport{}, portals: map[string]*ServePortalState{}, updateChecker: newUpdateChecker()}
	if err := s.rebuildRegistry(); err != nil {
		return nil, nil, GenerateResult{}, err
	}
	if !s.translationReadOnly && !externallyReachableHost(options.Host) {
		s.agentConsole = newAgentConsole(lazyCodexProvider{}, options.RepositoryRoot)
	}
	return s, s.model, s.result, nil
}

func canonicalPortalKey() string { return "" }

func (s *documentationServer) rebuildRegistry() error {
	canonical, err := BuildDocumentationModel(s.options)
	if err != nil {
		return err
	}
	// Starting serve directly on an independent translation root keeps a
	// single-locale read-only portal. Translation sources are updated only by
	// the explicit translation workflow.
	for _, profile := range canonical.SiteConfig.Translations {
		if root, rootErr := safeTranslationRoot(canonical.RepositoryRoot, profile.Root); rootErr == nil && filepath.Clean(root) == filepath.Clean(canonical.RootDirectory) {
			state := &ServePortalState{Locale: canonical.SiteConfig.Project.Locale, BaseURL: "/", Root: canonical.RootDirectory, Portal: GeneratedPortal{OutputDirectory: s.options.OutputDirectory}, Status: portalRebuilding, options: s.options, model: canonical}
			state.PageMap = outputPageMap(canonical)
			result, genErr := s.generatePortal(state, false)
			if genErr != nil {
				return genErr
			}
			s.portals = map[string]*ServePortalState{canonicalPortalKey(): state}
			s.model, s.result, s.revision = canonical, result, state.revision
			s.changesCache = map[string]*ChangeSetReport{}
			s.configDigest = s.currentConfigDigest()
			s.translationReadOnly = true
			return nil
		}
	}
	s.translationReadOnly = false
	canonicalRoot := canonical.RootDirectory
	states := map[string]*ServePortalState{canonicalPortalKey(): {Locale: canonical.SiteConfig.Project.Locale, BaseURL: "/", Root: canonicalRoot, Portal: GeneratedPortal{OutputDirectory: s.options.OutputDirectory}, Status: portalRebuilding, options: s.options}}
	locales := make([]string, 0, len(canonical.SiteConfig.Translations))
	for locale := range canonical.SiteConfig.Translations {
		locales = append(locales, locale)
	}
	sort.Strings(locales)
	for _, locale := range locales {
		profile := canonical.SiteConfig.Translations[locale]
		root, rootErr := safeTranslationRoot(canonical.RepositoryRoot, profile.Root)
		state := &ServePortalState{Locale: locale, BaseURL: localeMountBase + locale + "/", Status: portalUnavailable, options: s.options}
		if rootErr == nil {
			state.Root = root
			state.Portal.OutputDirectory = filepath.Join(s.options.OutputDirectory, "_toudocu", "locales", locale)
			state.options.InputDirectory = root
			state.options.OutputDirectory = state.Portal.OutputDirectory
			state.options.Clean = false
		}
		states[locale] = state
	}

	// Build every model first so every shell receives only already-resolved URLs.
	states[canonicalPortalKey()].model = canonical
	for _, locale := range locales {
		state := states[locale]
		if state.Root == "" {
			continue
		}
		model, buildErr := BuildDocumentationModel(state.options)
		if buildErr != nil {
			_, _ = fmt.Fprintln(s.stderr, "Could not prepare locale portal", locale+":", buildErr)
			continue
		}
		state.model = model
	}
	populateLanguageTargets(states)
	canonicalState := states[canonicalPortalKey()]
	if _, genErr := s.generatePortal(canonicalState, true); genErr != nil {
		return genErr
	}
	for _, locale := range locales {
		state := states[locale]
		if state.model == nil {
			continue
		}
		if _, genErr := s.generatePortal(state, false); genErr != nil {
			_, _ = fmt.Fprintln(s.stderr, "Could not build locale portal", locale+":", genErr)
			state.Status = portalUnavailable
		}
	}
	s.portals = states
	s.model, s.revision = canonicalState.model, canonicalState.revision
	s.changesCache = map[string]*ChangeSetReport{}
	s.configDigest = s.currentConfigDigest()
	return nil
}

func outputPageMap(model *Model) map[string]string {
	pages := map[string]string{"index.html": "index.html", model.HealthOutputPath: model.HealthOutputPath}
	for _, document := range model.Documents {
		pages[document.OutputPath] = document.OutputPath
	}
	if model.ProjectChangelog != nil {
		pages[projectChangelogOutput] = projectChangelogOutput
	}
	if len(model.Knowledge.UseCases)+len(model.Knowledge.Flows) > 0 {
		pages[sectionCatalogOutput(SectionFlows)] = sectionCatalogOutput(SectionFlows)
		pages[sectionCatalogOutput(SectionUseCases)] = sectionCatalogOutput(SectionUseCases)
	}
	if len(model.Knowledge.Screens) > 0 {
		pages["screens/catalog.html"] = "screens/catalog.html"
		pages["traceability.html"] = "traceability.html"
		if model.ScreenMapEnabled {
			pages["screens/index.html"] = "screens/index.html"
		}
	}
	for _, document := range model.Documents {
		pages[document.SourcePath] = document.OutputPath
	}
	return pages
}

func populateLanguageTargets(states map[string]*ServePortalState) {
	for _, state := range states {
		if state.model != nil {
			state.PageMap = outputPageMap(state.model)
		}
	}
	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for key, state := range states {
		if state.model == nil {
			continue
		}
		state.model.languageTargets = map[string][]LanguageTarget{}
		for current := range state.PageMap {
			targets := make([]LanguageTarget, 0, len(keys))
			for _, targetKey := range keys {
				target := states[targetKey]
				output := "index.html"
				available := target.model != nil && target.Status != portalUnavailable
				if sourcePath, ok := sourcePathForOutput(state, current); ok && target.model != nil {
					if matched, exists := target.PageMap[sourcePath]; exists {
						output = matched
					} else {
						available = false
					}
				} else if target.model != nil {
					if _, exists := target.PageMap[current]; exists {
						output = current
					} else {
						available = false
					}
				}
				url := target.BaseURL
				if output != "index.html" {
					url += output
				}
				targets = append(targets, LanguageTarget{Locale: target.Locale, URL: url, Active: key == targetKey, Available: available})
			}
			state.model.languageTargets[current] = targets
		}
	}
}

func sourcePathForOutput(state *ServePortalState, output string) (string, bool) {
	if state.model == nil {
		return "", false
	}
	for source, document := range state.model.DocByPath {
		if document.OutputPath == output {
			return source, true
		}
	}
	return "", false
}

func (s *documentationServer) generatePortal(state *ServePortalState, canonical bool) (GenerateResult, error) {
	if state.model == nil {
		return GenerateResult{}, fmt.Errorf("portal model unavailable")
	}
	previousStatus := state.Status
	state.Status = portalRebuilding
	revision, err := s.rootRevision(state.model, state.options)
	if err != nil {
		return GenerateResult{}, err
	}
	state.model.serveRevision = ""
	state.model.updateCheckEnabled = false
	if canonical {
		state.model.serveRevision = revision
		state.model.updateCheckEnabled = !s.options.NoUpdateCheck
		state.model.agentConsoleEnabled = !externallyReachableHost(s.options.Host)
	}
	next := state.Portal.OutputDirectory + ".next"
	_ = os.RemoveAll(next)
	options := state.options
	options.OutputDirectory = next
	options.Clean = true
	var result GenerateResult
	if canonical {
		result, err = generateServeSite(state.model, options)
	} else {
		result, err = GenerateSite(state.model, options)
	}
	if err != nil {
		if previousStatus == portalReady {
			state.Status = portalReady
		} else {
			state.Status = portalUnavailable
		}
		return GenerateResult{}, err
	}
	previous := state.Portal.OutputDirectory + ".previous"
	_ = os.RemoveAll(previous)
	if _, statErr := os.Stat(state.Portal.OutputDirectory); statErr == nil {
		if err = os.Rename(state.Portal.OutputDirectory, previous); err != nil {
			return GenerateResult{}, err
		}
	}
	if err = os.Rename(next, state.Portal.OutputDirectory); err != nil {
		_ = os.Rename(previous, state.Portal.OutputDirectory)
		return GenerateResult{}, err
	}
	_ = os.RemoveAll(previous)
	result.OutputDirectory = state.Portal.OutputDirectory
	state.revision, state.Status = revision, portalReady
	if canonical {
		s.result = result
	}
	return result, nil
}

func (s *documentationServer) rebuild() (*Model, GenerateResult, error) {
	state := s.portals[canonicalPortalKey()]
	if state == nil {
		return nil, GenerateResult{}, fmt.Errorf("canonical portal unavailable")
	}
	model, err := BuildDocumentationModel(s.options)
	if err != nil {
		return nil, GenerateResult{}, err
	}
	state.model = model
	populateLanguageTargets(s.portals)
	result, err := s.generatePortal(state, !s.translationReadOnly)
	if err != nil {
		return nil, GenerateResult{}, err
	}
	s.model, s.result, s.revision = model, result, state.revision
	s.changesCache = map[string]*ChangeSetReport{}
	return model, result, nil
}

func (s *documentationServer) rootRevision(model *Model, options Options) (string, error) {
	workspace, err := newEditorWorkspace(options)
	if err != nil {
		return "", err
	}
	_, revision, err := workspace.scan(model)
	if err != nil {
		return "", err
	}
	isTranslation := false
	for _, profile := range model.SiteConfig.Translations {
		if root, rootErr := safeTranslationRoot(model.RepositoryRoot, profile.Root); rootErr == nil && filepath.Clean(root) == filepath.Clean(model.RootDirectory) {
			isTranslation = true
			break
		}
	}
	if !isTranslation {
		revision += "\n" + projectChangelogFingerprint(model.RepositoryRoot)
	}
	return contentDigest([]byte(revision)), nil
}

func rootInputRevision(options Options) (string, error) {
	workspace, err := newEditorWorkspace(options)
	if err != nil {
		return "", err
	}
	_, revision, err := workspace.scan(&Model{DocByPath: map[string]*Document{}})
	return revision, err
}
func (s *documentationServer) workspaceRevision(model *Model) (string, error) {
	return s.rootRevision(model, s.options)
}

func (s *documentationServer) currentConfigDigest() string {
	data, err := os.ReadFile(filepath.Join(s.options.RepositoryRoot, ".toudocu", "config.yml"))
	if err != nil {
		return "missing"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (s *documentationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasPrefix(r.URL.Path, agentConsoleAPIBase) || strings.HasPrefix(r.URL.Path, "/_toudocu/api/tasks/") {
		s.serveAgentConsole(w, r)
		return
	}
	if r.URL.Path == versionEndpoint {
		if s.translationReadOnly || s.options.NoUpdateCheck {
			http.NotFound(w, r)
			return
		}
		s.serveVersion(w, r)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.HasPrefix(r.URL.Path, localeMountBase) {
		s.serveLocale(w, r)
		return
	}
	if s.translationReadOnly && (strings.HasPrefix(r.URL.Path, editorAPIBase+"/") || r.URL.Path == editorUIPath || r.URL.Path == strings.TrimSuffix(editorUIPath, "/") || r.URL.Path == apiDocsUIPath || r.URL.Path == strings.TrimSuffix(apiDocsUIPath, "/") || r.URL.Path == rebuildEndpoint) {
		http.NotFound(w, r)
		return
	}
	if s.translationReadOnly && (r.URL.Path == reviewAPIBase || strings.HasPrefix(r.URL.Path, reviewAPIBase+"/")) {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == reviewAPIBase || strings.HasPrefix(r.URL.Path, reviewAPIBase+"/") {
		s.serveReviewAPI(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, editorAPIBase+"/") {
		s.serveEditorAPI(w, r)
		return
	}
	if r.URL.Path == changesAPIBase || strings.HasPrefix(r.URL.Path, changesAPIBase+"/") {
		s.serveChangesAPI(w, r)
		return
	}
	if r.URL.Path == changesUIPath || r.URL.Path == strings.TrimSuffix(changesUIPath, "/") {
		s.serveChangesUI(w, r)
		return
	}
	if r.URL.Path == editorUIPath || r.URL.Path == strings.TrimSuffix(editorUIPath, "/") {
		s.serveEditorUI(w, r)
		return
	}
	if r.URL.Path == apiDocsUIPath || r.URL.Path == strings.TrimSuffix(apiDocsUIPath, "/") {
		s.serveAPIDocsUI(w, r)
		return
	}
	for _, route := range editorServiceRouteRegistry {
		if r.URL.Path == route.Path {
			route.Handler(s, w, r)
			return
		}
	}
	s.serveSnapshot(w, r, s.portals[canonicalPortalKey()])
}

func (s *documentationServer) serveRebuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("X-Toudocu-Action") != "rebuild" {
		http.Error(w, "Rebuild request rejected", http.StatusForbidden)
		return
	}
	model, result, err := s.rebuild()
	if err != nil {
		_, _ = fmt.Fprintln(s.stderr, "Could not rebuild documentation:", err)
		http.Error(w, "Could not rebuild documentation: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]int{"documents": model.Stats.Documents, "errors": model.Stats.Errors, "pages": result.Pages, "warnings": model.Stats.Warnings})
}

func (s *documentationServer) serveLocale(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, localeMountBase)
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	state, ok := s.portals[parts[0]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if state.Status == portalUnavailable || state.model == nil {
		s.serveUnavailableLocale(w, state)
		return
	}
	clone := r.Clone(r.Context())
	clone.URL.Path = "/"
	if len(parts) == 2 {
		clone.URL.Path = "/" + parts[1]
	}
	s.serveSnapshot(w, clone, state)
}
func (s *documentationServer) serveSnapshot(w http.ResponseWriter, r *http.Request, state *ServePortalState) {
	if state == nil || state.Status == portalUnavailable {
		http.NotFound(w, r)
		return
	}
	requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestPath == "." || requestPath == "" {
		requestPath = "index.html"
	}
	target := filepath.Join(state.Portal.OutputDirectory, filepath.FromSlash(requestPath))
	if _, err := os.Stat(target); err != nil {
		serveMode := state.BaseURL == "/" && !s.translationReadOnly
		previous := state.model.serveMode
		state.model.serveMode = serveMode
		page := renderNotFoundPage(state.model, requestPath)
		state.model.serveMode = previous
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, page)
		}
		return
	}
	http.FileServer(http.Dir(state.Portal.OutputDirectory)).ServeHTTP(w, r)
}
func (s *documentationServer) serveUnavailableLocale(w http.ResponseWriter, state *ServePortalState) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	targets := make([]LanguageTarget, 0, len(s.portals))
	keys := make([]string, 0, len(s.portals))
	for key := range s.portals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		target := s.portals[key]
		targets = append(targets, LanguageTarget{Locale: target.Locale, URL: target.BaseURL, Active: target == state, Available: target.Status != portalUnavailable && target.model != nil})
	}
	ui := frontend.NewUI(state.Locale)
	_, _ = io.WriteString(w, `<!doctype html><html><head><meta charset="utf-8"><title>`+escapeHTML(ui.Text("locale.unavailableTitle", state.Locale))+`</title></head><body><main><h1>`+escapeHTML(state.Locale)+`</h1><p>`+escapeHTML(ui.Text("locale.unavailableStatus"))+`</p><p>`+escapeHTML(ui.Text("locale.unavailable"))+`</p>`+renderLanguageSelect(ui, targets)+`</main></body></html>`)
}

func (s *documentationServer) watch(ctx context.Context) {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if digest := s.currentConfigDigest(); digest != s.configDigest {
				if err := s.rebuildRegistry(); err != nil {
					_, _ = fmt.Fprintln(s.stderr, "Could not update locale registry:", err)
				}
				s.mu.Unlock()
				continue
			}
			for key, state := range s.portals {
				if key != canonicalPortalKey() && state.Root == "" {
					continue
				}
				candidate, err := rootInputRevision(state.options)
				if state.model != nil {
					candidate, err = s.rootRevision(state.model, state.options)
				}
				if err != nil || candidate == state.revision {
					continue
				}
				// A second fingerprint prevents publishing a snapshot from a file that
				// is still being written by an editor or another tool.
				time.Sleep(200 * time.Millisecond)
				stable, stableErr := rootInputRevision(state.options)
				if state.model != nil {
					stable, stableErr = s.rootRevision(state.model, state.options)
				}
				if stableErr != nil || stable != candidate {
					continue
				}
				if key == canonicalPortalKey() {
					if _, _, err = s.rebuild(); err != nil {
						_, _ = fmt.Fprintln(s.stderr, "Could not rebuild documentation after an external change:", err)
					}
				} else {
					model, buildErr := BuildDocumentationModel(state.options)
					if buildErr != nil {
						_, _ = fmt.Fprintln(s.stderr, "Could not rebuild locale portal", state.Locale+":", buildErr)
						continue
					}
					state.model = model
					populateLanguageTargets(s.portals)
					if _, genErr := s.generatePortal(state, false); genErr != nil {
						_, _ = fmt.Fprintln(s.stderr, "Could not rebuild locale portal", state.Locale+":", genErr)
					}
				}
			}
			s.mu.Unlock()
		}
	}
}

func browserURL(host string, port int) string {
	if host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port)) + "/"
}
func externallyReachableHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}
func serveDocumentation(options Options, stdout, stderr io.Writer) error {
	handler, model, result, err := newDocumentationServer(options, stderr)
	if err != nil {
		return err
	}
	if handler.agentConsole != nil {
		defer handler.agentConsole.Close()
	}
	address := net.JoinHostPort(options.Host, strconv.Itoa(options.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer func() { _ = listener.Close() }()
	localURL := browserURL(options.Host, options.Port)
	_, _ = fmt.Fprintf(stdout, "\nDocumentation server started.\nAddress:        %s\nDirectory:      %s\nPages:          %d\nDocuments:      %d\nWarnings:       %d\nErrors:         %d\n", localURL, result.OutputDirectory, result.Pages, model.Stats.Documents, model.Stats.Warnings, model.Stats.Errors)
	if externallyReachableHost(options.Host) {
		_, _ = fmt.Fprintf(stdout, "Local network: http://<computer-IP-address>:%d/\nWarning: the server is reachable from the network without authentication or TLS.\n", options.Port)
	}
	if options.Open {
		if err := openGeneratedSite(localURL); err != nil {
			_, _ = fmt.Fprintln(stderr, "Failed to open the browser automatically:", err)
		}
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	watchContext, stopWatcher := context.WithCancel(context.Background())
	defer stopWatcher()
	go handler.watch(watchContext)
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
