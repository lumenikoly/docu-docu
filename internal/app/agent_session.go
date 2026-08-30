package toudocu

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

const (
	agentMessageLimit = 65536
	agentQueueLimit   = 32
)

type AgentSessionStatus string

const (
	AgentSessionIdle     AgentSessionStatus = "idle"
	AgentSessionRunning  AgentSessionStatus = "running"
	AgentSessionStopping AgentSessionStatus = "stopping"
	AgentSessionFailed   AgentSessionStatus = "failed"
)

type AgentPendingMessage struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	State    string `json:"state"`
	Reason   string `json:"reason"`
	Position int    `json:"position"`
	NotSent  bool   `json:"notSent"`
	policy   AgentTurnPolicy
}
type AgentSessionSnapshot struct {
	Settings   AgentSettings         `json:"settings"`
	Status     AgentSessionStatus    `json:"status"`
	ActiveTurn string                `json:"activeTurn,omitempty"`
	Pending    []AgentPendingMessage `json:"pending,omitempty"`
	Approvals  []AgentApproval       `json:"approvals,omitempty"`
	Failure    string                `json:"failure,omitempty"`
}

type AgentSessionManager struct {
	mu          sync.Mutex
	provider    AgentProvider
	session     AgentProviderSession
	status      AgentSessionStatus
	turnID      string
	pending     []AgentPendingMessage
	approvals   map[string]AgentApproval
	commands    map[string]struct{}
	completed   *AgentEvent
	nextPending uint64
	failure     string
	listeners   map[chan AgentEvent]struct{}
}

func NewAgentSessionManager(provider AgentProvider) *AgentSessionManager {
	return &AgentSessionManager{provider: provider, listeners: map[chan AgentEvent]struct{}{}}
}

func (m *AgentSessionManager) Subscribe() (<-chan AgentEvent, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan AgentEvent, 64)
	m.listeners[ch] = struct{}{}
	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if _, ok := m.listeners[ch]; ok {
			delete(m.listeners, ch)
			close(ch)
		}
	}
}

func (m *AgentSessionManager) publish(event AgentEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for listener := range m.listeners {
		select {
		case listener <- event:
		default:
		}
	}
}

func (m *AgentSessionManager) Start(ctx context.Context, cwd, taskID string, preset AgentLaunchPreset) error {
	return m.StartConfigured(ctx, AgentLaunch{CWD: cwd, TaskID: taskID, Preset: preset, Provider: m.provider.Name()})
}

func (m *AgentSessionManager) StartConfigured(ctx context.Context, launch AgentLaunch) error {
	return m.startConfigured(ctx, launch, "")
}

func (m *AgentSessionManager) ResumeConfigured(ctx context.Context, launch AgentLaunch, threadID string) error {
	if threadID == "" || len(threadID) > 4096 {
		return errors.New("invalid agent thread id")
	}
	return m.startConfigured(ctx, launch, threadID)
}

func (m *AgentSessionManager) startConfigured(ctx context.Context, launch AgentLaunch, threadID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		if m.status == AgentSessionFailed {
			if m.failure == ErrAgentStopUnconfirmed.Error() {
				return ErrAgentStopUnconfirmed
			}
			return errors.New("failed agent session requires cleanup")
		}
		return errors.New("agent session is already active")
	}
	if launch.Preset == "" {
		launch.Preset = AgentLaunchDefault
	}
	if !validLaunchPreset(launch.Preset) {
		return fmt.Errorf("unsupported agent launch preset %q", launch.Preset)
	}
	if launch.Provider == "" {
		launch.Provider = m.provider.Name()
	}
	var session AgentProviderSession
	var err error
	if threadID == "" {
		session, err = m.provider.Start(ctx, launch)
	} else if provider, ok := m.provider.(AgentHistoryProvider); ok {
		session, err = provider.Resume(ctx, launch, threadID)
	} else {
		return errors.New("agent provider does not support history")
	}
	if err != nil {
		return err
	}
	m.session, m.status, m.turnID, m.pending, m.failure = session, AgentSessionIdle, "", nil, ""
	m.approvals, m.commands, m.completed = map[string]AgentApproval{}, map[string]struct{}{}, nil
	go m.consume(session)
	return nil
}

func (m *AgentSessionManager) Send(ctx context.Context, text string, policy AgentTurnPolicy) error {
	if len([]byte(text)) > agentMessageLimit {
		return errors.New("agent message exceeds 65536 bytes")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil || m.status == AgentSessionFailed {
		return errors.New("agent session is not available")
	}
	if m.status == AgentSessionStopping {
		return errors.New("session_stopping")
	}
	if policy == AgentTurnReadOnly && !m.session.Settings().Capabilities.ReadOnlyTurns {
		return errors.New("agent session does not support read-only turns")
	}
	if m.turnID == "" {
		turnID, err := m.session.StartTurn(ctx, text, policy)
		if err != nil {
			if errors.Is(err, ErrAgentDeliveryUncertain) {
				if queueErr := m.appendPending(text, policy, true); queueErr != nil {
					return queueErr
				}
				m.failLocked(context.Background(), err)
			}
			return err
		}
		m.turnID, m.status = turnID, AgentSessionRunning
		return nil
	}
	if policy == AgentTurnNormal && m.session.Settings().Capabilities.Steering {
		err := m.session.Steer(ctx, m.turnID, text)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrAgentDeliveryUncertain) {
			return m.appendPending(text, policy, true)
		}
	}
	return m.appendPending(text, policy, false)
}

func (m *AgentSessionManager) appendPending(text string, policy AgentTurnPolicy, notSent bool) error {
	if len(m.pending) == agentQueueLimit {
		return errors.New("agent message queue is full")
	}
	m.nextPending++
	state, reason := "queued", "turn_in_progress"
	if notSent {
		state, reason = "not-sent", "delivery_uncertain"
	}
	m.pending = append(m.pending, AgentPendingMessage{ID: fmt.Sprintf("pending-%d", m.nextPending), Text: text, State: state, Reason: reason, Position: len(m.pending) + 1, NotSent: notSent, policy: policy})
	return nil
}

func (m *AgentSessionManager) CancelPending(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.pending {
		if m.pending[i].ID != id {
			continue
		}
		m.pending = append(m.pending[:i], m.pending[i+1:]...)
		m.repositionPendingLocked()
		return nil
	}
	return errors.New("pending message not found")
}

func (m *AgentSessionManager) StopResponse(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil || m.turnID == "" {
		return nil
	}
	if err := m.session.Interrupt(ctx, m.turnID); err != nil {
		m.failLocked(ctx, err)
		return err
	}
	return nil
}

func (m *AgentSessionManager) Approve(ctx context.Context, requestID string, decision AgentApprovalDecision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil || !m.session.Settings().Capabilities.Approvals {
		return errors.New("agent session does not accept approvals")
	}
	if _, ok := m.approvals[requestID]; !ok {
		return nil
	}
	if err := m.session.Approve(ctx, requestID, decision); err != nil {
		return err
	}
	delete(m.approvals, requestID)
	return nil
}

type AgentStopConflict struct{ Queued, NotSent int }

func (e *AgentStopConflict) Error() string { return "pending messages require discardPending=true" }

func (m *AgentSessionManager) Stop(ctx context.Context, discardPending bool) error {
	m.mu.Lock()
	if m.session == nil {
		m.mu.Unlock()
		return nil
	}
	if len(m.pending) > 0 && !discardPending {
		conflict := &AgentStopConflict{}
		for _, pending := range m.pending {
			if pending.NotSent {
				conflict.NotSent++
			} else {
				conflict.Queued++
			}
		}
		m.mu.Unlock()
		return conflict
	}
	m.status = AgentSessionStopping
	session := m.session
	m.mu.Unlock()
	err := session.Stop(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != session {
		return nil
	}
	if err != nil {
		m.status, m.failure = AgentSessionFailed, ErrAgentStopUnconfirmed.Error()
		return ErrAgentStopUnconfirmed
	}
	m.session, m.pending, m.commands, m.completed, m.turnID, m.status, m.failure = nil, nil, nil, nil, "", "", ""
	return nil
}
func (m *AgentSessionManager) Shutdown(ctx context.Context) error { return m.Stop(ctx, true) }

func (m *AgentSessionManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return nil
	}
	if m.status != AgentSessionFailed {
		return errors.New("only a failed agent session can be cleaned up")
	}
	m.session, m.pending, m.approvals, m.commands, m.completed, m.turnID, m.status, m.failure = nil, nil, nil, nil, nil, "", "", ""
	return nil
}

func (m *AgentSessionManager) Snapshot() (AgentSessionSnapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == nil {
		return AgentSessionSnapshot{}, false
	}
	approvals := make([]AgentApproval, 0, len(m.approvals))
	for _, approval := range m.approvals {
		approvals = append(approvals, approval)
	}
	return AgentSessionSnapshot{Settings: m.session.Settings(), Status: m.status, ActiveTurn: m.turnID, Pending: append([]AgentPendingMessage(nil), m.pending...), Approvals: approvals, Failure: m.failure}, true
}

func (m *AgentSessionManager) consume(session AgentProviderSession) {
	for event := range session.Events() {
		if event.Type == AgentEventApproval && event.Approval != nil {
			m.mu.Lock()
			if m.session == session {
				m.approvals[event.Approval.RequestID] = *event.Approval
			}
			m.mu.Unlock()
		}
		if event.Type == AgentEventCommandStarted {
			m.mu.Lock()
			if m.session == session {
				m.commands[event.ItemID] = struct{}{}
			}
			m.mu.Unlock()
		}
		if event.Type == AgentEventCommandFinished {
			m.mu.Lock()
			var completed *AgentEvent
			if m.session == session {
				delete(m.commands, event.ItemID)
				if len(m.commands) == 0 {
					completed, m.completed = m.completed, nil
				}
			}
			m.mu.Unlock()
			m.publish(event)
			if completed != nil {
				m.completeTurn(session, *completed)
			}
			continue
		}
		if event.Type != AgentEventTurnCompleted {
			m.publish(event)
			continue
		}
		m.mu.Lock()
		if m.session != session {
			m.mu.Unlock()
			return
		}
		if len(m.commands) > 0 {
			m.completed = &event
			m.mu.Unlock()
			continue
		}
		m.mu.Unlock()
		m.completeTurn(session, event)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session == session && m.status != AgentSessionFailed {
		m.status, m.failure = AgentSessionFailed, "provider_crash"
		for i := range m.pending {
			m.pending[i].NotSent = true
		}
		if err := session.Stop(context.Background()); err != nil {
			m.failure = ErrAgentStopUnconfirmed.Error()
		}
	}
}

func (m *AgentSessionManager) completeTurn(session AgentProviderSession, event AgentEvent) {
	m.mu.Lock()
	if m.session != session {
		m.mu.Unlock()
		return
	}
	m.turnID, m.status = "", AgentSessionIdle
	nextIndex := -1
	for i := range m.pending {
		if !m.pending[i].NotSent {
			nextIndex = i
			break
		}
	}
	if nextIndex >= 0 {
		next := m.pending[nextIndex]
		m.pending = append(m.pending[:nextIndex], m.pending[nextIndex+1:]...)
		m.repositionPendingLocked()
		turnID, err := session.StartTurn(context.Background(), next.Text, next.policy)
		if err != nil {
			next.NotSent, next.State, next.Reason = true, "not-sent", "delivery_uncertain"
			m.pending = append(m.pending, next)
			m.repositionPendingLocked()
			m.failLocked(context.Background(), err)
		} else {
			m.turnID, m.status = turnID, AgentSessionRunning
		}
	}
	m.mu.Unlock()
	m.publish(event)
}

func (m *AgentSessionManager) repositionPendingLocked() {
	for i := range m.pending {
		m.pending[i].Position = i + 1
	}
}

func (m *AgentSessionManager) failLocked(ctx context.Context, cause error) {
	m.status, m.failure = AgentSessionFailed, cause.Error()
	for i := range m.pending {
		m.pending[i].NotSent = true
	}
	if err := m.session.Stop(ctx); err != nil {
		m.failure = ErrAgentStopUnconfirmed.Error()
	}
}
