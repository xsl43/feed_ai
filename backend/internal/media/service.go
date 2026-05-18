package media

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"feedsystem_ai_go/internal/config"
	"feedsystem_ai_go/internal/media/lock"
	"feedsystem_ai_go/internal/media/storage"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	chunkUploadKeyPrefix = "upload:chunked:"
	chunkUploadTTL       = 24 * time.Hour
)

// MediaService 媒体文件服务
type MediaService struct {
	db           *gorm.DB
	rdb          *redis.Client
	minioClient  *storage.MinIOClient
	mediaCfg     config.MediaConfig
}

func NewMediaService(db *gorm.DB, rdb *redis.Client, minioClient *storage.MinIOClient, mediaCfg config.MediaConfig) *MediaService {
	return &MediaService{
		db:          db,
		rdb:         rdb,
		minioClient: minioClient,
		mediaCfg:    mediaCfg,
	}
}

// InitChunkedUpload 初始化分片上传，返回 uploadId
func (s *MediaService) InitChunkedUpload() string {
	uploadID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), randomHex(8))
	redisKey := chunkUploadKeyPrefix + uploadID
	if s.rdb != nil {
		s.rdb.Set(context.Background(), redisKey, "INIT", chunkUploadTTL)
	}
	return uploadID
}

// UploadChunk 上传分片
// uploadID: 上传会话 ID
// chunkIndex: 分片序号从0开始
// data: 分片数据
func (s *MediaService) UploadChunk(uploadID string, chunkIndex int, data []byte) error {
	redisKey := chunkUploadKeyPrefix + uploadID + ":chunks"
	if s.rdb != nil {
		// 检查上传会话是否存在
		sessionKey := chunkUploadKeyPrefix + uploadID
		exists, _ := s.rdb.Exists(context.Background(), sessionKey).Result()
		if exists == 0 {
			return fmt.Errorf("上传会话不存在或已过期")
		}

		// 记录分片状态
		fieldName := fmt.Sprintf("chunk_%d", chunkIndex)
		s.rdb.HSet(context.Background(), redisKey, fieldName, "uploaded")
		s.rdb.Expire(context.Background(), redisKey, chunkUploadTTL)
	}

	return nil
}

// CompleteUpload 完成分片上传，合并分片并上传到 MinIO
func (s *MediaService) CompleteUpload(uploadID, filename string, chunks [][]byte, userID *uint, md5Hash string) (*MediaFileRecord, error) {
	// 1. MD5 去重锁
	if s.rdb != nil && md5Hash != "" {
		dedupLock := lock.NewDistributedLock(s.rdb, "dedup:"+md5Hash)
		acquired, err := dedupLock.TryLock(0, 30*time.Second)
		if err != nil || !acquired {
			return nil, fmt.Errorf("检测到重复文件 (MD5 相同)")
		}
		defer dedupLock.Unlock()
	}

	// 2. 合并分片到临时文件
	tmpPath := filepath.Join(s.mediaCfg.UploadDir, uploadID+"_"+filename)
	if err := os.MkdirAll(filepath.Dir(tmpPath), 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %w", err)
	}

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer tmpFile.Close()

	for _, chunk := range chunks {
		if _, err := tmpFile.Write(chunk); err != nil {
			os.Remove(tmpPath)
			return nil, fmt.Errorf("写入分片失败: %w", err)
		}
	}
	tmpFile.Close()

	// 3. 上传到 MinIO
	fileURL, err := s.minioClient.UploadLocalFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("MinIO 上传失败: %w", err)
	}

	// 4. 清理本地临时文件
	os.Remove(tmpPath)

	// 5. 清理 Redis 上传状态
	if s.rdb != nil {
		redisKey := chunkUploadKeyPrefix + uploadID
		chunksKey := redisKey + ":chunks"
		s.rdb.Del(context.Background(), redisKey, chunksKey)
	}

	// 6. 写入数据库
	record := &MediaFileRecord{
		UserID:     userID,
		Filename:   filename,
		FilePath:   fileURL,
		Status:     "COMPLETED",
		UploadTime: time.Now(),
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("数据库写入失败: %w", err)
	}

	log.Printf("✅ 分片上传完成: %s", filename)
	return record, nil
}

// UploadFile 简单上传（不分片，小文件直接用）
func (s *MediaService) UploadFile(fileHeader *multipart.FileHeader, userID *uint) (*MediaFileRecord, error) {
	fileURL, err := s.minioClient.UploadFile(fileHeader)
	if err != nil {
		return nil, err
	}

	record := &MediaFileRecord{
		UserID:     userID,
		Filename:   fileHeader.Filename,
		FilePath:   fileURL,
		FileSize:   fileHeader.Size,
		Status:     "COMPLETED",
		UploadTime: time.Now(),
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, fmt.Errorf("数据库写入失败: %w", err)
	}

	// 清除 Redis 缓存
	if s.rdb != nil {
		userIDStr := "anon"
		if userID != nil {
			userIDStr = fmt.Sprintf("%d", *userID)
		}
		s.rdb.Del(context.Background(), "media:list:user:"+userIDStr)
	}

	log.Printf("✅ 文件上传完成: %s → %s", fileHeader.Filename, fileURL)
	return record, nil
}

// ListByUser 获取用户的媒体文件列表 (Redis 缓存)
func (s *MediaService) ListByUser(ctx context.Context, userID *uint) ([]MediaFileRecord, error) {
	cacheKey := "media:list:user:"
	if userID == nil {
		cacheKey += "anon"
	} else {
		cacheKey += fmt.Sprintf("%d", *userID)
	}

	// 尝试从 Redis 缓存获取
	if s.rdb != nil {
		cached, err := s.rdb.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			// 实际项目中应反序列化 JSON
			// 这里简化为直接查库
			log.Printf("⚡ 命中 Redis 缓存")
		}
	}

	// 查询数据库
	var records []MediaFileRecord
	query := s.db.Order("upload_time DESC")
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	// 写入缓存（30分钟）
	if s.rdb != nil {
		// 简化：缓存记录数，实际应缓存 JSON
		s.rdb.Set(ctx, cacheKey, fmt.Sprintf("%d records", len(records)), 30*time.Minute)
	}

	return records, nil
}

// DeleteMedia 删除媒体文件
func (s *MediaService) DeleteMedia(mediaID, userID uint) error {
	var record MediaFileRecord
	if err := s.db.First(&record, mediaID).Error; err != nil {
		return fmt.Errorf("文件不存在")
	}

	// 权限检查
	if record.UserID != nil && *record.UserID != userID {
		return fmt.Errorf("无权删除他人的文件")
	}

	// 从 MinIO 删除
	if record.FilePath != "" {
		s.minioClient.RemoveFile(record.FilePath)
	}

	// 从数据库删除
	if err := s.db.Delete(&record).Error; err != nil {
		return err
	}

	// 清除缓存
	if s.rdb != nil {
		userIDStr := "anon"
		if record.UserID != nil {
			userIDStr = fmt.Sprintf("%d", *record.UserID)
		}
		s.rdb.Del(context.Background(), "media:list:user:"+userIDStr)
	}

	log.Printf("✅ 文件已删除: ID=%d", mediaID)
	return nil
}

// ComputeMD5 计算文件的 MD5 哈希
func ComputeMD5(reader io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func randomHex(n int) string {
	const letters = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

