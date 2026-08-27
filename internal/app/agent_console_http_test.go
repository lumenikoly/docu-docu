package toudocu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	frontend "toudocu/internal/site"
	"toudocu/internal/skillinstall"
)

type unavailableAgentProvider struct{}

func (unavailableAgentProvider) Name() string                    { return "codex" }
func (unavailableAgentProvider) Capabilities() AgentCapabilities { return AgentCapabilities{} }
func (unavailableAgentProvider) Start(context.Context, AgentLaunch) (AgentProviderSession, error) {
	return nil, errors.New("structured integration unavailable")
}

type recoveringAgentProvider struct {
	fail    bool
	session *consoleSpySession
}

func (p *recoveringAgentProvider) Name() string { return "codex" }
func (p *recoveringAgentProvider) Capabilities() AgentCapabilities {
	return p.session.settings.Capabilities
}
func (p *recoveringAgentProvider) Start(_ context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	if p.fail {
		return nil, errors.New("structured integration unavailable")
	}
	p.session.settings.Launch = launch
	return p.session, nil
}

func TestProjectTerminalIndependent(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	server.agentConsole.startShell = func() error {
		server.agentConsole.terminal.mu.Lock()
		defer server.agentConsole.terminal.mu.Unlock()
		server.agentConsole.terminal.active = true
		return nil
	}
	if err := server.agentConsole.startStructured(context.Background(), "", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	if err := server.agentConsole.startProjectTerminal(); err != nil {
		t.Fatal(err)
	}
}

type consoleSpySession struct {
	*fakeAgentSession
	interrupts, approvals atomic.Int32
}

func (s *consoleSpySession) Interrupt(context.Context, string) error { s.interrupts.Add(1); return nil }
func (s *consoleSpySession) Approve(context.Context, string, AgentApprovalDecision) error {
	s.approvals.Add(1)
	return nil
}

type consoleSpyProvider struct{ session *consoleSpySession }

func (p *consoleSpyProvider) Name() string                    { return "codex" }
func (p *consoleSpyProvider) Capabilities() AgentCapabilities { return p.session.settings.Capabilities }
func (p *consoleSpyProvider) Start(_ context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	p.session.settings.Launch = launch
	return p.session, nil
}
func (p *consoleSpyProvider) Threads(_ context.Context, _ string) ([]AgentThread, error) {
	return []AgentThread{{ID: "thread-history", Preview: "Previous work", CreatedAt: 1, UpdatedAt: 2}}, nil
}
func (p *consoleSpyProvider) Resume(_ context.Context, launch AgentLaunch, _ string) (AgentProviderSession, error) {
	p.session.settings.Launch = launch
	return p.session, nil
}

func agentConsoleTestServer(t *testing.T) (*documentationServer, *consoleSpySession) {
	t.Helper()
	session := &consoleSpySession{fakeAgentSession: newFakeAgentSession()}
	session.settings.Capabilities.Approvals = true
	server := &documentationServer{agentConsole: newAgentConsole(&consoleSpyProvider{session: session}, t.TempDir())}
	server.agentConsole.preferences = AgentPreferenceStore{Dir: t.TempDir()}
	return server, session
}

func agentConsoleRequest(method, target, action, body string) *http.Request {
	request := httptest.NewRequest(method, "http://127.0.0.1"+target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://127.0.0.1")
	request.Header.Set("X-Toudocu-Action", action)
	return request
}

func TestAgentConsoleLoopback(t *testing.T) {
	options, _ := serveTestOptions(t)
	server, _, _, err := newDocumentationServer(options, &strings.Builder{})
	if err != nil || server.agentConsole == nil {
		t.Fatalf("loopback console: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, agentConsoleAPIBase+"/", nil)
	request.Host = "127.0.0.1"
	request.Header.Set("Origin", "http://127.0.0.1")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAgentConsoleRuntimeIsolation(t *testing.T) {
	options, _ := serveTestOptions(t)
	options.Host = "0.0.0.0"
	server, _, _, err := newDocumentationServer(options, &strings.Builder{})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, agentConsoleAPIBase+"/", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound || server.model.agentConsoleEnabled {
		t.Fatalf("network runtime exposed agent console: %d", response.Code)
	}
}

func TestAgentConsoleBootstrap(t *testing.T) {
	value := frontend.PageBootstrap{SchemaVersion: 1, Runtime: frontend.RuntimeServe, Page: frontend.PageReference{Kind: "document", Path: "index.html"}, Portal: frontend.PortalReference{AssetBase: "assets/", DataBase: "data/"}, Capabilities: frontend.Capabilities{AgentConsole: true}, Endpoints: &frontend.Endpoints{AgentConsole: agentConsoleAPIBase}}
	if _, err := frontend.MarshalBootstrap(value); err != nil {
		t.Fatal(err)
	}
	value.Endpoints.AgentConsole = "https://example.test/agent"
	if _, err := frontend.MarshalBootstrap(value); err == nil {
		t.Fatal("cross-origin Agent Console endpoint accepted")
	}
}

func TestAgentConsoleSetup(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodGet, agentConsoleAPIBase+"/", "", "")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	var output struct {
		Setup agentConsoleSetup `json:"setup"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &output) != nil {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(output.Setup.AvailableProviders) != 1 || output.Setup.SelectedProvider != "codex" || output.Setup.Preference.LaunchPreset != AgentLaunchDefault || output.Setup.Skill.Diagnostic == "" {
		t.Fatalf("setup=%+v", output.Setup)
	}
}

func TestAgentConsoleHistoryAndResume(t *testing.T) {
	server, session := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodGet, agentConsoleAPIBase+"/history", "", "")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "thread-history") {
		t.Fatalf("history status=%d body=%s", response.Code, response.Body.String())
	}
	request = agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/resume", "agent-session-resume", `{"threadID":"thread-history"}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || session.settings.Launch.CWD == "" {
		t.Fatalf("resume status=%d settings=%+v body=%s", response.Code, session.settings, response.Body.String())
	}
}

func TestAgentConsoleSkillSetup(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/skill", "agent-skill-change", `{"operation":"install","confirmed":false}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed=%d", response.Code)
	}
	request = agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/skill", "agent-skill-change", `{"operation":"install","confirmed":true}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("install=%d body=%s", response.Code, response.Body.String())
	}
	if setup := server.agentConsole.setup("en"); setup.Skill.State != skillinstall.Installed {
		t.Fatalf("skill=%+v", setup.Skill)
	}
}

func TestAgentConsoleAccessPreference(t *testing.T) {
	server, session := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/preference", "agent-preference-save", `{"preset":"full-access","confirmed":false}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("unconfirmed status=%d", response.Code)
	}
	request = agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/preference", "agent-preference-save", `{"preset":"full-access","confirmed":true}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save status=%d body=%s", response.Code, response.Body.String())
	}
	request = agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || session.settings.Launch.Preset != AgentLaunchFullAccess {
		t.Fatalf("start=%d preset=%q", response.Code, session.settings.Launch.Preset)
	}
}

func TestAgentConsoleStartValidatesProvider(t *testing.T) {
	server, session := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{"provider":"other"}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || session.settings.Launch.Provider != "" {
		t.Fatalf("status=%d launch=%+v body=%s", response.Code, session.settings.Launch, response.Body.String())
	}
	request = agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{"provider":"codex"}`)
	response = httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || session.settings.Launch.Provider != "codex" {
		t.Fatalf("status=%d launch=%+v body=%s", response.Code, session.settings.Launch, response.Body.String())
	}
}

func TestAgentConsoleRejectsProcessArguments(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{"taskID":"TASK-X-001","executable":"sh"}`)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAgentConsoleOrigin(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{}`)
	request.Header.Set("Origin", "https://evil.example")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d", response.Code)
	}
}

func TestAgentConsoleOriginRejectsRebindingHost(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	request := agentConsoleRequest(http.MethodPost, agentConsoleAPIBase+"/start", "agent-session-start", `{}`)
	request.Host = "evil.example"
	request.Header.Set("Origin", "http://evil.example")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d", response.Code)
	}
	read := httptest.NewRequest(http.MethodGet, "http://evil.example"+agentConsoleAPIBase+"/", nil)
	read.Header.Set("Origin", "http://evil.example")
	readResponse := httptest.NewRecorder()
	server.ServeHTTP(readResponse, read)
	if readResponse.Code != http.StatusForbidden {
		t.Fatalf("GET status=%d", readResponse.Code)
	}
}

func TestAgentConsoleStateHidesProcessBoundary(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	if err := server.agentConsole.manager.Start(context.Background(), "/secret/repository", "TASK-X-001", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1"+agentConsoleAPIBase+"/", nil)
	request.Header.Set("Origin", "http://127.0.0.1")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	body := response.Body.String()
	if strings.Contains(body, "/secret/repository") || strings.Contains(body, `"provider"`) || strings.Contains(body, `"cwd"`) {
		t.Fatalf("process boundary leaked: %s", body)
	}
}

func TestAgentConsoleBufferLimit(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	for i := 0; i < agentEventLimit+10; i++ {
		server.agentConsole.publish(agentConsoleMessage{Kind: "event"})
	}
	backlog, _, cancel := server.agentConsole.subscribe(0)
	defer cancel()
	if len(backlog) != agentEventLimit || backlog[0].Sequence != 11 {
		t.Fatalf("buffer=%d first=%d", len(backlog), backlog[0].Sequence)
	}
	large := agentConsoleEvent{Type: AgentEventCommandOutput, Text: strings.Repeat("x", agentBufferLimit+1)}
	server.agentConsole.publish(agentConsoleMessage{Kind: "event", Event: &large})
	if server.agentConsole.eventBytes > agentBufferLimit {
		t.Fatalf("buffer bytes=%d", server.agentConsole.eventBytes)
	}
}

func TestAgentConsoleCleanup(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	server.agentConsole.publish(agentConsoleMessage{Kind: "event"})
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < 100; i++ {
			server.agentConsole.publish(agentConsoleMessage{Kind: "late-event"})
		}
	}()
	go func() { defer wait.Done(); <-start; server.agentConsole.Close() }()
	close(start)
	wait.Wait()
	server.agentConsole.publish(agentConsoleMessage{Kind: "late-event"})
	if len(server.agentConsole.events) != 0 || len(server.agentConsole.manager.listeners) != 0 {
		t.Fatal("Agent Console retained ephemeral state")
	}
	connection, peer := net.Pipe()
	defer func() { _ = peer.Close() }()
	if server.agentConsole.registerConnection(connection) {
		t.Fatal("connection registered after cleanup")
	}
}

func TestAgentConsoleReconnect(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	for i := 0; i < 3; i++ {
		server.agentConsole.publish(agentConsoleMessage{Kind: "event"})
	}
	backlog, _, cancel := server.agentConsole.subscribe(2)
	defer cancel()
	if len(backlog) != 1 || backlog[0].Sequence != 3 {
		t.Fatalf("backlog=%v", backlog)
	}
}

func TestAgentConsoleReplayGap(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	for i := 0; i < agentEventLimit+2; i++ {
		server.agentConsole.publish(agentConsoleMessage{Kind: "event"})
	}
	backlog, _, cancel := server.agentConsole.subscribe(1)
	defer cancel()
	if len(backlog) != agentEventLimit+1 || backlog[0].Kind != "replay_gap" || backlog[0].ReplayGap.After != 1 || backlog[0].ReplayGap.Before != 3 {
		t.Fatalf("backlog=%+v", backlog[:1])
	}
	if _, active := server.agentConsole.manager.Snapshot(); active {
		t.Fatal("replay gap changed session state")
	}
}

func TestAgentConsoleCommandProjection(t *testing.T) {
	server, _ := agentConsoleTestServer(t)
	root := server.agentConsole.cwd
	event := server.agentConsole.normalizeAgentEvent(AgentEvent{Type: AgentEventCommandFinished, ItemID: "cmd-1", Command: "go test ./...", CWD: filepath.Join(root, "internal", "app"), Status: "completed", ExitCode: intPointer(0), DurationMillis: 12, ApprovalState: "approved", Text: strings.Repeat("x", agentMessageLimit+4)})
	if event.ItemID != "cmd-1" || event.CWD != "internal/app" || event.DurationMillis != 12 || event.ApprovalState != "approved" || !event.Truncated || len(event.Text) != agentMessageLimit {
		t.Fatalf("event=%+v", event)
	}
	leaked := server.agentConsole.normalizeAgentEvent(AgentEvent{CWD: filepath.Dir(root)})
	if leaked.CWD != "" {
		t.Fatalf("outside cwd leaked: %q", leaked.CWD)
	}
}

func intPointer(value int) *int { return &value }

func TestAgentConsoleWebSocket(t *testing.T) {
	TestAgentConsoleStateHidesProcessBoundary(t)
	server, session := agentConsoleTestServer(t)
	if err := server.agentConsole.manager.Start(context.Background(), server.agentConsole.cwd, "", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	parsed, _ := url.Parse(httpServer.URL)
	conn, err := net.Dial("tcp", parsed.Host)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
	_, _ = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Version: 13\r\nSec-WebSocket-Key: %s\r\nOrigin: %s\r\n\r\n", agentConsoleWS, parsed.Host, key, httpServer.URL)
	reader := bufio.NewReader(conn)
	line, _ := reader.ReadString('\n')
	if !strings.Contains(line, "101") {
		t.Fatalf("handshake: %s", line)
	}
	for {
		line, _ = reader.ReadString('\n')
		if line == "\r\n" {
			break
		}
	}
	payload, _ := json.Marshal(agentWSInput{Action: "message", Text: "hello"})
	if _, err = conn.Write(maskedWebSocketFrame(false, 1, nil)); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Write(maskedWebSocketFrame(true, 0, payload)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for len(session.turnTexts()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := session.turnTexts(); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("turns=%v", got)
	}
	interrupt, _ := json.Marshal(agentWSInput{Action: "interrupt"})
	approval, _ := json.Marshal(agentWSInput{Action: "approval", RequestID: "approval-1", Decision: AgentApprovalAccept})
	session.events <- AgentEvent{Type: AgentEventApproval, Approval: &AgentApproval{RequestID: "approval-1", Kind: "command", Reason: "test"}}
	deadline = time.Now().Add(time.Second)
	for {
		snapshot, _ := server.agentConsole.manager.Snapshot()
		if len(snapshot.Approvals) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("approval was not projected")
		}
		time.Sleep(time.Millisecond)
	}
	_, _ = conn.Write(maskedWebSocketFrame(true, 9, []byte("ping")))
	_, _ = conn.Write(maskedWebSocketFrame(true, 1, interrupt))
	_, _ = conn.Write(maskedWebSocketFrame(true, 1, approval))
	deadline = time.Now().Add(time.Second)
	for (session.interrupts.Load() == 0 || session.approvals.Load() == 0) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if session.interrupts.Load() != 1 || session.approvals.Load() != 1 {
		t.Fatalf("interrupts=%d approvals=%d", session.interrupts.Load(), session.approvals.Load())
	}
}

func maskedWebSocketFrame(fin bool, opcode byte, payload []byte) []byte {
	first := opcode
	if fin {
		first |= 0x80
	}
	frame := []byte{first, 0x80 | byte(len(payload)), 1, 2, 3, 4}
	for i, b := range payload {
		frame = append(frame, b^frame[2+i%4])
	}
	return frame
}

func TestAgentConsoleWebSocketLongFrame(t *testing.T) {
	var output bytes.Buffer
	writer := &webSocketWriter{writer: bufio.NewReadWriter(bufio.NewReader(&output), bufio.NewWriter(&output))}
	if err := writer.writeJSON(map[string]string{"text": strings.Repeat("x", 65536)}); err != nil {
		t.Fatal(err)
	}
	encoded := output.Bytes()
	if len(encoded) < 10 || encoded[1] != 127 {
		t.Fatalf("invalid long frame header: %v", encoded[:2])
	}
}

func TestAgentConsoleWebSocketRejectsFragmentedControl(t *testing.T) {
	frame := maskedWebSocketFrame(false, 9, nil)
	if _, _, _, err := readWebSocketFrame(bufio.NewReader(bytes.NewReader(frame))); err == nil {
		t.Fatal("fragmented ping accepted")
	}
}
