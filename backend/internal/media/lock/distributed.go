package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DistributedLock 基于 Redis 的分布式锁 (类似 Redisson)
// 用于 MD5 内容级去重
type DistributedLock struct {
	rdb *redis.Client
	key string
}

// NewDistributedLock 创建分布式锁
func NewDistributedLock(rdb *redis.Client, key string) *DistributedLock {
	return &DistributedLock{rdb: rdb, key: "lock:" + key}
}

// TryLock 尝试获取锁
// waitTime: 最长等待时间，-1 表示一直等待
// leaseTime: 锁的过期时间
func (l *DistributedLock) TryLock(waitTime, leaseTime time.Duration) (bool, error) {
	if l.rdb == nil {
		return true, nil // Redis 不可用时直接放行
	}

	ctx := context.Background()
	deadline := time.Now().Add(waitTime)

	for {
		// SET NX PX 实现加锁
		ok, err := l.rdb.SetNX(ctx, l.key, "locked", leaseTime).Result()
		if err != nil {
			return false, fmt.Errorf("获取锁失败: %w", err)
		}
		if ok {
			return true, nil
		}

		// 超时检查
		if waitTime >= 0 && time.Now().After(deadline) {
			return false, nil
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// Unlock 释放锁
func (l *DistributedLock) Unlock() error {
	if l.rdb == nil {
		return nil
	}
	return l.rdb.Del(context.Background(), l.key).Err()
}

// ForceUnlock 强制释放锁 (管理员操作)
func (l *DistributedLock) ForceUnlock() error {
	if l.rdb == nil {
		return nil
	}
	return l.rdb.Del(context.Background(), l.key).Err()
}
