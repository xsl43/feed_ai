package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"feedsystem_ai_go/internal/config"
)

// AIService 提供 DeepSeek AI 分析、ASR 语音转文字、FFmpeg 音频提取
type AIService struct {
	cfg      config.AIConfig
	mediaCfg config.MediaConfig
	client   *http.Client
	mu       sync.RWMutex
}

func NewAIService(aiCfg config.AIConfig, mediaCfg config.MediaConfig) *AIService {
	return &AIService{
		cfg:      aiCfg,
		mediaCfg: mediaCfg,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// GetConfig 返回当前 AI 配置（API Key 脱敏）
func (s *AIService) GetConfig() config.AIConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfg := s.cfg
	if len(cfg.APIKey) > 8 {
		cfg.APIKey = cfg.APIKey[:4] + "****" + cfg.APIKey[len(cfg.APIKey)-4:]
	}
	return cfg
}

// UpdateConfig 运行时更新 AI 配置
func (s *AIService) UpdateConfig(cfg config.AIConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.APIKey != "" {
		s.cfg.APIKey = cfg.APIKey
	}
	if cfg.BaseURL != "" {
		s.cfg.BaseURL = cfg.BaseURL
	}
	if cfg.Model != "" {
		s.cfg.Model = cfg.Model
	}
	if cfg.ASRModel != "" {
		s.cfg.ASRModel = cfg.ASRModel
	}
}

// getConfig 内部读取方法（不加锁，由调用方控制）
func (s *AIService) getConfigInternal() config.AIConfig {
	return s.cfg
}

// =========================================
// 视频 → 音频提取 (FFmpeg)
// =========================================

// ExtractAudio 从视频中提取 MP3 音频
func (s *AIService) ExtractAudio(videoPath, outputPath string) error {
	cmd := exec.Command(s.mediaCfg.FFmpegPath,
		"-y",
		"-i", videoPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-q:a", "2",
		outputPath,
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg 启动失败: %w", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ffmpeg 转换失败: %w", err)
		}
		return nil
	case <-time.After(15 * time.Minute):
		cmd.Process.Kill()
		return fmt.Errorf("ffmpeg 转换超时")
	}
}

// =========================================
// 语音转文字 (SiliconFlow ASR)
// =========================================

// TranscribeAudio 使用 SiliconFlow ASR 将音频转为文字
func (s *AIService) TranscribeAudio(audioPath string) (string, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("打开音频文件失败: %w", err)
	}
	defer file.Close()

	// 构建 multipart 请求
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	s.mu.RLock()
	baseURL := s.cfg.BaseURL
	apiKey := s.cfg.APIKey
	asrModel := s.cfg.ASRModel
	s.mu.RUnlock()

	// 添加 model 参数
	if err := writer.WriteField("model", asrModel); err != nil {
		return "", err
	}
	writer.Close()

	url := baseURL + "/audio/transcriptions"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ASR 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("ASR 返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ASR 响应解析失败: %w", err)
	}

	return result.Text, nil
}

// =========================================
// AI 智能总结 (DeepSeek via SiliconFlow)
// =========================================

const systemPrompt = `# Role
你是一位拥有认知心理学背景的资深信息架构师。你的专长是从杂乱的语音转录文本中提取高价值信息，并进行逻辑重构。

# Input Context
用户将提供一段由视频生成的语音识别（ASR）文本。文本可能包含口语废话、重复、语气词或识别错误。

# Goals
请忽略文本中的噪音，对内容进行深度降噪和逻辑精炼，最终输出一份结构清晰、语气专业的分析报告。

# Constraints
1. **必须**严格遵守下方的输出格式。
2. 语气保持客观、理性、犀利。
3. 如果文本内容过短或无意义，直接输出"无法提取有效信息"。
4. 禁止输出任何开场白或结束语（如"好的，我来分析..."），直接输出 Markdown 内容。

# Output Format (Markdown)
请严格按照以下模块输出：

## 核心摘要
（精简概括视频到底讲了什么，直击本质，全面贴切，但要一针见血地概括视频主旨。）

## 深度洞察
（提取 3-5 个核心观点，每个观点使用三级标题格式，如下所示：）

### 1. [这里提炼一个 4-8 字的强观点标题]
不要复述原话。请用专业的语言解释这个观点背后的逻辑、动因或对观众的启示。分析要犀利，直击本质。

### 2. [第二个强观点标题]
（此处填写对应的深度分析...）

### 3. [第三个强观点标题]
（此处填写对应的深度分析...）(后续标题和分析同理)

## 原始内容精选
> "引用视频中原本的最有价值的一句原话（修正错别字后）"
> "引用第二句有价值的原话"（如果有，不一定必须精选，后续同理，但原始内容精选最多三个）

## 🏷 领域标签
#标签1 #标签2 #标签3`

// Summarize 使用 DeepSeek 对文本进行智能总结
func (s *AIService) Summarize(content string) (string, error) {
	s.mu.RLock()
	baseURL := s.cfg.BaseURL
	apiKey := s.cfg.APIKey
	model := s.cfg.Model
	s.mu.RUnlock()

	url := baseURL + "/chat/completions"

	reqBody := map[string]interface{}{
		"model":  model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": "请对以下视频提取的文字进行总结，不需要废话，直接列出核心观点：\n" + content},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("AI 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("AI 返回错误 (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("AI 响应解析失败: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("AI 返回空结果")
	}

	return result.Choices[0].Message.Content, nil
}

// =========================================
// 视频全链路分析: 提取音频 → 语音转文字 → AI总结
// =========================================

// AnalyzeResult 分析结果
type AnalyzeResult struct {
	Transcript string
	Summary    string
}

// AnalyzeVideo 全链路分析
func (s *AIService) AnalyzeVideo(videoPath string) (*AnalyzeResult, error) {
	// 1. 生成临时 MP3 路径
	tmpDir := os.TempDir()
	mp3Path := filepath.Join(tmpDir, fmt.Sprintf("ai_temp_%d.mp3", time.Now().UnixNano()))
	defer os.Remove(mp3Path)

	fmt.Printf("🎵 [AI] 正在提取音频: %s\n", videoPath)

	// 2. 提取音频
	if err := s.ExtractAudio(videoPath, mp3Path); err != nil {
		return nil, fmt.Errorf("音频提取失败: %w", err)
	}

	// 3. 语音转文字
	fmt.Println("🎤 [AI] 正在语音转文字...")
	text, err := s.TranscribeAudio(mp3Path)
	if err != nil {
		return nil, fmt.Errorf("语音转文字失败: %w", err)
	}

	if strings.HasPrefix(text, "❌") {
		return &AnalyzeResult{Transcript: text, Summary: text}, nil
	}

	// 4. AI 总结
	fmt.Println("🤖 [AI] 正在生成智能总结...")
	summary, err := s.Summarize(text)
	if err != nil {
		return nil, fmt.Errorf("AI 总结失败: %w", err)
	}

	return &AnalyzeResult{
		Transcript: text,
		Summary:    summary,
	}, nil
}
