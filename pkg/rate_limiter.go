package rate_limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(key string) bool
	AllowN(key string, n int) bool
	Reset(key string)
}

// RedisRateLimiter Redis实现的限流器
type RedisRateLimiter struct {
	client *redis.Client
	window time.Duration
	limit  int
}

// LocalRateLimiter 本地内存限流器
type LocalRateLimiter struct {
	// 每个key的限流状态
	limiter map[string]*TokenBucket
	mu      sync.RWMutex
}

// TokenBucket 令牌桶算法实现
type TokenBucket struct {
	capacity   int       // 桶容量
	tokens     int       // 当前令牌数
	refillRate int       // 每秒补充令牌数
	lastRefill time.Time // 上次补充时间
	mu         sync.Mutex
}

// NewRedisRateLimiter 创建Redis限流器
func NewRedisRateLimiter(client *redis.Client, window time.Duration, limit int) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		window: window,
		limit:  limit,
	}
}

// NewLocalRateLimiter 创建本地限流器
func NewLocalRateLimiter() *LocalRateLimiter {
	return &LocalRateLimiter{
		limiter: make(map[string]*TokenBucket),
	}
}

// Allow 检查是否允许请求
func (r *RedisRateLimiter) Allow(key string) bool {
	ctx := context.Background()
	current := time.Now().Unix()

	// 使用滑动窗口算法
	pipe := r.client.Pipeline()

	// 清理过期的记录
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", current-int64(r.window.Seconds())))

	// 添加当前请求
	z := &redis.Z{Score: float64(current), Member: fmt.Sprintf("%d", current)}
	pipe.ZAdd(ctx, key, z)

	// 获取窗口内的请求数
	pipe.ZCard(ctx, key)

	// 设置过期时间
	pipe.Expire(ctx, key, r.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true // Redis出错时允许通过
	}

	// 获取当前窗口内的请求数
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return true
	}

	return int64(count) <= int64(r.limit)
}

// AllowN 检查是否允许N个请求
func (r *RedisRateLimiter) AllowN(key string, n int) bool {
	ctx := context.Background()
	current := time.Now().Unix()

	pipe := r.client.Pipeline()

	// 清理过期记录
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", current-int64(r.window.Seconds())))

	// 添加N个当前请求
	for i := 0; i < n; i++ {
		z := &redis.Z{Score: float64(current), Member: fmt.Sprintf("%d", current+int64(i))}
		pipe.ZAdd(ctx, key, z)
	}

	// 获取窗口内的请求数
	pipe.ZCard(ctx, key)

	// 设置过期时间
	pipe.Expire(ctx, key, r.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return true
	}

	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return true
	}

	return int64(count) <= int64(r.limit)
}

// Reset 重置限流器
func (r *RedisRateLimiter) Reset(key string) {
	ctx := context.Background()
	r.client.Del(ctx, key)
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// refill 补充令牌
func (tb *TokenBucket) refill() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tokensToAdd := int(elapsed * float64(tb.refillRate))

	tb.tokens += tokensToAdd
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	tb.lastRefill = now
}

// Take 尝试获取令牌
func (tb *TokenBucket) Take(n int) bool {
	tb.refill()

	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// Allow 检查是否允许请求
func (r *LocalRateLimiter) Allow(key string) bool {
	return r.AllowN(key, 1)
}

// AllowN 检查是否允许N个请求
func (r *LocalRateLimiter) AllowN(key string, n int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	bucket, exists := r.limiter[key]
	if !exists {
		// 创建新的令牌桶：每秒10个令牌，桶容量100
		bucket = NewTokenBucket(100, 10)
		r.limiter[key] = bucket
	}

	return bucket.Take(n)
}

// Reset 重置限流器
func (r *LocalRateLimiter) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.limiter, key)
}

// SetRateLimit 为特定key设置限流参数
func (r *LocalRateLimiter) SetRateLimit(key string, capacity, refillRate int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.limiter[key] = NewTokenBucket(capacity, refillRate)
}
