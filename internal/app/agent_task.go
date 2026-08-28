package toudocu

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
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
	Instruction         string `json:"instruction"`
	ReadOnlyInstruction bool   `json:"readOnlyInstruction"`
}

func (s *documentationServer) resolveTaskActions(taskID string) (taskActionProjection, error) {
	model, item, _, content, err := s.taskDocument(taskID)
	if err != nil {
		return taskActionProjection{}, err
	}
	state := taskWorkspaceState(item, taskWorkspaceReadiness(model, item, model.strictPolicy, workItemsByID(model)))
	projection := taskActionProjection{SchemaVersion: 1, Task: taskActionTask{ID: item.ID, Status: string(item.statusName), WorkspaceState: state, Digest: contentDigest(content)}, Agent: taskActionAgent{Relation: "none", Status: "off"}, Actions: preparedTaskActions(state)}
	if s.agentConsole == nil {
		return projection, nil
	}
	snapshot, active := s.agentConsole.manager.Snapshot()
	if active {
		projection.Agent = taskActionAgent{Relation: "other-task", Status: string(snapshot.Status), NeedsAttention: len(snapshot.Approvals) > 0}
		if snapshot.Settings.Launch.TaskID == taskID {
			projection.Agent.Relation = "current-task"
		}
	}
	for index := range projection.Actions {
		action := &projection.Actions[index]
		delivery := TaskActionDelivery{Type: "agent-console", Available: true}
		if active && snapshot.Settings.Launch.TaskID != taskID {
			delivery.Available = false
			delivery.UnavailableReason = "busy_other_task"
		} else if action.policy == AgentTurnReadOnly && active && !snapshot.Settings.Capabilities.ReadOnlyTurns || action.policy == AgentTurnReadOnly && !active && !s.agentConsole.provider.Capabilities().ReadOnlyTurns {
			delivery.Available = false
			delivery.UnavailableReason = "read_only_unavailable"
		} else if active && snapshot.Settings.Launch.TaskID == taskID && action.ID == "continue-work" {
			delivery.OpenSession = true
			action.Label = "Open agent"
		}
		action.Deliveries = append([]TaskActionDelivery{delivery}, action.Deliveries...)
	}
	return projection, nil
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
	if delivery == "handoff" {
		if action.mutates {
			if err = s.markTaskInProgress(document, content, expectedDigest); err != nil {
				return taskActionResult{}, err
			}
		}
		result.Handoff = &taskActionHandoff{Instruction: prompt, ReadOnlyInstruction: action.policy == AgentTurnReadOnly}
	} else {
		if s.agentConsole == nil {
			return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: "Agent Console is unavailable"}
		}
		snapshot, active := s.agentConsole.manager.Snapshot()
		if active && snapshot.Settings.Launch.TaskID != taskID {
			return taskActionResult{}, &agentTaskConflict{code: "busy_other_task", message: "Agent Session belongs to another task"}
		}
		if action.policy == AgentTurnReadOnly && active && !snapshot.Settings.Capabilities.ReadOnlyTurns || action.policy == AgentTurnReadOnly && !active && !s.agentConsole.provider.Capabilities().ReadOnlyTurns {
			return taskActionResult{}, &agentTaskConflict{code: "unavailable_delivery", message: "Agent Console cannot provide a read-only turn"}
		}
		if actionID == "continue-work" && active {
			result.OpenSession = true
		} else {
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
