package toudocu

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"toudocu/internal/skillinstall"
	"toudocu/skills"
)

const (
	agentConsoleAPIBase = "/_toudocu/api/agent-console"
	agentConsoleWS      = agentConsoleAPIBase + "/ws"
	agentEventLimit     = 256
	agentBufferLimit    = 3 << 20
)

type agentConsoleMessage struct {
	Sequence  uint64                    `json:"sequence"`
	Kind      string                    `json:"kind"`
	Event     *agentConsoleEvent        `json:"event,omitempty"`
	State     *agentConsoleSessionState `json:"state,omitempty"`
	ReplayGap *agentReplayGap           `json:"replayGap,omitempty"`
	Terminal  *agentTerminalEvent       `json:"terminal,omitempty"`
}

type agentReplayGap struct {
	After  uint64 `json:"after"`
	Before uint64 `json:"before"`
}

type agentConsoleEvent struct {
	Type           AgentEventType `json:"type"`
	ThreadID       string         `json:"threadID,omitempty"`
	TurnID         string         `json:"turnID,omitempty"`
	ItemID         string         `json:"itemID,omitempty"`
	Text           string         `json:"text,omitempty"`
	Command        string         `json:"command,omitempty"`
	Status         string         `json:"status,omitempty"`
	ExitCode       *int           `json:"exitCode,omitempty"`
	CWD            string         `json:"cwd,omitempty"`
	DurationMillis int64          `json:"durationMillis,omitempty"`
	ApprovalState  string         `json:"approvalState,omitempty"`
	Truncated      bool           `json:"truncated,omitempty"`
	Approval       *AgentApproval `json:"approval,omitempty"`
}

type agentConsoleSettings struct {
	TaskID          string                 `json:"taskID,omitempty"`
	Preset          AgentLaunchPreset      `json:"preset"`
	EffectiveAccess EffectiveAccessSummary `json:"effectiveAccess"`
	Capabilities    AgentCapabilities      `json:"capabilities"`
	Model           string                 `json:"model,omitempty"`
	Effort          string                 `json:"effort,omitempty"`
}

type agentConsoleSessionState struct {
	Active       bool                  `json:"active"`
	Settings     *agentConsoleSettings `json:"settings,omitempty"`
	Status       AgentSessionStatus    `json:"status,omitempty"`
	ActiveTurn   string                `json:"activeTurn,omitempty"`
	Pending      []AgentPendingMessage `json:"pending,omitempty"`
	Approvals    []AgentApproval       `json:"approvals,omitempty"`
	Failure      string                `json:"failure,omitempty"`
	Verification *TaskVerifyReport     `json:"verification,omitempty"`
	Terminal     agentTerminalState    `json:"terminal"`
}

type agentConsole struct {
	manager      *AgentSessionManager
	cwd          string
	mu           sync.Mutex
	next         uint64
	events       []agentConsoleMessage
	eventSizes   []int
	eventBytes   int
	closed       bool
	clients      map[chan agentConsoleMessage]struct{}
	connections  map[net.Conn]struct{}
	stopEvents   func()
	provider     AgentProvider
	preferences  AgentPreferenceStore
	verification *TaskVerifyReport
	terminal     *agentPTY
	startShell   func() error
}

func newAgentConsole(provider AgentProvider, cwd string) *agentConsole {
	preferences, _ := NewAgentPreferenceStore()
	console := &agentConsole{manager: NewAgentSessionManager(provider), provider: provider, preferences: preferences, cwd: cwd, clients: map[chan agentConsoleMessage]struct{}{}, connections: map[net.Conn]struct{}{}}
	console.terminal = newAgentPTY(func(event agentTerminalEvent) {
		console.publish(agentConsoleMessage{Kind: "terminal", Terminal: &event})
		console.publishState()
	})
	console.startShell = func() error { return console.terminal.StartShell(console.cwd) }
	events, stopEvents := console.manager.Subscribe()
	console.stopEvents = stopEvents
	go func() {
		for event := range events {
			normalized := console.normalizeAgentEvent(event)
			console.publish(agentConsoleMessage{Kind: "event", Event: &normalized})
		}
	}()
	return console
}

func (c *agentConsole) startStructured(ctx context.Context, taskID string, preset AgentLaunchPreset) error {
	if _, active := c.manager.Snapshot(); active {
		return errors.New("agent session is already active")
	}
	if preset != "" && !validLaunchPreset(preset) {
		return fmt.Errorf("unsupported agent launch preset %q", preset)
	}
	preference := c.preferences.Load(c.cwd)
	err := c.manager.StartConfigured(ctx, AgentLaunch{CWD: c.cwd, TaskID: taskID, Preset: preset, Model: preference.Model, Effort: preference.Effort})
	return err
}

func (c *agentConsole) resumeStructured(ctx context.Context, threadID string, preset AgentLaunchPreset) error {
	if _, active := c.manager.Snapshot(); active {
		return errors.New("agent session is already active")
	}
	preference := c.preferences.Load(c.cwd)
	return c.manager.ResumeConfigured(ctx, AgentLaunch{CWD: c.cwd, Preset: preset, Model: preference.Model, Effort: preference.Effort}, threadID)
}

func (c *agentConsole) startProjectTerminal() error { return c.startShell() }

func (c *agentConsole) stopTerminal(ctx context.Context) error {
	return c.terminal.Stop(ctx)
}

type agentConsoleSetup struct {
	AvailableProviders []string          `json:"availableProviders"`
	SelectedProvider   string            `json:"selectedProvider"`
	Preference         AgentPreferences  `json:"preference"`
	Skill              agentConsoleSkill `json:"skill"`
	Models             []AgentModel      `json:"models,omitempty"`
}

type agentConsoleSkill struct {
	State      skillinstall.State `json:"state"`
	Diagnostic string             `json:"diagnostic"`
	Command    string             `json:"command,omitempty"`
}

func (c *agentConsole) setup(language string) agentConsoleSetup {
	ru := strings.HasPrefix(strings.ToLower(language), "ru")
	message := func(en, russian string) string {
		if ru {
			return russian
		}
		return en
	}
	setup := agentConsoleSetup{AvailableProviders: []string{c.provider.Name()}, SelectedProvider: c.provider.Name(), Preference: c.preferences.Load(c.cwd)}
	bundle, err := skills.Load()
	if err != nil {
		setup.Skill = agentConsoleSkill{State: skillinstall.InvalidManifest, Diagnostic: message("Bundled Toudocu skill is unavailable.", "Встроенный навык Toudocu недоступен.")}
		return setup
	}
	targets, err := skillinstall.ResolveTargets(c.provider.Name(), skillinstall.Project, c.cwd, "")
	if err != nil || len(targets) != 1 {
		setup.Skill = agentConsoleSkill{State: skillinstall.UnsafePath, Diagnostic: message("Toudocu skill target is unavailable.", "Целевой каталог навыка Toudocu недоступен.")}
		return setup
	}
	snapshot := skillinstall.Inspect(targets[0], bundle)
	setup.Skill.State = snapshot.State
	switch snapshot.State {
	case skillinstall.Installed:
		setup.Skill.Diagnostic = message("Toudocu skill is ready.", "Навык Toudocu готов к работе.")
	case skillinstall.NotInstalled:
		setup.Skill.Diagnostic, setup.Skill.Command = message("Toudocu skill must be installed before prepared workflows can run.", "Перед запуском подготовленных действий установите навык Toudocu."), "toudocu skill install --agent "+c.provider.Name()
	case skillinstall.Outdated:
		setup.Skill.Diagnostic, setup.Skill.Command = message("Toudocu skill must be updated before prepared workflows can run.", "Перед запуском подготовленных действий обновите навык Toudocu."), "toudocu skill update --agent "+c.provider.Name()
	default:
		setup.Skill.Diagnostic = message("Toudocu skill requires manual repair: ", "Навык Toudocu требует ручного исправления: ") + string(snapshot.State) + "."
	}
	return setup
}

func (c *agentConsole) changeSkill(operation skillinstall.Operation) error {
	if operation != skillinstall.Install && operation != skillinstall.Update {
		return errors.New("unsupported skill operation")
	}
	bundle, err := skills.Load()
	if err != nil {
		return err
	}
	targets, err := skillinstall.ResolveTargets(c.provider.Name(), skillinstall.Project, c.cwd, "")
	if err != nil || len(targets) != 1 {
		return errors.New("skill target is unavailable")
	}
	plan := skillinstall.BuildPlan(operation, targets[0], bundle)
	if plan.Conflict {
		return errors.New(plan.Code + ": " + plan.Message)
	}
	result := skillinstall.Execute(plan, Version)
	return result.Error
}

func (c *agentConsole) normalizeAgentEvent(event AgentEvent) agentConsoleEvent {
	var approval *AgentApproval
	if event.Approval != nil {
		approval = &AgentApproval{RequestID: boundedAgentText(event.Approval.RequestID, 4096), Kind: boundedAgentText(event.Approval.Kind, 4096), Reason: boundedAgentText(event.Approval.Reason, agentMessageLimit)}
	}
	text, truncated := boundedAgentTail(event.Text, agentMessageLimit)
	cwd := ""
	if relative, err := filepath.Rel(c.cwd, event.CWD); event.CWD != "" && err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		cwd = filepath.ToSlash(relative)
	}
	return agentConsoleEvent{Type: event.Type, ThreadID: boundedAgentText(event.ThreadID, 4096), TurnID: boundedAgentText(event.TurnID, 4096), ItemID: boundedAgentText(event.ItemID, 4096), Text: text, Command: boundedAgentText(event.Command, agentMessageLimit), CWD: cwd, Status: boundedAgentText(event.Status, 4096), ExitCode: event.ExitCode, DurationMillis: event.DurationMillis, ApprovalState: boundedAgentText(event.ApprovalState, 64), Truncated: truncated, Approval: approval}
}

func boundedAgentTail(value string, limit int) (string, bool) {
	if len(value) <= limit {
		return value, false
	}
	return string([]byte(value)[len(value)-limit:]), true
}

func boundedAgentText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return string([]byte(value)[:limit])
}

func normalizeAgentState(state AgentSessionSnapshot, active bool) agentConsoleSessionState {
	normalized := agentConsoleSessionState{Active: active, Status: state.Status, ActiveTurn: state.ActiveTurn, Pending: state.Pending, Approvals: state.Approvals, Failure: boundedAgentText(state.Failure, agentMessageLimit)}
	if active {
		normalized.Settings = &agentConsoleSettings{TaskID: state.Settings.Launch.TaskID, Preset: state.Settings.Launch.Preset, EffectiveAccess: state.Settings.EffectiveAccess, Capabilities: state.Settings.Capabilities, Model: state.Settings.Launch.Model, Effort: state.Settings.Launch.Effort}
	}
	return normalized
}

type lazyCodexProvider struct{}

func (lazyCodexProvider) Name() string { return "codex" }
func (lazyCodexProvider) Capabilities() AgentCapabilities {
	return AgentCapabilities{Steering: true, Interrupt: true, Approvals: true, ReadOnlyTurns: true}
}
func (lazyCodexProvider) Start(ctx context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	provider, err := NewCodexProvider()
	if err != nil {
		return nil, err
	}
	return provider.Start(ctx, launch)
}
func (lazyCodexProvider) Models(ctx context.Context, cwd string) ([]AgentModel, error) {
	provider, err := NewCodexProvider()
	if err != nil {
		return nil, err
	}
	return provider.Models(ctx, cwd)
}
func (lazyCodexProvider) Threads(ctx context.Context, cwd string) ([]AgentThread, error) {
	provider, err := NewCodexProvider()
	if err != nil {
		return nil, err
	}
	return provider.Threads(ctx, cwd)
}
func (lazyCodexProvider) Resume(ctx context.Context, launch AgentLaunch, threadID string) (AgentProviderSession, error) {
	provider, err := NewCodexProvider()
	if err != nil {
		return nil, err
	}
	return provider.Resume(ctx, launch, threadID)
}

func (c *agentConsole) publish(message agentConsoleMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.next++
	message.Sequence = c.next
	size := 0
	if encoded, err := json.Marshal(message); err == nil {
		size = len(encoded)
	}
	if size <= agentBufferLimit {
		c.events = append(c.events, message)
		c.eventSizes = append(c.eventSizes, size)
		c.eventBytes += size
	}
	for len(c.events) > agentEventLimit || c.eventBytes > agentBufferLimit {
		c.eventBytes -= c.eventSizes[0]
		c.events, c.eventSizes = c.events[1:], c.eventSizes[1:]
	}
	for client := range c.clients {
		select {
		case client <- message:
		default:
		}
	}
}

func (c *agentConsole) publishState() {
	state, ok := c.manager.Snapshot()
	normalized := normalizeAgentState(state, ok)
	c.mu.Lock()
	normalized.Verification = c.verification
	normalized.Terminal = c.terminal.Snapshot(true)
	c.mu.Unlock()
	c.publish(agentConsoleMessage{Kind: "state", State: &normalized})
}

func (c *agentConsole) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	for client := range c.clients {
		close(client)
		delete(c.clients, client)
	}
	for connection := range c.connections {
		_ = connection.Close()
		delete(c.connections, connection)
	}
	c.events, c.eventSizes, c.eventBytes = nil, nil, 0
	c.mu.Unlock()
	c.stopEvents()
	terminalContext, terminalCancel := context.WithTimeout(context.Background(), 3*time.Second)
	_ = c.terminal.Stop(terminalContext)
	terminalCancel()
	_ = c.manager.Shutdown(context.Background())
}

func (c *agentConsole) registerConnection(connection net.Conn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		_ = connection.Close()
		return false
	}
	c.connections[connection] = struct{}{}
	return true
}

func (c *agentConsole) subscribe(since uint64) ([]agentConsoleMessage, <-chan agentConsoleMessage, func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	backlog := make([]agentConsoleMessage, 0, len(c.events))
	if since > 0 && len(c.events) > 0 && since+1 < c.events[0].Sequence {
		backlog = append(backlog, agentConsoleMessage{Sequence: c.events[0].Sequence - 1, Kind: "replay_gap", ReplayGap: &agentReplayGap{After: since, Before: c.events[0].Sequence}})
	}
	for _, event := range c.events {
		if event.Sequence > since {
			backlog = append(backlog, event)
		}
	}
	ch := make(chan agentConsoleMessage, 64)
	if c.closed {
		close(ch)
		return backlog, ch, func() {}
	}
	c.clients[ch] = struct{}{}
	return backlog, ch, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if _, ok := c.clients[ch]; ok {
			delete(c.clients, ch)
			close(ch)
		}
	}
}

func (s *documentationServer) serveAgentConsole(w http.ResponseWriter, r *http.Request) {
	if s.agentConsole == nil {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == agentConsoleWS {
		s.agentConsole.serveWebSocket(w, r)
		return
	}
	if path == agentConsoleAPIBase && r.Method == http.MethodGet {
		if !agentRequestOriginAllowed(r) {
			writeEditorError(w, http.StatusForbidden, "origin_forbidden", "Agent Console requires a loopback same-origin request", nil)
			return
		}
		state, active := s.agentConsole.manager.Snapshot()
		normalized := normalizeAgentState(state, active)
		s.agentConsole.mu.Lock()
		normalized.Verification = s.agentConsole.verification
		normalized.Terminal = s.agentConsole.terminal.Snapshot(true)
		s.agentConsole.mu.Unlock()
		setup := s.agentConsole.setup(r.Header.Get("Accept-Language"))
		if provider, ok := s.agentConsole.provider.(AgentModelProvider); ok {
			setup.Models, _ = provider.Models(r.Context(), s.agentConsole.cwd)
		}
		writeChangesJSON(w, http.StatusOK, struct {
			SchemaVersion int                      `json:"schemaVersion"`
			Setup         agentConsoleSetup        `json:"setup"`
			State         agentConsoleSessionState `json:"state"`
		}{1, setup, normalized})
		return
	}
	if path == agentConsoleAPIBase+"/history" && r.Method == http.MethodGet {
		if !agentRequestOriginAllowed(r) {
			writeEditorError(w, http.StatusForbidden, "origin_forbidden", "Agent Console requires a loopback same-origin request", nil)
			return
		}
		provider, ok := s.agentConsole.provider.(AgentHistoryProvider)
		if !ok {
			writeEditorError(w, http.StatusNotImplemented, "history_unavailable", "Agent provider does not support history", nil)
			return
		}
		threads, err := provider.Threads(r.Context(), s.agentConsole.cwd)
		if err != nil {
			writeEditorError(w, http.StatusConflict, "agent_history_failed", err.Error(), nil)
			return
		}
		writeChangesJSON(w, http.StatusOK, struct {
			SchemaVersion int           `json:"schemaVersion"`
			Threads       []AgentThread `json:"threads"`
		}{1, threads})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeEditorError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	action := "agent-session-start"
	if path == agentConsoleAPIBase+"/stop" {
		action = "agent-session-stop"
	} else if path == agentConsoleAPIBase+"/resume" {
		action = "agent-session-resume"
	} else if path == agentConsoleAPIBase+"/cleanup" {
		action = "agent-session-cleanup"
	} else if path == agentConsoleAPIBase+"/pending/cancel" {
		action = "agent-pending-cancel"
	} else if path == agentConsoleAPIBase+"/preference" {
		action = "agent-preference-save"
	} else if path == agentConsoleAPIBase+"/skill" {
		action = "agent-skill-change"
	} else if path == agentConsoleAPIBase+"/verify" {
		action = "agent-task-verify"
	} else if path == agentConsoleAPIBase+"/verification/send" {
		action = "agent-verification-send"
	} else if path == agentConsoleAPIBase+"/terminal/start" {
		action = "project-terminal-start"
	} else if path == agentConsoleAPIBase+"/terminal/stop" {
		action = "project-terminal-stop"
	}
	if !agentRequestOriginAllowed(r) {
		writeEditorError(w, http.StatusForbidden, "origin_forbidden", "Agent Console requires a loopback same-origin request", nil)
		return
	}
	if !requireEditorJSONAction(w, r, action) {
		return
	}
	var err error
	switch {
	case path == agentConsoleAPIBase+"/start":
		var input struct {
			TaskID   string            `json:"taskID"`
			Provider string            `json:"provider"`
			Preset   AgentLaunchPreset `json:"preset"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		if input.TaskID != "" && !workItemIDRE.MatchString(input.TaskID) {
			writeEditorError(w, http.StatusBadRequest, "invalid_task", "Invalid task ID", nil)
			return
		}
		if input.Provider != "" && input.Provider != s.agentConsole.provider.Name() {
			writeEditorError(w, http.StatusBadRequest, "invalid_provider", "Provider is not available", nil)
			return
		}
		if input.Preset == "" {
			input.Preset = s.agentConsole.preferences.Load(s.agentConsole.cwd).LaunchPreset
		}
		err = s.agentConsole.startStructured(r.Context(), input.TaskID, input.Preset)
	case path == agentConsoleAPIBase+"/resume":
		var input struct {
			ThreadID string            `json:"threadID"`
			Preset   AgentLaunchPreset `json:"preset"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		if input.ThreadID == "" || len(input.ThreadID) > 4096 {
			writeEditorError(w, http.StatusBadRequest, "invalid_thread", "Invalid agent thread ID", nil)
			return
		}
		if input.Preset == "" {
			input.Preset = s.agentConsole.preferences.Load(s.agentConsole.cwd).LaunchPreset
		}
		err = s.agentConsole.resumeStructured(r.Context(), input.ThreadID, input.Preset)
	case path == agentConsoleAPIBase+"/stop":
		var input struct {
			DiscardPending bool `json:"discardPending"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		err = s.agentConsole.manager.Stop(r.Context(), input.DiscardPending)
	case path == agentConsoleAPIBase+"/cleanup":
		err = s.agentConsole.manager.Cleanup()
	case path == agentConsoleAPIBase+"/pending/cancel":
		var input struct {
			ID string `json:"id"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		if input.ID == "" {
			writeEditorError(w, http.StatusBadRequest, "invalid_pending_id", "Pending message ID is required", nil)
			return
		}
		err = s.agentConsole.manager.CancelPending(input.ID)
	case path == agentConsoleAPIBase+"/preference":
		var input struct {
			Preset    AgentLaunchPreset `json:"preset"`
			Model     string            `json:"model"`
			Effort    string            `json:"effort"`
			Confirmed bool              `json:"confirmed"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		current := s.agentConsole.preferences.Load(s.agentConsole.cwd)
		if input.Preset == AgentLaunchFullAccess && current.LaunchPreset != AgentLaunchFullAccess && !input.Confirmed {
			writeEditorError(w, http.StatusConflict, "full_access_confirmation_required", "Full access requires confirmation for this repository", nil)
			return
		}
		err = s.agentConsole.preferences.Save(s.agentConsole.cwd, AgentPreferences{LaunchPreset: input.Preset, Model: input.Model, Effort: input.Effort})
	case path == agentConsoleAPIBase+"/skill":
		var input struct {
			Operation skillinstall.Operation `json:"operation"`
			Confirmed bool                   `json:"confirmed"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		if !input.Confirmed {
			writeEditorError(w, http.StatusConflict, "skill_confirmation_required", "Skill installation or update requires confirmation", nil)
			return
		}
		err = s.agentConsole.changeSkill(input.Operation)
	case path == agentConsoleAPIBase+"/verify":
		var input struct {
			TaskID    string `json:"taskID"`
			Confirmed bool   `json:"confirmed"`
		}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		if !input.Confirmed {
			writeEditorError(w, http.StatusConflict, "verification_confirmation_required", "Task verification requires confirmation", nil)
			return
		}
		if !workItemIDRE.MatchString(input.TaskID) {
			writeEditorError(w, http.StatusBadRequest, "invalid_task", "Invalid task ID", nil)
			return
		}
		snapshot, active := s.agentConsole.manager.Snapshot()
		if !active || snapshot.Settings.Launch.TaskID != input.TaskID {
			writeEditorError(w, http.StatusConflict, "verification_task_mismatch", "Verification requires the active session task", nil)
			return
		}
		report := s.verifyAgentTask(input.TaskID)
		writeChangesJSON(w, http.StatusOK, report)
		return
	case path == agentConsoleAPIBase+"/verification/send":
		err = s.sendVerificationFailure(r.Context())
	case path == agentConsoleAPIBase+"/terminal/start":
		var input struct{}
		if !decodeEditorJSON(w, r, &input) {
			return
		}
		err = s.agentConsole.startProjectTerminal()
	case path == agentConsoleAPIBase+"/terminal/stop":
		err = s.agentConsole.stopTerminal(r.Context())
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		var taskConflict *agentTaskConflict
		if errors.As(err, &taskConflict) {
			writeEditorError(w, http.StatusConflict, taskConflict.code, taskConflict.message, map[string]string{"command": taskConflict.command})
			return
		}
		var conflict *AgentStopConflict
		if errors.As(err, &conflict) {
			writeEditorError(w, http.StatusConflict, "pending_messages", err.Error(), map[string]int{"queued": conflict.Queued, "notSent": conflict.NotSent})
			return
		}
		writeEditorError(w, http.StatusConflict, "agent_action_failed", err.Error(), nil)
		return
	}
	s.agentConsole.publishState()
	writeChangesJSON(w, http.StatusOK, map[string]int{"schemaVersion": 1})
}

func (s *documentationServer) serveTaskActions(w http.ResponseWriter, r *http.Request) {
	if !s.taskActionsEnabled {
		http.NotFound(w, r)
		return
	}
	if !agentRequestOriginAllowed(r) {
		writeEditorError(w, http.StatusForbidden, "origin_forbidden", "Task actions require a loopback same-origin request", nil)
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/_toudocu/api/tasks/"), "/"), "/")
	if len(parts) < 2 || len(parts) > 3 || parts[1] != "actions" || !workItemIDRE.MatchString(parts[0]) {
		writeEditorError(w, http.StatusBadRequest, "invalid_task_action", "Invalid task action path", nil)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodGet {
		projection, err := s.resolveTaskActions(parts[0])
		if err != nil {
			writeEditorError(w, http.StatusNotFound, "task_not_found", err.Error(), nil)
			return
		}
		writeChangesJSON(w, http.StatusOK, projection)
		return
	}
	if len(parts) != 3 || r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		writeEditorError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
		return
	}
	if !requireEditorJSONAction(w, r, "task-action-execute") {
		return
	}
	var input struct {
		Delivery       string `json:"delivery"`
		ExpectedDigest string `json:"expectedDigest"`
		Input          struct {
			Text string `json:"text"`
		} `json:"input"`
	}
	if !decodeEditorJSON(w, r, &input) {
		return
	}
	preset := AgentLaunchDefault
	if s.agentConsole != nil {
		preset = s.agentConsole.preferences.Load(s.agentConsole.cwd).LaunchPreset
	}
	result, err := s.executeTaskAction(r.Context(), parts[0], parts[2], input.Delivery, input.ExpectedDigest, input.Input.Text, preset)
	if err != nil {
		var conflict *agentTaskConflict
		if errors.As(err, &conflict) {
			status := http.StatusConflict
			if conflict.code == "invalid_action" || conflict.code == "invalid_input" || conflict.code == "unavailable_delivery" && input.Delivery != "agent-console" {
				status = http.StatusBadRequest
			}
			writeEditorError(w, status, conflict.code, conflict.message, nil)
			return
		}
		writeEditorError(w, http.StatusConflict, "task_action_failed", err.Error(), nil)
		return
	}
	writeChangesJSON(w, http.StatusOK, result)
}

type agentWSInput struct {
	Action    string                `json:"action"`
	Text      string                `json:"text,omitempty"`
	Policy    AgentTurnPolicy       `json:"policy,omitempty"`
	RequestID string                `json:"requestID,omitempty"`
	Decision  AgentApprovalDecision `json:"decision,omitempty"`
	Columns   int                   `json:"columns,omitempty"`
	Rows      int                   `json:"rows,omitempty"`
}

func (c *agentConsole) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || !agentRequestOriginAllowed(r) || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") || r.Header.Get("Sec-WebSocket-Version") != "13" {
		writeEditorError(w, http.StatusForbidden, "websocket_forbidden", "A same-origin WebSocket is required", nil)
		return
	}
	key := strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key"))
	if decoded, err := base64.StdEncoding.DecodeString(key); err != nil || len(decoded) != 16 {
		writeEditorError(w, http.StatusBadRequest, "invalid_websocket_key", "Invalid WebSocket key", nil)
		return
	}
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "WebSocket unsupported", http.StatusInternalServerError)
		return
	}
	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return
	}
	if !c.registerConnection(conn) {
		return
	}
	defer func() { c.mu.Lock(); delete(c.connections, conn); c.mu.Unlock(); _ = conn.Close() }()
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	_, _ = rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n")
	if rw.Flush() != nil {
		return
	}
	since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
	backlog, messages, cancel := c.subscribe(since)
	defer cancel()
	writer := &webSocketWriter{writer: rw}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, message := range backlog {
			if writer.writeJSON(message) != nil {
				return
			}
		}
		c.publishState()
		for message := range messages {
			if writer.writeJSON(message) != nil {
				return
			}
		}
	}()
	var fragmented []byte
	fragmenting := false
	for {
		fin, opcode, payload, frameErr := readWebSocketFrame(rw.Reader)
		if frameErr != nil {
			return
		}
		switch opcode {
		case 8:
			if !fin {
				return
			}
			_ = writer.writeFrame(8, payload)
			return
		case 9:
			if !fin {
				return
			}
			if writer.writeFrame(10, payload) != nil {
				return
			}
			continue
		case 1:
			if fragmenting {
				return
			}
			fragmented = append(fragmented, payload...)
			fragmenting = !fin
		case 0:
			if !fragmenting {
				return
			}
			fragmented = append(fragmented, payload...)
			fragmenting = !fin
		default:
			continue
		}
		if len(fragmented) > agentMessageLimit {
			return
		}
		if !fin {
			continue
		}
		payload, fragmented = fragmented, nil
		var input agentWSInput
		decoder := json.NewDecoder(strings.NewReader(string(payload)))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			continue
		}
		switch input.Action {
		case "terminal-input":
			err = c.terminal.Write(input.Text)
		case "terminal-resize":
			err = c.terminal.Resize(input.Columns, input.Rows)
		case "terminal-interrupt":
			err = c.terminal.Interrupt()
		case "message":
			if input.Policy == "" {
				input.Policy = AgentTurnNormal
			}
			if input.Text == "" || input.Policy != AgentTurnNormal && input.Policy != AgentTurnReadOnly {
				err = errors.New("invalid agent message")
			} else {
				err = c.manager.Send(context.Background(), input.Text, input.Policy)
			}
		case "interrupt":
			err = c.manager.StopResponse(context.Background())
		case "approval":
			if input.RequestID == "" || input.Decision != AgentApprovalAccept && input.Decision != AgentApprovalDecline && input.Decision != AgentApprovalCancel {
				err = errors.New("invalid approval response")
			} else {
				err = c.manager.Approve(context.Background(), input.RequestID, input.Decision)
			}
		default:
			err = errors.New("unsupported agent action")
		}
		if err == nil {
			c.publishState()
		} else {
			event := agentConsoleEvent{Type: AgentEventError, Text: boundedAgentText(err.Error(), agentMessageLimit)}
			c.publish(agentConsoleMessage{Kind: "event", Event: &event})
		}
		select {
		case <-done:
			return
		default:
		}
	}
}

func agentRequestOriginAllowed(r *http.Request) bool {
	host := r.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.Trim(host, "[]")
	return (strings.EqualFold(host, "localhost") || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()) && editorOriginAllowed(r)
}

type webSocketWriter struct {
	mu     sync.Mutex
	writer *bufio.ReadWriter
}

func (w *webSocketWriter) writeJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return w.writeFrame(1, payload)
}

func (w *webSocketWriter) writeFrame(opcode byte, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	header := []byte{0x80 | opcode}
	if len(payload) < 126 {
		header = append(header, byte(len(payload)))
	} else if len(payload) <= 65535 {
		header = append(header, 126, byte(len(payload)>>8), byte(len(payload)))
	} else {
		header = append(header, 127, 0, 0, 0, 0, byte(uint64(len(payload))>>24), byte(uint64(len(payload))>>16), byte(uint64(len(payload))>>8), byte(len(payload)))
	}
	if _, err := w.writer.Write(header); err != nil {
		return err
	}
	if _, err := w.writer.Write(payload); err != nil {
		return err
	}
	return w.writer.Flush()
}

func readWebSocketFrame(reader *bufio.Reader) (bool, byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return false, 0, nil, err
	}
	opcode := header[0] & 0x0f
	if header[0]&0x70 != 0 || header[1]&0x80 == 0 {
		return false, 0, nil, errors.New("invalid websocket frame")
	}
	if opcode >= 8 && header[0]&0x80 == 0 {
		return false, 0, nil, errors.New("fragmented websocket control frame")
	}
	length := uint64(header[1] & 0x7f)
	if length == 126 {
		var size uint16
		if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
			return false, 0, nil, err
		}
		length = uint64(size)
	}
	if length == 127 {
		if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
			return false, 0, nil, err
		}
	}
	if length > agentMessageLimit || opcode >= 8 && length > 125 {
		return false, 0, nil, errors.New("websocket frame too large")
	}
	mask := make([]byte, 4)
	if _, err := io.ReadFull(reader, mask); err != nil {
		return false, 0, nil, err
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return false, 0, nil, err
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return header[0]&0x80 != 0, opcode, payload, nil
}
