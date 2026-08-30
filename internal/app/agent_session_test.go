package toudocu

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeAgentProvider struct{ session *fakeAgentSession }

func (p *fakeAgentProvider) Name() string                    { return "fake" }
func (p *fakeAgentProvider) Capabilities() AgentCapabilities { return p.session.settings.Capabilities }
func (p *fakeAgentProvider) Start(_ context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	p.session.settings.Launch = launch
	return p.session, nil
}

type fakeAgentSession struct {
	mu                              sync.Mutex
	settings                        AgentSettings
	events                          chan AgentEvent
	turns                           []string
	policies                        []AgentTurnPolicy
	startErr                        error
	steerErr, interruptErr, stopErr error
	stopped                         bool
}

func newFakeAgentSession() *fakeAgentSession {
	return &fakeAgentSession{settings: AgentSettings{Capabilities: AgentCapabilities{Steering: true, Interrupt: true, ReadOnlyTurns: true}}, events: make(chan AgentEvent, 8)}
}
func (s *fakeAgentSession) Settings() AgentSettings   { return s.settings }
func (s *fakeAgentSession) Events() <-chan AgentEvent { return s.events }
func (s *fakeAgentSession) StartTurn(_ context.Context, text string, policy AgentTurnPolicy) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startErr != nil {
		return "", s.startErr
	}
	s.turns = append(s.turns, text)
	s.policies = append(s.policies, policy)
	return "turn-" + text, nil
}
func (s *fakeAgentSession) turnTexts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.turns...)
}
func (s *fakeAgentSession) Steer(context.Context, string, string) error                  { return s.steerErr }
func (s *fakeAgentSession) Interrupt(context.Context, string) error                      { return s.interruptErr }
func (s *fakeAgentSession) Approve(context.Context, string, AgentApprovalDecision) error { return nil }
func (s *fakeAgentSession) Stop(context.Context) error                                   { s.stopped = true; return s.stopErr }

func TestAgentSessionLifecycle(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	if err := m.Start(context.Background(), ".", "TASK-1", AgentLaunchDefault); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(context.Background(), ".", "TASK-2", AgentLaunchDefault); err == nil {
		t.Fatal("second session started")
	}
	snapshot, ok := m.Snapshot()
	if !ok || snapshot.Settings.Launch.TaskID != "TASK-1" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if err := m.Stop(context.Background(), false); err != nil {
		t.Fatal(err)
	}
}
func TestAgentSessionTaskBinding(t *testing.T) { TestAgentSessionLifecycle(t) }

func TestAgentSessionMessages(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	if err := m.Send(context.Background(), "one", AgentTurnNormal); err != nil {
		t.Fatal(err)
	}
	s.steerErr = errors.New("not steerable")
	if err := m.Send(context.Background(), "two", AgentTurnNormal); err != nil {
		t.Fatal(err)
	}
	s.events <- AgentEvent{Type: AgentEventTurnCompleted}
	deadline := time.Now().Add(time.Second)
	for len(s.turnTexts()) < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	turns := s.turnTexts()
	if len(turns) != 2 || turns[1] != "two" {
		t.Fatalf("turns = %v", turns)
	}
}
func TestAgentSessionOrdering(t *testing.T)      { TestAgentSessionMessages(t) }
func TestAgentSessionSteeringQueue(t *testing.T) { TestAgentSessionMessages(t) }

func TestAgentSessionWaitsForRunningCommand(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	events, unsubscribe := m.Subscribe()
	defer unsubscribe()
	waitFor := func(want AgentEventType) {
		t.Helper()
		for {
			select {
			case event := <-events:
				if event.Type == want {
					return
				}
			case <-time.After(time.Second):
				t.Fatalf("timed out waiting for %s", want)
			}
		}
	}
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	s.events <- AgentEvent{Type: AgentEventCommandStarted, ItemID: "command-1"}
	s.events <- AgentEvent{Type: AgentEventTurnCompleted, TurnID: "turn-one"}
	s.events <- AgentEvent{Type: AgentEventCommandOutput, ItemID: "command-1", Text: "still running"}

	waitFor(AgentEventCommandOutput)
	snapshot, _ := m.Snapshot()
	if snapshot.Status != AgentSessionRunning || snapshot.ActiveTurn == "" {
		t.Fatalf("command marked idle: %+v", snapshot)
	}

	s.events <- AgentEvent{Type: AgentEventCommandFinished, ItemID: "command-1"}
	waitFor(AgentEventTurnCompleted)
	snapshot, _ = m.Snapshot()
	if snapshot.Status != AgentSessionIdle || snapshot.ActiveTurn != "" {
		t.Fatalf("completed command still running: %+v", snapshot)
	}
}

func TestAgentSessionQueuedReadOnlyPolicy(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "active", AgentTurnNormal)
	_ = m.Send(context.Background(), "read only", AgentTurnReadOnly)
	s.events <- AgentEvent{Type: AgentEventTurnCompleted}
	deadline := time.Now().Add(time.Second)
	for len(s.turnTexts()) < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.policies) != 2 || s.policies[1] != AgentTurnReadOnly {
		t.Fatalf("policies=%v", s.policies)
	}
}

func TestAgentSessionQueueLimits(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "active", AgentTurnNormal)
	for i := 0; i < agentQueueLimit; i++ {
		if err := m.Send(context.Background(), "queued", AgentTurnNormal); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Send(context.Background(), "overflow", AgentTurnNormal); err == nil {
		t.Fatal("queue accepted overflow")
	}
	if err := m.Send(context.Background(), string(make([]byte, agentMessageLimit+1)), AgentTurnNormal); err == nil {
		t.Fatal("accepted oversized message")
	}
}

func TestAgentDeliveryUncertain(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	s.steerErr = ErrAgentDeliveryUncertain
	_ = m.Send(context.Background(), "maybe", AgentTurnNormal)
	snapshot, _ := m.Snapshot()
	if len(snapshot.Pending) != 1 || !snapshot.Pending[0].NotSent {
		t.Fatalf("pending = %+v", snapshot.Pending)
	}
}

func TestAgentDeliveryUncertainStart(t *testing.T) {
	s := newFakeAgentSession()
	s.startErr = ErrAgentDeliveryUncertain
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	if !errors.Is(m.Send(context.Background(), "maybe", AgentTurnNormal), ErrAgentDeliveryUncertain) {
		t.Fatal("uncertain start was not reported")
	}
	snapshot, _ := m.Snapshot()
	if snapshot.Status != AgentSessionFailed || len(snapshot.Pending) != 1 || !snapshot.Pending[0].NotSent {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestAgentTurnInterrupt(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	if err := m.StopResponse(context.Background()); err != nil || s.stopped {
		t.Fatalf("err=%v stopped=%v", err, s.stopped)
	}
}
func TestAgentInterruptUnconfirmed(t *testing.T) {
	s := newFakeAgentSession()
	s.interruptErr = errors.New("unconfirmed")
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	if err := m.StopResponse(context.Background()); err == nil {
		t.Fatal("interrupt succeeded")
	}
	snapshot, _ := m.Snapshot()
	if snapshot.Status != AgentSessionFailed || !s.stopped {
		t.Fatalf("snapshot=%+v stopped=%v", snapshot, s.stopped)
	}
}

func TestAgentSessionStop(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	_ = m.Send(context.Background(), "two", AgentTurnNormal)
	if err := m.Stop(context.Background(), false); err == nil {
		t.Fatal("discard was not required")
	}
	if err := m.Stop(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}
func TestAgentSessionShutdown(t *testing.T) { TestAgentSessionStop(t) }
func TestAgentPendingDiscard(t *testing.T)  { TestAgentSessionStop(t) }
func TestAgentFailedMessages(t *testing.T)  { TestAgentDeliveryUncertain(t) }
func TestAgentStopUnconfirmed(t *testing.T) {
	s := newFakeAgentSession()
	s.stopErr = errors.New("no confirmation")
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	if !errors.Is(m.Stop(context.Background(), true), ErrAgentStopUnconfirmed) {
		t.Fatal("missing provider_stop_unconfirmed")
	}
	if !errors.Is(m.Start(context.Background(), ".", "", AgentLaunchDefault), ErrAgentStopUnconfirmed) {
		t.Fatal("session started after unconfirmed provider stop")
	}
}

func TestAgentPreferenceStore(t *testing.T) {
	store := AgentPreferenceStore{Dir: t.TempDir()}
	root := t.TempDir()
	if got := store.Load(root).LaunchPreset; got != AgentLaunchDefault {
		t.Fatalf("default = %q", got)
	}
	if err := store.Save(root, AgentPreferences{LaunchPreset: AgentLaunchFullAccess}); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(root).LaunchPreset; got != AgentLaunchFullAccess {
		t.Fatalf("saved = %q", got)
	}
}
func TestAgentInvalidPreferences(t *testing.T) {
	store := AgentPreferenceStore{Dir: t.TempDir()}
	if err := store.Save(".", AgentPreferences{LaunchPreset: "bad"}); err == nil {
		t.Fatal("accepted invalid preset")
	}
	path := filepath.Join(store.Dir, "agent-preferences.json")
	bad := []byte(`{"version":99,"roots":{}}`)
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(".").LaunchPreset; got != AgentLaunchDefault {
		t.Fatalf("fallback = %q", got)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(bad) {
		t.Fatalf("invalid file was rewritten: %q, %v", got, err)
	}
}

func TestAgentPreferenceStoreUserLocal(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewAgentPreferenceStore()
	if err != nil {
		t.Fatal(err)
	}
	if store.Dir != filepath.Join(want, "toudocu") {
		t.Fatalf("dir = %q", store.Dir)
	}
}

func TestAgentProviderCrash(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	_ = m.Send(context.Background(), "two", AgentTurnNormal)
	close(s.events)
	deadline := time.Now().Add(time.Second)
	for {
		snapshot, _ := m.Snapshot()
		if snapshot.Status == AgentSessionFailed {
			if len(snapshot.Pending) != 1 || !snapshot.Pending[0].NotSent {
				t.Fatalf("snapshot = %+v", snapshot)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("crash was not observed")
		}
		time.Sleep(time.Millisecond)
	}
}
func TestAgentReadOnlyCapability(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.ReadOnlyTurns = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	if m.Send(context.Background(), "read", AgentTurnReadOnly) == nil {
		t.Fatal("read-only turn accepted")
	}
}
func TestAgentEffectiveAccessSummary(t *testing.T) { TestAgentSessionLifecycle(t) }

func TestAgentPendingCancel(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "active", AgentTurnNormal)
	_ = m.Send(context.Background(), "one", AgentTurnNormal)
	_ = m.Send(context.Background(), "two", AgentTurnNormal)
	snapshot, _ := m.Snapshot()
	if snapshot.Pending[0].ID == "" || snapshot.Pending[0].Position != 1 {
		t.Fatalf("pending=%+v", snapshot.Pending)
	}
	if err := m.CancelPending(snapshot.Pending[0].ID); err != nil {
		t.Fatal(err)
	}
	snapshot, _ = m.Snapshot()
	if len(snapshot.Pending) != 1 || snapshot.Pending[0].Position != 1 || snapshot.Pending[0].Text != "two" {
		t.Fatalf("pending=%+v", snapshot.Pending)
	}
}

func TestAgentApprovalSnapshot(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Approvals = true
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	s.events <- AgentEvent{Type: AgentEventApproval, Approval: &AgentApproval{RequestID: "approval-1", Kind: "command", Reason: "needed"}}
	deadline := time.Now().Add(time.Second)
	for {
		snapshot, _ := m.Snapshot()
		if len(snapshot.Approvals) == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("approval not projected")
		}
		time.Sleep(time.Millisecond)
	}
	if err := m.Approve(context.Background(), "approval-1", AgentApprovalAccept); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := m.Snapshot()
	if len(snapshot.Approvals) != 0 {
		t.Fatalf("approvals=%+v", snapshot.Approvals)
	}
	if err := m.Approve(context.Background(), "approval-1", AgentApprovalAccept); err != nil {
		t.Fatal(err)
	}
}

func TestAgentFailedSessionRequiresCleanup(t *testing.T) {
	s := newFakeAgentSession()
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	close(s.events)
	deadline := time.Now().Add(time.Second)
	for {
		snapshot, _ := m.Snapshot()
		if snapshot.Status == AgentSessionFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("session did not fail")
		}
		time.Sleep(time.Millisecond)
	}
	if m.Start(context.Background(), ".", "", AgentLaunchDefault) == nil {
		t.Fatal("started before cleanup")
	}
	if err := m.Cleanup(); err != nil {
		t.Fatal(err)
	}
}

type blockingStopSession struct {
	*fakeAgentSession
	started chan struct{}
	release chan struct{}
}

func (s *blockingStopSession) Stop(context.Context) error { close(s.started); <-s.release; return nil }

type blockingStopProvider struct{ session *blockingStopSession }

func (p *blockingStopProvider) Name() string { return "fake" }
func (p *blockingStopProvider) Capabilities() AgentCapabilities {
	return p.session.settings.Capabilities
}
func (p *blockingStopProvider) Start(_ context.Context, launch AgentLaunch) (AgentProviderSession, error) {
	p.session.settings.Launch = launch
	return p.session, nil
}

func TestAgentSessionStopping(t *testing.T) {
	s := &blockingStopSession{fakeAgentSession: newFakeAgentSession(), started: make(chan struct{}), release: make(chan struct{})}
	m := NewAgentSessionManager(&blockingStopProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	done := make(chan error, 1)
	go func() { done <- m.Stop(context.Background(), true) }()
	<-s.started
	if err := m.Send(context.Background(), "late", AgentTurnNormal); err == nil || err.Error() != "session_stopping" {
		t.Fatalf("send error=%v", err)
	}
	close(s.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestAgentSessionStopConflict(t *testing.T) {
	s := newFakeAgentSession()
	s.settings.Capabilities.Steering = false
	m := NewAgentSessionManager(&fakeAgentProvider{session: s})
	_ = m.Start(context.Background(), ".", "", AgentLaunchDefault)
	_ = m.Send(context.Background(), "active", AgentTurnNormal)
	_ = m.Send(context.Background(), "queued", AgentTurnNormal)
	m.mu.Lock()
	_ = m.appendPending("uncertain", AgentTurnNormal, true)
	m.mu.Unlock()
	var conflict *AgentStopConflict
	if err := m.Stop(context.Background(), false); !errors.As(err, &conflict) || conflict.Queued != 1 || conflict.NotSent != 1 {
		t.Fatalf("conflict=%+v err=%v", conflict, err)
	}
}

func TestAgentApprovalIdempotency(t *testing.T) { TestAgentApprovalSnapshot(t) }
