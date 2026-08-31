package toudocu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenCodeProviderDetect(t *testing.T) {
	old := os.Getenv("PATH")
	t.Setenv("PATH", t.TempDir())
	if _, err := NewOpenCodeProvider(); err == nil {
		t.Fatal("missing opencode executable was detected")
	}
	t.Setenv("PATH", old)
}

func TestOpenCodeProviderInvocation(t *testing.T) { testOpenCodeLifecycle(t) }
func TestOpenCodeSessionLifecycle(t *testing.T)   { testOpenCodeLifecycle(t) }
func TestOpenCodeInterrupt(t *testing.T)          { testOpenCodeLifecycle(t) }
func TestOpenCodeAgentEvents(t *testing.T)        { testOpenCodeLifecycle(t) }
func TestFakeOpenCodeServer(t *testing.T)         { testOpenCodeLifecycle(t) }
func TestOpenCodeReconnect(t *testing.T)          { testOpenCodeLifecycle(t) }

func TestOpenCodeCapabilities(t *testing.T) {
	p := &OpenCodeProvider{}
	got := p.Capabilities()
	if !got.Interrupt || got.Steering || got.Approvals || got.ReadOnlyTurns {
		t.Fatalf("dishonest capabilities: %+v", got)
	}
}
func TestParseOpenCodeModels(t *testing.T) {
	models, err := parseOpenCodeModels("openai/gpt-5.6-sol\n" + `{"id":"gpt-5.6-sol","providerID":"openai","name":"GPT-5.6 Sol","capabilities":{"reasoning":true},"options":{},"variants":{"none":{},"medium":{},"high":{},"max":{}}}`)
	if err != nil || len(models) != 1 || models[0].DisplayName != "GPT-5.6 Sol" || models[0].DefaultReasoningEffort != "medium" || len(models[0].SupportedReasoningEfforts) != 4 || models[0].SupportedReasoningEfforts[0].ReasoningEffort != "none" || models[0].SupportedReasoningEfforts[3].ReasoningEffort != "max" {
		t.Fatalf("models = %+v", models)
	}
}
func TestOpenCodeAccessPreset(t *testing.T) {
	settings := (&openCodeSession{settings: AgentSettings{Launch: AgentLaunch{Preset: AgentLaunchFullAccess}}}).Settings()
	if settings.Launch.Preset != AgentLaunchFullAccess || settings.EffectiveAccess.Known {
		t.Fatalf("access is overstated: %+v", settings)
	}
}
func TestOpenCodePreferences(t *testing.T) {
	store := AgentPreferenceStore{Dir: t.TempDir()}
	root := t.TempDir()
	want := AgentPreferences{LaunchPreset: AgentLaunchFullAccess, Model: "openai/gpt-5.4"}
	if err := store.Save(root, want); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(root); got != want {
		t.Fatalf("preferences = %+v, want %+v", got, want)
	}
}

func TestOpenCodeErrorCompletesTurn(t *testing.T) {
	s := &openCodeSession{sessionID: "session-1", events: make(chan AgentEvent, 2), done: make(chan struct{})}
	s.normalize("session.error", []byte(`{"sessionID":"session-1","error":"failed"}`))
	if <-s.events; (<-s.events).Type != AgentEventTurnCompleted {
		t.Fatal("error did not complete turn")
	}
}

func TestOpenCodeReadinessRetriesHungProbe(t *testing.T) {
	var probes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if probes.Add(1) == 1 {
			<-r.Context().Done()
			return
		}
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()
	s := &openCodeSession{baseURL: server.URL, cwd: t.TempDir(), client: server.Client(), done: make(chan struct{})}
	if err := s.waitReady(context.Background()); err != nil || probes.Load() < 2 {
		t.Fatalf("readiness probes = %d, err = %v", probes.Load(), err)
	}
}

func testOpenCodeLifecycle(t *testing.T) {
	var streams atomic.Int32
	aborted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("directory") == "" {
			t.Error("directory query is missing")
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"id":"session-1"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/session/session-1/prompt_async":
			var body struct {
				Model   struct{ ProviderID, ModelID string }
				Variant string
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model.ProviderID != "openai" || body.Model.ModelID != "gpt-5.4" || body.Variant != "xhigh" {
				t.Errorf("model = %+v", body.Model)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/session/session-1/abort":
			select {
			case aborted <- struct{}{}:
			default:
			}
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/event":
			w.Header().Set("Content-Type", "text/event-stream")
			if streams.Add(1) == 1 {
				_, _ = fmt.Fprintln(w, `data: {"type":"message.part.delta","properties":{"sessionID":"session-1","partID":"part-1","delta":"hello"}}`)
				return
			}
			_, _ = fmt.Fprintln(w, `data: {"type":"message.part.updated","properties":{"sessionID":"session-1","part":{"id":"tool-1","type":"tool","callID":"call-1","state":{"status":"completed","input":{"command":"go test ./..."},"output":"ok","time":{"start":1,"end":3}}}}}`)
			_, _ = fmt.Fprintln(w, `data: {"type":"file.edited","properties":{"sessionID":"session-1","path":"a.go"}}`)
			_, _ = fmt.Fprintln(w, `data: {"type":"session.idle","properties":{"sessionID":"session-1"}}`)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p := &OpenCodeProvider{baseURL: server.URL, client: server.Client()}
	v, err := p.Start(context.Background(), AgentLaunch{CWD: t.TempDir(), Provider: "opencode", Model: "openai/gpt-5.4", Effort: "xhigh"})
	if err != nil {
		t.Fatal(err)
	}
	s := v.(*openCodeSession)
	turn, err := s.StartTurn(context.Background(), "hello", AgentTurnNormal)
	if err != nil || turn == "" {
		t.Fatalf("start turn: %q %v", turn, err)
	}
	seen := map[AgentEventType]bool{}
	deadline := time.After(2 * time.Second)
	for !seen[AgentEventTurnCompleted] {
		select {
		case event := <-s.Events():
			seen[event.Type] = true
			if event.Type == AgentEventCommandStarted && !strings.Contains(event.Command, "go test") {
				t.Fatalf("command not normalized: %+v", event)
			}
		case <-deadline:
			t.Fatalf("events not normalized after %d streams: %+v", streams.Load(), seen)
		}
	}
	for _, kind := range []AgentEventType{AgentEventMessageDelta, AgentEventCommandStarted, AgentEventCommandOutput, AgentEventCommandFinished, AgentEventFilesChanged, AgentEventTurnCompleted} {
		if !seen[kind] {
			t.Errorf("missing %s", kind)
		}
	}
	if err := s.Interrupt(context.Background(), turn); err != nil {
		t.Fatal(err)
	}
	select {
	case <-aborted:
	default:
		t.Fatal("abort endpoint not called")
	}
	if err := s.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
