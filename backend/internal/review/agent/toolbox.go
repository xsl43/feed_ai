package agent

import (
	"os"

	appai "feedsystem_ai_go/internal/ai"
	"feedsystem_ai_go/internal/review"
)

// VideoContext holds per-video data needed by tools.
type VideoContext struct {
	Title      string
	Desc       string
	CoverPath  string
	VideoPath  string
	FramePaths []string
}

// ToolboxDeps holds all external services needed by tool implementations.
type ToolboxDeps struct {
	ReviewService *review.ReviewService
	AIService     *appai.AIService
}

// NewToolbox creates a ToolRegistry with all tools registered.
// includeEscalate controls whether the escalate tool (post-review only) is included.
func NewToolbox(deps ToolboxDeps, vctx VideoContext, includeEscalate bool) *ToolRegistry {
	reg := NewToolRegistry()

	frameDir, _ := os.MkdirTemp("", "agent_frames_*")

	reg.Register(&extractMoreFramesTool{
		reviewService: deps.ReviewService,
		videoPath:     vctx.VideoPath,
		frameDir:      frameDir,
	})
	reg.Register(&transcribeAudioTool{
		reviewService: deps.ReviewService,
		aiService:     deps.AIService,
		videoPath:     vctx.VideoPath,
	})
	reg.Register(&ocrFramesTool{
		reviewService: deps.ReviewService,
		framePaths:    vctx.FramePaths,
	})
	reg.Register(&enlargeFrameTool{
		reviewService: deps.ReviewService,
		framePaths:    vctx.FramePaths,
	})
	reg.Register(&reviewBetterModelTool{
		reviewService: deps.ReviewService,
		title:         vctx.Title,
		desc:          vctx.Desc,
		coverPath:     vctx.CoverPath,
		framePaths:    vctx.FramePaths,
	})
	reg.Register(&fullReviewTool{
		reviewService: deps.ReviewService,
		title:         vctx.Title,
		desc:          vctx.Desc,
		coverPath:     vctx.CoverPath,
		framePaths:    vctx.FramePaths,
	})
	reg.Register(&markCheckedTool{})
	reg.Register(&doneTool{})

	if includeEscalate {
		reg.Register(&escalateTool{})
	}

	return reg
}
