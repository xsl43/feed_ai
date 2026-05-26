package video

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"feedsystem_ai_go/internal/account"
	"feedsystem_ai_go/internal/apierror"
	"feedsystem_ai_go/internal/middleware/jwt"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	service        *VideoService
	accountService *account.AccountService
}

func NewVideoHandler(service *VideoService, accountService *account.AccountService) *VideoHandler {
	return &VideoHandler{service: service, accountService: accountService}
}

func (vh *VideoHandler) PublishVideo(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	username, err := jwt.GetUsername(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	video := &Video{
		AuthorID:    authorId,
		Username:    username,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CoverURL:    req.CoverURL,
		CreateTime:  time.Now(),
	}
	if err := vh.service.Publish(c.Request.Context(), video); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}

func (vh *VideoHandler) UploadVideo(c *gin.Context) {
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	// 1. 扩展名白名单
	allowedExts := map[string]bool{".mp4": true, ".mov": true, ".avi": true, ".mkv": true, ".webm": true}
	ext := strings.ToLower(filepath.Ext(f.Filename))
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的视频格式，仅允许: mp4, mov, avi, mkv, webm"})
		return
	}

	// 2. 文件大小（从 config 读取）
	cfg := vh.service.GetReviewConfig()
	maxSize := int64(cfg.MaxVideoSizeMB) << 20
	if maxSize <= 0 {
		maxSize = 500 << 20 // 兜底
	}
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("文件大小超限，最大 %dMB", cfg.MaxVideoSizeMB)})
		return
	}

	// 3. 保存文件
	date := time.Now().Format("20060102")
	relDir := filepath.Join("videos", fmt.Sprintf("%d", authorId), date)
	root := filepath.Join(".run", "uploads")
	absDir := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDir, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. 魔数校验
	if !validateMagicBytes(absPath, ext) {
		os.Remove(absPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件格式伪装：扩展名与实际内容不符"})
		return
	}

	// 5. ffprobe 流完整性 + 时长校验
	probe, err := probeVideo(absPath)
	if err != nil {
		os.Remove(absPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("视频文件无效: %v", err)})
		return
	}
	if probe.Duration < float64(cfg.MinVideoDurationSec) || probe.Duration > float64(cfg.MaxVideoDurationSec) {
		os.Remove(absPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("视频时长需在 %ds ~ %ds 之间", cfg.MinVideoDurationSec, cfg.MaxVideoDurationSec)})
		return
	}

	urlPath := path.Join("/static", "videos", fmt.Sprintf("%d", authorId), date, filename)
	c.JSON(http.StatusOK, gin.H{
		"url":      buildAbsoluteURL(c, urlPath),
		"play_url": buildAbsoluteURL(c, urlPath),
	})
}

func (vh *VideoHandler) UploadCover(c *gin.Context) {
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	// 1. 扩展名白名单
	ext := strings.ToLower(filepath.Ext(f.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图片格式，仅允许: jpg, jpeg, png, webp, gif"})
		return
	}

	// 2. 文件大小（从 config 读取）
	cfg := vh.service.GetReviewConfig()
	maxSize := int64(cfg.MaxCoverSizeMB) << 20
	if maxSize <= 0 {
		maxSize = 20 << 20
	}
	if f.Size <= 0 || f.Size > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("封面大小超限，最大 %dMB", cfg.MaxCoverSizeMB)})
		return
	}

	// 3. 保存
	date := time.Now().Format("20060102")
	relDir := filepath.Join("covers", fmt.Sprintf("%d", authorId), date)
	root := filepath.Join(".run", "uploads")
	absDir := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	filename = filename + ext
	absPath := filepath.Join(absDir, filename)

	if err := c.SaveUploadedFile(f, absPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. 魔数校验
	if !validateImageMagicBytes(absPath, ext) {
		os.Remove(absPath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片格式伪装：扩展名与实际内容不符"})
		return
	}

	urlPath := path.Join("/static", "covers", fmt.Sprintf("%d", authorId), date, filename)
	c.JSON(http.StatusOK, gin.H{
		"url":       buildAbsoluteURL(c, urlPath),
		"cover_url": buildAbsoluteURL(c, urlPath),
	})
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func buildAbsoluteURL(c *gin.Context, p string) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if xf := c.GetHeader("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, p)
}

// videoMagicBytes maps video format extensions to magic byte patterns
var videoMagicBytes = map[string][]byte{
	".mp4":  {0x00, 0x00, 0x00},
	".mov":  {0x00, 0x00, 0x00},
	".avi":  {0x52, 0x49, 0x46, 0x46},
	".mkv":  {0x1A, 0x45, 0xDF, 0xA3},
	".webm": {0x1A, 0x45, 0xDF, 0xA3},
}

// validateMagicBytes reads first 16 bytes of file and verifies magic bytes match extension
func validateMagicBytes(filePath string, ext string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 16)
	n, err := f.Read(header)
	if err != nil || n < 12 {
		return false
	}

	switch ext {
	case ".mp4", ".mov":
		return bytes.Equal(header[4:8], []byte("ftyp"))
	case ".avi":
		return bytes.HasPrefix(header, []byte("RIFF"))
	case ".mkv", ".webm":
		return bytes.HasPrefix(header, []byte{0x1A, 0x45, 0xDF, 0xA3})
	default:
		return true
	}
}

// imageMagicBytes maps image format extensions to magic byte patterns
var imageMagicBytes = map[string][]byte{
	".jpg":  {0xFF, 0xD8, 0xFF},
	".jpeg": {0xFF, 0xD8, 0xFF},
	".png":  {0x89, 0x50, 0x4E, 0x47},
	".webp": {0x52, 0x49, 0x46, 0x46},
	".gif":  {0x47, 0x49, 0x46, 0x38},
}

// validateImageMagicBytes verifies image file header magic bytes
func validateImageMagicBytes(filePath string, ext string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 12)
	n, err := f.Read(header)
	if err != nil || n < 4 {
		return false
	}

	switch ext {
	case ".jpg", ".jpeg":
		return header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF
	case ".png":
		return bytes.HasPrefix(header, []byte{0x89, 0x50, 0x4E, 0x47})
	case ".webp":
		return bytes.HasPrefix(header, []byte("RIFF")) &&
			bytes.Contains(header[8:12], []byte("WEBP"))
	case ".gif":
		return bytes.HasPrefix(header, []byte("GIF8"))
	default:
		return true
	}
}

// videoProbe holds metadata extracted by ffprobe
type videoProbe struct {
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	Codec    string  `json:"codec"`
}

// probeVideo uses ffprobe to validate video stream integrity and return metadata
func probeVideo(filePath string) (*videoProbe, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration:stream=codec_type,codec_name,width,height",
		"-of", "json",
		filePath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe 解析失败: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("ffprobe 输出解析失败: %w", err)
	}

	probe := &videoProbe{}
	if d, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
		probe.Duration = d
	}

	for _, s := range result.Streams {
		if s.CodecType == "video" {
			probe.Width = s.Width
			probe.Height = s.Height
			probe.Codec = s.CodecName
			break
		}
	}

	if probe.Codec == "" {
		return nil, fmt.Errorf("未找到视频流")
	}
	return probe, nil
}

func (vh *VideoHandler) DeleteVideo(c *gin.Context) {
	var req DeleteVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	authorId, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.Delete(c.Request.Context(), req.ID, authorId); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "video deleted"})
}

func (vh *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req ListByAuthorIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	videos, err := vh.service.ListByAuthorID(c.Request.Context(), req.AuthorID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if videos == nil {
		videos = []Video{}
	}
	c.JSON(200, videos)
}

func (vh *VideoHandler) GetDetail(c *gin.Context) {
	var req GetDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	viewerID, _ := jwt.GetAccountID(c)
	video, err := vh.service.GetDetail(c.Request.Context(), req.ID, viewerID)
	if err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}

func (vh *VideoHandler) UpdateLikesCount(c *gin.Context) {
	var req UpdateLikesCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	if err := vh.service.UpdateLikesCount(c.Request.Context(), req.ID, req.LikesCount); err != nil {
		c.JSON(apierror.ClassifyHTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "likes count updated"})
}
