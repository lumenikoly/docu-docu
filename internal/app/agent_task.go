package toudocu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
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
	report := BuildTaskReady(model, taskID, model.strictPolicy)
	if item.statusName != WorkItemReady || !report.ReadyForWork {
		message := "Task is not Ready for work"
		if len(report.Issues) > 0 {
			message += ": " + report.Issues[0].Message
		}
		return nil, nil, &agentTaskConflict{code: "task_not_ready", message: message, command: "toudocu task ready " + taskID + " docs --repository-root ."}
	}
	return document, content, nil
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

type taskActionTask struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	WorkspaceState string `json:"workspaceState"`
	Digest         string `json:"digest"`
}

type taskActionProjection struct {
	SchemaVersion int               `json:"schemaVersion"`
	Task          taskActionTask    `json:"task"`
	Agent         taskActionAgent   `json:"agent"`
	Actions       []AgentTaskAction `json:"actions"`
}

type taskActionAgent struct {
	Relation       string `json:"relation"`
	Status         string `json:"status"`
	NeedsAttention bool   `json:"needsAttention"`
}

type taskActionResult struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ActionID      string                `json:"actionID"`
	Delivery      string                `json:"delivery"`
	OpenSession   bool                  `json:"openSession,omitempty"`
	Handoff       *taskActionHandoff    `json:"handoff,omitempty"`
	Projection    *taskActionProjection `json:"projection,omitempty"`
}

type taskActionHandoff struct {
	SchemaVersion      int    `json:"schemaVersion"`
	TaskID             string `json:"taskID"`
	ActionID           string `json:"actionID"`
	MediaType          string `json:"mediaType"`
	Text               string `json:"text"`
	Truncated          bool   `json:"truncated"`
	FullContextCommand string `json:"fullContextCommand"`
}

func (s *documentationServer) buildTaskActionHandoff(model *Model, item *WorkItem, action AgentTaskAction, instruction string, readiness taskReadinessSummary) (taskActionHandoff, error) {
	docsPath, err := filepath.Rel(s.options.RepositoryRoot, s.options.InputDirectory)
	if err != nil {
		return taskActionHandoff{}, err
	}
	command := fmt.Sprintf("toudocu task context %s %s --repository-root . --format json", item.ID, filepath.ToSlash(docsPath))

	criteria := make([]string, 0, len(item.Criteria))
	for _, criterion := range item.Criteria {
		criteria = append(criteria, "- "+criterion.Text)
	}
	primary := "# Toudocu task handoff\n\n## Action\n\n" + instruction + "\n\nUse the authoritative Toudocu task contract and current repository state.\n"
	if action.ID != "fix-problems" {
		primary += "Do not change the task contract without explicit user approval.\n"
	}
	primary += "Do not mark the task Done automatically.\n"
	if action.policy == AgentTurnReadOnly {
		primary += "\nThis action is intended to be read-only.\nDo not modify repository files.\nToudocu cannot enforce the permissions of an external agent.\n"
	}
	primary += fmt.Sprintf("\n## Task\n\n- ID: `%s`\n- Status: `%s`\n- Document: `%s`\n\n## Acceptance criteria\n\n%s\n", item.ID, item.statusName, filepath.ToSlash(filepath.Join(docsPath, item.Document)), strings.Join(criteria, "\n"))

	secondary := ""
	context, contextErr := BuildTaskContext(model, item.ID)
	if contextErr == nil {
		secondary = handoffSecondaryContext(context)
	} else if item.statusName == WorkItemDraft {
		secondary = handoffDraftContext(item, readiness)
	} else {
		return taskActionHandoff{}, contextErr
	}
	suffix := fmt.Sprintf("\n## Complete context\n\nFor the complete current context run:\n\n`%s`\n", command)
	text := primary + secondary + suffix
	truncated := len(text) > agentMessageLimit
	if truncated {
		text = primary + suffix
		if len(text) > agentMessageLimit {
			text = truncateUTF8(primary, agentMessageLimit-len(suffix)) + suffix
		}
	}
	return taskActionHandoff{SchemaVersion: 1, TaskID: item.ID, ActionID: action.ID, MediaType: "text/markdown", Text: text, Truncated: truncated, FullContextCommand: command}, nil
}

func handoffSecondaryContext(report TaskContextReport) string {
	var body strings.Builder
	if report.Task.Result != "" {
		fmt.Fprintf(&body, "\n## Goal\n\n%s\n", report.Task.Result)
	}
	if len(report.Task.RepositoryPaths) > 0 {
		body.WriteString("\n## Scope\n\n")
		for _, path := range report.Task.RepositoryPaths {
			fmt.Fprintf(&body, "- `%s`\n", path)
		}
	}
	if len(report.Dependencies) > 0 {
		body.WriteString("\n## Dependencies\n\n")
		for _, dependency := range report.Dependencies {
			fmt.Fprintf(&body, "- `%s`: %s\n", dependency.ID, dependency.statusName)
		}
	}
	if len(report.RequiredReads) > 0 {
		body.WriteString("\n## Required context\n\n")
		for _, path := range report.RequiredReads {
			fmt.Fprintf(&body, "- `%s`\n", path)
		}
	}
	if len(report.Issues) > 0 {
		body.WriteString("\n## Readiness issues\n\n")
		for _, issue := range report.Issues {
			fmt.Fprintf(&body, "- %s\n", issue.Message)
		}
	}
	if len(report.Task.Checks) > 0 {
		body.WriteString("\n## Verification\n\n")
		for _, check := range report.Task.Checks {
			for _, command := range check.Commands {
				fmt.Fprintf(&body, "- `%s`: `%s`\n", check.Target, command)
			}
		}
	}
	return body.String()
}

func handoffDraftContext(item *WorkItem, readiness taskReadinessSummary) string {
	report := TaskContextReport{Task: *item, Issues: readiness.Issues, RequiredReads: []string{item.Document}}
	return handoffSecondaryContext(report)
}

func truncateUTF8(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}

func (s *documentationServer) resolveTaskActions(taskID string) (taskActionProjection, error) {
	model, item, _, content, err := s.taskDocument(taskID)
	if err != nil {
		return taskActionProjection{}, err
	}
	return s.resolveTaskActionsFrom(model, item, content), nil
}

func (s *documentationServer) resolveTaskActionsSnapshot(taskID string) (taskActionProjection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.model == nil {
		return taskActionProjection{}, errors.New("portal model is unavailable")
	}
	item, err := findWorkItem(s.model, taskID)
	if err != nil {
		return taskActionProjection{}, err
	}
	document := s.model.DocByPath[item.Document]
	if document == nil {
		return taskActionProjection{}, errors.New("task document is unavailable")
	}
	return s.resolveTaskActionsFrom(s.model, item, []byte(document.Content)), nil
}

func (s *documentationServer) resolveTaskActionsFrom(model *Model, item *WorkItem, content []byte) taskActionProjection {
	state := taskWorkspaceState(item, taskWorkspaceReadiness(model, item, model.strictPolicy, workItemsByID(model)))
	projection := taskActionProjection{SchemaVersion: 1, Task: taskActionTask{ID: item.ID, Status: string(item.statusName), WorkspaceState: state, Digest: contentDigest(content)}, Agent: taskActionAgent{Relation: "none", Status: "off"}, Actions: preparedTaskActions(state)}
	if s.agentConsole == nil {
		return projection
	}
	snapshot, active := s.agentConsole.manager.Snapshot()
	if active {
		projection.Agent = taskActionAgent{Relation: "other-task", Status: string(snapshot.Status), NeedsAttention: len(snapshot.Approvals) > 0}
		if snapshot.Settings.Launch.TaskID == "" {
			projection.Agent.Relation = "unbound"
		} else if snapshot.Settings.Launch.TaskID == item.ID {
			projection.Agent.Relation = "current-task"
		}
	}
	reason := taskActionAgentUnavailable(snapshot, active, item.ID)
	actions := projection.Actions[:0]
	for index := range projection.Actions {
		action := projection.Actions[index]
		if active && snapshot.Settings.Launch.TaskID == item.ID && action.ID == "continue-work" && reason != "" {
			continue
		}
		delivery := TaskActionDelivery{Type: "agent-console", Available: true}
		if reason != "" {
			delivery.Available = false
			delivery.UnavailableReason = reason
		} else if action.policy == AgentTurnReadOnly && active && !snapshot.Settings.Capabilities.ReadOnlyTurns || action.policy == AgentTurnReadOnly && !active && !s.agentConsole.provider.Capabilities().ReadOnlyTurns {
			delivery.Available = false
			delivery.UnavailableReason = "read_only_unavailable"
		}
		action.Deliveries = append([]TaskActionDelivery{delivery}, action.Deliveries...)
		actions = append(actions, action)
	}
	projection.Actions = actions
	return projection
}

func taskActionAgentUnavailable(snapshot AgentSessionSnapshot, active bool, taskID string) string {
	if !active {
		return ""
	}
	if snapshot.Settings.Launch.TaskID == "" {
		return "busy_unbound_session"
	}
	if snapshot.Settings.Launch.TaskID != taskID {
		return "busy_other_task"
	}
	if len(snapshot.Approvals) > 0 {
		return "agent_needs_attention"
	}
	switch snapshot.Status {
	case AgentSessionRunning:
		return "agent_running"
	case AgentSessionStopping:
		return "agent_stopping"
	case AgentSessionFailed:
		return "agent_failed"
	default:
		return ""
	}
}

func taskActionUnavailableMessage(code string) string {
	switch code {
	case "busy_unbound_session":
		return "Agent Session is not bound to this task"
	case "busy_other_task":
		return "Agent Session belongs to another task"
	case "agent_needs_attention":
		return "Agent Session requires a decision before another task action"
	case "agent_running":
		return "Agent Session is already working on this task"
	case "agent_stopping":
		return "Agent Session is stopping"
	case "agent_failed":
		return "Agent Session failed and requires cleanup"
	default:
		return "Task action delivery is unavailable"
	}
}

func (s *documentationServer) executeTaskAction(ctx context.Context, taskID, actionID, delivery, expectedDigest, input string, preset AgentLaunchPreset) (taskActionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input = strings.TrimSpace(input)
	action, ok := agentTaskActions[actionID]
	if !ok {
		return taskActionResult{}, &agentTaskConflict{code: "invalid_action", message: "Unsupported task action"}
	}
	if delivery != "agent-console" && delivery != "handoff" {
		return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: "Unsupported task action delivery"}
	}
	if action.Input == "text" && input == "" {
		return taskActionResult{}, &agentTaskConflict{code: "invalid_input", message: "Text input is required"}
	}
	if len([]byte(input)) > agentMessageLimit {
		return taskActionResult{}, &agentTaskConflict{code: "invalid_input", message: "Text input exceeds 65536 bytes"}
	}
	model, item, document, content, err := s.taskDocument(taskID)
	if err != nil {
		return taskActionResult{}, err
	}
	if expectedDigest != "" && contentDigest(content) != expectedDigest {
		return taskActionResult{}, &agentTaskConflict{code: "stale_digest", message: "Task changed; refresh Task Workspace and try again"}
	}
	readiness := taskWorkspaceReadiness(model, item, model.strictPolicy, workItemsByID(model))
	state := taskWorkspaceState(item, readiness)
	if !action.states[state] {
		message := "Task action is not allowed in state " + state
		if issues := blockingReadinessIssues(readiness.Issues, model.strictPolicy); len(issues) > 0 {
			message += ": " + issues[0].Message
		}
		return taskActionResult{}, &agentTaskConflict{code: "invalid_state", message: message}
	}
	if action.mutates {
		document, content, err = s.ensureAgentTaskReady(taskID, expectedDigest)
		if err != nil {
			return taskActionResult{}, err
		}
	}
	result := taskActionResult{SchemaVersion: 1, ActionID: actionID, Delivery: delivery}
	prompt := action.build(taskID, input)
	if action.Input == "none" && input != "" {
		prompt += "\n\nAdditional instruction for this run:\n" + input
	}
	if delivery == "handoff" {
		if action.mutates {
			if err = s.markTaskInProgress(document, content, expectedDigest); err != nil {
				return taskActionResult{}, err
			}
			model, item, _, _, err = s.taskDocument(taskID)
			if err != nil {
				return taskActionResult{}, err
			}
			readiness = taskWorkspaceReadiness(model, item, model.strictPolicy, workItemsByID(model))
		}
		handoff, handoffErr := s.buildTaskActionHandoff(model, item, action, prompt, readiness)
		if handoffErr != nil {
			return taskActionResult{}, handoffErr
		}
		result.Handoff = &handoff
	} else {
		if s.agentConsole == nil {
			return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: "Agent Console is unavailable"}
		}
		snapshot, active := s.agentConsole.manager.Snapshot()
		if code := taskActionAgentUnavailable(snapshot, active, taskID); code != "" {
			return taskActionResult{}, &agentTaskConflict{code: code, message: taskActionUnavailableMessage(code)}
		}
		if action.policy == AgentTurnReadOnly && active && !snapshot.Settings.Capabilities.ReadOnlyTurns || action.policy == AgentTurnReadOnly && !active && !s.agentConsole.provider.Capabilities().ReadOnlyTurns {
			return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: "Agent Console cannot provide a read-only turn"}
		}
		started := false
		if !active {
			if err = s.agentConsole.startStructured(ctx, taskID, preset); err != nil {
				return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: err.Error()}
			}
			started = true
		}
		if action.mutates {
			if err = s.markTaskInProgress(document, content, expectedDigest); err != nil {
				if started {
					_ = s.agentConsole.manager.Stop(ctx, true)
				}
				return taskActionResult{}, err
			}
		}
		if err = s.agentConsole.manager.Send(ctx, prompt, action.policy); err != nil {
			return taskActionResult{}, err
		}
		s.agentConsole.publishState()
	}
	projection, err := s.resolveTaskActions(taskID)
	if err == nil {
		result.Projection = &projection
	}
	return result, nil
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
