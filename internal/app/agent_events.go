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
	RequestID string
	Kind      string
	Reason    string
}

type AgentEvent struct {
	Type     AgentEventType
	ThreadID string
	TurnID   string
	ItemID   string
	Text     string
	Command  string
	CWD      string
	Status   string
	ExitCode *int
	Approval *AgentApproval
}
