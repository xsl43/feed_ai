package review

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const textReviewPrompt = `# Role
你是一个专业的内容安全审核员。你的任务是审核用户提交的文本内容是否违反平台规定。

# 审核标准
- 涉政：涉及政治敏感话题、攻击国家制度、分裂言论、领导人攻击等
- 色情：涉及色情描写、性暗示、低俗内容等
- 暴恐：涉及暴力、恐怖主义、血腥描述、违法活动教唆等
- 辱骂：人身攻击、侮辱性言论、网络暴力、仇恨言论等
- 广告：垃圾广告、恶意营销、引流二维码/链接、刷单等

# 输出要求
请严格按以下JSON格式输出，不要包含任何其他内容：
{"status":"approved或rejected","confidence":0.95,"reason":"审核理由","categories":["涉政","广告"]}`

const imageReviewPrompt = `# Role
你是一个专业的内容安全审核员。请审核这张图片是否包含违规内容。

# 审核标准
- 色情：裸露、性暗示、低俗姿势
- 暴力：血腥、暴力场景、武器展示
- 违规信息：二维码、广告、政治敏感符号、违法内容
- 恐怖：恐怖组织标识、令人不适的画面

# 输出要求
请严格按以下JSON格式输出，不要包含任何其他内容：
{"status":"approved或rejected","confidence":0.95,"reason":"审核理由","categories":["色情","暴力"]}`

type ReviewService struct {
	cfg       ReviewConfig
	client    *http.Client
	dfaFilter *DFAFilter
}

func NewReviewService(cfg ReviewConfig) *ReviewService {
	svc := &ReviewService{
		cfg:    cfg,
		client: &http.Client{Timeout: 60 * time.Second},
	}
	// DFA敏感词过滤器（文件不存在不阻塞启动）
	filter, err := NewDFAFilter("configs/sensitive_words.txt")
	if err == nil {
		svc.dfaFilter = filter
	} else {
		log.Printf("[Review] DFA敏感词库加载失败: %v", err)
	}
	return svc
}

func (s *ReviewService) IsEnabled() bool {
	return s.cfg.Enabled && s.cfg.APIKey != ""
}

func (s *ReviewService) UpdateConfig(cfg ReviewConfig) {
	s.cfg = cfg
}

func (s *ReviewService) GetConfig() ReviewConfig {
	return s.cfg
}

// SensitiveWordCheck 敏感词检测，返回命中的词列表
func (s *ReviewService) SensitiveWordCheck(text string) []string {
	if s.dfaFilter == nil {
		return nil
	}
	return s.dfaFilter.Check(text)
}

// Classify applies the confidence decision matrix to an AI review result.
// Returns the final status: "approved", "rejected", or "manual_review".
func (s *ReviewService) Classify(result *ReviewResult) string {
	if result == nil {
		return "manual_review"
	}
	cfg := s.cfg

	if result.Confidence >= cfg.ConfidenceThreshold {
		return result.Status
	}

	if result.Confidence >= cfg.ManualReviewThreshold {
		return "manual_review"
	}

	// Low confidence: reject if AI says rejected, otherwise manual
	if result.Status == "rejected" {
		return "rejected"
	}
	return "manual_review"
}

// ReviewTextWithRetry calls ReviewText with retry on failure.
func (s *ReviewService) ReviewTextWithRetry(title, content string) (*ReviewResult, error) {
	return s.reviewWithRetry(func() (*ReviewResult, error) {
		return s.ReviewText(title, content)
	})
}

// ReviewImageWithRetry calls ReviewImage with retry on failure.
func (s *ReviewService) ReviewImageWithRetry(imagePath string) (*ReviewResult, error) {
	return s.reviewWithRetry(func() (*ReviewResult, error) {
		return s.ReviewImage(imagePath)
	})
}

func (s *ReviewService) reviewWithRetry(fn func() (*ReviewResult, error)) (*ReviewResult, error) {
	var lastErr error
	for i := 0; i <= s.cfg.MaxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
		log.Printf("[Review] 第%d次尝试失败: %v", i+1, err)
		if i < s.cfg.MaxRetries {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	return nil, fmt.Errorf("审核重试%d次后仍失败: %w", s.cfg.MaxRetries, lastErr)
}

// ReviewText sends text content to LLM for moderation.
func (s *ReviewService) ReviewText(title, content string) (*ReviewResult, error) {
	input := title
	if content != "" {
		input = fmt.Sprintf("标题：%s\n内容：%s", title, content)
	}
	return s.callTextLLM(input, textReviewPrompt)
}

// ReviewImage sends an image file to a vision model for moderation.
func (s *ReviewService) ReviewImage(imagePath string) (*ReviewResult, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("读取图片失败: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(imagePath))
	mime := "image/jpeg"
	switch ext {
	case ".png":
		mime = "image/png"
	case ".webp":
		mime = "image/webp"
	case ".gif":
		mime = "image/gif"
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)

	return s.callVisionLLM(dataURL, imageReviewPrompt)
}

// ExtractFrames extracts N evenly-spaced frames from a video using FFmpeg.
func (s *ReviewService) ExtractFrames(videoPath, outputDir string, numFrames int) ([]string, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	outputPattern := filepath.Join(outputDir, "frame_%d.jpg")

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=1/30,select='not(mod(n\\,floor(30*N/%d)))'", numFrames),
		"-frames:v", fmt.Sprintf("%d", numFrames),
		"-q:v", "3",
		outputPattern,
	)
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ffmpeg启动失败: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return nil, fmt.Errorf("ffmpeg抽帧失败: %w", err)
		}
	case <-time.After(5 * time.Minute):
		cmd.Process.Kill()
		return nil, fmt.Errorf("ffmpeg抽帧超时")
	}

	var frames []string
	for i := 1; i <= numFrames; i++ {
		p := filepath.Join(outputDir, fmt.Sprintf("frame_%d.jpg", i))
		if _, err := os.Stat(p); err == nil {
			frames = append(frames, p)
		}
	}
	return frames, nil
}

// ReviewFrames reviews multiple video frames and returns the worst result.
func (s *ReviewService) ReviewFrames(framePaths []string) (*ReviewResult, error) {
	worst := &ReviewResult{Status: "approved", Confidence: 1.0}
	anySucceeded := false
	for _, fp := range framePaths {
		r, err := s.ReviewImage(fp)
		if err != nil {
			continue
		}
		anySucceeded = true
		if r.Status == "rejected" && r.Confidence > worst.Confidence {
			worst = r
		}
	}
	if !anySucceeded {
		return nil, fmt.Errorf("所有视频帧审核均失败 (共%d帧)", len(framePaths))
	}
	return worst, nil
}

// callTextLLM sends a text moderation request to the LLM.
func (s *ReviewService) callTextLLM(text, systemPrompt string) (*ReviewResult, error) {
	url := s.cfg.BaseURL + "/chat/completions"

	reqBody := map[string]interface{}{
		"model":       s.cfg.TextModel,
		"stream":      false,
		"max_tokens":  256,
		"temperature": 0,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("请审核以下内容：\n%s", text)},
		},
	}

	return s.doLLMCall(url, reqBody)
}

// callVisionLLM sends an image moderation request to the vision model.
func (s *ReviewService) callVisionLLM(imageDataURL, systemPrompt string) (*ReviewResult, error) {
	url := s.cfg.BaseURL + "/chat/completions"

	type imageContent struct {
		Type     string `json:"type"`
		ImageURL struct {
			URL string `json:"url"`
		} `json:"image_url,omitempty"`
		Text string `json:"text,omitempty"`
	}

	reqBody := map[string]interface{}{
		"model":       s.cfg.VisionModel,
		"stream":      false,
		"max_tokens":  256,
		"temperature": 0,
		"messages": []map[string]interface{}{
			{
				"role": "system",
				"content": []imageContent{
					{Type: "text", Text: systemPrompt},
				},
			},
			{
				"role": "user",
				"content": []imageContent{
					{Type: "image_url", ImageURL: struct{ URL string `json:"url"` }{URL: imageDataURL}},
					{Type: "text", Text: "请审核这张图片是否违规，输出JSON格式结果。"},
				},
			},
		},
	}

	return s.doLLMCall(url, reqBody)
}

func (s *ReviewService) doLLMCall(url string, reqBody interface{}) (*ReviewResult, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("审核API请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("审核API错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("审核API响应解析失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("审核API返回空结果")
	}

	content := result.Choices[0].Message.Content
	content = extractJSON(content)

	var review ReviewResult
	if err := json.Unmarshal([]byte(content), &review); err != nil {
		return &ReviewResult{
			Status:     "approved",
			Confidence: 0.5,
			Reason:     "审核结果解析失败，默认放行: " + content,
		}, nil
	}

	if review.Status != "approved" && review.Status != "rejected" {
		review.Status = "approved"
	}

	return &review, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "{"); idx >= 0 {
		s = s[idx:]
	}
	if idx := strings.LastIndex(s, "}"); idx >= 0 {
		s = s[:idx+1]
	}
	return s
}
