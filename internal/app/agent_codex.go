package toudocu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

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
	return AgentCapabilities{Steering: true, Interrupt: true, Approvals: true}
}

func (p *CodexProvider) Start(ctx context.Context, cwd string, preferences AgentPreferences) (AgentSession, error) {
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolve agent cwd: %w", err)
	}
	cmd := exec.CommandContext(ctx, p.executable, p.args...)
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
	go io.Copy(io.Discard, stderr) // app-server protocol is stdout-only.

	session := &codexSession{
		cmd: cmd, stdin: stdin, decoder: json.NewDecoder(stdout),
		events: make(chan AgentEvent, 64), pending: make(map[string]chan rpcMessage),
		done:     make(chan struct{}),
		settings: AgentSettings{Provider: p.Name(), AccessPreset: preferences.Access, EffectiveAccess: preferences.Access, Capabilities: p.Capabilities()},
	}
	go session.read()
	if _, err := session.request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]string{"name": "toudocu", "title": "Toudocu", "version": Version},
	}); err != nil {
		session.Stop()
		return nil, fmt.Errorf("initialize codex app-server: %w", err)
	}
	if err := session.notify("initialized", map[string]any{}); err != nil {
		session.Stop()
		return nil, err
	}
	params := map[string]any{"cwd": absoluteCWD, "ephemeral": true}
	if sandbox := codexSandbox(preferences.Access); sandbox != "" {
		params["sandbox"] = sandbox
	}
	result, err := session.request(ctx, "thread/start", params)
	if err != nil {
		session.Stop()
		return nil, fmt.Errorf("start codex thread: %w", err)
	}
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(result, &started); err != nil || started.Thread.ID == "" {
		session.Stop()
		return nil, errors.New("codex thread/start returned no thread id")
	}
	session.threadID = started.Thread.ID
	session.emit(AgentEvent{Type: AgentEventSessionStarted, ThreadID: session.threadID})
	return session, nil
}

func codexSandbox(access AgentAccessPreset) string {
	switch access {
	case AgentAccessReadOnly:
		return "read-only"
	case AgentAccessWorkspace:
		return "workspace-write"
	case AgentAccessFull:
		return "danger-full-access"
	default:
		return ""
	}
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
	done       chan struct{}

	mu      sync.Mutex
	nextID  int64
	pending map[string]chan rpcMessage
	closed  bool
}

func (s *codexSession) Events() <-chan AgentEvent { return s.events }
func (s *codexSession) Settings() AgentSettings   { return s.settings }

func (s *codexSession) StartTurn(ctx context.Context, prompt string) (string, error) {
	result, err := s.request(ctx, "turn/start", map[string]any{
		"threadId": s.threadID,
		"input":    []map[string]string{{"type": "text", "text": prompt}},
	})
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

func (s *codexSession) Stop() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	_ = s.stdin.Close()
	s.mu.Unlock()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait()
	<-s.done
	close(s.events)
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
	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		return nil, ctx.Err()
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

func (s *codexSession) read() {
	defer close(s.done)
	for {
		var message rpcMessage
		if err := s.decoder.Decode(&message); err != nil {
			if !errors.Is(err, io.EOF) {
				s.emit(AgentEvent{Type: AgentEventError, Text: err.Error()})
			}
			return
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

func (s *codexSession) normalize(message rpcMessage) {
	var params struct {
		ThreadID string `json:"threadId"`
		TurnID   string `json:"turnId"`
		ItemID   string `json:"itemId"`
		Delta    string `json:"delta"`
		Diff     string `json:"diff"`
		Reason   string `json:"reason"`
		Turn     struct {
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
			} else {
				event.Type, event.Text = AgentEventCommandFinished, params.Item.AggregatedOutput
			}
		case "fileChange":
			event.Type = AgentEventFilesChanged
		default:
			return
		}
	case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
		event.Type = AgentEventApproval
		event.Approval = &AgentApproval{RequestID: string(message.ID), Kind: message.Method, Reason: params.Reason}
	default:
		return
	}
	s.emit(event)
}

func (s *codexSession) emit(event AgentEvent) {
	defer func() { _ = recover() }()
	s.events <- event
}
