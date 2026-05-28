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

// PublishingAgent runs the ReAct Agent during video publishing (Phase 2).
type PublishingAgent struct {
	rs   *review.ReviewService
	cfg  EngineConfig
	deps ToolboxDeps
}

// NewPublishingAgent creates a new PublishingAgent.
func NewPublishingAgent(rs *review.ReviewService, cfg EngineConfig, deps ToolboxDeps) *PublishingAgent {
	return &PublishingAgent{rs: rs, cfg: cfg, deps: deps}
}

// Run executes Phase 2 (Agent ReAct) after Phase 0/1 results are known.
func (a *PublishingAgent) Run(
	ctx context.Context,
	title, desc string,
	phase0Hits []string,
	phase1Result *review.ReviewResult,
	phase1Details map[string]*review.ReviewResult,
	videoPath, coverPath string,
	framePaths []string,
) (*AgentTrace, error) {
	// Build toolbox with video-specific context
	tools := NewToolbox(a.deps, VideoContext{
		Title:      title,
		Desc:       desc,
		CoverPath:  coverPath,
		VideoPath:  videoPath,
		FramePaths: framePaths,
	}, false)

	engine := &Engine{
		cfg:    a.cfg,
		tools:  tools,
		client: &http.Client{Timeout: 120 * time.Second},
	}

	// Build the initial user message from Phase 0 and Phase 1 results
	msg := buildPublishingContext(title, desc, phase0Hits, phase1Result, phase1Details, framePaths)

	return engine.Run(ctx, PublishingAgentSystemPrompt, msg)
}

func buildPublishingContext(
	title, desc string,
	phase0Hits []string,
	phase1Result *review.ReviewResult,
	phase1Details map[string]*review.ReviewResult,
	framePaths []string,
) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("## 视频信息\n标题: %s\n描述: %s\n抽帧数: %d\n\n", title, desc, len(framePaths)))

	// Phase 0
	b.WriteString("## Phase 0 — 硬性关卡结果\n")
	if len(phase0Hits) > 0 {
		jsonHits, _ := json.Marshal(phase0Hits)
		b.WriteString(fmt.Sprintf("状态: **不通过**\n敏感词命中: %s\n", string(jsonHits)))
	} else {
		b.WriteString("状态: 通过 ✓（无敏感词命中）\n")
	}

	// Phase 1
	b.WriteString("\n## Phase 1 — 强制基础审核结果\n")
	if phase1Details != nil {
		for dim, r := range phase1Details {
			if r == nil {
				b.WriteString(fmt.Sprintf("- **%s**: 执行失败\n", dim))
				continue
			}
			emoji := "✓"
			if r.Status == "rejected" {
				emoji = "✗"
			}
			b.WriteString(fmt.Sprintf("- **%s**: %s %s (置信度: %.2f) 理由: %s\n",
				dim, r.Status, emoji, r.Confidence, r.Reason))
		}
	}

	b.WriteString(fmt.Sprintf("\n汇总结果: %s (置信度: %.2f)\n", phase1Result.Status, phase1Result.Confidence))

	// Guidance
	b.WriteString("\n## 你的任务\n")
	b.WriteString("根据以上结果决定是否需要进一步审核。如果证据充分，调用 done 结束。")
	b.WriteString("如果某维度存疑，调用相应工具深入调查。优先使用低成本工具。\n")

	return b.String()
}
