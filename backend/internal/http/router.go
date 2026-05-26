package http

import (
	"context"
	"feedsystem_ai_go/internal/account"
	"feedsystem_ai_go/internal/config"
	"feedsystem_ai_go/internal/feed"
	"feedsystem_ai_go/internal/message"
	"feedsystem_ai_go/internal/middleware/admin"
	"feedsystem_ai_go/internal/middleware/jwt"
	"feedsystem_ai_go/internal/middleware/rabbitmq"
	"feedsystem_ai_go/internal/middleware/ratelimit"
	rediscache "feedsystem_ai_go/internal/middleware/redis"
	"feedsystem_ai_go/internal/social"
	"feedsystem_ai_go/internal/video"
	"feedsystem_ai_go/internal/worker"
	"log"
	"time"

	appai "feedsystem_ai_go/internal/ai"
	"feedsystem_ai_go/internal/media"
	mediastorage "feedsystem_ai_go/internal/media/storage"
	"feedsystem_ai_go/internal/review"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB, cache *rediscache.Client, rdb *redis.Client, rmq *rabbitmq.RabbitMQ, cfg config.Config) *gin.Engine {
	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Printf("SetTrustedProxies failed: %v", err)
	}
	if len(cfg.Server.AdminIDs) > 0 {
		admin.SetAdminIDs(cfg.Server.AdminIDs)
	}
	r.Static("/static", "./.run/uploads")
	// rate_limit
	loginLimiter := ratelimit.Limit(cache, "account_login", 10, time.Minute, ratelimit.KeyByIP)
	registerLimiter := ratelimit.Limit(cache, "account_register", 5, time.Hour, ratelimit.KeyByIP)

	likeLimiter := ratelimit.Limit(cache, "like_write", 30, time.Minute, ratelimit.KeyByAccount)
	commentLimiter := ratelimit.Limit(cache, "comment_write", 10, time.Minute, ratelimit.KeyByAccount)
	socialLimiter := ratelimit.Limit(cache, "social_write", 20, time.Minute, ratelimit.KeyByAccount)

	// account
	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository, cache)
	accountHandler := account.NewAccountHandler(accountService)
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", registerLimiter, accountHandler.CreateAccount)
		accountGroup.POST("/login", loginLimiter, accountHandler.Login)
		accountGroup.POST("/changePassword", accountHandler.ChangePassword)
		accountGroup.POST("/findByID", accountHandler.FindByID)
		accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
		accountGroup.POST("/refresh", accountHandler.Refresh)
	}
	protectedAccountGroup := accountGroup.Group("")
	protectedAccountGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedAccountGroup.POST("/logout", accountHandler.Logout)
		protectedAccountGroup.POST("/rename", accountHandler.Rename)
		protectedAccountGroup.POST("/uploadAvatar", accountHandler.UploadAvatar)
		protectedAccountGroup.POST("/updateProfile", accountHandler.UpdateProfile)
	}
	// review service - created early so it can be injected into video/comment/AI
	reviewCfg := review.ReviewConfig{
		Enabled:               cfg.Review.Enabled,
		TextModel:             cfg.Review.TextModel,
		VisionModel:           cfg.Review.VisionModel,
		SampleFrames:          cfg.Review.SampleFrames,
		FrameReviewMode:       cfg.Review.FrameReviewMode,
		ConfidenceThreshold:   cfg.Review.ConfidenceThreshold,
		ManualReviewThreshold: cfg.Review.ManualReviewThreshold,
		MaxRetries:            cfg.Review.MaxRetries,
		APIKey:                cfg.AI.APIKey,
		BaseURL:               cfg.AI.BaseURL,
		MaxVideoSizeMB:        cfg.Review.MaxVideoSizeMB,
		MaxCoverSizeMB:        cfg.Review.MaxCoverSizeMB,
		MaxVideoDurationSec:   cfg.Review.MaxVideoDurationSec,
		MinVideoDurationSec:   cfg.Review.MinVideoDurationSec,
		EnableAudioReview:     cfg.Review.EnableAudioReview,
		EnableOCRReview:       cfg.Review.EnableOCRReview,
		MaxConcurrentFrames:   cfg.Review.MaxConcurrentFrames,
		MaxConcurrentVideos:   cfg.Review.MaxConcurrentVideos,
	}
	reviewService := review.NewReviewService(reviewCfg)

	// video
	videoRepository := video.NewVideoRepository(db)
	popularityMQ, err := rabbitmq.NewPopularityMQ(rmq)
	if err != nil {
		log.Printf("PopularityMQ init failed (mq disabled): %v", err)
		popularityMQ = nil
	}
	videoService := video.NewVideoService(videoRepository, cache, popularityMQ)
	videoService.SetReviewService(reviewService)
	videoHandler := video.NewVideoHandler(videoService, accountService)
	videoGroup := r.Group("/video")
	{
		videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)
		videoGroup.POST("/getDetail", videoHandler.GetDetail)
	}
	protectedVideoGroup := videoGroup.Group("")
	protectedVideoGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedVideoGroup.POST("/uploadVideo", videoHandler.UploadVideo)
		protectedVideoGroup.POST("/uploadCover", videoHandler.UploadCover)
		protectedVideoGroup.POST("/publish", videoHandler.PublishVideo)
		protectedVideoGroup.POST("/delete", videoHandler.DeleteVideo)
	}
	// like
	likeMQ, err := rabbitmq.NewLikeMQ(rmq)
	if err != nil {
		log.Printf("LikeMQ init failed (mq disabled): %v", err)
		likeMQ = nil
	}
	likeRepository := video.NewLikeRepository(db)
	likeService := video.NewLikeService(likeRepository, videoRepository, cache, likeMQ, popularityMQ)
	likeHandler := video.NewLikeHandler(likeService)
	likeGroup := r.Group("/like")
	protectedLikeGroup := likeGroup.Group("")
	protectedLikeGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedLikeGroup.POST("/like", likeLimiter, likeHandler.Like)
		protectedLikeGroup.POST("/unlike", likeLimiter, likeHandler.Unlike)
		protectedLikeGroup.POST("/isLiked", likeHandler.IsLiked)
		protectedLikeGroup.POST("/listMyLikedVideos", likeHandler.ListMyLikedVideos)
	}
	// comment
	commentRepository := video.NewCommentRepository(db)
	commentMQ, err := rabbitmq.NewCommentMQ(rmq)
	if err != nil {
		log.Printf("CommentMQ init failed (mq disabled): %v", err)
		commentMQ = nil
	}
	commentService := video.NewCommentService(commentRepository, videoRepository, cache, commentMQ, popularityMQ)
	commentService.SetReviewService(reviewService)
	commentHandler := video.NewCommentHandler(commentService, accountService)
	commentGroup := r.Group("/comment")
	{
		commentGroup.POST("/listAll", commentHandler.GetAllComments)
	}
	protectedCommentGroup := commentGroup.Group("")
	protectedCommentGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedCommentGroup.POST("/publish", commentLimiter, commentHandler.PublishComment)
		protectedCommentGroup.POST("/delete", commentLimiter, commentHandler.DeleteComment)
	}
	// social
	socialMQ, err := rabbitmq.NewSocialMQ(rmq)
	if err != nil {
		log.Printf("SocialMQ init failed (mq disabled): %v", err)
		socialMQ = nil
	}
	socialRepository := social.NewSocialRepository(db)
	socialService := social.NewSocialService(socialRepository, accountRepository, socialMQ)
	socialHandler := social.NewSocialHandler(socialService)
	socialGroup := r.Group("/social")
	protectedSocialGroup := socialGroup.Group("")
	protectedSocialGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedSocialGroup.POST("/follow", socialLimiter, socialHandler.Follow)
		protectedSocialGroup.POST("/unfollow", socialLimiter, socialHandler.Unfollow)
		protectedSocialGroup.POST("/getAllFollowers", socialHandler.GetAllFollowers)
		protectedSocialGroup.POST("/getAllVloggers", socialHandler.GetAllVloggers)
		protectedSocialGroup.POST("/getCounts", socialHandler.GetCounts)
	}

	accountGroup.POST("/getProfile", func(c *gin.Context) {
		var req account.GetProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if req.AccountID == 0 {
			c.JSON(400, gin.H{"error": "account_id is required"})
			return
		}
		acc, err := accountService.FindByID(c.Request.Context(), req.AccountID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		videoCount, _ := videoRepository.CountByAuthor(c.Request.Context(), req.AccountID)
		totalLikes, _ := videoRepository.TotalLikesByAuthor(c.Request.Context(), req.AccountID)
		followerCount, _ := socialRepository.CountFollowers(c.Request.Context(), req.AccountID)
		vloggerCount, _ := socialRepository.CountVloggers(c.Request.Context(), req.AccountID)

		c.JSON(200, account.GetProfileResponse{
			Account:    account.FindByIDResponse{ID: acc.ID, Username: acc.Username, AvatarURL: acc.AvatarURL, Bio: acc.Bio},
			VideoCount: videoCount, TotalLikes: totalLikes,
			FollowerCount: followerCount, VloggerCount: vloggerCount,
		})
	})
	// feed
	feedRepository := feed.NewFeedRepository(db)
	feedService := feed.NewFeedService(feedRepository, likeRepository, cache)
	feedHandler := feed.NewFeedHandler(feedService)
	feedGroup := r.Group("/feed")
	feedGroup.Use(jwt.SoftJWTAuth(accountRepository, cache))
	{
		feedGroup.POST("/listLatest", feedHandler.ListLatest)
		feedGroup.POST("/listLikesCount", feedHandler.ListLikesCount)
		feedGroup.POST("/listByPopularity", feedHandler.ListByPopularity)
		feedGroup.POST("/listByTag", feedHandler.ListByTag)
	}
	protectedFeedGroup := feedGroup.Group("")
	protectedFeedGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedFeedGroup.POST("/listByFollowing", feedHandler.ListByFollowing)
	}
	// message
	messageRepo := message.NewRepository(db)
	messageService := message.NewService(messageRepo)
	messageHandler := message.NewHandler(messageService)
	messageGroup := r.Group("/message")
	protectedMessageGroup := messageGroup.Group("")
	protectedMessageGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		protectedMessageGroup.POST("/send", messageHandler.Send)
		protectedMessageGroup.POST("/list", messageHandler.List)
	}
	//worker
	timelineMQ, err := rabbitmq.NewTimelineMQ(rmq)
	if err != nil {
		log.Printf("timelineMQ init failed (mq disabled): %v", err)
		timelineMQ = nil
	}
	worker.StartOutboxPoller(db, timelineMQ)
	worker.StartConsumer(timelineMQ, "video.timeline.update.queue", cache)

	// SSE notification
	if rmq != nil && rmq.Ch != nil {
		rmq.DeclareTopic("like.events", "notification.like", "like.like")
		rmq.DeclareTopic("comment.events", "notification.comment", "comment.publish")
		rmq.DeclareTopic("social.events", "notification.social", "social.follow")
	}
	sseHub := worker.NewSSEHub(db)
	notifGroup := r.Group("/notification")
	notifGroup.Use(sseHub.SSERequireAuth())
	sseHub.RegisterRoutes(r, notifGroup)

	go func() {
		if rmq != nil && rmq.Ch != nil {
			hub := sseHub
			ctx := context.Background()
			// consume from like queue
			go func() {
				w := worker.NewNotificationWorker(rmq.Ch, db, "notification.like", hub)
				if err := w.Run(ctx); err != nil {
					log.Printf("notification-like worker: %v", err)
				}
			}()
			go func() {
				w := worker.NewNotificationWorker(rmq.Ch, db, "notification.comment", hub)
				if err := w.Run(ctx); err != nil {
					log.Printf("notification-comment worker: %v", err)
				}
			}()
			go func() {
				w := worker.NewNotificationWorker(rmq.Ch, db, "notification.social", hub)
				if err := w.Run(ctx); err != nil {
					log.Printf("notification-social worker: %v", err)
				}
			}()
		} else {
			log.Printf("Notification SSE disabled (MQ not available)")
		}
	}()

	// ========== AI 智能分析模块 ==========
	minioClient, minioErr := mediastorage.NewMinIOClient(cfg.MinIO)
	aiService := appai.NewAIService(cfg.AI, cfg.Media)
	aiHandler := appai.NewAIHandler(db, aiService, rdb)
	aiHandler.SetReviewService(reviewService)
	aiGroup := r.Group("/ai")
	aiGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		aiGroup.POST("/analyze", aiHandler.TriggerAnalysis)
		aiGroup.POST("/transcribe", aiHandler.TranscribeOnly)
		aiGroup.POST("/summarize", aiHandler.SummarizeText)
		aiGroup.GET("/status/:id", aiHandler.GetAnalysisStatus)
		aiGroup.GET("/audio/:id", aiHandler.DownloadAudio)
		aiGroup.GET("/config", aiHandler.GetConfig)
		aiGroup.POST("/config", aiHandler.UpdateConfig)
	}

	// ========== 媒体文件管理模块 ==========
	if minioErr == nil {
		mediaService := media.NewMediaService(db, rdb, minioClient, cfg.Media)
		mediaHandler := media.NewMediaHandler(mediaService, rdb)
		mediaGroup := r.Group("/media")
		mediaGroup.Use(jwt.JWTAuth(accountRepository, cache))
		{
			mediaGroup.POST("/init-upload", mediaHandler.InitUpload)
			mediaGroup.POST("/upload", mediaHandler.Upload)
			mediaGroup.POST("/upload-chunk", mediaHandler.UploadChunk)
			mediaGroup.POST("/complete-upload", mediaHandler.CompleteChunkUpload)
			mediaGroup.GET("/list", mediaHandler.List)
			mediaGroup.DELETE("/delete", mediaHandler.Delete)
			mediaGroup.POST("/check-duplicate", mediaHandler.CheckDuplicate)
		}
	} else {
		log.Printf("MinIO not available, media upload disabled: %v", minioErr)
	}

	// ========== 内容审核模块 ==========
	reviewHandler := NewReviewHandler(db, reviewService, videoService)
	reviewGroup := r.Group("/review")
	reviewGroup.Use(jwt.JWTAuth(accountRepository, cache))
	{
		reviewGroup.GET("/config", reviewHandler.GetReviewConfig)
		reviewGroup.GET("/status/:videoId", reviewHandler.GetVideoReviewStatus)
		reviewGroup.POST("/resubmit", reviewHandler.ReSubmitVideo)
	}
	// 人工审核端点（需要管理员权限）
	adminReviewGroup := r.Group("/review")
	adminReviewGroup.Use(jwt.JWTAuth(accountRepository, cache))
	adminReviewGroup.Use(admin.RequireAdmin())
	{
		adminReviewGroup.POST("/config", reviewHandler.UpdateReviewConfig)
		adminReviewGroup.POST("/video/:id/approve", reviewHandler.ApproveVideo)
		adminReviewGroup.POST("/video/:id/reject", reviewHandler.RejectVideo)
		adminReviewGroup.GET("/pending", reviewHandler.GetPendingVideos)
	}

	// 事后复审 Worker (热门/举报/流量突增)
	if reviewService.IsEnabled() {
		reviewWorker := worker.NewReviewWorker(db, reviewService)
		reviewWorker.Start()
	}

	return r
}
