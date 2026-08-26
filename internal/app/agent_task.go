package toudocu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"toudocu/internal/skillinstall"
)

var readyTaskStatusRE = regexp.MustCompile(`(?mi)^(?:-[ \t]+)?status:[ \t]*ready[ \t]*$`)

type agentTaskConflict struct {
	code, message, command string
}

func (e *agentTaskConflict) Error() string { return e.message }

func (s *documentationServer) taskDocument(taskID string) (*Model, *WorkItem, *Document, []byte, error) {
	model, err := BuildDocumentationModel(s.options)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	item, err := findWorkItem(model, taskID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	document := model.DocByPath[item.Document]
	if document == nil {
		return nil, nil, nil, nil, errors.New("task document is unavailable")
	}
	content, err := os.ReadFile(document.AbsolutePath)
	return model, item, document, content, err
}

func (s *documentationServer) ensureAgentTaskReady(taskID, digest string) (*Document, []byte, error) {
	model, item, document, content, err := s.taskDocument(taskID)
	if err != nil {
		return nil, nil, err
	}
	if digest == "" || contentDigest(content) != digest {
		return nil, nil, &agentTaskConflict{code: "stale_digest", message: "Task changed; refresh Task Workspace and try again"}
	}
	report := BuildTaskReady(model, taskID, true)
	if item.statusName != WorkItemReady || !report.ReadyForWork {
		message := "Task is not Ready for work"
		if len(report.Issues) > 0 {
			message += ": " + report.Issues[0].Message
		}
		return nil, nil, &agentTaskConflict{code: "task_not_ready", message: message, command: "toudocu task ready " + taskID + " docs --repository-root ."}
	}
	setup := s.agentConsole.setup("en")
	if setup.Skill.State != skillinstall.Installed {
		return nil, nil, &agentTaskConflict{code: "skill_not_ready", message: setup.Skill.Diagnostic, command: setup.Skill.Command}
	}
	return document, content, nil
}

func (s *documentationServer) startTask(ctx context.Context, taskID, digest string, preset AgentLaunchPreset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	document, content, err := s.ensureAgentTaskReady(taskID, digest)
	if err != nil {
		return err
	}
	return s.startValidatedTask(ctx, taskID, digest, preset, document, content)
}

func (s *documentationServer) startValidatedTask(ctx context.Context, taskID, digest string, preset AgentLaunchPreset, document *Document, content []byte) error {
	if err := s.agentConsole.startStructured(ctx, taskID, preset); err != nil {
		return err
	}
	if err := s.markTaskInProgress(document, content, digest); err != nil {
		_ = s.agentConsole.manager.Stop(ctx, true)
		return err
	}
	return s.agentConsole.manager.Send(ctx, fmt.Sprintf(agentTaskActions["start-work"].Prompt, taskID), AgentTurnNormal)
}

func (s *documentationServer) markTaskInProgress(document *Document, content []byte, digest string) error {
	updated := readyTaskStatusRE.ReplaceAll(content, []byte("status: in-progress"))
	if string(updated) == string(content) {
		return errors.New("task status metadata is not Ready")
	}
	updatedRE := regexp.MustCompile(`(?m)^updated:[ \t]*[^\r\n]+$`)
	updated = updatedRE.ReplaceAll(updated, []byte("updated: "+time.Now().UTC().Format("2006-01-02")))
	_, err := s.workspace.save(document.SourcePath, updated, digest)
	return err
}

func (s *documentationServer) runTaskAction(ctx context.Context, taskID, actionID, question string, preset AgentLaunchPreset) error {
	action, ok := agentTaskActions[actionID]
	if !ok || actionID == "start-work" {
		return errors.New("unsupported task action")
	}
	model, item, _, _, err := s.taskDocument(taskID)
	if err != nil {
		return err
	}
	state := taskWorkspaceState(item, taskWorkspaceReadiness(model, item, model.strictPolicy, workItemsByID(model)))
	allowed := false
	for _, candidate := range preparedTaskActions(state) {
		allowed = allowed || candidate.ID == actionID
	}
	if !allowed {
		return errors.New("task action is not allowed in the current state")
	}
	_, active := s.agentConsole.manager.Snapshot()
	if !active {
		if err = s.agentConsole.startStructured(ctx, taskID, preset); err != nil {
			return err
		}
	}
	prompt := fmt.Sprintf(action.Prompt, taskID)
	if actionID == "ask" && strings.TrimSpace(question) != "" {
		prompt += "\n\nQuestion: " + strings.TrimSpace(question)
	}
	return s.agentConsole.manager.Send(ctx, prompt, action.Policy)
}

func workItemsByID(model *Model) map[string]*WorkItem {
	result := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		item := &model.Knowledge.WorkItems[index]
		result[item.ID] = item
	}
	return result
}

func (s *documentationServer) verifyAgentTask(taskID string) TaskVerifyReport {
	model, err := BuildDocumentationModel(s.options)
	if err != nil {
		return TaskVerifyReport{SchemaVersion: 1, Kind: "task-verify", Status: "blocked", ValidationIssues: []Issue{{Severity: "error", Code: "model-build-failed", Message: err.Error()}}}
	}
	runner := s.taskRunner
	if runner == nil {
		runner = osCommandRunner{}
	}
	report := executeTaskVerify(model, Options{TaskID: taskID, VerifyMode: "run", Format: "json", Timeout: s.options.Timeout}, io.Discard, io.Discard, runner)
	s.agentConsole.mu.Lock()
	s.agentConsole.verification = &report
	s.agentConsole.mu.Unlock()
	s.agentConsole.publishState()
	return report
}

func (s *documentationServer) sendVerificationFailure(ctx context.Context) error {
	s.agentConsole.mu.Lock()
	report := s.agentConsole.verification
	s.agentConsole.mu.Unlock()
	if report == nil || report.Status != "failed" {
		return errors.New("latest verification did not fail")
	}
	data, _ := json.Marshal(report)
	prefix := "Review this failed Toudocu task verification and propose a fix. Do not make changes until the user asks.\n\n"
	return s.agentConsole.manager.Send(ctx, prefix+boundedAgentText(string(data), agentMessageLimit-len(prefix)), AgentTurnNormal)
}
