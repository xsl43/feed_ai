package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"feedsystem_ai_go/internal/middleware/rabbitmq"
	rediscache "feedsystem_ai_go/internal/middleware/redis"
	"feedsystem_ai_go/internal/review"
	"feedsystem_ai_go/internal/apierror"

	"gorm.io/gorm"
)

type VideoService struct {
	repo          *VideoRepository
	cache         *rediscache.Client
	cacheTTL      time.Duration
	popularityMQ  *rabbitmq.PopularityMQ
	reviewService *review.ReviewService
}

func NewVideoService(repo *VideoRepository, cache *rediscache.Client, popularityMQ *rabbitmq.PopularityMQ) *VideoService {
	return &VideoService{repo: repo, cache: cache, cacheTTL: 5 * time.Minute, popularityMQ: popularityMQ}
}

func (vs *VideoService) SetReviewService(rs *review.ReviewService) {
	vs.reviewService = rs
}

// GetReviewConfig returns the review config (for handlers to read size limits, etc.)
func (vs *VideoService) GetReviewConfig() review.ReviewConfig {
	if vs.reviewService == nil {
		return review.ReviewConfig{
			MaxVideoSizeMB: 500,
			MaxCoverSizeMB: 20,
		}
	}
	return vs.reviewService.GetConfig()
}

func (vs *VideoService) Publish(ctx context.Context, video *Video) error {
	if video == nil {
		return errors.New("video is nil")
	}
	video.Title = strings.TrimSpace(video.Title)
	video.PlayURL = strings.TrimSpace(video.PlayURL)
	video.CoverURL = strings.TrimSpace(video.CoverURL)

	if video.Title == "" {
		return errors.New("title is required")
	}
	if video.PlayURL == "" {
		return errors.New("play url is required")
	}
	if video.CoverURL == "" {
		return errors.New("cover url is required")
	}

	reviewEnabled := vs.reviewService != nil && vs.reviewService.IsEnabled()

	if reviewEnabled {
		video.ReviewStatus = "pending"
	}

	//事务保证视频写入库和消息写入本地消息表的一致性
	err := vs.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(video).Error; err != nil {
			return err
		}

		if !reviewEnabled {
			msg := OutboxMsg{
				VideoID:    video.ID,
				EventType:  "video_published",
				Status:     "pending",
				CreateTime: video.CreateTime,
			}

			if err := tx.Create(&msg).Error; err != nil {
				return err
			}
		}

		tags := ExtractTags(video.Title + " " + video.Description)
		for _, tagName := range tags {
			var tag Tag
			tx.Where("name = ?", tagName).FirstOrCreate(&tag, Tag{Name: tagName})
			tx.Create(&VideoTag{VideoID: video.ID, TagID: tag.ID})
		}
		return nil
	})
	if err != nil {
		return err
	}

	if reviewEnabled {
		go vs.ReviewAndPublishVideo(video)
	}

	return nil
}

// ReviewAndPublishVideo runs the full review pipeline and updates video status.
func (vs *VideoService) ReviewAndPublishVideo(v *Video) {
	// Stage 0: Sensitive word pre-check
	if vs.reviewService != nil {
		hits := vs.reviewService.SensitiveWordCheck(v.Title + " " + v.Description)
		if len(hits) > 0 {
			vs.applyReviewResult(v.ID, "manual_review", &review.ReviewResult{
				Status:     "rejected",
				Confidence: 0.9,
				Reason:     fmt.Sprintf("敏感词命中: %v", hits),
				Categories: []string{"敏感词"},
			})
			return
		}
	}

	// Stage 1: Determine frame paths if frame review is needed
	var framePaths []string
	coverPath := urlToLocalPath(v.CoverURL)

	// Extract frames if frame review mode is on or auto
	if vs.reviewService != nil && vs.reviewService.GetConfig().FrameReviewEnabled() {
		videoPath := urlToLocalPath(v.PlayURL)
		if videoPath != "" {
			tmpDir, err := os.MkdirTemp("", "frames_*")
			if err == nil {
				defer os.RemoveAll(tmpDir)
				frames, err := vs.reviewService.ExtractFrames(videoPath, tmpDir, vs.reviewService.GetConfig().SampleFrames)
				if err == nil {
					framePaths = frames
				}
			}
		}
	}

	// Stage 2: Concurrent review (text + cover + frames)
	result, _ := vs.reviewService.ReviewAllDimensions(
		v.Title,
		v.Description,
		coverPath,
		"", // videoPath not needed separately
		framePaths,
	)

	// Stage 3: Classify and apply
	finalStatus := vs.reviewService.Classify(result)
	vs.applyReviewResult(v.ID, finalStatus, result)

	if finalStatus == "approved" {
		_ = vs.repo.CreateMsg(context.Background(), &OutboxMsg{
			VideoID:    v.ID,
			EventType:  "video_published",
			CreateTime: time.Now(),
			Status:     "pending",
		})
	}
}

func (vs *VideoService) applyReviewResult(videoID uint, status string, result *review.ReviewResult) {
	categories := ""
	if len(result.Categories) > 0 {
		categories = result.Categories[0]
		for i := 1; i < len(result.Categories); i++ {
			categories += "," + result.Categories[i]
		}
	}
	vs.repo.db.Model(&Video{}).Where("id = ?", videoID).Updates(map[string]interface{}{
		"review_status":     status,
		"review_reason":     result.Reason,
		"review_confidence": result.Confidence,
		"review_categories": categories,
		"last_review_time":  time.Now(),
	})
}

func urlToLocalPath(url string) string {
	idx := strings.Index(url, "/static/")
	if idx < 0 {
		return ""
	}
	return filepath.Join(".run", "uploads", url[idx+len("/static/"):])
}

func (vs *VideoService) Delete(ctx context.Context, id uint, authorID uint) error {
	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if video == nil {
		return errors.New("video not found")
	}
	if video.AuthorID != authorID {
		return apierror.ErrUnauthorized
	}
	if err := vs.repo.DeleteVideo(ctx, id); err != nil {
		return err
	}
	if vs.cache != nil {
		cacheKey := vs.cache.Key("video:detail:id=%d", id)
		_ = vs.cache.Del(context.Background(), cacheKey)
	}
	return nil
}

func (vs *VideoService) ListByAuthorID(ctx context.Context, authorID uint) ([]Video, error) {
	videos, err := vs.repo.ListByAuthorID(ctx, int64(authorID))
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (vs *VideoService) GetDetail(ctx context.Context, id uint, viewerID uint) (*Video, error) {
	video, err := vs.getDetailInternal(ctx, id)
	if err != nil {
		return nil, err
	}
	if video.ReviewStatus != "approved" && video.AuthorID != viewerID {
		return nil, errors.New("video not found")
	}
	return video, nil
}

func (vs *VideoService) getDetailInternal(ctx context.Context, id uint) (*Video, error) {
	cacheKey := vs.cache.Key("video:detail:id=%d", id)

	getCached := func() (*Video, bool) {
		opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		b, err := vs.cache.GetBytes(opCtx, cacheKey)
		if err != nil {
			return nil, false
		}
		var cached Video
		if err := json.Unmarshal(b, &cached); err != nil {
			return nil, false
		}
		return &cached, true
	}

	setCached := func(video *Video) {
		b, err := json.Marshal(video)
		if err != nil {
			return
		}
		opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		_ = vs.cache.SetBytes(opCtx, cacheKey, b, vs.cacheTTL)
	}

	if vs.cache != nil {
		if v, ok := getCached(); ok {
			return v, nil
		}

		opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		b, err := vs.cache.GetBytes(opCtx, cacheKey)
		cancel()
		if err == nil {
			var cached Video
			if err := json.Unmarshal(b, &cached); err == nil {
				return &cached, nil
			}
		} else if rediscache.IsMiss(err) {
			lockKey := "lock:" + cacheKey

			lockCtx, lockCancel := context.WithTimeout(ctx, 50*time.Millisecond)
			token, locked, lockErr := vs.cache.Lock(lockCtx, lockKey, 2*time.Second)
			lockCancel()

			if lockErr == nil && locked {
				defer func() { _ = vs.cache.Unlock(context.Background(), lockKey, token) }()

				if v, ok := getCached(); ok {
					return v, nil
				}

				video, err := vs.repo.GetByID(ctx, id)
				if err != nil {
					return nil, err
				}
				setCached(video)
				return video, nil
			}

			// 没拿到锁：等待别人回填缓存
			for i := 0; i < 5; i++ {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(20 * time.Millisecond):
				}
				if v, ok := getCached(); ok {
					return v, nil
				}
			}
		}
	}

	video, err := vs.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if vs.cache != nil {
		setCached(video)
	}
	return video, nil
}

func (vs *VideoService) UpdateLikesCount(ctx context.Context, id uint, likesCount int64) error {
	if err := vs.repo.UpdateLikesCount(ctx, id, likesCount); err != nil {
		return err
	}
	return nil
}

func (vs *VideoService) UpdatePopularity(ctx context.Context, id uint, change int64) error {
	if err := vs.repo.UpdatePopularity(ctx, id, change); err != nil {
		return err
	}

	if vs.popularityMQ != nil {
		if err := vs.popularityMQ.Update(ctx, id, change); err == nil {
			return nil
		}
	}

	if vs.cache != nil {
		// 1) 详情缓存：直接失效（最简单靠谱）
		_ = vs.cache.Del(context.Background(), vs.cache.Key("video:detail:id=%d", id))

		// 2) 热榜：写到“时间窗ZSET”，不要用 detail key
		now := time.Now().UTC().Truncate(time.Minute)
		windowKey := vs.cache.Key("hot:video:1m:%s", now.Format("200601021504"))
		member := strconv.FormatUint(uint64(id), 10)

		opCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()

		_ = vs.cache.ZincrBy(opCtx, windowKey, member, float64(change))
		_ = vs.cache.Expire(opCtx, windowKey, 2*time.Hour)
	}
	return nil
}
