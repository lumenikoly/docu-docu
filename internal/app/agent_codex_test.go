package toudocu

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCodexProviderDetect(t *testing.T) {
	dir := t.TempDir()
	name := "codex"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.Link(os.Args[0], filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	provider, err := NewCodexProvider()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(provider.executable) != name {
		t.Fatalf("executable = %q", provider.executable)
	}
	if fmt.Sprint(provider.args) != "[app-server --stdio]" {
		t.Fatalf("args = %v", provider.args)
	}
}

func TestAgentProviderContract(t *testing.T) {
	var _ AgentProvider = (*CodexProvider)(nil)
	var _ AgentHistoryProvider = (*CodexProvider)(nil)
	var _ AgentProviderSession = (*codexSession)(nil)
	capabilities := (&CodexProvider{}).Capabilities()
	if !capabilities.Steering || !capabilities.Interrupt || !capabilities.Approvals {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func TestCodexHistoryAndResume(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	cwd := t.TempDir()
	t.Setenv("TOUDOCU_EXPECT_CWD", cwd)
	provider := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	threads, err := provider.Threads(context.Background(), cwd)
	if err != nil || len(threads) != 1 || threads[0].ID != "thread-history" {
		t.Fatalf("threads=%+v err=%v", threads, err)
	}
	session, err := provider.Resume(context.Background(), AgentLaunch{CWD: cwd, Preset: AgentLaunchDefault, Provider: "codex"}, threads[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Stop(context.Background()) }()
}

func TestAgentProviderPreferences(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	provider := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	session, err := provider.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchFullAccess, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = session.Stop(context.Background()) }()
	settings := session.Settings()
	if settings.Launch.Provider != "codex" || settings.Launch.Preset != AgentLaunchFullAccess || !settings.EffectiveAccess.Known || !settings.EffectiveAccess.Unrestricted {
		t.Fatalf("settings = %+v", settings)
	}
}

func TestAgentCapabilities(t *testing.T) { TestAgentProviderContract(t) }

func TestCodexDefaults(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	t.Setenv("TOUDOCU_EXPECT_CODEX", "default")
	p := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	s, err := p.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchDefault, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Stop(context.Background()) }()
	if s.Settings().EffectiveAccess.Known || s.Settings().Capabilities.ReadOnlyTurns {
		t.Fatalf("settings = %+v", s.Settings())
	}
}

func TestCodexFullAccess(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	t.Setenv("TOUDOCU_EXPECT_CODEX", "full")
	p := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	s, err := p.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchFullAccess, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Stop(context.Background()) }()
}

func TestCodexReadOnlyTurn(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	t.Setenv("TOUDOCU_EXPECT_CODEX", "readonly")
	p := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	v, err := p.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchFullAccess, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	s := v.(*codexSession)
	defer func() { _ = s.Stop(context.Background()) }()
	if _, err := s.StartTurn(context.Background(), "read", AgentTurnReadOnly); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartTurn(context.Background(), "normal", AgentTurnNormal); err != nil {
		t.Fatal(err)
	}
}
func TestCodexSandboxRestore(t *testing.T) { TestCodexReadOnlyTurn(t) }

func TestCodexSandboxRestoreRetry(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	t.Setenv("TOUDOCU_EXPECT_CODEX", "restore-retry")
	p := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	v, err := p.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchFullAccess, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	s := v.(*codexSession)
	defer func() { _ = s.Stop(context.Background()) }()
	if _, err := s.StartTurn(context.Background(), "read", AgentTurnReadOnly); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartTurn(context.Background(), "normal-fails", AgentTurnNormal); err == nil {
		t.Fatal("restore unexpectedly succeeded")
	}
	if _, err := s.StartTurn(context.Background(), "normal-retry", AgentTurnNormal); err != nil {
		t.Fatal(err)
	}
}
func TestCodexSessionStop(t *testing.T) { TestCodexDefaults(t) }

func TestCodexSessionOutlivesStartContext(t *testing.T) {
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	p := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	ctx, cancel := context.WithCancel(context.Background())
	v, err := p.Start(ctx, AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchDefault, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	s := v.(*codexSession)
	defer func() { _ = s.Stop(context.Background()) }()
	cancel()

	turnCtx, turnCancel := context.WithTimeout(context.Background(), time.Second)
	defer turnCancel()
	if _, err := s.StartTurn(turnCtx, "still running", AgentTurnNormal); err != nil {
		t.Fatal(err)
	}
}

func TestCodexProviderInvocation(t *testing.T)    { testCodexLifecycle(t, false) }
func TestCodexAppServerLifecycle(t *testing.T)    { testCodexLifecycle(t, false) }
func TestCodexProtocolCompatibility(t *testing.T) { testCodexLifecycle(t, true) }
func TestCodexAgentEvents(t *testing.T)           { testCodexLifecycle(t, true) }
func TestFakeCodexAppServer(t *testing.T)         { testCodexLifecycle(t, true) }

func testCodexLifecycle(t *testing.T, checkEvents bool) {
	t.Helper()
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	provider := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	sessionValue, err := provider.Start(ctx, AgentLaunch{CWD: t.TempDir(), Preset: AgentLaunchDefault, Provider: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	session := sessionValue.(*codexSession)
	defer func() { _ = session.Stop(context.Background()) }()
	turnID, err := session.StartTurn(ctx, "implement", AgentTurnNormal)
	if err != nil || turnID != "turn-1" {
		t.Fatalf("turn = %q, err = %v", turnID, err)
	}
	if err := session.Steer(ctx, turnID, "also test"); err != nil {
		t.Fatal(err)
	}
	if err := session.Interrupt(ctx, turnID); err != nil {
		t.Fatal(err)
	}
	if !checkEvents {
		return
	}

	want := map[AgentEventType]bool{
		AgentEventSessionStarted: false, AgentEventTurnStarted: false,
		AgentEventMessageDelta: false, AgentEventCommandStarted: false,
		AgentEventCommandOutput: false, AgentEventCommandFinished: false,
		AgentEventFilesChanged: false, AgentEventApproval: false,
		AgentEventTurnCompleted: false,
	}
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-session.Events():
			if event.Type == AgentEventApproval {
				if event.Approval == nil || event.Approval.RequestID != "99" {
					t.Fatalf("approval = %+v", event)
				}
				if err := session.Approve(ctx, event.Approval.RequestID, AgentApprovalAccept); err != nil {
					t.Fatal(err)
				}
			}
			if _, ok := want[event.Type]; ok {
				want[event.Type] = true
			}
			complete := true
			for _, seen := range want {
				complete = complete && seen
			}
			if complete {
				return
			}
		case <-deadline:
			t.Fatalf("missing normalized events: %v", want)
		}
	}
}

func TestCodexAppServerHelper(t *testing.T) {
	if os.Getenv("TOUDOCU_FAKE_CODEX") != "1" {
		return
	}
	reader := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	initialized := false
	restoreFailed := false
	for reader.Scan() {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(reader.Bytes(), &request); err != nil {
			os.Exit(2)
		}
		if request.Method == "" {
			continue
		}
		switch request.Method {
		case "initialize":
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"userAgent": "fake"}})
		case "initialized":
			initialized = true
		case "thread/start":
			if !initialized {
				os.Exit(3)
			}
			var params map[string]any
			_ = json.Unmarshal(request.Params, &params)
			if params["ephemeral"] != false {
				os.Exit(10)
			}
			expect := os.Getenv("TOUDOCU_EXPECT_CODEX")
			if expect == "default" && (params["sandbox"] != nil || params["approvalPolicy"] != nil) {
				os.Exit(4)
			}
			if (expect == "full" || expect == "readonly" || expect == "restore-retry") && (params["sandbox"] != "danger-full-access" || params["approvalPolicy"] != "never") {
				os.Exit(5)
			}
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "thread/list":
			cwd := os.Getenv("TOUDOCU_EXPECT_CWD")
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"data": []map[string]any{{"id": "thread-history", "preview": "Previous work", "createdAt": 1, "updatedAt": 2, "cwd": cwd}}}})
		case "thread/read":
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"thread": map[string]any{"id": "thread-history", "cwd": os.Getenv("TOUDOCU_EXPECT_CWD")}}})
		case "thread/resume":
			var params struct {
				ThreadID string `json:"threadId"`
				CWD      string `json:"cwd"`
			}
			_ = json.Unmarshal(request.Params, &params)
			if params.ThreadID != "thread-history" || params.CWD != os.Getenv("TOUDOCU_EXPECT_CWD") {
				os.Exit(11)
			}
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"thread": map[string]any{"id": params.ThreadID}}})
		case "turn/start":
			if os.Getenv("TOUDOCU_EXPECT_CODEX") == "readonly" || os.Getenv("TOUDOCU_EXPECT_CODEX") == "restore-retry" {
				var params struct {
					Input []struct {
						Text string `json:"text"`
					} `json:"input"`
					SandboxPolicy codexSandboxPolicy `json:"sandboxPolicy"`
				}
				_ = json.Unmarshal(request.Params, &params)
				if len(params.Input) == 0 || (params.Input[0].Text == "read" && params.SandboxPolicy.Type != "readOnly") || (params.Input[0].Text == "normal" && params.SandboxPolicy.Type != "dangerFullAccess") {
					os.Exit(6)
				}
				if params.Input[0].Text == "normal-fails" {
					if params.SandboxPolicy.Type != "dangerFullAccess" {
						os.Exit(7)
					}
					restoreFailed = true
					_ = encoder.Encode(map[string]any{"id": request.ID, "error": map[string]any{"code": -1, "message": "retry"}})
					continue
				}
				if params.Input[0].Text == "normal-retry" && (!restoreFailed || params.SandboxPolicy.Type != "dangerFullAccess") {
					os.Exit(8)
				}
			}
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"turn": map[string]any{"id": "turn-1"}}})
			_ = encoder.Encode(notification("turn/started", map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "inProgress"}}))
			_ = encoder.Encode(notification("item/agentMessage/delta", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "msg-1", "delta": "working"}))
			_ = encoder.Encode(notification("item/started", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "startedAtMs": 1, "item": map[string]any{"id": "cmd-1", "type": "commandExecution", "command": "go test ./...", "cwd": "/repo", "status": "inProgress"}}))
			_ = encoder.Encode(notification("item/commandExecution/outputDelta", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "cmd-1", "delta": "ok\n"}))
			_ = encoder.Encode(notification("item/completed", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "completedAtMs": 2, "item": map[string]any{"id": "cmd-1", "type": "commandExecution", "command": "go test ./...", "status": "completed", "exitCode": 0}}))
			_ = encoder.Encode(notification("turn/diff/updated", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "diff": "diff --git"}))
			_ = encoder.Encode(map[string]any{"id": 99, "method": "item/commandExecution/requestApproval", "params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "cmd-2", "reason": "network", "startedAtMs": 3}})
			_ = encoder.Encode(notification("unknown/newEvent", map[string]any{"future": true}))
			_ = encoder.Encode(notification("turn/completed", map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}}))
		case "turn/steer":
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"turnId": "turn-1"}})
		case "turn/interrupt":
			_ = encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{}})
		}
	}
	if reader.Err() != nil {
		os.Exit(9)
	}
	os.Exit(0)
}

func notification(method string, params any) map[string]any {
	return map[string]any{"method": method, "params": params}
}
