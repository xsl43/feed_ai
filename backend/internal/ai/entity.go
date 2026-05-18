package ai

import "time"

// AnalysisTaskMsg RocketMQ 消息体
type AnalysisTaskMsg struct {
	MediaID uint   `json:"media_id"`
	Action  string `json:"action"` // "START_ANALYSIS"
}

// MediaFile AI分析结果
type MediaFile struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         *uint      `gorm:"index" json:"user_id"`
	Filename       string     `json:"filename"`
	Status         string     `json:"status"` // UPLOADED, PROCESSING, COMPLETED, FAILED
	FilePath       string     `json:"file_path"`
	FileSize       int64      `json:"file_size"`
	AiSummary      string     `gorm:"type:text" json:"ai_summary"`
	TranscriptText string     `gorm:"type:text" json:"transcript_text"`
	CoverURL       string     `json:"cover_url"`
	UploadTime     time.Time  `json:"upload_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (MediaFile) TableName() string {
	return "media_files"
}
