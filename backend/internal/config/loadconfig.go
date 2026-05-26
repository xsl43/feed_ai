package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server              ServerConfig        `yaml:"server"`
	Database            DatabaseConfig      `yaml:"database"`
	Redis               RedisConfig         `yaml:"redis"`
	RabbitMQ            RabbitMQConfig      `yaml:"rabbitmq"`
	ObservabilityConfig ObservabilityConfig `yaml:"observability"`
	AI                  AIConfig            `yaml:"ai"`
	Media               MediaConfig         `yaml:"media"`
	MinIO               MinIOConfig         `yaml:"minio"`
	Review              ReviewConfig        `yaml:"review"`
}

type ServerConfig struct {
	Port     int   `yaml:"port"`
	AdminIDs []uint `yaml:"admin_ids"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ObservabilityConfig struct {
	Pprof PprofConfig `yaml:"pprof"`
}
type PprofConfig struct {
	Enabled    bool   `yaml:"enabled"`
	ApiAddr    string `yaml:"api_addr"`
	WorkerAddr string `yaml:"worker_addr"`
}

// AI 相关配置 (DeepSeek + ASR)
type AIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
	ASRModel string `yaml:"asr_model"`
}

// 媒体处理配置
type MediaConfig struct {
	UploadDir    string `yaml:"upload_dir"`
	FFmpegPath   string `yaml:"ffmpeg_path"`
	YtDlpPath    string `yaml:"ytdlp_path"`
	MaxFileSizeMB int64 `yaml:"max_file_size_mb"`
}

// 内容审核配置
type ReviewConfig struct {
	Enabled               bool    `yaml:"enabled"`
	TextModel             string  `yaml:"text_model"`
	VisionModel           string  `yaml:"vision_model"`
	SampleFrames          int     `yaml:"sample_frames"`
	FrameReviewMode       string  `yaml:"frame_review_mode"`
	ConfidenceThreshold   float64 `yaml:"confidence_threshold"`
	ManualReviewThreshold float64 `yaml:"manual_review_threshold"`
	MaxRetries            int     `yaml:"max_retries"`
	MaxVideoSizeMB        int     `yaml:"max_video_size_mb"`
	MaxCoverSizeMB        int     `yaml:"max_cover_size_mb"`
	MaxVideoDurationSec   int     `yaml:"max_video_duration_sec"`
	MinVideoDurationSec   int     `yaml:"min_video_duration_sec"`
	EnableAudioReview     bool    `yaml:"enable_audio_review"`
	EnableOCRReview       bool    `yaml:"enable_ocr_review"`
	MaxConcurrentFrames   int     `yaml:"max_concurrent_frames"`
	MaxConcurrentVideos   int     `yaml:"max_concurrent_videos"`
}

// MinIO 对象存储配置
type MinIOConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

func Load(filename string) (Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", filename, err)
	}

	ApplyEnvOverrides(&cfg)
	return cfg, nil
}

func ApplyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("MYSQL_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("MYSQL_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("MYSQL_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("MYSQL_ROOT_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("MYSQL_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("MYSQL_DATABASE"); v != "" {
		cfg.Database.DBName = v
	}
	if v := os.Getenv("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := os.Getenv("REDIS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Redis.Port = port
		}
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if db, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = db
		}
	}
	if v := os.Getenv("RABBITMQ_HOST"); v != "" {
		cfg.RabbitMQ.Host = v
	}
	if v := os.Getenv("RABBITMQ_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.RabbitMQ.Port = port
		}
	}
	if v := os.Getenv("RABBITMQ_USER"); v != "" {
		cfg.RabbitMQ.Username = v
	}
	if v := os.Getenv("RABBITMQ_PASS"); v != "" {
		cfg.RabbitMQ.Password = v
	}
}

// bool用来表示是否使用了默认配置，true表示使用了默认配置
func LoadLocalDev(filename string) (Config, bool, error) {
	cfg, err := Load(filename)
	if err == nil {
		return cfg, false, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return DefaultLocalConfig(), true, nil
	}
	return Config{}, false, err
}

func DefaultLocalConfig() Config {
	cfg := Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "123456",
			DBName:   "feedsystem",
		},
		Redis: RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "123456",
			DB:       0,
		},
		RabbitMQ: RabbitMQConfig{
			Host:     "localhost",
			Port:     5672,
			Username: "admin",
			Password: "password123",
		},
		ObservabilityConfig: ObservabilityConfig{
			Pprof: PprofConfig{
				Enabled:    true,
				ApiAddr:    "localhost:6060",
				WorkerAddr: "localhost:6061",
			},
		},
		AI: AIConfig{
			APIKey:   "",
			BaseURL:  "https://api.siliconflow.cn/v1",
			Model:    "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B",
			ASRModel: "TeleAI/TeleSpeechASR",
		},
		Media: MediaConfig{
			UploadDir:     "./.run/uploads",
			FFmpegPath:    "ffmpeg",
			YtDlpPath:     "yt-dlp",
			MaxFileSizeMB: 2048,
		},
		MinIO: MinIOConfig{
			Endpoint:  "localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "media",
			UseSSL:    false,
		},
		Review: ReviewConfig{
			Enabled:               true,
			TextModel:             "deepseek-ai/DeepSeek-V3",
			VisionModel:           "Qwen/Qwen2.5-VL-32B-Instruct",
			SampleFrames:          5,
			FrameReviewMode:       "auto",
			ConfidenceThreshold:   0.7,
			ManualReviewThreshold: 0.5,
			MaxRetries:            3,
			MaxVideoSizeMB:        500,
			MaxCoverSizeMB:        20,
			MaxVideoDurationSec:   1800,
			MinVideoDurationSec:   15,
			EnableAudioReview:     true,
			EnableOCRReview:       true,
			MaxConcurrentFrames:   3,
			MaxConcurrentVideos:   10,
		},
	}
	ApplyEnvOverrides(&cfg)
	return cfg
}
