package toudocu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func agentTaskTestServer(t *testing.T, task string) (*documentationServer, string, *consoleSpySession) {
	t.Helper()
	status := regexp.MustCompile(`(?m)^- Status: (.+)$`).FindStringSubmatch(task)[1]
	task = regexp.MustCompile(`(?m)^- (?:Status|Type|Module|Use case): .+\n`).ReplaceAllString(task, "")
	task = strings.Replace(task, "# TASK-AUTH-021: Add verification workflow\n", "<!-- toudocu\nid: TASK-AUTH-021\nstatus: "+strings.ToLower(status)+"\ntaskType: feature\nmodule: MOD-AUTH\nuseCase: UC-AUTH-01\nupdated: 2026-08-26\n-->\n\n# TASK-AUTH-021: Add verification workflow\n\nFixture task description.\n", 1)
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
	server := &documentationServer{options: options, workspace: workspace, model: model, agentConsole: console}
	return server, filepath.Join(docs, "work", "TASK-AUTH-021.md"), session
}

func TestStartTaskDigest(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Ready"))
	content, _ := os.ReadFile(path)
	err := server.startTask(context.Background(), "TASK-AUTH-021", contentDigest(content)+"stale", AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "stale_digest" {
		t.Fatalf("error=%v", err)
	}
}

func TestStartTaskReadiness(t *testing.T) {
	server, path, _ := agentTaskTestServer(t, completeTaskFixture("Draft"))
	content, _ := os.ReadFile(path)
	err := server.startTask(context.Background(), "TASK-AUTH-021", contentDigest(content), AgentLaunchDefault)
	var conflict *agentTaskConflict
	if !errors.As(err, &conflict) || conflict.code != "task_not_ready" {
		t.Fatalf("error=%v", err)
	}
}

func TestStartTask(t *testing.T) {
	server, path, session := agentTaskTestServer(t, completeTaskFixture("Ready"))
	content, _ := os.ReadFile(path)
	if err := server.startValidatedTask(context.Background(), "TASK-AUTH-021", contentDigest(content), AgentLaunchDefault, &Document{SourcePath: "work/TASK-AUTH-021.md"}, content); err != nil {
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

func TestAgentTaskActions(t *testing.T) {
	for _, state := range []string{"ready", "in-progress", "waiting", "needs-attention", "draft"} {
		if len(preparedTaskActions(state)) == 0 {
			t.Fatalf("no actions for %s", state)
		}
	}
	if len(preparedTaskActions("done")) != 0 || agentTaskActions["ask"].Policy != AgentTurnReadOnly || agentTaskActions["clarify"].Policy == AgentTurnReadOnly {
		t.Fatal("invalid action policy or terminal-state actions")
	}
}

func TestAgentPromptRegistry(t *testing.T) {
	for id, action := range agentTaskActions {
		if action.ID != id || strings.Count(action.Prompt, "%s") != 1 {
			t.Fatalf("invalid action %s: %+v", id, action)
		}
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
