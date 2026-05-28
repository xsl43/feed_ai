package agent

import "encoding/json"

// RoundTrace records a single round of the ReAct loop.
type RoundTrace struct {
	Round       int    `json:"round"`
	Thought     string `json:"thought"`
	Action      string `json:"action"`
	ActionInput string `json:"action_input"`
	Observation string `json:"observation,omitempty"`
}

// AgentTrace holds the full Agent execution trace.
type AgentTrace struct {
	Rounds       []RoundTrace `json:"rounds"`
	FinalVerdict string       `json:"final_verdict,omitempty"`
	FinalReason  string       `json:"final_reason,omitempty"`
}

// AddRound appends a new round (without observation).
func (t *AgentTrace) AddRound(round int, thought, action, actionInput string) {
	t.Rounds = append(t.Rounds, RoundTrace{
		Round:       round,
		Thought:     thought,
		Action:      action,
		ActionInput: actionInput,
	})
}

// SetObservation sets the observation for the last round.
func (t *AgentTrace) SetObservation(obs string) {
	if len(t.Rounds) > 0 {
		t.Rounds[len(t.Rounds)-1].Observation = obs
	}
}

// ToJSON serializes the trace to JSON string.
func (t *AgentTrace) ToJSON() string {
	b, _ := json.Marshal(t)
	return string(b)
}

// AgentTraceFromJSON deserializes a trace from JSON string.
func AgentTraceFromJSON(s string) (*AgentTrace, error) {
	var t AgentTrace
	if err := json.Unmarshal([]byte(s), &t); err != nil {
		return nil, err
	}
	return &t, nil
}
