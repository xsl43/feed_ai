package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	appai "feedsystem_ai_go/internal/ai"
	"feedsystem_ai_go/internal/review"
)

// ============================================================
// extract_more_frames — 追加抽帧审核
// ============================================================

type extractMoreFramesTool struct {
	reviewService *review.ReviewService
	videoPath     string
	frameDir      string
}

func (t *extractMoreFramesTool) Name() string { return "extract_more_frames" }
func (t *extractMoreFramesTool) Description() string {
	return "追加抽取N帧进行画面审核。适用场景：某帧可疑需要更多上下文。输入: {\"n\": 3}"
}

func (t *extractMoreFramesTool) Execute(ctx context.Context, input string) (string, error) {
	var params struct {
		N int `json:"n"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil || params.N < 1 {
		params.N = 3
	}
	if params.N > 10 {
		params.N = 10
	}

	frames, err := t.reviewService.ExtractFrames(t.videoPath, t.frameDir, params.N)
	if err != nil {
		return "", err
	}
	if len(frames) == 0 {
		return `{"status": "ok", "frames_extracted": 0, "message": "未能成功抽帧"}`, nil
	}

	result, err := t.reviewService.ReviewFrames(frames)
	if err != nil {
		return "", err
	}

	out, _ := json.Marshal(map[string]interface{}{
		"status":            "ok",
		"frames_extracted":  len(frames),
		"review_status":     result.Status,
		"review_confidence": result.Confidence,
		"review_reason":     result.Reason,
	})
	return string(out), nil
}

// ============================================================
// transcribe_audio — ASR语音转文字 + 文字审核
// ============================================================

type transcribeAudioTool struct {
	reviewService *review.ReviewService
	aiService     *appai.AIService
	videoPath     string
}

func (t *transcribeAudioTool) Name() string { return "transcribe_audio" }
func (t *transcribeAudioTool) Description() string {
	return "提取视频音频并ASR转文字，然后审核文字内容。适用场景：怀疑音频中有违规内容。输入: {}"
}

func (t *transcribeAudioTool) Execute(ctx context.Context, _ string) (string, error) {
	if t.aiService == nil {
		return "", fmt.Errorf("ASR服务不可用")
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	tmpDir, err := os.MkdirTemp("", "agent_asr_*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	mp3Path := filepath.Join(tmpDir, "audio.mp3")

	if err := extractAudioFFmpeg(t.videoPath, mp3Path); err != nil {
		return `{"status": "ok", "has_audio": false, "message": "视频无音频轨道"}`, nil
	}

	transcript, err := t.aiService.TranscribeAudio(mp3Path)
	if err != nil {
		return "", fmt.Errorf("ASR转录失败: %w", err)
	}
	if transcript == "" {
		return `{"status": "ok", "has_audio": false, "message": "视频无有效音频内容"}`, nil
	}

	result, err := t.reviewService.ReviewText(transcript, "")
	if err != nil {
		return "", err
	}

	out, _ := json.Marshal(map[string]interface{}{
		"status":            "ok",
		"transcript_length": len([]rune(transcript)),
		"review_status":     result.Status,
		"review_confidence": result.Confidence,
		"review_reason":     result.Reason,
	})
	return string(out), nil
}

func extractAudioFFmpeg(videoPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-y", "-i", videoPath,
		"-vn", "-acodec", "libmp3lame", "-q:a", "2",
		outputPath,
	)
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg启动失败: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Minute):
		cmd.Process.Kill()
		return fmt.Errorf("ffmpeg音频提取超时")
	}
}

// ============================================================
// ocr_frames — 对帧图片OCR提取文字后审核
// ============================================================

type ocrFramesTool struct {
	reviewService *review.ReviewService
	framePaths    []string
}

func (t *ocrFramesTool) Name() string { return "ocr_frames" }
func (t *ocrFramesTool) Description() string {
	return "对指定帧进行OCR文字提取和审核。适用场景：画面中有文字。输入: {\"frame_ids\": [1, 3]}"
}

func (t *ocrFramesTool) Execute(ctx context.Context, input string) (string, error) {
	var params struct {
		FrameIDs []int `json:"frame_ids"`
	}
	json.Unmarshal([]byte(input), &params)
	if len(params.FrameIDs) == 0 {
		params.FrameIDs = []int{1, 2}
	}

	var ocrTexts []string
	for _, id := range params.FrameIDs {
		idx := id - 1
		if idx < 0 || idx >= len(t.framePaths) {
			continue
		}
		fp := t.framePaths[idx]
		data, err := os.ReadFile(fp)
		if err != nil {
			continue
		}
		mimeType := http.DetectContentType(data)
		dataURL := "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)

		text, err := t.reviewService.ExtractTextFromFrame(dataURL)
		if err != nil {
			log.Printf("[OCR] 提取文字失败 %s: %v", fp, err)
			continue
		}
		if text != "" {
			ocrTexts = append(ocrTexts, text)
		}
	}

	if len(ocrTexts) == 0 {
		return `{"status": "ok", "ocr_results": 0, "message": "未检测到文字内容"}`, nil
	}

	combined := strings.Join(ocrTexts, "\n---\n")
	result, err := t.reviewService.ReviewText(combined, "")
	if err != nil {
		return "", err
	}

	out, _ := json.Marshal(map[string]interface{}{
		"status":           "ok",
		"ocr_results":      len(ocrTexts),
		"review_status":    result.Status,
		"review_confidence": result.Confidence,
		"review_reason":    result.Reason,
	})
	return string(out), nil
}

// ============================================================
// enlarge_frame — 单帧放大审核
// ============================================================

type enlargeFrameTool struct {
	reviewService *review.ReviewService
	framePaths    []string
}

func (t *enlargeFrameTool) Name() string { return "enlarge_frame" }
func (t *enlargeFrameTool) Description() string {
	return "对指定帧进行高精度审核。适用场景：某帧细节可疑。输入: {\"frame_id\": 1}"
}

func (t *enlargeFrameTool) Execute(ctx context.Context, input string) (string, error) {
	var params struct {
		FrameID int `json:"frame_id"`
	}
	if err := json.Unmarshal([]byte(input), &params); err != nil || params.FrameID < 1 {
		return "", fmt.Errorf("需要有效的 frame_id")
	}

	idx := params.FrameID - 1
	if idx < 0 || idx >= len(t.framePaths) {
		return "", fmt.Errorf("frame_id %d 超出范围 (共%d帧)", params.FrameID, len(t.framePaths))
	}

	f, err := os.Open(t.framePaths[idx])
	if err != nil {
		return "", fmt.Errorf("打开帧图片失败: %w", err)
	}
	defer f.Close()

	result, err := t.reviewService.ReviewImage(f)
	if err != nil {
		return "", err
	}

	out, _ := json.Marshal(map[string]interface{}{
		"status":           "ok",
		"frame_id":         params.FrameID,
		"review_status":    result.Status,
		"review_confidence": result.Confidence,
		"review_reason":    result.Reason,
	})
	return string(out), nil
}

// ============================================================
// review_with_better_model — 高精度模型复审全部维度
// ============================================================

type reviewBetterModelTool struct {
	reviewService *review.ReviewService
	title         string
	desc          string
	coverPath     string
	framePaths    []string
}

func (t *reviewBetterModelTool) Name() string { return "review_with_better_model" }
func (t *reviewBetterModelTool) Description() string {
	return "使用高精度模型重新审核全部维度。成本高，仅在普通审核存疑时使用。输入: {}"
}

func (t *reviewBetterModelTool) Execute(ctx context.Context, _ string) (string, error) {
	result, allResults := t.reviewService.ReviewAllDimensions(t.title, t.desc, t.coverPath, "", t.framePaths)
	out, _ := json.Marshal(map[string]interface{}{
		"status":    "ok",
		"verdict":   result.Status,
		"confidence": result.Confidence,
		"reason":    result.Reason,
		"details":   allResults,
	})
	return string(out), nil
}

// ============================================================
// full_review — 全部维度审核
// ============================================================

type fullReviewTool struct {
	reviewService *review.ReviewService
	title         string
	desc          string
	coverPath     string
	framePaths    []string
}

func (t *fullReviewTool) Name() string { return "full_review" }
func (t *fullReviewTool) Description() string {
	return "启用所有审核维度进行全面审核。成本高。输入: {}"
}

func (t *fullReviewTool) Execute(ctx context.Context, _ string) (string, error) {
	result, allResults := t.reviewService.ReviewAllDimensions(t.title, t.desc, t.coverPath, "", t.framePaths)
	out, _ := json.Marshal(map[string]interface{}{
		"status":    "ok",
		"verdict":   result.Status,
		"confidence": result.Confidence,
		"reason":    result.Reason,
		"details":   allResults,
	})
	return string(out), nil
}

// ============================================================
// mark_checked — 标记帧已确认无问题
// ============================================================

type markCheckedTool struct{}

func (t *markCheckedTool) Name() string { return "mark_checked" }
func (t *markCheckedTool) Description() string {
	return "标记帧已确认无问题（零成本，无API调用）。输入: {\"frame_ids\": [1]}"
}
func (t *markCheckedTool) Execute(_ context.Context, input string) (string, error) {
	var params struct {
		FrameIDs []int `json:"frame_ids"`
	}
	json.Unmarshal([]byte(input), &params)
	return fmt.Sprintf(`{"status": "ok", "marked_frames": %v}`, params.FrameIDs), nil
}

// ============================================================
// done — 结束审核（引擎拦截，不实际执行）
// ============================================================

type doneTool struct{}

func (t *doneTool) Name() string        { return "done" }
func (t *doneTool) Description() string { return "结束审核，输出最终判断" }
func (t *doneTool) Execute(_ context.Context, _ string) (string, error) {
	return "", nil
}

// ============================================================
// escalate — 升级人工审核（仅 Post-Review Agent 使用）
// ============================================================

type escalateTool struct{}

func (t *escalateTool) Name() string { return "escalate" }
func (t *escalateTool) Description() string {
	return "升级为人工审核。适用场景：无法确定需要人工介入。输入: {\"reason\": \"原因\"}"
}
func (t *escalateTool) Execute(_ context.Context, input string) (string, error) {
	return `{"status": "escalated", "message": "已升级至人工审核"}`, nil
}
