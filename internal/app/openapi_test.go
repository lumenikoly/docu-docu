package toudocu

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func validOpenAPISource(extra string) []byte {
	return []byte("openapi: 3.1.0\ninfo:\n  title: Test\n  version: '1'\npaths:\n  /items/{id}:\n    get:\n      operationId: getItem\n      parameters:\n        - name: id\n          in: path\n          required: true\n          schema: {type: string}\n      responses:\n        '200': {description: ok}\ncomponents:\n  schemas:\n    Item: {type: object}\n" + extra)
}

func openAPIIssueCodes(issues []Issue) map[string]bool {
	result := map[string]bool{}
	for _, issue := range issues {
		result[issue.Code] = true
	}
	return result
}

func TestOpenAPIValidation(t *testing.T) {
	if issues := validateOpenAPIContract("contracts/test.openapi.yaml", validOpenAPISource("")); len(issues) != 0 {
		t.Fatalf("valid contract issues: %#v", issues)
	}
	tests := []struct {
		name string
		src  string
		code string
	}{
		{"syntax", "openapi: [", "openapi-syntax-error"},
		{"root", "- openapi", "openapi-invalid-root"},
		{"version", "info: {title: Test, version: '1'}\npaths: {}", "openapi-invalid-version"},
		{"malformed version", "openapi: 3.1.invalid\ninfo: {title: Test, version: '1'}\npaths: {}", "openapi-invalid-version"},
		{"info", "openapi: 3.1.0\npaths: {}", "openapi-missing-info"},
		{"responses", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX}}}", "openapi-missing-responses"},
		{"operation id", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {responses: {'200': {description: ok}}}}}", "openapi-missing-operation-id"},
		{"duplicate", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: same, responses: {'200': {description: ok}}}}, '/y': {get: {operationId: same, responses: {'200': {description: ok}}}}}", "openapi-duplicate-operation-id"},
		{"path parameter", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x/{id}': {get: {operationId: getX, responses: {'200': {description: ok}}}}}", "openapi-missing-path-parameter"},
		{"path item key", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {fetch: {operationId: getX, responses: {'200': {description: ok}}}}}", "openapi-invalid-path-item-key"},
		{"response status", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX, responses: {'20': {description: ok}}}}}", "openapi-invalid-response-status"},
		{"response object", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX, responses: {'200': ok}}}}", "openapi-invalid-response"},
		{"response description", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX, responses: {'200': {content: {}}}}}}", "openapi-missing-response-description"},
		{"ref", "openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX, responses: {'200': {description: ok, content: {application/json: {schema: {$ref: '#/components/schemas/Missing'}}}}}}}}", "openapi-unresolved-internal-ref"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issues := validateOpenAPIContract("contracts/test.openapi.yaml", []byte(test.src))
			if !openAPIIssueCodes(issues)[test.code] {
				t.Fatalf("missing %s in %#v", test.code, issues)
			}
			if issues[0].Line == 0 {
				t.Fatalf("diagnostic has no line: %#v", issues[0])
			}
		})
	}
	external := []byte("openapi: 3.1.0\ninfo: {title: Test, version: '1'}\npaths: {'/x': {get: {operationId: getX, responses: {'200': {description: ok, content: {application/json: {schema: {$ref: 'https://invalid.example/schema.yaml'}}}}}}}}")
	if issues := validateOpenAPIContract("contracts/external.openapi.yaml", external); len(issues) != 0 {
		t.Fatalf("external refs must not be resolved: %#v", issues)
	}
}

func TestEditorOpenAPIDiagnostics(t *testing.T) {
	workspace := &editorWorkspace{}
	diagnostics, err := workspace.diagnostics("contracts/test.openapi.json", []byte(`{"openapi":"3.1.0","info":{"title":"T","version":"1"},"paths":{"/x":{"get":{"operationId":"x"}}}}`))
	if err != nil || len(diagnostics) != 1 || diagnostics[0].Code != "openapi-missing-responses" || diagnostics[0].Line == 0 || diagnostics[0].Column == 0 {
		t.Fatalf("diagnostics=%#v err=%v", diagnostics, err)
	}
	plain, err := workspace.diagnostics("contracts/plain.yaml", []byte("not: openapi"))
	if err != nil || len(plain) != 0 {
		t.Fatalf("plain YAML changed behavior: %#v %v", plain, err)
	}
}

func readContractOperations(t *testing.T, name string) map[string]map[string]struct{} {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "docs", "contracts", name))
	if err != nil {
		t.Fatal(err)
	}
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	return openAPIOperations(documentMapping(&document))
}

func registryOperations(routes []apiRoute) map[string]map[string]struct{} {
	result := map[string]map[string]struct{}{}
	for _, route := range routes {
		result[route.Path] = map[string]struct{}{}
		for _, method := range route.Methods {
			result[route.Path][method] = struct{}{}
		}
	}
	return result
}

func TestOpenAPIContracts(t *testing.T) {
	for _, name := range []string{"editor.openapi.yaml", "changes.openapi.yaml", "agent-feedback.openapi.yaml"} {
		content, err := os.ReadFile(filepath.Join("..", "..", "docs", "contracts", name))
		if err != nil {
			t.Fatal(err)
		}
		contract, issues := parseOpenAPIContract("contracts/"+name, content)
		if len(issues) != 0 || contract.Version != "3.1.0" || contract.Title == "" {
			t.Fatalf("%s: contract=%#v issues=%#v", name, contract, issues)
		}
		for _, token := range []string{"schemaVersion", "example:", "application/json"} {
			if !strings.Contains(string(content), token) {
				t.Fatalf("%s missing %s", name, token)
			}
		}
		if name == "agent-feedback.openapi.yaml" {
			for _, token := range []string{
				"const: agent-discussion-create", "const: agent-discussion-update", "const: agent-discussion-delete",
				"const: agent-message-create", "const: agent-message-update", "const: agent-message-delete",
				"const: agent-delivery-next", "const: agent-delivery-response",
				"additionalProperties: false", "x-maxBytes: 65536", "atomically queue one editable request",
				"RequestOrigin:", "FetchMetadata:", `security: [{RequestOrigin: []}, {FetchMetadata: []}]`,
				`"415": {$ref: "#/components/responses/AgentError"}`,
			} {
				if !strings.Contains(string(content), token) {
					t.Fatalf("%s missing strict input contract %q", name, token)
				}
			}
			if strings.Contains(string(content), "maxLength:") {
				t.Fatal("agent feedback byte limits must not be represented as Unicode maxLength")
			}
			if got := strings.Count(string(content), `"503":`); got != 10 {
				t.Fatalf("agent feedback must document 503 for every operation: got %d", got)
			}
			createStart := strings.Index(string(content), "operationId: createAgentDiscussion")
			if createStart < 0 {
				t.Fatal("agent feedback create discussion operation is missing")
			}
			createEnd := strings.Index(string(content)[createStart:], "/_toudocu/api/agent/discussions/{discussionId}")
			if createEnd < 0 || !strings.Contains(string(content)[createStart:createStart+createEnd], `"404":`) {
				t.Fatal("agent feedback create discussion must document a missing target")
			}
		}
	}
}

func TestOpenAPIContractParity(t *testing.T) {
	editorRoutes := append(append([]apiRoute{}, editorRouteRegistry...), editorServiceRouteRegistry...)
	for _, route := range append(append([]apiRoute{}, editorRoutes...), allChangesRouteRegistry()...) {
		if route.Handler == nil {
			t.Fatalf("route %s has no handler", route.Path)
		}
	}
	if got, want := registryOperations(editorRoutes), readContractOperations(t, "editor.openapi.yaml"); !reflect.DeepEqual(got, want) {
		t.Fatalf("editor registry/spec mismatch\nregistry=%#v\nspec=%#v", got, want)
	}
	if got, want := registryOperations(allChangesRouteRegistry()), readContractOperations(t, "changes.openapi.yaml"); !reflect.DeepEqual(got, want) {
		t.Fatalf("changes registry/spec mismatch\nregistry=%#v\nspec=%#v", got, want)
	}
	if got, want := registryOperations(reviewRouteRegistry), readContractOperations(t, "agent-feedback.openapi.yaml"); !reflect.DeepEqual(got, want) {
		t.Fatalf("agent feedback registry/spec mismatch\nregistry=%#v\nspec=%#v", got, want)
	}
}

func TestContractDocumentLinksToSelectedSwaggerSpecOnlyInServe(t *testing.T) {
	document := &Document{SourcePath: "contracts/editor-http.md", Links: []Link{{Destination: "editor.openapi.yaml"}}}
	model := &Model{serveMode: true, openAPIContracts: []OpenAPIContract{{Path: "contracts/editor.openapi.yaml", Title: "Editor"}}}
	button := renderOpenAPIContractButton(model, document)
	if !strings.Contains(button, "Open in Swagger UI") || !strings.Contains(button, "spec=contracts%2Feditor.openapi.yaml") {
		t.Fatalf("button=%q", button)
	}
	model.serveMode = false
	if button := renderOpenAPIContractButton(model, document); button != "" {
		t.Fatalf("static button=%q", button)
	}
}

func TestAPIDocsUI(t *testing.T) {
	server, _ := changesHTTPServer(t)
	server.model = &Model{Project: ProjectInfo{Title: "Test"}}
	server.model.openAPIContracts = []OpenAPIContract{{Path: "contracts/editor.openapi.yaml", Title: "Editor", Version: "3.1.0"}, {Path: "contracts/changes.openapi.yaml", Title: "Changes", Version: "3.1.0"}}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, apiDocsUIPath, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "swagger-ui-bundle.js") || !strings.Contains(response.Body.String(), "contracts/editor.openapi.yaml") || !strings.Contains(response.Body.String(), "contracts/changes.openapi.yaml") {
		t.Fatalf("API docs response: %d %s", response.Code, response.Body.String())
	}
	for header, value := range map[string]string{"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"} {
		if response.Header().Get(header) != value {
			t.Fatalf("%s=%q", header, response.Header().Get(header))
		}
	}
	if csp := response.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") || !strings.Contains(csp, "connect-src 'self'") {
		t.Fatalf("CSP=%q", csp)
	}
	head := httptest.NewRecorder()
	server.ServeHTTP(head, httptest.NewRequest(http.MethodHead, apiDocsUIPath, nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD: %d %q", head.Code, head.Body.String())
	}
	hidden := httptest.NewRecorder()
	server.ServeHTTP(hidden, httptest.NewRequest(http.MethodGet, apiDocsUIPath, nil))
	if hidden.Code != http.StatusOK {
		t.Fatalf("locale API docs=%d", hidden.Code)
	}
}

func TestSwaggerUIVendoredAssets(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "web", "package.json"))
	if err != nil || !strings.Contains(string(manifest), `"swagger-ui-dist": "5.32.12"`) {
		t.Fatalf("Swagger UI version is not pinned: %v", err)
	}
	checksums, err := os.ReadFile(filepath.Join("..", "site", "assets", "generated", "swagger-ui.checksums.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(checksums)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			t.Fatalf("bad checksum line %q", line)
		}
		content, err := os.ReadFile(filepath.Join("..", "site", "assets", "generated", parts[1]))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != parts[0] {
			t.Fatalf("checksum mismatch for %s", parts[1])
		}
	}
	initializer, err := os.ReadFile(filepath.Join("..", "..", "web", "src", "features", "api-docs.ts"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{`get("spec")`, `"urls.primaryName": primaryName`} {
		if !strings.Contains(string(initializer), token) {
			t.Fatalf("api-docs.js missing %q", token)
		}
	}
	if strings.Contains(string(initializer), "https://") || !strings.Contains(string(initializer), `supportedSubmitMethods: ["get", "head"]`) {
		t.Fatalf("unsafe initializer contract")
	}
}

func TestStaticSiteExcludesAPIDocs(t *testing.T) {
	repository := t.TempDir()
	docs := filepath.Join(repository, "docs")
	ensureTestDocumentationVersion(t, docs)
	if err := os.MkdirAll(filepath.Join(docs, "contracts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "index.md"), []byte("# Test\n\nDocs.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docs, "contracts", "test.openapi.yaml"), validOpenAPISource(""), 0o644); err != nil {
		t.Fatal(err)
	}
	model, err := BuildDocumentationModel(Options{InputDirectory: docs, RepositoryRoot: repository})
	if err != nil {
		t.Fatal(err)
	}
	staticOutput := filepath.Join(repository, "static")
	if _, err = GenerateSite(model, Options{InputDirectory: docs, RepositoryRoot: repository, OutputDirectory: staticOutput}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staticOutput, "contracts", "test.openapi.yaml")); err != nil {
		t.Fatalf("static spec missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staticOutput, "assets", "swagger-ui.css")); !os.IsNotExist(err) {
		t.Fatalf("static output contains Swagger UI: %v", err)
	}
	index, err := os.ReadFile(filepath.Join(staticOutput, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(index), apiDocsUIPath) {
		t.Fatal("static navigation contains API docs")
	}
	serveOutput := filepath.Join(repository, "serve")
	if _, err = generateServeSite(model, Options{InputDirectory: docs, RepositoryRoot: repository, OutputDirectory: serveOutput}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(serveOutput, "assets", "swagger-ui.css")); err != nil {
		t.Fatalf("serve Swagger UI missing: %v", err)
	}
	serveIndex, err := os.ReadFile(filepath.Join(serveOutput, "index.html"))
	if err != nil || !strings.Contains(string(serveIndex), apiDocsUIPath) {
		t.Fatalf("serve navigation missing API docs: %v", err)
	}
}

func TestIncompleteServerExcludesAPIDocs(t *testing.T) {
	server := &documentationServer{model: &Model{Project: ProjectInfo{Title: "Locale"}}}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, apiDocsUIPath, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("incomplete server API docs: %d", response.Code)
	}
}

func TestChangesHTTPContract(t *testing.T) {
	server, _ := changesHTTPServer(t)
	for _, test := range []struct {
		method string
		url    string
		status int
		allow  string
	}{
		{http.MethodPost, changesAPIBase, http.StatusMethodNotAllowed, "GET, HEAD"},
		{http.MethodHead, changesAPIBase + "/file", http.StatusMethodNotAllowed, "GET"},
		{http.MethodGet, changesAPIBase + "/missing", http.StatusNotFound, ""},
	} {
		response := httptest.NewRecorder()
		server.ServeHTTP(response, httptest.NewRequest(test.method, test.url, nil))
		if response.Code != test.status || !strings.Contains(response.Body.String(), `"schemaVersion":1`) || !strings.Contains(response.Body.String(), `"diagnostics"`) {
			t.Fatalf("%s %s: %d %s", test.method, test.url, response.Code, response.Body.String())
		}
		if test.allow != "" && response.Header().Get("Allow") != test.allow {
			t.Fatalf("Allow=%q, want %q", response.Header().Get("Allow"), test.allow)
		}
	}
	invalidSide := httptest.NewRecorder()
	server.ServeHTTP(invalidSide, httptest.NewRequest(http.MethodGet, changesAPIBase+"/content?side=invalid&path=docs/modules/MOD-CORE.md", nil))
	if invalidSide.Code != http.StatusBadRequest || invalidSide.Header().Get("Content-Type") != "application/json; charset=utf-8" || !strings.Contains(invalidSide.Body.String(), "invalid-change-side") {
		t.Fatalf("invalid side: %d %#v %s", invalidSide.Code, invalidSide.Header(), invalidSide.Body.String())
	}
}
