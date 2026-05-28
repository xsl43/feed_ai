package video

import "time"

type Video struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	AuthorID     uint      `gorm:"index;not null" json:"author_id"`
	Username     string    `gorm:"type:varchar(255);not null" json:"username"`
	Title        string    `gorm:"type:varchar(255);not null" json:"title"`
	Description  string    `gorm:"type:varchar(255);" json:"description,omitempty"`
	PlayURL      string    `gorm:"type:varchar(255);not null" json:"play_url"`
	CoverURL     string    `gorm:"type:varchar(255);not null" json:"cover_url"`
	CreateTime   time.Time `gorm:"autoCreateTime;index:idx_videos_create_time,sort:desc;index:idx_videos_popularity_time_id,priority:2,sort:desc" json:"create_time"`
	LikesCount   int64     `gorm:"column:likes_count;not null;default:0;index:idx_videos_likes_count_id,priority:1,sort:desc" json:"likes_count"`
	Popularity   int64     `gorm:"column:popularity;not null;default:0;index:idx_videos_popularity_time_id,priority:1,sort:desc" json:"popularity"`
	ReviewStatus     string  `gorm:"type:varchar(20);default:pending;index" json:"review_status"`
	ReviewReason     string  `gorm:"type:text" json:"review_reason,omitempty"`
	ReviewConfidence float64 `gorm:"type:decimal(5,4);default:0" json:"review_confidence,omitempty"`
	ReviewCategories string  `gorm:"type:varchar(255)" json:"review_categories,omitempty"`
	RetryCount       int     `gorm:"default:0" json:"retry_count,omitempty"`
	PlayCount        int64     `gorm:"column:play_count;not null;default:0" json:"play_count"`
	ReportCount      int       `gorm:"column:report_count;not null;default:0" json:"report_count"`
	LastReviewTime   *time.Time `gorm:"column:last_review_time" json:"last_review_time,omitempty"`
	ReviewPriority   int       `gorm:"column:review_priority;default:0" json:"review_priority,omitempty"`
	// Agent 审核追踪
	AgentTrace   string `gorm:"column:agent_trace;type:json" json:"agent_trace,omitempty"`
	AgentRounds  int    `gorm:"column:agent_rounds;default:0" json:"agent_rounds,omitempty"`
	AgentVerdict string `gorm:"column:agent_verdict;type:varchar(30)" json:"agent_verdict,omitempty"`
	Phase0Result string `gorm:"column:phase0_result;type:json" json:"phase0_result,omitempty"`
	Phase1Result string `gorm:"column:phase1_result;type:json" json:"phase1_result,omitempty"`
}

type PublishVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PlayURL     string `json:"play_url"`
	CoverURL    string `json:"cover_url"`
}

type DeleteVideoRequest struct {
	ID uint `json:"id"`
}

type ListByAuthorIDRequest struct {
	AuthorID uint `json:"author_id"`
}

type GetDetailRequest struct {
	ID uint `json:"id"`
}

type UpdateLikesCountRequest struct {
	ID         uint  `json:"id"`
	LikesCount int64 `json:"likes_count"`
}

type ReSubmitVideoRequest struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type OutboxMsg struct {
	ID         uint      `gorm:"primaryKey"`
	VideoID    uint      `gorm:"index"`
	EventType  string    `gorm:"type:varchar(50)"`
	CreateTime time.Time `gorm:"autoCreateTime"`
	Status     string    `gorm:"type:varchar(50);index"`
}
