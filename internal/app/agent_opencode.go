package toudocu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type structuredAgentProvider struct{ providers map[string]AgentProvider }

func newStructuredAgentProvider() AgentProvider {
	providers := map[string]AgentProvider{"codex": lazyCodexProvider{}}
	if provider, err := NewOpenCodeProvider(); err == nil {
		providers[provider.Name()] = provider
	}
	return &structuredAgentProvider{providers: providers}
}

func (p *structuredAgentProvider) Name() string { return "codex" }
func (p *structuredAgentProvider) Names() []string {
	names := []string{"codex"}
	if _, ok := p.providers["opencode"]; ok {
		names = append(names, "opencode")
	}
	return names
}
func (p *structuredAgentProvider) Capabilities() AgentCapabilities {
	return p.providers[p.Name()].Capabilities()
}
func (p *structuredAgentProvider) ModelsFor(ctx context.Context, cwd, name string) ([]AgentModel, error) {
	provider, ok := p.providers[name].(AgentModelProvider)
	if !ok {
		return nil, nil
	}
	return provider.Models(ctx, cwd)
}
func (p *structuredAgentProvider) Start(ctx context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	provider := p.providers[launch.Provider]
	if provider == nil {
		return nil, fmt.Errorf("agent provider %q is not available", launch.Provider)
	}
	return provider.Start(ctx, launch)
}

type OpenCodeProvider struct {
	executable string
	args       []string
	client     *http.Client
	baseURL    string
}

func NewOpenCodeProvider() (*OpenCodeProvider, error) {
	executable, err := exec.LookPath("opencode")
	if err != nil {
		return nil, fmt.Errorf("find opencode: %w", err)
	}
	return &OpenCodeProvider{executable: executable, client: &http.Client{}}, nil
}
func (p *OpenCodeProvider) Name() string { return "opencode" }
func (p *OpenCodeProvider) Capabilities() AgentCapabilities {
	return AgentCapabilities{Interrupt: true}
}
func (p *OpenCodeProvider) Models(ctx context.Context, cwd string) ([]AgentModel, error) {
	cmd := exec.CommandContext(ctx, p.executable, append(p.args, "models")...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list opencode models: %w", err)
	}
	return parseOpenCodeModels(string(output)), nil
}

func parseOpenCodeModels(output string) []AgentModel {
	models := make([]AgentModel, 0)
	seen := map[string]bool{}
	for _, id := range strings.Fields(output) {
		if id != "" && !seen[id] {
			seen[id] = true
			models = append(models, AgentModel{ID: id, DisplayName: id, SupportedReasoningEfforts: []AgentReasoningEffort{}})
		}
	}
	return models
}
func (p *OpenCodeProvider) Start(ctx context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	abs, err := filepath.Abs(launch.CWD)
	if err != nil {
		return nil, err
	}
	s := &openCodeSession{events: make(chan AgentEvent, 64), done: make(chan struct{}), cwd: abs, settings: AgentSettings{Launch: launch, Capabilities: p.Capabilities()}, client: p.client}
	s.streamContext, s.cancelStream = context.WithCancel(context.Background())
	if s.client == nil {
		s.client = &http.Client{}
	}
	if p.baseURL == "" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, err
		}
		port := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()
		args := append(append([]string{}, p.args...), "serve", "--hostname", "127.0.0.1", "--port", strconv.Itoa(port))
		cmd := exec.Command(p.executable, args...)
		configureProcessTree(cmd)
		cmd.Cancel = nil
		cmd.Dir = abs
		if launch.Preset == AgentLaunchFullAccess {
			cmd.Env = append(os.Environ(), `OPENCODE_PERMISSION={"*":"allow"}`)
		}
		stderr, _ := cmd.StderrPipe()
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start opencode serve: %w", err)
		}
		s.cmd = cmd
		s.baseURL = "http://127.0.0.1:" + strconv.Itoa(port)
		go io.Copy(io.Discard, stderr)
		go func() { err := cmd.Wait(); s.finish(err) }()
		if err := s.waitReady(ctx); err != nil {
			_ = s.Stop(context.Background())
			return nil, err
		}
	} else {
		s.baseURL = strings.TrimRight(p.baseURL, "/")
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := s.request(ctx, http.MethodPost, "/session", map[string]any{}, &created); err != nil || created.ID == "" {
		_ = s.Stop(context.Background())
		if err == nil {
			err = errors.New("opencode session create returned no id")
		}
		return nil, err
	}
	s.sessionID = created.ID
	s.settings.Launch.Provider = p.Name()
	s.emit(AgentEvent{Type: AgentEventSessionStarted, ThreadID: created.ID})
	go s.subscribe()
	return s, nil
}

type openCodeSession struct {
	baseURL, cwd, sessionID string
	client                  *http.Client
	cmd                     *exec.Cmd
	settings                AgentSettings
	events                  chan AgentEvent
	done                    chan struct{}
	stopOnce                sync.Once
	turn                    atomic.Uint64
	streamContext           context.Context
	cancelStream            context.CancelFunc
}

func (s *openCodeSession) Settings() AgentSettings   { return s.settings }
func (s *openCodeSession) Events() <-chan AgentEvent { return s.events }
func (s *openCodeSession) waitReady(ctx context.Context) error {
	for {
		var sessions json.RawMessage
		if s.request(ctx, http.MethodGet, "/session", nil, &sessions) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.done:
			return errors.New("opencode serve exited before ready")
		case <-time.After(25 * time.Millisecond):
		}
	}
}
func (s *openCodeSession) StartTurn(ctx context.Context, prompt string, policy AgentTurnPolicy) (string, error) {
	if policy != AgentTurnNormal {
		return "", errors.New("opencode provider does not support read-only turns")
	}
	id := fmt.Sprintf("%s-%d", s.sessionID, s.turn.Add(1))
	body := map[string]any{"parts": []map[string]string{{"type": "text", "text": prompt}}}
	if model := s.settings.Launch.Model; model != "" {
		providerID, modelID, ok := strings.Cut(model, "/")
		if !ok || providerID == "" || modelID == "" {
			return "", fmt.Errorf("invalid opencode model %q", model)
		}
		body["model"] = map[string]string{"providerID": providerID, "modelID": modelID}
	}
	if err := s.request(ctx, http.MethodPost, "/session/"+url.PathEscape(s.sessionID)+"/prompt_async", body, nil); err != nil {
		return "", err
	}
	s.emit(AgentEvent{Type: AgentEventTurnStarted, ThreadID: s.sessionID, TurnID: id})
	return id, nil
}
func (s *openCodeSession) Steer(context.Context, string, string) error {
	return errors.New("opencode provider does not support steering")
}
func (s *openCodeSession) Interrupt(ctx context.Context, _ string) error {
	return s.request(ctx, http.MethodPost, "/session/"+url.PathEscape(s.sessionID)+"/abort", nil, nil)
}
func (s *openCodeSession) Approve(context.Context, string, AgentApprovalDecision) error {
	return errors.New("opencode provider does not support approvals")
}
func (s *openCodeSession) Stop(ctx context.Context) error {
	if s.sessionID != "" {
		_ = s.Interrupt(ctx, "")
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = killProcessTree(s.cmd)
	}
	s.finish(nil)
	return nil
}
func (s *openCodeSession) finish(_ error) {
	s.stopOnce.Do(func() { s.cancelStream(); close(s.done); close(s.events) })
}
func (s *openCodeSession) emit(event AgentEvent) {
	select {
	case s.events <- event:
	case <-s.done:
	}
}
func (s *openCodeSession) request(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path+"?directory="+url.QueryEscape(s.cwd), body)
	if err != nil {
		return err
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("opencode %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if output == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(output)
}
func (s *openCodeSession) subscribe() {
	for {
		select {
		case <-s.done:
			return
		default:
		}
		req, _ := http.NewRequestWithContext(s.streamContext, http.MethodGet, s.baseURL+"/event?directory="+url.QueryEscape(s.cwd), nil)
		resp, err := s.client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			s.readEvents(resp.Body)
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
		select {
		case <-s.done:
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}
func (s *openCodeSession) readEvents(reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), agentMessageLimit)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event struct {
			Type       string          `json:"type"`
			Properties json.RawMessage `json:"properties"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event) == nil {
			s.normalize(event.Type, event.Properties)
		}
	}
}
func (s *openCodeSession) normalize(kind string, raw json.RawMessage) {
	var p struct {
		SessionID, MessageID, PartID, Delta, Path string
		Status                                    struct {
			Type string `json:"type"`
		} `json:"status"`
		Error any `json:"error"`
		Part  struct {
			ID, Type, Text, Tool, CallID string
			State                        struct {
				Status, Output, Error, Title string
				Input                        map[string]any
				Time                         struct{ Start, End int64 }
			}
		} `json:"part"`
	}
	if json.Unmarshal(raw, &p) != nil || p.SessionID != "" && p.SessionID != s.sessionID {
		return
	}
	switch kind {
	case "message.part.delta":
		if p.Delta != "" {
			s.emit(AgentEvent{Type: AgentEventMessageDelta, ThreadID: s.sessionID, ItemID: p.PartID, Text: p.Delta})
		}
	case "message.part.updated":
		part := p.Part
		if part.Type == "text" && part.Text != "" {
			s.emit(AgentEvent{Type: AgentEventMessageDelta, ThreadID: s.sessionID, ItemID: part.ID, Text: part.Text})
		}
		if part.Type == "tool" {
			command, _ := part.State.Input["command"].(string)
			item := part.CallID
			if item == "" {
				item = part.ID
			}
			switch part.State.Status {
			case "pending", "running":
				s.emit(AgentEvent{Type: AgentEventCommandStarted, ThreadID: s.sessionID, ItemID: item, Command: command, CWD: s.cwd, Status: part.State.Status})
			case "completed", "error":
				s.emit(AgentEvent{Type: AgentEventCommandStarted, ThreadID: s.sessionID, ItemID: item, Command: command, CWD: s.cwd, Status: "running"})
				text := part.State.Output
				if text == "" {
					text = part.State.Error
				}
				if text != "" {
					s.emit(AgentEvent{Type: AgentEventCommandOutput, ThreadID: s.sessionID, ItemID: item, Text: text})
				}
				s.emit(AgentEvent{Type: AgentEventCommandFinished, ThreadID: s.sessionID, ItemID: item, Command: command, CWD: s.cwd, Status: part.State.Status, DurationMillis: part.State.Time.End - part.State.Time.Start})
			}
		}
	case "file.edited", "file.watcher.updated":
		s.emit(AgentEvent{Type: AgentEventFilesChanged, ThreadID: s.sessionID, Text: p.Path})
	case "session.idle":
		s.emit(AgentEvent{Type: AgentEventTurnCompleted, ThreadID: s.sessionID, Status: "completed"})
	case "session.status":
		if p.Status.Type == "idle" {
			s.emit(AgentEvent{Type: AgentEventTurnCompleted, ThreadID: s.sessionID, Status: "completed"})
		}
	case "session.error":
		data, _ := json.Marshal(p.Error)
		s.emit(AgentEvent{Type: AgentEventError, ThreadID: s.sessionID, Text: string(data)})
	}
}
