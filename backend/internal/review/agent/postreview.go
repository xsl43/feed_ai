package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"feedsystem_ai_go/internal/review"
)

// TriageAgent performs lightweight pre-review triage.
type TriageAgent struct {
	cfg  EngineConfig
	deps ToolboxDeps
}

// NewTriageAgent creates a TriageAgent.
func NewTriageAgent(cfg EngineConfig, deps ToolboxDeps) *TriageAgent {
	if cfg.MaxRounds > 1 {
		cfg.MaxRounds = 1
	}
	return &TriageAgent{cfg: cfg, deps: deps}
}

// Triage runs a single-round check. Returns "skip" or "review".
func (a *TriageAgent) Triage(ctx context.Context, title, desc, triggerReason, previousVerdict string) (string, error) {
	tools := NewToolbox(a.deps, VideoContext{Title: title, Desc: desc}, true)
	engine := &Engine{cfg: a.cfg, tools: tools, client: &http.Client{Timeout: 120 * time.Second}}

	msg := fmt.Sprintf(`## 触发信息
触发原因: %s
原始审核结果: %s
视频标题: %s
视频描述: %s

判断是否需要深入复审。输出 skip (放行) 或 review (深入复审)。`, triggerReason, previousVerdict, title, desc)

	trace, err := engine.Run(ctx, TriageAgentSystemPrompt, msg)
	if err != nil {
		return "review", err // On error, default to review (safer)
	}
	return trace.FinalVerdict, nil
}

// PostReviewAgent performs deep review of published content.
type PostReviewAgent struct {
	cfg    EngineConfig
	deps   ToolboxDeps
	triage *TriageAgent
}

// NewPostReviewAgent creates a PostReviewAgent.
func NewPostReviewAgent(cfg EngineConfig, deps ToolboxDeps, triage *TriageAgent) *PostReviewAgent {
	return &PostReviewAgent{cfg: cfg, deps: deps, triage: triage}
}

// ReviewResult holds the outcome of post-review.
type ReviewResult struct {
	Trace   *AgentTrace
	Verdict string // "safe", "risky", "violation", "escalated"
}

// Review runs the full post-review pipeline: triage → deep review.
func (a *PostReviewAgent) Review(
	ctx context.Context,
	title, desc string,
	triggerReason, previousVerdict string,
	prevReview *review.ReviewResult,
	prevTrace string,
	videoPath, coverPath string,
	framePaths []string,
) (*ReviewResult, error) {
	// Step 1: Triage
	triageVerdict, err := a.triage.Triage(ctx, title, desc, triggerReason, previousVerdict)
	if err != nil {
		return &ReviewResult{Verdict: "risky"}, err
	}
	if triageVerdict == "skip" {
		return &ReviewResult{Verdict: "safe"}, nil
	}

	// Step 2: Deep ReAct review
	tools := NewToolbox(a.deps, VideoContext{
		Title:      title,
		Desc:       desc,
		CoverPath:  coverPath,
		VideoPath:  videoPath,
		FramePaths: framePaths,
	}, true)

	engine := &Engine{cfg: a.cfg, tools: tools, client: &http.Client{Timeout: 120 * time.Second}}

	msg := buildPostReviewContext(title, desc, triggerReason, previousVerdict, prevReview, prevTrace)

	trace, err := engine.Run(ctx, PostReviewAgentSystemPrompt, msg)
	if err != nil {
		return &ReviewResult{
			Trace:   trace,
			Verdict: "risky",
		}, err
	}

	verdict := trace.FinalVerdict
	if verdict != "safe" && verdict != "risky" && verdict != "violation" && verdict != "escalated" {
		verdict = "risky"
	}

	return &ReviewResult{Trace: trace, Verdict: verdict}, nil
}

func buildPostReviewContext(
	title, desc, triggerReason, previousVerdict string,
	prevReview *review.ReviewResult,
	prevTrace string,
) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("## 复审触发\n原因: %s\n原始审核结果: %s\n\n", triggerReason, previousVerdict))
	b.WriteString(fmt.Sprintf("## 视频信息\n标题: %s\n描述: %s\n\n", title, desc))

	if prevReview != nil {
		b.WriteString("## 原始审核记录\n")
		b.WriteString(fmt.Sprintf("状态: %s (置信度: %.2f)\n", prevReview.Status, prevReview.Confidence))
		b.WriteString(fmt.Sprintf("理由: %s\n", prevReview.Reason))
		if len(prevReview.Categories) > 0 {
			cats, _ := json.Marshal(prevReview.Categories)
			b.WriteString(fmt.Sprintf("分类: %s\n", string(cats)))
		}
	}

	if prevTrace != "" {
		b.WriteString("\n## 原始 Agent 审核过程\n")
		b.WriteString(prevTrace)
		b.WriteString("\n")
	}

	b.WriteString("\n## 你的任务\n")
	b.WriteString("作为事后复审 Agent，请根据触发原因和原始审核记录，决定是否需要深入审核。")
	b.WriteString("如果触发原因是举报且内容可信，应优先进行全面复审。调用 done 时，输出 verdict 为 safe/risky/violation。\n")

	return b.String()
}
