package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// EngineConfig holds configuration for the ReAct engine.
type EngineConfig struct {
	Model     string
	BaseURL   string
	APIKey    string
	MaxRounds int
	Timeout   time.Duration
}

// Engine runs the ReAct loop.
type Engine struct {
	cfg    EngineConfig
	tools  *ToolRegistry
	client *http.Client
}

func NewEngine(cfg EngineConfig, tools *ToolRegistry) *Engine {
	if cfg.MaxRounds <= 0 {
		cfg.MaxRounds = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 120 * time.Second
	}
	return &Engine{
		cfg:    cfg,
		tools:  tools,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var (
	reThought     = regexp.MustCompile(`(?is)Thought:\s*(.+?)(?:\n(?:Action|Observation|$)|$)`)
	reAction      = regexp.MustCompile(`(?i)Action:\s*(\w+)`)
	reActionInput = regexp.MustCompile(`(?is)Action\s*Input:\s*(\{.+\})`)
)

// Run executes the ReAct loop and returns the full trace.
func (e *Engine) Run(ctx context.Context, systemPrompt, userMessage string) (*AgentTrace, error) {
	messages := []message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	trace := &AgentTrace{}

	for round := 1; round <= e.cfg.MaxRounds; round++ {
		select {
		case <-ctx.Done():
			trace.FinalVerdict = "manual_review"
			trace.FinalReason = "Agent context timeout"
			return trace, ctx.Err()
		default:
		}

		// Call LLM with current conversation
		llmResponse, err := e.callLLM(ctx, messages)
		if err != nil {
			log.Printf("[Agent] LLM call failed round %d: %v", round, err)
			// Retry once
			llmResponse, err = e.callLLM(ctx, messages)
			if err != nil {
				trace.FinalVerdict = "manual_review"
				trace.FinalReason = fmt.Sprintf("LLM API error: %v", err)
				return trace, err
			}
		}

		thought := extractFirstGroup(reThought, llmResponse)
		action := extractFirstGroup(reAction, llmResponse)
		actionInput := extractFirstGroup(reActionInput, llmResponse)

		if action == "" {
			log.Printf("[Agent] Failed to parse action from response: %s", llmResponse)
			// Send clarification
			messages = append(messages,
				message{Role: "assistant", Content: llmResponse},
				message{Role: "user", Content: "请按照要求的格式输出：Thought / Action / Action Input。不清楚的地方可以追问，但不能省略格式。"},
			)
			continue
		}

		trace.AddRound(round, thought, action, actionInput)

		// Control actions
		if action == "done" {
			trace.FinalVerdict, trace.FinalReason = parseDoneVerdict(actionInput)
			return trace, nil
		}
		if action == "escalate" {
			trace.FinalVerdict = "escalated"
			trace.FinalReason = parseEscalateReason(actionInput)
			return trace, nil
		}

		// Execute tool
		tool, ok := e.tools.Get(action)
		if !ok {
			obs := fmt.Sprintf("错误：未知工具 '%s'。可用工具：%s", action, e.tools.ToolsDescription())
			trace.SetObservation(obs)
			messages = append(messages,
				message{Role: "assistant", Content: llmResponse},
				message{Role: "user", Content: fmt.Sprintf("Observation: %s", obs)},
			)
			continue
		}

		observation, err := tool.Execute(ctx, actionInput)
		if err != nil {
			observation = fmt.Sprintf("工具执行错误: %v", err)
		}
		trace.SetObservation(observation)

		// Append conversation
		messages = append(messages,
			message{Role: "assistant", Content: llmResponse},
			message{Role: "user", Content: fmt.Sprintf("Observation: %s", observation)},
		)
	}

	trace.FinalVerdict = "manual_review"
	trace.FinalReason = fmt.Sprintf("Max rounds (%d) exceeded without done", e.cfg.MaxRounds)
	return trace, nil
}

// callLLM sends the full conversation to the LLM and returns the response text.
func (e *Engine) callLLM(ctx context.Context, messages []message) (string, error) {
	url := e.cfg.BaseURL + "/chat/completions"

	msgs := make([]map[string]string, len(messages))
	for i, m := range messages {
		msgs[i] = map[string]string{"role": m.Role, "content": m.Content}
	}

	reqBody := map[string]interface{}{
		"model":       e.cfg.Model,
		"stream":      false,
		"max_tokens":  1024,
		"temperature": 0,
		"messages":    msgs,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+e.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM API请求失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LLM API错误 (HTTP %d): %s", resp.StatusCode, buf.String())
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return "", fmt.Errorf("LLM响应解析失败: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM返回空结果")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func extractFirstGroup(re *regexp.Regexp, s string) string {
	matches := re.FindStringSubmatch(s)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func parseDoneVerdict(input string) (verdict, reason string) {
	var v struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "manual_review", "verdict parse failed: " + input
	}
	return v.Verdict, v.Reason
}

func parseEscalateReason(input string) string {
	var v struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(input), &v); err != nil {
		return "escalation reason parse failed: " + input
	}
	return v.Reason
}
