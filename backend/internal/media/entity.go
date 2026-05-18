package media

import (
	"time"
)

// MediaFileRecord 媒体文件数据库模型
type MediaFileRecord struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         *uint      `gorm:"index" json:"user_id"`
	Filename       string     `json:"filename"`
	FilePath       string     `json:"file_path"`
	FileSize       int64      `json:"file_size"`
	Status         string     `json:"status"` // UPLOADED, PROCESSING, COMPLETED, FAILED
	AiSummary      string     `gorm:"type:text" json:"ai_summary"`
	TranscriptText string     `gorm:"type:text" json:"transcript_text"`
	CoverURL       string     `json:"cover_url"`
	UploadTime     time.Time  `json:"upload_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (MediaFileRecord) TableName() string {
	return "media_files"
}
