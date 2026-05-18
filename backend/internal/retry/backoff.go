package retry

import (
	"log"
	"time"
)

// WithBackoff 指数退避重试
// fn: 要执行的函数，返回 nil 表示成功
// maxRetries: 最大重试次数
// baseDelay: 初始延迟时间，每次重试翻倍
func WithBackoff(fn func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			delay := baseDelay * time.Duration(1<<(i-1))
			log.Printf("⏳ 第 %d/%d 次重试，等待 %v...", i, maxRetries, delay)
			time.Sleep(delay)
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("⚠️ 第 %d 次尝试失败: %v", i+1, err)
	}
	return lastErr
}

// WithBackoffMax 指数退避重试（带最大延迟限制）
func WithBackoffMax(fn func() error, maxRetries int, baseDelay, maxDelay time.Duration) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			delay := baseDelay * time.Duration(1<<(i-1))
			if delay > maxDelay {
				delay = maxDelay
			}
			log.Printf("⏳ 第 %d/%d 次重试，等待 %v...", i, maxRetries, delay)
			time.Sleep(delay)
		}

		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		log.Printf("⚠️ 第 %d 次尝试失败: %v", i+1, err)
	}
	return lastErr
}
