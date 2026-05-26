package worker

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// ReviewWorker handles post-publication review tasks
type ReviewWorker struct {
	db     *gorm.DB
	stopCh chan struct{}
}

// NewReviewWorker creates a post-review worker
func NewReviewWorker(db *gorm.DB) *ReviewWorker {
	return &ReviewWorker{
		db:     db,
		stopCh: make(chan struct{}),
	}
}

// Start begins the periodic review tasks
func (w *ReviewWorker) Start() {
	// Check trending content every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				w.checkHotVideos()
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

	log.Println("[ReviewWorker] 事后审核定时任务已启动")
}

// Stop gracefully shuts down the worker
func (w *ReviewWorker) Stop() {
	close(w.stopCh)
}

// checkHotVideos finds trending approved videos for re-review
func (w *ReviewWorker) checkHotVideos() {
	var ids []uint
	w.db.Table("videos").
		Select("id").
		Where("review_status = ? AND popularity > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 10000, time.Now().Add(-6*time.Hour)).
		Limit(50).
		Pluck("id", &ids)

	if len(ids) > 0 {
		log.Printf("[ReviewWorker] 热门内容触发二次审核: %d videos", len(ids))
	}
}

// checkSurgeVideos finds videos with sudden view spikes
func (w *ReviewWorker) checkSurgeVideos() {
	var ids []uint
	w.db.Table("videos").
		Select("id").
		Where("review_status = ? AND play_count > ? AND (last_review_time < ? OR last_review_time IS NULL)",
			"approved", 50000, time.Now().Add(-1*time.Hour)).
		Limit(50).
		Pluck("id", &ids)

	if len(ids) > 0 {
		log.Printf("[ReviewWorker] 播放量突增触发二次审核: %d videos", len(ids))
	}
}
