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
	"sort"
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
func (p *structuredAgentProvider) Threads(ctx context.Context, cwd string) ([]AgentThread, error) {
	provider, ok := p.providers[p.Name()].(AgentHistoryProvider)
	if !ok {
		return nil, errors.New("agent provider does not support history")
	}
	return provider.Threads(ctx, cwd)
}
func (p *structuredAgentProvider) Resume(ctx context.Context, launch AgentLaunch, threadID string) (AgentProviderSession, error) {
	if launch.Provider == "" {
		launch.Provider = p.Name()
	}
	provider, ok := p.providers[launch.Provider].(AgentHistoryProvider)
	if !ok {
		return nil, errors.New("agent provider does not support history")
	}
	return provider.Resume(ctx, launch, threadID)
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
	cmd := exec.CommandContext(ctx, p.executable, append(p.args, "models", "--verbose")...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list opencode models: %w", err)
	}
	return parseOpenCodeModels(string(output))
}

func parseOpenCodeModels(output string) ([]AgentModel, error) {
	type metadata struct {
		ID           string `json:"id"`
		ProviderID   string `json:"providerID"`
		Name         string `json:"name"`
		Capabilities struct {
			Reasoning bool `json:"reasoning"`
		} `json:"capabilities"`
		Options struct {
			ReasoningEffort string `json:"reasoningEffort"`
		} `json:"options"`
		Variants map[string]json.RawMessage `json:"variants"`
	}

	models := make([]AgentModel, 0)
	for remaining := strings.TrimSpace(output); remaining != ""; {
		lineEnd := strings.IndexByte(remaining, '\n')
		if lineEnd < 0 {
			return nil, errors.New("invalid verbose opencode model output")
		}
		modelID := strings.TrimSpace(remaining[:lineEnd])
		remaining = strings.TrimSpace(remaining[lineEnd+1:])
		decoder := json.NewDecoder(strings.NewReader(remaining))
		var item metadata
		if err := decoder.Decode(&item); err != nil {
			return nil, fmt.Errorf("decode opencode model %q: %w", modelID, err)
		}
		remaining = strings.TrimSpace(remaining[int(decoder.InputOffset()):])
		if item.ProviderID+"/"+item.ID != modelID {
			return nil, fmt.Errorf("opencode model metadata does not match %q", modelID)
		}

		variants := make([]string, 0, len(item.Variants))
		if item.Capabilities.Reasoning {
			for name := range item.Variants {
				variants = append(variants, name)
			}
			order := map[string]int{"none": 0, "minimal": 1, "low": 2, "medium": 3, "high": 4, "xhigh": 5, "max": 6}
			sort.Slice(variants, func(i, j int) bool {
				left, leftKnown := order[variants[i]]
				right, rightKnown := order[variants[j]]
				if leftKnown != rightKnown {
					return leftKnown
				}
				if leftKnown {
					return left < right
				}
				return variants[i] < variants[j]
			})
		}
		efforts := make([]AgentReasoningEffort, 0, len(variants))
		for _, name := range variants {
			efforts = append(efforts, AgentReasoningEffort{ReasoningEffort: name})
		}
		models = append(models, AgentModel{
			ID: modelID, DisplayName: item.Name, SupportedReasoningEfforts: efforts,
			DefaultReasoningEffort: openCodeDefaultVariant(variants, item.Options.ReasoningEffort),
		})
	}
	return models, nil
}

func openCodeDefaultVariant(variants []string, configured string) string {
	for _, candidate := range []string{configured, "medium", "high"} {
		for _, variant := range variants {
			if variant == candidate {
				return variant
			}
		}
	}
	if len(variants) > 0 {
		return variants[0]
	}
	return ""
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
		readyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err = s.waitReady(readyCtx)
		cancel()
		if err != nil {
			_ = s.Stop(context.Background())
			return nil, fmt.Errorf("wait for opencode serve: %w", err)
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
		probeCtx, cancel := context.WithTimeout(ctx, time.Second)
		err := s.request(probeCtx, http.MethodGet, "/session", nil, &sessions)
		cancel()
		if err == nil {
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
	if variant := s.settings.Launch.Effort; variant != "" {
		body["variant"] = variant
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
		s.emit(AgentEvent{Type: AgentEventTurnCompleted, ThreadID: s.sessionID, Status: "failed"})
	}
}
