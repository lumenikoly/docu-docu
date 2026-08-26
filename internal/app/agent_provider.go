package toudocu

import "context"

type AgentAccessPreset string

const (
	AgentAccessReadOnly  AgentAccessPreset = "read-only"
	AgentAccessWorkspace AgentAccessPreset = "workspace-write"
	AgentAccessFull      AgentAccessPreset = "full-access"
)

type AgentPreferences struct {
	Access AgentAccessPreset
}

type AgentCapabilities struct {
	Steering  bool
	Interrupt bool
	Approvals bool
}

type AgentSettings struct {
	Provider        string
	AccessPreset    AgentAccessPreset
	EffectiveAccess AgentAccessPreset
	Capabilities    AgentCapabilities
}

type AgentProvider interface {
	Name() string
	Capabilities() AgentCapabilities
	Start(context.Context, string, AgentPreferences) (AgentSession, error)
}

type AgentSession interface {
	Settings() AgentSettings
	Events() <-chan AgentEvent
	StartTurn(context.Context, string) (string, error)
	Steer(context.Context, string, string) error
	Interrupt(context.Context, string) error
	Approve(context.Context, string, AgentApprovalDecision) error
	Stop() error
}
