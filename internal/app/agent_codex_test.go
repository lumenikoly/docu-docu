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
	var _ AgentSession = (*codexSession)(nil)
	capabilities := (&CodexProvider{}).Capabilities()
	if !capabilities.Steering || !capabilities.Interrupt || !capabilities.Approvals {
		t.Fatalf("capabilities = %+v", capabilities)
	}
}

func TestAgentProviderPreferences(t *testing.T) {
	for preset, want := range map[AgentAccessPreset]string{
		AgentAccessReadOnly: "read-only", AgentAccessWorkspace: "workspace-write", AgentAccessFull: "danger-full-access",
	} {
		if got := codexSandbox(preset); got != want {
			t.Fatalf("codexSandbox(%q) = %q", preset, got)
		}
	}
	t.Setenv("TOUDOCU_FAKE_CODEX", "1")
	provider := &CodexProvider{executable: os.Args[0], args: []string{"-test.run=TestCodexAppServerHelper", "--"}}
	session, err := provider.Start(context.Background(), t.TempDir(), AgentPreferences{Access: AgentAccessReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Stop()
	settings := session.Settings()
	if settings.Provider != "codex" || settings.AccessPreset != AgentAccessReadOnly || settings.EffectiveAccess != AgentAccessReadOnly {
		t.Fatalf("settings = %+v", settings)
	}
}

func TestAgentCapabilities(t *testing.T) { TestAgentProviderContract(t) }

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
	sessionValue, err := provider.Start(ctx, t.TempDir(), AgentPreferences{Access: AgentAccessWorkspace})
	if err != nil {
		t.Fatal(err)
	}
	session := sessionValue.(*codexSession)
	defer session.Stop()
	turnID, err := session.StartTurn(ctx, "implement")
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
			encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"userAgent": "fake"}})
		case "initialized":
			initialized = true
		case "thread/start":
			if !initialized {
				os.Exit(3)
			}
			encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "turn/start":
			encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"turn": map[string]any{"id": "turn-1"}}})
			encoder.Encode(notification("turn/started", map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "inProgress"}}))
			encoder.Encode(notification("item/agentMessage/delta", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "msg-1", "delta": "working"}))
			encoder.Encode(notification("item/started", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "startedAtMs": 1, "item": map[string]any{"id": "cmd-1", "type": "commandExecution", "command": "go test ./...", "cwd": "/repo", "status": "inProgress"}}))
			encoder.Encode(notification("item/commandExecution/outputDelta", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "cmd-1", "delta": "ok\n"}))
			encoder.Encode(notification("item/completed", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "completedAtMs": 2, "item": map[string]any{"id": "cmd-1", "type": "commandExecution", "command": "go test ./...", "status": "completed", "exitCode": 0}}))
			encoder.Encode(notification("turn/diff/updated", map[string]any{"threadId": "thread-1", "turnId": "turn-1", "diff": "diff --git"}))
			encoder.Encode(map[string]any{"id": 99, "method": "item/commandExecution/requestApproval", "params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "cmd-2", "reason": "network", "startedAtMs": 3}})
			encoder.Encode(notification("unknown/newEvent", map[string]any{"future": true}))
			encoder.Encode(notification("turn/completed", map[string]any{"threadId": "thread-1", "turn": map[string]any{"id": "turn-1", "status": "completed"}}))
		case "turn/steer":
			encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{"turnId": "turn-1"}})
		case "turn/interrupt":
			encoder.Encode(map[string]any{"id": request.ID, "result": map[string]any{}})
		}
	}
	os.Exit(0)
}

func notification(method string, params any) map[string]any {
	return map[string]any{"method": method, "params": params}
}
