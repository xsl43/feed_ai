package worker

import (
	"log"
	"time"

	"feedsystem_ai_go/internal/review"

	"gorm.io/gorm"
)

// ReviewWorker handles post-publication review tasks
type ReviewWorker struct {
	db      *gorm.DB
	review  *review.ReviewService
	stopCh  chan struct{}
}

// NewReviewWorker creates a post-review worker
func NewReviewWorker(db *gorm.DB, rs *review.ReviewService) *ReviewWorker {
	return &ReviewWorker{
		db:     db,
		review: rs,
		stopCh: make(chan struct{}),
	}
}

// Start begins the periodic review tasks
func (w *ReviewWorker) Start() {
	// Check hot/reported content every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkHotVideos()
				w.checkReportedVideos()
			case <-w.stopCh:
				return
			}
		}
	}()

	// Check view surge every hour
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkSurgeVideos()
			case <-w.stopCh:
				return
			}
		}
	}()

	log.Println("[ReviewWorker] 事后审核定时任务已启动 (热门/举报:5min, 流量突增:1h)")
}

// Stop gracefully shuts down the worker
func (w *ReviewWorker) Stop() {
	close(w.stopCh)
}

type videoRecord struct {
	ID          uint
	Title       string
	Description string
	CoverURL    string
}

// checkHotVideos finds trending approved videos for re-review
func (w *ReviewWorker) checkHotVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url").
		Where("review_status = ? AND popularity > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 10000, time.Now().Add(-6*time.Hour)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 热门内容触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v.ID, v.Title, v.Description, v.CoverURL)
	}
}

// checkSurgeVideos finds videos with sudden play spikes
func (w *ReviewWorker) checkSurgeVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url").
		Where("review_status = ? AND play_count > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 50000, time.Now().Add(-1*time.Hour)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 播放量突增触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v.ID, v.Title, v.Description, v.CoverURL)
	}
}

// checkReportedVideos finds videos with high report counts
func (w *ReviewWorker) checkReportedVideos() {
	var videos []videoRecord
	w.db.Table("videos").
		Select("id, title, description, cover_url").
		Where("review_status = ? AND report_count >= ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 10, time.Now().Add(-30*time.Minute)).
		Limit(20).
		Find(&videos)

	if len(videos) == 0 {
		return
	}
	log.Printf("[ReviewWorker] 被举报内容触发复审: %d videos", len(videos))
	for _, v := range videos {
		w.reviewVideo(v.ID, v.Title, v.Description, v.CoverURL)
	}
}

// reviewVideo runs text review on a single video and updates its status
func (w *ReviewWorker) reviewVideo(id uint, title, desc, coverURL string) {
	result, err := w.review.ReviewTextWithRetry(title, desc)
	if err != nil {
		log.Printf("[ReviewWorker] 复审失败 videoID=%d: %v", id, err)
		return
	}

	finalStatus := w.review.Classify(result)
	categories := ""
	if len(result.Categories) > 0 {
		categories = result.Categories[0]
		for i := 1; i < len(result.Categories); i++ {
			categories += "," + result.Categories[i]
		}
	}

	w.db.Table("videos").Where("id = ?", id).Updates(map[string]interface{}{
		"review_status":     finalStatus,
		"review_reason":     result.Reason,
		"review_confidence": result.Confidence,
		"review_categories": categories,
		"last_review_time":  time.Now(),
	})

	if finalStatus == "rejected" {
		log.Printf("[ReviewWorker] videoID=%d 复审拒绝: %s (confidence=%.2f)", id, result.Reason, result.Confidence)
	} else {
		log.Printf("[ReviewWorker] videoID=%d 复审通过", id)
	}
}
