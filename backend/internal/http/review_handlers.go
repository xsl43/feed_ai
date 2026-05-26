package http

import (
	"feedsystem_ai_go/internal/review"
	"feedsystem_ai_go/internal/video"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewHandler struct {
	db           *gorm.DB
	service      *review.ReviewService
	videoService *video.VideoService
}

func NewReviewHandler(db *gorm.DB, service *review.ReviewService, videoService *video.VideoService) *ReviewHandler {
	return &ReviewHandler{db: db, service: service, videoService: videoService}
}

// GetPendingVideos GET /review/pending - 管理员获取待审核视频列表
func (h *ReviewHandler) GetPendingVideos(c *gin.Context) {
	var videos []video.Video
	h.db.Where("review_status = ?", "manual_review").
		Order("review_priority DESC, create_time ASC").
		Limit(100).
		Find(&videos)
	if videos == nil {
		videos = []video.Video{}
	}
	c.JSON(http.StatusOK, videos)
}

// ApproveVideo POST /review/video/:id/approve - 管理员人工通过
func (h *ReviewHandler) ApproveVideo(c *gin.Context) {
	h.manualReview(c, "approved")
}

// RejectVideo POST /review/video/:id/reject - 管理员人工拒绝
func (h *ReviewHandler) RejectVideo(c *gin.Context) {
	h.manualReview(c, "rejected")
}

func (h *ReviewHandler) manualReview(c *gin.Context, status string) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的ID"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	var v video.Video
	if err := h.db.First(&v, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "视频不存在"})
		return
	}

	oldStatus := v.ReviewStatus
	v.ReviewStatus = status
	if req.Reason != "" {
		v.ReviewReason = "人工审核: " + req.Reason
	} else {
		v.ReviewReason = "人工审核"
	}
	v.RetryCount = 0
	h.db.Save(&v)

	// If manually approved, write outbox message to make video visible in feed
	if status == "approved" && oldStatus != "approved" {
		msg := video.OutboxMsg{
			VideoID:    v.ID,
			EventType:  "video_published",
			Status:     "pending",
			CreateTime: v.CreateTime,
		}
		h.db.Create(&msg)
	}

	c.JSON(http.StatusOK, gin.H{"message": "审核完成", "status": status})
}

// GetVideoReviewStatus GET /review/status/:videoId - 作者查询审核状态
func (h *ReviewHandler) GetVideoReviewStatus(c *gin.Context) {
	idStr := c.Param("videoId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "无效的ID"})
		return
	}

	var v video.Video
	if err := h.db.First(&v, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "视频不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                v.ID,
		"review_status":     v.ReviewStatus,
		"review_reason":     v.ReviewReason,
		"review_confidence": v.ReviewConfidence,
	})
}

// ReSubmitVideo POST /review/resubmit - 作者重新提交被拒视频
func (h *ReviewHandler) ReSubmitVideo(c *gin.Context) {
	var req video.ReSubmitVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var v video.Video
	if err := h.db.First(&v, req.ID).Error; err != nil {
		c.JSON(404, gin.H{"error": "视频不存在"})
		return
	}

	if v.ReviewStatus != "rejected" && v.ReviewStatus != "manual_review" {
		c.JSON(400, gin.H{"error": "只能重新提交被拒绝或在人工审核队列中的视频"})
		return
	}

	v.Title = req.Title
	v.Description = req.Description
	v.ReviewStatus = "pending"
	v.ReviewReason = ""
	v.ReviewConfidence = 0
	v.RetryCount = 0
	h.db.Save(&v)

	if h.service.IsEnabled() && h.videoService != nil {
		go h.videoService.ReviewAndPublishVideo(&v)
	}

	c.JSON(http.StatusOK, gin.H{"message": "已重新提交审核"})
}

// GetReviewConfig GET /review/config
func (h *ReviewHandler) GetReviewConfig(c *gin.Context) {
	cfg := h.service.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"enabled":                 cfg.Enabled,
		"text_model":              cfg.TextModel,
		"vision_model":            cfg.VisionModel,
		"sample_frames":           cfg.SampleFrames,
		"frame_review_mode":       cfg.FrameReviewMode,
		"confidence_threshold":    cfg.ConfidenceThreshold,
		"manual_review_threshold": cfg.ManualReviewThreshold,
		"max_retries":             cfg.MaxRetries,
	})
}

// UpdateReviewConfig POST /review/config
func (h *ReviewHandler) UpdateReviewConfig(c *gin.Context) {
	var req struct {
		Enabled               *bool    `json:"enabled"`
		TextModel             *string  `json:"text_model"`
		VisionModel           *string  `json:"vision_model"`
		SampleFrames          *int     `json:"sample_frames"`
		FrameReviewMode       *string  `json:"frame_review_mode"`
		ConfidenceThreshold   *float64 `json:"confidence_threshold"`
		ManualReviewThreshold *float64 `json:"manual_review_threshold"`
		MaxRetries            *int     `json:"max_retries"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	cfg := h.service.GetConfig()
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.TextModel != nil {
		cfg.TextModel = *req.TextModel
	}
	if req.VisionModel != nil {
		cfg.VisionModel = *req.VisionModel
	}
	if req.SampleFrames != nil {
		cfg.SampleFrames = *req.SampleFrames
	}
	if req.FrameReviewMode != nil {
		cfg.FrameReviewMode = *req.FrameReviewMode
	}
	if req.ConfidenceThreshold != nil {
		cfg.ConfidenceThreshold = *req.ConfidenceThreshold
	}
	if req.ManualReviewThreshold != nil {
		cfg.ManualReviewThreshold = *req.ManualReviewThreshold
	}
	if req.MaxRetries != nil {
		cfg.MaxRetries = *req.MaxRetries
	}
	h.service.UpdateConfig(cfg)

	c.JSON(http.StatusOK, gin.H{"message": "审核配置已更新"})
}
