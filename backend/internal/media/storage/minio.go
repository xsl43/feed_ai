package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"feedsystem_ai_go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient 封装 MinIO 操作
type MinIOClient struct {
	client     *minio.Client
	bucket     string
	endpoint   string
}

// NewMinIOClient 创建 MinIO 客户端
func NewMinIOClient(cfg config.MinIOConfig) (*MinIOClient, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("MinIO 连接失败: %w", err)
	}

	// 检查并创建 Bucket
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("MinIO Bucket 检查失败: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("MinIO Bucket 创建失败: %w", err)
		}
		log.Printf("✅ MinIO Bucket '%s' 已创建", cfg.Bucket)
	}

	return &MinIOClient{
		client:   client,
		bucket:   cfg.Bucket,
		endpoint: cfg.Endpoint,
	}, nil
}

// UploadFile 上传 multipart.FileHeader 到 MinIO，返回访问 URL
func (m *MinIOClient) UploadFile(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 生成唯一文件名
	ext := filepath.Ext(fileHeader.Filename)
	objectName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), randomString(8), ext)

	_, err = m.client.PutObject(context.Background(), m.bucket, objectName, file, fileHeader.Size, minio.PutObjectOptions{
		ContentType: fileHeader.Header.Get("Content-Type"),
	})
	if err != nil {
		return "", fmt.Errorf("MinIO 上传失败: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s", m.endpoint, m.bucket, objectName)
	log.Printf("✅ MinIO 上传成功: %s", url)
	return url, nil
}

// UploadReader 上传 io.Reader 到 MinIO
func (m *MinIOClient) UploadReader(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	_, err := m.client.PutObject(context.Background(), m.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("MinIO 上传失败: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s", m.endpoint, m.bucket, objectName)
	return url, nil
}

// UploadLocalFile 上传本地文件到 MinIO
func (m *MinIOClient) UploadLocalFile(localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()

	objectName := filepath.Base(localPath)
	stat, _ := file.Stat()
	size := stat.Size()

	_, err = m.client.PutObject(context.Background(), m.bucket, objectName, file, size, minio.PutObjectOptions{
		ContentType: "video/mp4",
	})
	if err != nil {
		return "", fmt.Errorf("MinIO 上传失败: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s", m.endpoint, m.bucket, objectName)
	log.Printf("✅ MinIO 本地上传成功: %s", url)
	return url, nil
}

// RemoveFile 从 MinIO 删除文件
func (m *MinIOClient) RemoveFile(fileURL string) error {
	objectName := fileURL[strings.LastIndex(fileURL, "/")+1:]
	err := m.client.RemoveObject(context.Background(), m.bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("MinIO 删除失败: %w", err)
	}
	log.Printf("🗑 MinIO 文件已删除: %s", objectName)
	return nil
}

// GetPresignedURL 生成预签名下载 URL
func (m *MinIOClient) GetPresignedURL(objectName string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedGetObject(context.Background(), m.bucket, objectName, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

