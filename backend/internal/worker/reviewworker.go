package worker

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"feedsystem_ai_go/internal/review"
	"feedsystem_ai_go/internal/review/agent"

	"gorm.io/gorm"
)

// ReviewWorker handles post-publication review tasks using the PostReviewAgent.
type ReviewWorker struct {
	db              *gorm.DB
	review          *review.ReviewService
	postReviewAgent *agent.PostReviewAgent
	stopCh          chan struct{}
}

// NewReviewWorker creates a post-review worker.
func NewReviewWorker(db *gorm.DB, rs *review.ReviewService, pra *agent.PostReviewAgent) *ReviewWorker {
	return &ReviewWorker{
		db:              db,
		review:          rs,
		postReviewAgent: pra,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the periodic review tasks.
func (w *ReviewWorker) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkHotVideos()
				w.checkReportedVideos()
				w.checkRiskyVideos()
			case <-w.stopCh:
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkSurgeVideos()
				w.checkSpotVideos()
			case <-w.stopCh:
				return
			}
		}
	}()

	log.Println("[ReviewWorker] 事后复审定时任务已启动 (Agent模式)")
}

// Stop gracefully shuts down the worker.
func (w *ReviewWorker) Stop() {
	close(w.stopCh)
}

type videoRecord struct {
	ID              uint
	Title           string
	Description     string
	CoverURL        string
	PlayURL         string
	ReviewStatus    string
	ReviewReason    string
	ReviewConfidence float64
	ReviewCategories string
	AgentTrace      string
}

// checkRiskyVideos finds videos marked as risky for re-review.
func (w *ReviewWorker) checkRiskyVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url, play_url, review_status, review_reason, review_confidence, review_categories, agent_trace").
		Where("review_status = ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"manual_review", time.Now().Add(-30*time.Minute)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] risky内容触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v, "risky内容定时复审")
	}
}

// checkHotVideos finds trending approved videos for re-review.
func (w *ReviewWorker) checkHotVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url, play_url, review_status, review_reason, review_confidence, review_categories, agent_trace").
		Where("review_status = ? AND popularity > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 10000, time.Now().Add(-6*time.Hour)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 热门内容触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v, "热门内容定时复审")
	}
}

// checkSurgeVideos finds videos with sudden play spikes.
func (w *ReviewWorker) checkSurgeVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url, play_url, review_status, review_reason, review_confidence, review_categories, agent_trace").
		Where("review_status = ? AND play_count > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 50000, time.Now().Add(-1*time.Hour)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 播放量突增触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v, "播放量突增复审")
	}
}

// checkReportedVideos finds videos with high report counts.
func (w *ReviewWorker) checkReportedVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url, play_url, review_status, review_reason, review_confidence, review_categories, agent_trace").
		Where("review_status = ? AND report_count >= ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 5, time.Now().Add(-30*time.Minute)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 被举报内容触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v, "用户举报触发复审")
	}
}

// checkSpotVideos randomly samples videos for spot-check.
func (w *ReviewWorker) checkSpotVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url, play_url, review_status, review_reason, review_confidence, review_categories, agent_trace").
		Where("review_status = ?", "approved").
		Order("RAND()").
		Limit(10).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 定时抽检: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v, "定时随机抽检")
	}
}

// reviewVideo runs the Agent-based post-review pipeline on a single video.
func (w *ReviewWorker) reviewVideo(v videoRecord, triggerReason string) {
	if w.postReviewAgent == nil {
		log.Printf("[ReviewWorker] PostReviewAgent未配置，跳过复审 videoID=%d", v.ID)
		return
	}

	// Parse previous review result
	var prevResult *review.ReviewResult
	if v.ReviewReason != "" {
		var cats []string
		if v.ReviewCategories != "" {
			cats = strings.Split(v.ReviewCategories, ",")
		}
		prevResult = &review.ReviewResult{
			Status:     v.ReviewStatus,
			Confidence: v.ReviewConfidence,
			Reason:     v.ReviewReason,
			Categories: cats,
		}
	}

	// Resolve paths
	coverPath := urlToLocalPath(v.CoverURL)
	videoPath := urlToLocalPath(v.PlayURL)

	// Extract frames for review
	var framePaths []string
	if videoPath != "" && w.review != nil {
		tmpDir, err := os.MkdirTemp("", "postreview_frames_*")
		if err == nil {
			defer os.RemoveAll(tmpDir)
			frames, err := w.review.ExtractFrames(videoPath, tmpDir, 5)
			if err == nil {
				framePaths = frames
			}
		}
	}

	// Run Agent-based review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	result, err := w.postReviewAgent.Review(ctx,
		v.Title, v.Description,
		triggerReason, v.ReviewStatus,
		prevResult, v.AgentTrace,
		videoPath, coverPath, framePaths,
	)

	if err != nil {
		log.Printf("[ReviewWorker] Agent复审失败 videoID=%d: %v", v.ID, err)
	}

	// Apply result to database
	w.applyPostReviewResult(v.ID, result)
}

// applyPostReviewResult updates the video with post-review results.
func (w *ReviewWorker) applyPostReviewResult(videoID uint, result *agent.ReviewResult) {
	if result == nil {
		return
	}

	status := "manual_review"
	switch result.Verdict {
	case "safe":
		status = "approved"
	case "risky":
		status = "manual_review"
	case "violation":
		status = "rejected"
	case "escalated":
		status = "manual_review"
	}

	updates := map[string]interface{}{
		"review_status":    status,
		"last_review_time": time.Now(),
	}

	if result.Trace != nil {
		updates["agent_trace"] = result.Trace.ToJSON()
		updates["agent_rounds"] = len(result.Trace.Rounds)
		updates["agent_verdict"] = result.Verdict

		// Serialize trace to phase results for future reference
		phaseResult, _ := json.Marshal(map[string]interface{}{
			"post_review_verdict": result.Verdict,
			"rounds":              len(result.Trace.Rounds),
			"reason":              result.Trace.FinalReason,
		})
		updates["phase1_result"] = string(phaseResult)
	}

	w.db.Table("videos").Where("id = ?", videoID).Updates(updates)

	if status == "rejected" {
		log.Printf("[ReviewWorker] videoID=%d 复审违规，已下架: %s", videoID, result.Trace.FinalReason)
	} else {
		log.Printf("[ReviewWorker] videoID=%d 复审结果: %s → %s", videoID, result.Verdict, status)
	}
}

func urlToLocalPath(url string) string {
	idx := strings.Index(url, "/static/")
	if idx < 0 {
		return ""
	}
	return filepath.Join(".run", "uploads", url[idx+len("/static/"):])
}
