package toudocu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const agentStderrLimit = 64 << 10

type CodexProvider struct {
	executable string
	args       []string
}

func NewCodexProvider() (*CodexProvider, error) {
	executable, err := exec.LookPath("codex")
	if err != nil {
		return nil, fmt.Errorf("find codex: %w", err)
	}
	return &CodexProvider{executable: executable, args: []string{"app-server", "--stdio"}}, nil
}

func (p *CodexProvider) Name() string { return "codex" }

func (p *CodexProvider) Capabilities() AgentCapabilities {
	return AgentCapabilities{Steering: true, Interrupt: true, Approvals: true, ReadOnlyTurns: true}
}

func (p *CodexProvider) connect(ctx context.Context, cwd string) (*codexSession, error) {
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve agent cwd: %w", err)
	}
	cmd := exec.Command(p.executable, p.args...)
	configureProcessTree(cmd)
	cmd.Cancel = nil
	cmd.Dir = absoluteCWD
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}
	stderrBuffer := newTailBuffer(agentStderrLimit)
	stderrDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(stderrBuffer, stderr)
		stderrDone <- copyErr
	}()

	session := &codexSession{
		cmd: cmd, stdin: stdin, decoder: json.NewDecoder(stdout),
		events: make(chan AgentEvent, 64), pending: make(map[string]chan rpcMessage),
		waitDone: make(chan struct{}),
		stderr:   stderrBuffer,
	}
	go session.run(stderrDone)
	if _, err := session.request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]string{"name": "toudocu", "title": "Toudocu", "version": Version},
	}); err != nil {
		_ = session.Stop(context.Background())
		return nil, fmt.Errorf("initialize codex app-server: %w", err)
	}
	if err := session.notify("initialized", map[string]any{}); err != nil {
		_ = session.Stop(context.Background())
		return nil, err
	}
	return session, nil
}

func (p *CodexProvider) Models(ctx context.Context, cwd string) ([]AgentModel, error) {
	session, err := p.connect(ctx, cwd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Stop(context.Background()) }()
	result, err := session.request(ctx, "model/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []AgentModel `json:"data"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (p *CodexProvider) Threads(ctx context.Context, cwd string) ([]AgentThread, error) {
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve agent cwd: %w", err)
	}
	session, err := p.connect(ctx, absoluteCWD)
	if err != nil {
		return nil, err
	}
	defer func() { _ = session.Stop(context.Background()) }()
	return session.listThreads(ctx, absoluteCWD)
}

func (s *codexSession) listThreads(ctx context.Context, cwd string) ([]AgentThread, error) {
	// ponytail: first page only; add cursor pagination when 50 recent runs stop being enough.
	result, err := s.request(ctx, "thread/list", map[string]any{
		"cwd": cwd, "limit": 50, "sortKey": "updated_at", "sortDirection": "desc",
		"sourceKinds": []string{"appServer", "cli", "vscode"},
	})
	if err != nil {
		return nil, err
	}
	var response struct {
		Data []AgentThread `json:"data"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (p *CodexProvider) Resume(ctx context.Context, launch AgentLaunch, threadID string) (AgentProviderSession, error) {
	absoluteCWD, err := filepath.Abs(launch.CWD)
	if err != nil {
		return nil, fmt.Errorf("resolve agent cwd: %w", err)
	}
	session, err := p.connect(ctx, absoluteCWD)
	if err != nil {
		return nil, err
	}
	result, err := session.request(ctx, "thread/read", map[string]any{"threadId": threadID, "includeTurns": false})
	if err != nil {
		_ = session.Stop(context.Background())
		return nil, err
	}
	var stored struct {
		Thread struct {
			CWD string `json:"cwd"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &stored); err != nil || stored.Thread.CWD != absoluteCWD {
		_ = session.Stop(context.Background())
		return nil, errors.New("codex thread is not available for this repository")
	}
	p.configureSession(session, launch, absoluteCWD)
	params := map[string]any{"threadId": threadID, "cwd": absoluteCWD}
	if launch.Preset == AgentLaunchFullAccess {
		params["sandbox"] = "danger-full-access"
		params["approvalPolicy"] = "never"
	}
	result, err = session.request(ctx, "thread/resume", params)
	if err != nil {
		_ = session.Stop(context.Background())
		return nil, fmt.Errorf("resume codex thread: %w", err)
	}
	var resumed struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &resumed); err != nil || resumed.Thread.ID != threadID {
		_ = session.Stop(context.Background())
		return nil, errors.New("codex thread/resume returned an unexpected thread id")
	}
	session.mu.Lock()
	session.threadID = threadID
	session.mu.Unlock()
	session.emit(AgentEvent{Type: AgentEventSessionStarted, ThreadID: threadID})
	return session, nil
}

func (p *CodexProvider) Start(ctx context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	absoluteCWD, err := filepath.Abs(launch.CWD)
	if err != nil {
		return nil, fmt.Errorf("resolve agent cwd: %w", err)
	}
	session, err := p.connect(ctx, absoluteCWD)
	if err != nil {
		return nil, err
	}
	p.configureSession(session, launch, absoluteCWD)
	params := map[string]any{"cwd": absoluteCWD, "ephemeral": false}
	if launch.Preset == AgentLaunchFullAccess {
		params["sandbox"] = "danger-full-access"
		params["approvalPolicy"] = "never"
	}
	result, err := session.request(ctx, "thread/start", params)
	if err != nil {
		_ = session.Stop(context.Background())
		return nil, fmt.Errorf("start codex thread: %w", err)
	}
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &started); err != nil || started.Thread.ID == "" {
		_ = session.Stop(context.Background())
		return nil, errors.New("codex thread/start returned no thread id")
	}
	session.mu.Lock()
	session.threadID = started.Thread.ID
	session.mu.Unlock()
	session.emit(AgentEvent{Type: AgentEventSessionStarted, ThreadID: session.threadID})
	return session, nil
}

func (p *CodexProvider) configureSession(session *codexSession, launch AgentLaunch, cwd string) {
	session.settings = AgentSettings{Launch: launch, Capabilities: p.Capabilities()}
	session.settings.Launch.CWD = cwd
	if launch.Preset == AgentLaunchFullAccess {
		session.baseline = &codexSandboxPolicy{Type: "dangerFullAccess"}
		session.settings.EffectiveAccess = EffectiveAccessSummary{Known: true, Unrestricted: true}
	} else {
		session.settings.Capabilities.ReadOnlyTurns = false
	}
}

type codexSandboxPolicy struct {
	Type string `json:"type"`
}

type rpcMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type codexSession struct {
	cmd        *exec.Cmd
	stdin      io.WriteCloser
	decoder    *json.Decoder
	events     chan AgentEvent
	threadID   string
	activeTurn string
	settings   AgentSettings

	mu             sync.Mutex
	eventMu        sync.RWMutex
	stopMu         sync.Mutex
	nextID         int64
	pending        map[string]chan rpcMessage
	closed         bool
	stopped        bool
	waitDone       chan struct{}
	waitErr        error
	readErr        error
	stderrErr      error
	stderr         *tailBuffer
	baseline       *codexSandboxPolicy
	readOnlyTurn   bool
	commandStarted map[string]int64
}

func (s *codexSession) Events() <-chan AgentEvent { return s.events }
func (s *codexSession) Settings() AgentSettings   { return s.settings }

func (s *codexSession) StartTurn(ctx context.Context, prompt string, policy AgentTurnPolicy) (string, error) {
	params := map[string]any{
		"threadId": s.threadID,
		"input":    []map[string]string{{"type": "text", "text": prompt}},
	}
	if s.settings.Launch.Model != "" {
		params["model"] = s.settings.Launch.Model
	}
	if s.settings.Launch.Effort != "" {
		params["effort"] = s.settings.Launch.Effort
	}
	if policy == AgentTurnReadOnly {
		if s.baseline == nil {
			return "", errors.New("read-only turns require a restorable baseline sandbox")
		}
		params["sandboxPolicy"] = codexSandboxPolicy{Type: "readOnly"}
	} else if s.readOnlyTurn {
		params["sandboxPolicy"] = *s.baseline
	}
	result, err := s.request(ctx, "turn/start", params)
	if err != nil {
		return "", err
	}
	var response struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(result, &response); err != nil || response.Turn.ID == "" {
		return "", errors.New("codex turn/start returned no turn id")
	}
	s.mu.Lock()
	s.activeTurn = response.Turn.ID
	s.readOnlyTurn = policy == AgentTurnReadOnly
	s.mu.Unlock()
	return response.Turn.ID, nil
}

func (s *codexSession) Steer(ctx context.Context, turnID, prompt string) error {
	_, err := s.request(ctx, "turn/steer", map[string]any{
		"threadId": s.threadID, "expectedTurnId": turnID,
		"input": []map[string]string{{"type": "text", "text": prompt}},
	})
	return err
}

func (s *codexSession) Interrupt(ctx context.Context, turnID string) error {
	_, err := s.request(ctx, "turn/interrupt", map[string]string{"threadId": s.threadID, "turnId": turnID})
	return err
}

func (s *codexSession) Approve(_ context.Context, requestID string, decision AgentApprovalDecision) error {
	if decision != AgentApprovalAccept && decision != AgentApprovalDecline && decision != AgentApprovalCancel {
		return fmt.Errorf("unsupported approval decision %q", decision)
	}
	return s.write(map[string]any{"id": json.RawMessage(requestID), "result": map[string]any{"decision": decision}})
}

func (s *codexSession) Stop(ctx context.Context) error {
	s.stopMu.Lock()
	defer s.stopMu.Unlock()
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	if !s.closed {
		s.closed = true
		_ = s.stdin.Close()
	}
	s.mu.Unlock()
	select {
	case <-s.waitDone:
	case <-time.After(500 * time.Millisecond):
		if err := killProcessTree(s.cmd); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return err
		}
		select {
		case <-s.waitDone:
		case <-ctx.Done():
			return ErrAgentStopUnconfirmed
		}
	}
	s.mu.Lock()
	s.stopped = true
	s.mu.Unlock()
	return nil
}

func (s *codexSession) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	s.mu.Lock()
	s.nextID++
	numericID := s.nextID
	id := strconv.FormatInt(numericID, 10)
	response := make(chan rpcMessage, 1)
	s.pending[id] = response
	s.mu.Unlock()
	if err := s.write(map[string]any{"id": numericID, "method": method, "params": params}); err != nil {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, err
	}
	select {
	case message := <-response:
		if message.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, message.Error.Message)
		}
		return message.Result, nil
	case <-s.waitDone:
		return nil, s.exitError()
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, fmt.Errorf("%w: %v", ErrAgentDeliveryUncertain, ctx.Err())
	}
}

func (s *codexSession) notify(method string, params any) error {
	return s.write(map[string]any{"method": method, "params": params})
}

func (s *codexSession) write(value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("agent session is closed")
	}
	return json.NewEncoder(s.stdin).Encode(value)
}

func (s *codexSession) run(stderrDone <-chan error) {
	s.readErr = s.read()
	s.stderrErr = <-stderrDone
	s.waitErr = s.cmd.Wait()
	err := s.exitError()
	s.mu.Lock()
	stopping := s.closed
	s.mu.Unlock()
	if !stopping {
		s.emit(AgentEvent{Type: AgentEventError, Text: err.Error()})
	}
	s.eventMu.Lock()
	close(s.events)
	s.eventMu.Unlock()
	close(s.waitDone)
}

func (s *codexSession) read() error {
	for {
		var message rpcMessage
		if err := s.decoder.Decode(&message); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if len(message.ID) > 0 && message.Method == "" {
			id := string(message.ID)
			s.mu.Lock()
			response := s.pending[id]
			delete(s.pending, id)
			s.mu.Unlock()
			if response != nil {
				response <- message
			}
			continue
		}
		s.normalize(message)
	}
}

func (s *codexSession) exitError() error {
	detail, truncated := s.stderr.snapshot()
	detail = strings.TrimSpace(detail)
	if truncated {
		detail = "[stderr truncated]\n" + detail
	}
	phase := ""
	s.mu.Lock()
	starting := s.threadID == ""
	s.mu.Unlock()
	if starting {
		phase = " during startup"
	}
	reason := "unexpectedly"
	if s.waitErr != nil {
		reason = s.waitErr.Error()
	} else if s.readErr != nil {
		reason = "stdout: " + s.readErr.Error()
	} else if s.stderrErr != nil {
		reason = "stderr: " + s.stderrErr.Error()
	}
	message := "Codex app-server exited" + phase + ": " + reason
	if detail != "" {
		message += "\n\n" + detail
	}
	return errors.New(message)
}

func (s *codexSession) normalize(message rpcMessage) {
	var params struct {
		ThreadID      string `json:"threadId"`
		TurnID        string `json:"turnId"`
		ItemID        string `json:"itemId"`
		Delta         string `json:"delta"`
		Diff          string `json:"diff"`
		Reason        string `json:"reason"`
		StartedAtMs   int64  `json:"startedAtMs"`
		CompletedAtMs int64  `json:"completedAtMs"`
		Turn          struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		} `json:"turn"`
		Item struct {
			ID               string `json:"id"`
			Type             string `json:"type"`
			Command          string `json:"command"`
			CWD              string `json:"cwd"`
			Status           string `json:"status"`
			AggregatedOutput string `json:"aggregatedOutput"`
			ExitCode         *int   `json:"exitCode"`
		} `json:"item"`
	}
	_ = json.Unmarshal(message.Params, &params)
	event := AgentEvent{ThreadID: params.ThreadID, TurnID: params.TurnID, ItemID: params.ItemID}
	switch message.Method {
	case "turn/started":
		event.Type, event.TurnID, event.Status = AgentEventTurnStarted, params.Turn.ID, params.Turn.Status
	case "turn/completed":
		event.Type, event.TurnID, event.Status = AgentEventTurnCompleted, params.Turn.ID, params.Turn.Status
		if params.Turn.Error != nil {
			event.Text = params.Turn.Error.Message
		}
	case "item/agentMessage/delta":
		event.Type, event.Text = AgentEventMessageDelta, params.Delta
	case "item/commandExecution/outputDelta":
		event.Type, event.Text = AgentEventCommandOutput, params.Delta
	case "turn/diff/updated":
		event.Type, event.Text = AgentEventFilesChanged, params.Diff
	case "item/started", "item/completed":
		event.ItemID, event.Command, event.CWD, event.Status, event.ExitCode = params.Item.ID, params.Item.Command, params.Item.CWD, params.Item.Status, params.Item.ExitCode
		switch params.Item.Type {
		case "commandExecution":
			if message.Method == "item/started" {
				event.Type = AgentEventCommandStarted
				s.mu.Lock()
				if s.commandStarted == nil {
					s.commandStarted = map[string]int64{}
				}
				s.commandStarted[event.ItemID] = params.StartedAtMs
				s.mu.Unlock()
			} else {
				event.Type, event.Text = AgentEventCommandFinished, params.Item.AggregatedOutput
				s.mu.Lock()
				started := s.commandStarted[event.ItemID]
				delete(s.commandStarted, event.ItemID)
				s.mu.Unlock()
				if params.CompletedAtMs >= started && started > 0 {
					event.DurationMillis = params.CompletedAtMs - started
				}
			}
		case "fileChange":
			event.Type = AgentEventFilesChanged
		default:
			return
		}
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		event.Type = AgentEventApproval
		event.ApprovalState = "pending"
		event.Approval = &AgentApproval{RequestID: string(message.ID), Kind: message.Method, Reason: params.Reason}
	default:
		return
	}
	s.emit(event)
}

func (s *codexSession) emit(event AgentEvent) {
	s.eventMu.RLock()
	defer s.eventMu.RUnlock()
	defer func() { _ = recover() }()
	s.events <- event
}
