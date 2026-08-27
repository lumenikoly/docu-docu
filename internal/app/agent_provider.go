package toudocu

import (
	"context"
	"errors"
)

type AgentLaunchPreset string

const (
	AgentLaunchDefault    AgentLaunchPreset = "default"
	AgentLaunchFullAccess AgentLaunchPreset = "full-access"
)

type AgentPreferences struct {
	LaunchPreset AgentLaunchPreset `json:"launchPreset"`
	Model        string            `json:"model,omitempty"`
	Effort       string            `json:"effort,omitempty"`
}

type AgentReasoningEffort struct {
	ReasoningEffort string `json:"reasoningEffort"`
	Description     string `json:"description"`
}

type AgentModel struct {
	ID                        string                 `json:"id"`
	DisplayName               string                 `json:"displayName"`
	Description               string                 `json:"description"`
	SupportedReasoningEfforts []AgentReasoningEffort `json:"supportedReasoningEfforts"`
	DefaultReasoningEffort    string                 `json:"defaultReasoningEffort"`
	IsDefault                 bool                   `json:"isDefault"`
}

type AgentCapabilities struct {
	Steering      bool `json:"steering"`
	Interrupt     bool `json:"interrupt"`
	Approvals     bool `json:"approvals"`
	ReadOnlyTurns bool `json:"readOnlyTurns"`
}

type EffectiveAccessSummary struct {
	Known        bool `json:"known"`
	Unrestricted bool `json:"unrestricted"`
}
type AgentLaunch struct {
	CWD      string            `json:"cwd"`
	TaskID   string            `json:"taskID,omitempty"`
	Preset   AgentLaunchPreset `json:"preset"`
	Provider string            `json:"provider"`
	Model    string            `json:"model,omitempty"`
	Effort   string            `json:"effort,omitempty"`
}

type AgentSettings struct {
	Launch          AgentLaunch            `json:"launch"`
	EffectiveAccess EffectiveAccessSummary `json:"effectiveAccess"`
	Capabilities    AgentCapabilities      `json:"capabilities"`
}

type AgentTurnPolicy string

const (
	AgentTurnNormal   AgentTurnPolicy = "normal"
	AgentTurnReadOnly AgentTurnPolicy = "filesystem-read-only"
)

var (
	ErrAgentDeliveryUncertain = errors.New("agent delivery result is uncertain")
	ErrAgentStopUnconfirmed   = errors.New("provider_stop_unconfirmed")
)

type AgentProvider interface {
	Name() string
	Capabilities() AgentCapabilities
	Start(context.Context, AgentLaunch) (AgentProviderSession, error)
}

type AgentModelProvider interface {
	Models(context.Context, string) ([]AgentModel, error)
}

type AgentProviderSession interface {
	Settings() AgentSettings
	Events() <-chan AgentEvent
	StartTurn(context.Context, string, AgentTurnPolicy) (string, error)
	Steer(context.Context, string, string) error
	Interrupt(context.Context, string) error
	Approve(context.Context, string, AgentApprovalDecision) error
	Stop(context.Context) error
}
