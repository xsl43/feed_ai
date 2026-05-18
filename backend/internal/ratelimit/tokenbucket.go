package ratelimit

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBucket 基于 Redis 的令牌桶限流器
type TokenBucket struct {
	rdb    *redis.Client
	key    string
	rate   int           // 每秒/每分钟产生令牌数
	window time.Duration // 时间窗口
}

// NewTokenBucket 创建令牌桶
// key: Redis key
// rate: 每秒产生的令牌数 (如果 window=1*time.Minute 则代表每分钟)
// window: 时间窗口
func NewTokenBucket(rdb *redis.Client, key string, rate int, window time.Duration) *TokenBucket {
	return &TokenBucket{rdb: rdb, key: key, rate: rate, window: window}
}

// Allow 尝试获取1个令牌，成功返回 true
func (tb *TokenBucket) Allow() bool {
	if tb.rdb == nil {
		return true // Redis 不可用时限流放行
	}

	ctx := context.Background()
	now := time.Now().UnixNano()
	// 使用 sorted set 实现滑动窗口
	windowStart := now - tb.window.Nanoseconds()

	pipe := tb.rdb.Pipeline()

	// 删除过期令牌
	pipe.ZRemRangeByScore(ctx, tb.key, "0", formatScore(windowStart))

	// 统计窗口内令牌数
	pipe.ZCard(ctx, tb.key)

	// 添加当前请求的时间戳
	member := redis.Z{Score: float64(now), Member: now}

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return true // Redis 异常时限流放行
	}

	// cmds[0] = ZRemRangeByScore 结果, cmds[1] = ZCard 结果
	if cardCmd, ok := cmds[1].(*redis.IntCmd); ok {
		currentCount := cardCmd.Val()
		if int(currentCount) >= tb.rate {
			return false
		}
	}

	// 添加当前请求
	tb.rdb.ZAdd(ctx, tb.key, member)
	tb.rdb.Expire(ctx, tb.key, tb.window+time.Second)

	return true
}

func formatScore(n int64) string {
	return time.Unix(0, n).UTC().Format(time.RFC3339Nano)
}
