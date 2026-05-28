package review

type ReviewResult struct {
	Status     string   `json:"status"`     // approved, rejected
	Confidence float64  `json:"confidence"` // 0.0 - 1.0
	Reason     string   `json:"reason"`
	Categories []string `json:"categories"`
}

type ReviewConfig struct {
	Enabled               bool
	TextModel             string  // 文本审核模型
	VisionModel           string  // 视觉审核模型
	SampleFrames          int
	FrameReviewMode       string  // "off", "on", "auto"
	ConfidenceThreshold   float64 // 高置信度阈值, >=此值AI自主决策
	ManualReviewThreshold float64 // 灰区下限, <此值且AI判定approved时转人工
	MaxRetries            int     // AI失败最大重试次数
	APIKey                string
	BaseURL               string
	MaxVideoSizeMB        int
	MaxCoverSizeMB        int
	MaxVideoDurationSec   int
	MinVideoDurationSec   int
	EnableAudioReview     bool
	EnableOCRReview       bool
	MaxConcurrentFrames   int
	MaxConcurrentVideos   int
	// Agent 配置
	AgentEnabled    bool
	AgentMaxRounds  int
	AgentTimeoutSec int
}

func (c ReviewConfig) FrameReviewEnabled() bool {
	return c.FrameReviewMode == "on" || c.FrameReviewMode == "auto"
}
