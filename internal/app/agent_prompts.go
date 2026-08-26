package toudocu

type AgentTaskAction struct {
	ID     string          `json:"id"`
	Label  string          `json:"label"`
	Prompt string          `json:"-"`
	Policy AgentTurnPolicy `json:"-"`
}

var agentTaskActions = map[string]AgentTaskAction{
	"start-work":       {ID: "start-work", Label: "Start work", Prompt: "Implement task %s using its Toudocu task context and acceptance criteria.", Policy: AgentTurnNormal},
	"ask":              {ID: "ask", Label: "Ask", Prompt: "Inspect task %s and answer the user's question using its Toudocu task context.", Policy: AgentTurnReadOnly},
	"clarify":          {ID: "clarify", Label: "Clarify", Prompt: "$toudocu clarify %s", Policy: AgentTurnNormal},
	"explain-blocker":  {ID: "explain-blocker", Label: "Explain blocker", Prompt: "Explain what blocks task %s and what must change before work can start. Do not modify files.", Policy: AgentTurnReadOnly},
	"explain-problems": {ID: "explain-problems", Label: "Explain problems", Prompt: "Explain the readiness or contract problems for task %s. Do not modify files.", Policy: AgentTurnReadOnly},
	"next":             {ID: "next", Label: "Ask what to do next", Prompt: "Using task %s and its Toudocu context, recommend the next concrete step. Do not modify files.", Policy: AgentTurnReadOnly},
	"refresh-diff":     {ID: "refresh-diff", Label: "Refresh diff", Prompt: "$toudocu refresh diff\n\nWork item: %s", Policy: AgentTurnNormal},
}

func preparedTaskActions(state string) []AgentTaskAction {
	ids := map[string][]string{
		"ready":           {"start-work", "ask", "clarify"},
		"in-progress":     {"ask", "clarify", "explain-problems", "next", "refresh-diff"},
		"waiting":         {"ask", "explain-blocker"},
		"needs-attention": {"ask", "clarify", "explain-problems"},
		"draft":           {"ask", "clarify", "explain-problems"},
	}[state]
	result := make([]AgentTaskAction, 0, len(ids))
	for _, id := range ids {
		result = append(result, agentTaskActions[id])
	}
	return result
}
