package toudocu

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func agentTaskTestServer(t *testing.T, task string) (*documentationServer, string, *consoleSpySession) {
	t.Helper()
	status := regexp.MustCompile(`(?m)^- Status: (.+)$`).FindStringSubmatch(task)[1]
	task = regexp.MustCompile(`(?m)^- (?:Status|Type|Module|Use case): .+\n`).ReplaceAllString(task, "")
	task = strings.Replace(task, "# TASK-AUTH-021: Add verification workflow\n", "<!-- toudocu\nid: TASK-AUTH-021\nstatus: "+strings.ToLower(status)+"\ntaskType: feature\nmodule: MOD-AUTH\nuseCase: UC-AUTH-01\nupdated: 2026-08-26\n-->\n\n# TASK-AUTH-021: Add verification workflow\n\nFixture task description.\n", 1)
	for heading, section := range map[string]string{"## Result": "result", "## Behavior change": "behavior-change", "### Before": "before", "### After": "after", "## Scope": "scope", "## Out of scope": "out-of-scope", "## Acceptance criteria": "acceptance-criteria", "## Plan": "plan", "## Verification": "verification", "## Documentation impact": "documentation-impact"} {
		task = strings.Replace(task, heading, "<!-- toudocu:section "+section+" -->\n"+heading, 1)
	}
	model, docs := hierarchyModel(t, map[string]string{"work/TASK-AUTH-021.md": task})
	writeTestFile(t, model.RepositoryRoot, "new.go", "package fixture\n")
	options := Options{Command: "serve", InputDirectory: docs, OutputDirectory: filepath.Join(model.RepositoryRoot, "site"), RepositoryRoot: model.RepositoryRoot, RepositoryRef: "main", StaleDays: 0, Host: "127.0.0.1"}
	workspace, err := newEditorWorkspace(options)
	if err != nil {
		t.Fatal(err)
	}
	session := &consoleSpySession{fakeAgentSession: newFakeAgentSession()}
	session.settings.Capabilities.ReadOnlyTurns = true
	console := newAgentConsole(&consoleSpyProvider{session: session}, model.RepositoryRoot)
	server := &documentationServer{options: options, workspace: workspace, model: model, agentConsole: console, taskActionsEnabled: true}
	return server, filepath.Join(docs, "work", "TASK-AUTH-021.md"), session
}

func TestStartTaskDigest(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Ready"))
	content, _ := os.ReadFile(path)
	_, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "start-work", "handoff", contentDigest(content)+"stale", "", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "stale_digest" {
		t.Fatalf("error=%v", err)
	}
}

func TestTaskActionDigest(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	_, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "ask", "handoff", "stale", "question", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "stale_digest" {
		t.Fatalf("error=%v", err)
	}
}

func TestTaskActionStartWorkReadiness(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Draft"))
	content, _ := os.ReadFile(path)
	_, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "start-work", "handoff", contentDigest(content), "", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "invalid_state" {
		t.Fatalf("error=%v", err)
	}
}

func TestStartTask(t *testing.T) {
	server, path, session := agentTaskTestServer(t, completeTaskFixture("Ready"))
	content, _ := os.ReadFile(path)
	if _, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "start-work", "agent-console", contentDigest(content), "", AgentLaunchDefault); err != nil {
		t.Fatalf("%v\n%s", err, content)
	}
	updated, _ := os.ReadFile(path)
	if !strings.Contains(string(updated), "status: in-progress") && !strings.Contains(string(updated), "Status: in-progress") {
		t.Fatalf("status not updated:\n%s", updated)
	}
	if prompts := session.turnTexts(); session.settings.Launch.TaskID != "TASK-AUTH-021" || len(prompts) != 1 {
		t.Fatalf("session=%+v prompts=%+v", session.settings, prompts)
	}
}

func TestTaskActionResolver(t *testing.T) {
	for _, state := range []string{"ready", "in-progress", "waiting", "needs-attention", "draft", "blocked", "done"} {
		if len(preparedTaskActions(state)) == 0 {
			t.Fatalf("no actions for %s", state)
		}
	}
	if len(preparedTaskActions("cancelled")) != 0 || agentTaskActions["ask"].policy != AgentTurnReadOnly || agentTaskActions["clarify"].policy == AgentTurnReadOnly {
		t.Fatal("invalid action policy or terminal-state actions")
	}
}

func TestTaskActionRegistry(t *testing.T) {
	for id, action := range agentTaskActions {
		if action.ID != id || action.build == nil || action.build("TASK-X-001", "question") == "" {
			t.Fatalf("invalid action %s: %+v", id, action)
		}
	}
}

func TestTaskContinueWork(t *testing.T) {
	server, _, session := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "agent-console", "", "", AgentLaunchDefault)
	if err != nil || result.OpenSession || len(session.turnTexts()) != 1 {
		t.Fatalf("first continue: result=%+v prompts=%v err=%v", result, session.turnTexts(), err)
	}
	result, err = server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "agent-console", "", "", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "agent_running" || len(session.turnTexts()) != 1 {
		t.Fatalf("active continue: result=%+v prompts=%v err=%v", result, session.turnTexts(), err)
	}
	server.agentConsole.manager.mu.Lock()
	server.agentConsole.manager.status, server.agentConsole.manager.turnID = AgentSessionIdle, ""
	server.agentConsole.manager.mu.Unlock()
	result, err = server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "agent-console", "", "", AgentLaunchDefault)
	if err != nil || result.OpenSession || len(session.turnTexts()) != 2 {
		t.Fatalf("idle continue: result=%+v prompts=%v err=%v", result, session.turnTexts(), err)
	}
}

func TestTaskActionAddsOneTimeInstruction(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "handoff", "", "Check the migration first.", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || !strings.Contains(result.Handoff.Text, "Additional instruction for this run:\nCheck the migration first.") {
		t.Fatalf("handoff=%+v err=%v", result.Handoff, err)
	}
	result, err = server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "handoff", "", "", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || strings.Contains(result.Handoff.Text, "Check the migration first.") {
		t.Fatalf("next handoff=%+v err=%v", result.Handoff, err)
	}
}

func TestTaskActionHandoff(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "handoff", "", "", AgentLaunchDefault)
	if err != nil || result.Handoff == nil {
		t.Fatalf("handoff=%+v err=%v", result.Handoff, err)
	}
	for _, expected := range []string{"# Toudocu task handoff", "`TASK-AUTH-021`", "## Acceptance criteria", "docs/work/TASK-AUTH-021.md", result.Handoff.FullContextCommand} {
		if !strings.Contains(result.Handoff.Text, expected) {
			t.Fatalf("handoff missing %q:\n%s", expected, result.Handoff.Text)
		}
	}
	if result.Handoff.SchemaVersion != 1 || result.Handoff.MediaType != "text/markdown" || result.Handoff.ActionID != "continue-work" {
		t.Fatalf("handoff=%+v", result.Handoff)
	}
}

func TestTaskActionHandoffDraft(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("Draft"))
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "ask", "handoff", "", "What is missing?", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || !strings.Contains(result.Handoff.Text, "## Readiness issues") {
		t.Fatalf("handoff=%+v err=%v", result.Handoff, err)
	}
}

func TestTaskActionHandoffBounds(t *testing.T) {
	task := strings.Replace(completeTaskFixture("In-progress"), "The requested behavior is implemented.", strings.Repeat("🙂", agentMessageLimit), 1)
	server, _, _ := agentTaskTestServer(t, task)
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "handoff", "", "", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || !result.Handoff.Truncated || len(result.Handoff.Text) > agentMessageLimit || !utf8.ValidString(result.Handoff.Text) || !strings.Contains(result.Handoff.Text, result.Handoff.FullContextCommand) {
		t.Fatalf("bytes=%d handoff=%+v err=%v", len(result.Handoff.Text), result.Handoff, err)
	}
}

func TestTaskActionHandoffReadOnly(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "ask", "handoff", "", "Question", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || !strings.Contains(result.Handoff.Text, "intended to be read-only") || !strings.Contains(result.Handoff.Text, "cannot enforce") {
		t.Fatalf("handoff=%+v err=%v", result.Handoff, err)
	}
}

func TestTaskActionHandoffStartWork(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Ready"))
	content, _ := os.ReadFile(path)
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "start-work", "handoff", contentDigest(content), "", AgentLaunchDefault)
	updated, _ := os.ReadFile(path)
	if err != nil || result.Handoff == nil || !strings.Contains(string(updated), "status: in-progress") || !strings.Contains(result.Handoff.Text, "Status: `in-progress`") || result.Projection == nil || result.Projection.Task.Status != "in-progress" {
		t.Fatalf("handoff=%+v projection=%+v err=%v\n%s", result.Handoff, result.Projection, err, updated)
	}
}

func TestTaskActionHandoffIsolation(t *testing.T) {
	t.Setenv("TOUDOCU_HANDOFF_SECRET", "never-copy-this-secret")
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	writeTestFile(t, server.options.RepositoryRoot, "credentials.txt", "never-copy-this-secret")
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "continue-work", "handoff", "", "", AgentLaunchDefault)
	if err != nil || result.Handoff == nil || strings.Contains(result.Handoff.Text, "never-copy-this-secret") {
		t.Fatalf("handoff=%+v err=%v", result.Handoff, err)
	}
}

func TestTaskActionSessionBinding(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-OTHER-001", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	_, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "ask", "agent-console", "", "question", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "busy_other_task" {
		t.Fatalf("error=%v", err)
	}
	result, err := server.executeTaskAction(context.Background(), "TASK-AUTH-021", "ask", "handoff", "", "question", AgentLaunchDefault)
	if err != nil || result.Handoff == nil {
		t.Fatalf("handoff=%+v err=%v", result, err)
	}
}

func TestTaskActionReadOnlyDelivery(t *testing.T) {
	server, _, session := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	session.settings.Capabilities.ReadOnlyTurns = false
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-AUTH-021", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	projection, err := server.resolveTaskActions("TASK-AUTH-021")
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range projection.Actions {
		if action.ID == "ask" && (len(action.Deliveries) != 2 || action.Deliveries[0].Type != "agent-console" || action.Deliveries[0].Available || action.Deliveries[0].UnavailableReason != "read_only_unavailable" || action.Deliveries[1].Type != "handoff" || !action.Deliveries[1].Available) {
			t.Fatalf("ask deliveries=%+v", action.Deliveries)
		}
	}
}

func TestTaskActionProjectionKeepsBusyConsoleVisible(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-OTHER-001", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	projection, err := server.resolveTaskActions("TASK-AUTH-021")
	if err != nil {
		t.Fatal(err)
	}
	delivery := projection.Actions[0].Deliveries[0]
	if delivery.Type != "agent-console" || delivery.Available || delivery.UnavailableReason != "busy_other_task" || !projection.Actions[0].Deliveries[1].Available {
		t.Fatalf("deliveries=%+v", projection.Actions[0].Deliveries)
	}
}

func TestTaskActionProjectionIncludesAgentState(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	projection, err := server.resolveTaskActions("TASK-AUTH-021")
	if err != nil || projection.Agent.Relation != "none" || projection.Agent.Status != "off" {
		t.Fatalf("inactive agent=%+v err=%v", projection.Agent, err)
	}
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-AUTH-021", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	server.agentConsole.manager.mu.Lock()
	server.agentConsole.manager.status = AgentSessionRunning
	server.agentConsole.manager.turnID = "turn-1"
	server.agentConsole.manager.approvals["approval-1"] = AgentApproval{RequestID: "approval-1"}
	server.agentConsole.manager.mu.Unlock()
	projection, err = server.resolveTaskActions("TASK-AUTH-021")
	if err != nil || projection.Agent.Relation != "current-task" || projection.Agent.Status != "running" || !projection.Agent.NeedsAttention {
		t.Fatalf("active agent=%+v err=%v", projection.Agent, err)
	}
	for _, action := range projection.Actions {
		if action.ID == "continue-work" {
			t.Fatal("running projection contains continue-work")
		}
		if len(action.Deliveries) != 2 || action.Deliveries[0].Available || action.Deliveries[0].UnavailableReason != "agent_needs_attention" || !action.Deliveries[1].Available {
			t.Fatalf("running deliveries=%+v", action.Deliveries)
		}
	}
}

func TestTaskActionProjectionTreatsManualSessionAsUnbound(t *testing.T) {
	server, _, _ := agentTaskTestServer(t, completeTaskFixture("In-progress"))
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	projection, err := server.resolveTaskActions("TASK-AUTH-021")
	if err != nil || projection.Agent.Relation != "unbound" || projection.Actions[0].Deliveries[0].UnavailableReason != "busy_unbound_session" {
		t.Fatalf("projection=%+v err=%v", projection, err)
	}
}

func TestTaskActionsHTTP(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Ready"))
	server.agentConsole = nil
	get := agentConsoleRequest(http.MethodGet, "/_toudocu/api/tasks/TASK-AUTH-021/actions", "", "")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, get)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"start-work"`) || strings.Contains(response.Body.String(), `"agent-console"`) {
		t.Fatalf("GET status=%d body=%s", response.Code, response.Body.String())
	}
	content, _ := os.ReadFile(path)
	post := agentConsoleRequest(http.MethodPost, "/_toudocu/api/tasks/TASK-AUTH-021/actions/start-work", "task-action-execute", `{"delivery":"handoff","expectedDigest":"`+contentDigest(content)+`","input":{"text":""}}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, post)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"handoff"`) {
		t.Fatalf("POST status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestTaskActionsGETUsesPortalSnapshot(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Ready"))
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	get := agentConsoleRequest(http.MethodGet, "/_toudocu/api/tasks/TASK-AUTH-021/actions", "", "")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, get)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"start-work"`) {
		t.Fatalf("GET status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestTaskActionsHTTPErrors(t *testing.T) {
	tests := []struct {
		name, status, action, delivery, digest, input, code string
		prepare                                             func(*documentationServer, *consoleSpySession)
	}{
		{name: "invalid state", status: "In-progress", action: "start-work", delivery: "handoff", code: "invalid_state"},
		{name: "stale digest", status: "Ready", action: "start-work", delivery: "handoff", digest: "stale", code: "stale_digest"},
		{name: "busy other task", status: "In-progress", action: "ask", delivery: "agent-console", input: "question", code: "busy_other_task", prepare: func(server *documentationServer, _ *consoleSpySession) {
			if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-OTHER-001", AgentLaunchDefault); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "unavailable delivery", status: "In-progress", action: "ask", delivery: "agent-console", input: "question", code: "unavailable_delivery", prepare: func(server *documentationServer, session *consoleSpySession) {
			session.settings.Capabilities.ReadOnlyTurns = false
			if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-AUTH-021", AgentLaunchDefault); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, path, session := agentTaskTestServer(t, completeTaskFixture(test.status))
			if test.prepare != nil {
				test.prepare(server, session)
			}
			digest := test.digest
			if digest == "" {
				content, _ := os.ReadFile(path)
				digest = contentDigest(content)
			}
			body := fmt.Sprintf(`{"delivery":%q,"expectedDigest":%q,"input":{"text":%q}}`, test.delivery, digest, test.input)
			request := agentConsoleRequest(http.MethodPost, "/_toudocu/api/tasks/TASK-AUTH-021/actions/"+test.action, "task-action-execute", body)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestTaskActionsRuntimeIsolation(t *testing.T) {
	server := &documentationServer{}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, agentConsoleRequest(http.MethodGet, "/_toudocu/api/tasks/TASK-X-001/actions", "", ""))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAgentConsoleVerificationAuthorization(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/verify", "agent-task-verify", `{"taskID":"TASK-X-001","confirmed":false}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "verification_confirmation_required") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAgentConsoleVerificationRun(t *testing.T) {
	repositoryRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	server, session := agentConsoleTestServer(t)
	server.options = Options{InputDirectory: filepath.Join(repositoryRoot, "docs"), RepositoryRoot: repositoryRoot, StaleDays: 0}
	server.taskRunner = &fakeCommandRunner{outcomes: map[string]fakeCommandOutcome{}}
	server.agentConsole.cwd = repositoryRoot
	if err = server.agentConsole.manager.Start(context.Background(), repositoryRoot, "TASK-AGENT-008", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/verify", "agent-task-verify", `{"taskID":"TASK-AGENT-008","confirmed":true}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || server.agentConsole.verification == nil || server.agentConsole.verification.Mode != "run" || len(session.turnTexts()) != 0 {
		t.Fatalf("status=%d verification=%+v body=%s", response.Code, server.agentConsole.verification, response.Body.String())
	}
}

func TestAgentConsoleVerificationFailureToAgent(t *testing.T) {
	server, session := agentConsoleTestServer(t)
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "TASK-X-001", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	server.agentConsole.verification = &TaskVerifyReport{Status: "failed", Task: TaskVerifyTask{ID: "TASK-X-001"}}
	if err := server.sendVerificationFailure(context.Background()); err != nil {
		t.Fatal(err)
	}
	if prompts := session.turnTexts(); len(prompts) != 1 || !strings.Contains(prompts[0], "failed Toudocu task verification") {
		t.Fatalf("prompts=%v", prompts)
	}
}
