package toudocu

import (
	"fmt"
	"strings"
)

type TaskActionDelivery struct {
	Type              string `json:"type"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
	OpenSession       bool   `json:"openSession,omitempty"`
}

type AgentTaskAction struct {
	ID         string               `json:"id"`
	Label      string               `json:"label"`
	Input      string               `json:"input"`
	Deliveries []TaskActionDelivery `json:"deliveries,omitempty"`
	policy     AgentTurnPolicy
	mutates    bool
	states     map[string]bool
	build      func(string, string) string
}

func taskPrompt(format string) func(string, string) string {
	return func(taskID, _ string) string { return fmt.Sprintf(format, taskID) }
}

var agentTaskActions = map[string]AgentTaskAction{
	"start-work":    {ID: "start-work", Label: "Start work", Input: "none", mutates: true, states: states("ready"), build: taskPrompt("Implement task %s using its Toudocu task context and acceptance criteria.")},
	"continue-work": {ID: "continue-work", Label: "Continue work", Input: "none", states: states("in-progress"), build: taskPrompt("Continue task %s. Re-read its current Toudocu task context and repository state before acting; do not assume a previous provider thread is available.")},
	"ask": {ID: "ask", Label: "Ask", Input: "text", policy: AgentTurnReadOnly, states: states("ready", "in-progress", "waiting", "needs-attention", "draft", "blocked", "done"), build: func(taskID, input string) string {
		return fmt.Sprintf("Inspect task %s and answer the user's question using its Toudocu task context.\n\nQuestion: %s", taskID, strings.TrimSpace(input))
	}},
	"clarify":         {ID: "clarify", Label: "Clarify", Input: "none", states: states("ready", "in-progress", "needs-attention", "draft", "blocked"), build: taskPrompt("$toudocu clarify %s")},
	"explain-blocker": {ID: "explain-blocker", Label: "Explain blocker", Input: "none", policy: AgentTurnReadOnly, states: states("waiting", "blocked"), build: taskPrompt("Explain what blocks task %s and what must change before work can start. Do not modify files.")},
	"fix-problems":    {ID: "fix-problems", Label: "Fix problems", Input: "none", policy: AgentTurnNormal, states: states("needs-attention", "draft"), build: taskPrompt("Fix the readiness or contract problems for task %s. Modify the task contract and repository files as needed. Do not mark the task Done automatically.")},
	"next":            {ID: "next", Label: "Ask what to do next", Input: "none", policy: AgentTurnReadOnly, states: states("in-progress"), build: taskPrompt("Using task %s and its Toudocu context, recommend the next concrete step. Do not modify files.")},
	"refresh-diff":    {ID: "refresh-diff", Label: "Refresh diff", Input: "none", states: states("in-progress"), build: taskPrompt("$toudocu refresh diff\n\nWork item: %s")},
}

var taskActionOrder = map[string][]string{
	"ready":           {"start-work", "ask", "clarify"},
	"in-progress":     {"continue-work", "ask", "clarify", "next"},
	"waiting":         {"explain-blocker", "ask"},
	"needs-attention": {"clarify", "ask", "fix-problems"},
	"draft":           {"clarify", "ask", "fix-problems"},
	"blocked":         {"explain-blocker", "ask", "clarify"},
	"done":            {"ask"},
}

func states(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func preparedTaskActions(state string) []AgentTaskAction {
	result := []AgentTaskAction{}
	for _, id := range taskActionOrder[state] {
		action := agentTaskActions[id]
		if action.states[state] {
			action.Deliveries = []TaskActionDelivery{{Type: "handoff", Available: true}}
			result = append(result, action)
		}
	}
	return result
}
