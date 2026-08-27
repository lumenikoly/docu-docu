package toudocu

type AgentEventType string

const (
	AgentEventSessionStarted  AgentEventType = "session_started"
	AgentEventTurnStarted     AgentEventType = "turn_started"
	AgentEventTurnCompleted   AgentEventType = "turn_completed"
	AgentEventMessageDelta    AgentEventType = "message_delta"
	AgentEventCommandStarted  AgentEventType = "command_started"
	AgentEventCommandOutput   AgentEventType = "command_output"
	AgentEventCommandFinished AgentEventType = "command_finished"
	AgentEventFilesChanged    AgentEventType = "files_changed"
	AgentEventApproval        AgentEventType = "approval"
	AgentEventError           AgentEventType = "error"
)

type AgentApprovalDecision string

const (
	AgentApprovalAccept  AgentApprovalDecision = "accept"
	AgentApprovalDecline AgentApprovalDecision = "decline"
	AgentApprovalCancel  AgentApprovalDecision = "cancel"
)

type AgentApproval struct {
	RequestID string `json:"requestID"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
}

type AgentEvent struct {
	Type           AgentEventType `json:"type"`
	ThreadID       string         `json:"threadID,omitempty"`
	TurnID         string         `json:"turnID,omitempty"`
	ItemID         string         `json:"itemID,omitempty"`
	Text           string         `json:"text,omitempty"`
	Command        string         `json:"command,omitempty"`
	CWD            string         `json:"cwd,omitempty"`
	Status         string         `json:"status,omitempty"`
	ExitCode       *int           `json:"exitCode,omitempty"`
	DurationMillis int64          `json:"durationMillis,omitempty"`
	ApprovalState  string         `json:"approvalState,omitempty"`
	Approval       *AgentApproval `json:"approval,omitempty"`
}
