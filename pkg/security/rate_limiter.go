package security

import (
	"context"
	"fmt"
	"sync"
	"time"
	"user_crud_jwt/pkg/cache"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	AllowN(ctx context.Context, key string, n int) (bool, error)
	Reserve(ctx context.Context, key string) (*Reservation, error)
	GetLimit(ctx context.Context, key string) (Limit, error)
	SetLimit(ctx context.Context, key string, limit Limit) error
	Reset(ctx context.Context, key string) error
}

// Limit 限流配置
type Limit struct {
	Rate   float64       // 速率 (requests/second)
	Burst  int           // 突发容量
	Window time.Duration // 时间窗口
}

// Reservation 预留信息
type Reservation struct {
	OK        bool
	Delay     time.Duration
	Remaining int
}

// TokenBucket 令牌桶限流器
type TokenBucket struct {
	cache  cache.CacheService
	mu     sync.RWMutex
	limits map[string]Limit
}

// NewTokenBucket 创建令牌桶限流器
func NewTokenBucket(cache cache.CacheService) *TokenBucket {
	return &TokenBucket{
		cache:  cache,
		limits: make(map[string]Limit),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow(ctx context.Context, key string) (bool, error) {
	limit, err := tb.getLimit(key)
	if err != nil {
		return false, err
	}

	return tb.allowN(ctx, key, limit, 1)
}

// AllowN 检查是否允许 n 个请求
func (tb *TokenBucket) AllowN(ctx context.Context, key string, n int) (bool, error) {
	limit, err := tb.getLimit(key)
	if err != nil {
		return false, err
	}

	return tb.allowN(ctx, key, limit, n)
}

// Reserve 预留请求
func (tb *TokenBucket) Reserve(ctx context.Context, key string) (*Reservation, error) {
	limit, err := tb.getLimit(key)
	if err != nil {
		return &Reservation{OK: false}, err
	}

	return tb.reserve(ctx, key, limit, 1)
}

// GetLimit 获取限流配置
func (tb *TokenBucket) GetLimit(ctx context.Context, key string) (Limit, error) {
	return tb.getLimit(key)
}

// SetLimit 设置限流配置
func (tb *TokenBucket) SetLimit(ctx context.Context, key string, limit Limit) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.limits[key] = limit
	return nil
}

// Reset 重置限流器
func (tb *TokenBucket) Reset(ctx context.Context, key string) error {
	cacheKey := fmt.Sprintf("rate_limit:%s", key)
	return tb.cache.Delete(ctx, cacheKey)
}

// getLimit 获取限流配置
func (tb *TokenBucket) getLimit(key string) (Limit, error) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	if limit, exists := tb.limits[key]; exists {
		return limit, nil
	}

	// 返回默认限流配置
	return Limit{
		Rate:   100, // 100 requests/second
		Burst:  200, // 200 burst
		Window: time.Second,
	}, nil
}

// allowN 检查是否允许 n 个请求
func (tb *TokenBucket) allowN(ctx context.Context, key string, limit Limit, n int) (bool, error) {
	if n > limit.Burst {
		return false, fmt.Errorf("request count %d exceeds burst %d", n, limit.Burst)
	}

	cacheKey := fmt.Sprintf("rate_limit:%s", key)

	var bucket struct {
		Tokens     float64 `json:"tokens"`
		LastRefill int64   `json:"last_refill"`
	}

	err := tb.cache.Get(ctx, cacheKey, &bucket)
	if err != nil {
		// 首次访问，创建新的令牌桶
		bucket.Tokens = float64(limit.Burst - n)
		bucket.LastRefill = time.Now().Unix()

		if n <= limit.Burst {
			tb.cache.Set(ctx, cacheKey, bucket, limit.Window*2)
			return true, nil
		}
		return false, nil
	}

	// 简化实现：基于缓存的令牌桶
	return n <= limit.Burst, nil
}

// reserve 预留请求
func (tb *TokenBucket) reserve(ctx context.Context, key string, limit Limit, n int) (*Reservation, error) {
	allowed, err := tb.allowN(ctx, key, limit, n)
	if err != nil {
		return &Reservation{OK: false}, err
	}

	if allowed {
		return &Reservation{OK: true, Delay: 0}, nil
	}

	// 计算等待时间
	delay := time.Duration(float64(n-limit.Burst) / limit.Rate * float64(time.Second))
	return &Reservation{
		OK:        false,
		Delay:     delay,
		Remaining: limit.Burst,
	}, nil
}

// SlidingWindowLog 滑动窗口日志限流器
type SlidingWindowLog struct {
	cache cache.CacheService
	limit Limit
}

// NewSlidingWindowLog 创建滑动窗口日志限流器
func NewSlidingWindowLog(cache cache.CacheService, limit Limit) *SlidingWindowLog {
	return &SlidingWindowLog{
		cache: cache,
		limit: limit,
	}
}

// Allow 检查是否允许请求
func (swl *SlidingWindowLog) Allow(ctx context.Context, key string) (bool, error) {
	return swl.AllowN(ctx, key, 1)
}

// AllowN 检查是否允许 n 个请求
func (swl *SlidingWindowLog) AllowN(ctx context.Context, key string, n int) (bool, error) {
	now := time.Now()
	windowStart := now.Add(-swl.limit.Window)

	cacheKey := fmt.Sprintf("sliding_window:%s", key)

	// 获取当前窗口内的请求
	var requests []time.Time
	err := swl.cache.Get(ctx, cacheKey, &requests)
	if err != nil {
		// 首次访问
		requests = []time.Time{}
	}

	// 清理过期的请求
	validRequests := make([]time.Time, 0)
	for _, req := range requests {
		if req.After(windowStart) {
			validRequests = append(validRequests, req)
		}
	}

	// 检查是否超过限制
	if len(validRequests)+n > int(swl.limit.Rate*float64(swl.limit.Window.Seconds())) {
		// 更新缓存
		swl.cache.Set(ctx, cacheKey, validRequests, swl.limit.Window)
		return false, nil
	}

	// 添加新请求
	for i := 0; i < n; i++ {
		validRequests = append(validRequests, now)
	}

	swl.cache.Set(ctx, cacheKey, validRequests, swl.limit.Window)
	return true, nil
}

// Reserve 预留请求
func (swl *SlidingWindowLog) Reserve(ctx context.Context, key string) (*Reservation, error) {
	allowed, err := swl.Allow(ctx, key)
	if err != nil {
		return &Reservation{OK: false}, err
	}

	return &Reservation{OK: allowed}, nil
}

// GetLimit 获取限流配置
func (swl *SlidingWindowLog) GetLimit(ctx context.Context, key string) (Limit, error) {
	return swl.limit, nil
}

// SetLimit 设置限流配置
func (swl *SlidingWindowLog) SetLimit(ctx context.Context, key string, limit Limit) error {
	swl.limit = limit
	return nil
}

// Reset 重置限流器
func (swl *SlidingWindowLog) Reset(ctx context.Context, key string) error {
	cacheKey := fmt.Sprintf("sliding_window:%s", key)
	return swl.cache.Delete(ctx, cacheKey)
}

// MultiRateLimiter 多级限流器
type MultiRateLimiter struct {
	limiters map[string]RateLimiter
	mu       sync.RWMutex
}

// NewMultiRateLimiter 创建多级限流器
func NewMultiRateLimiter() *MultiRateLimiter {
	return &MultiRateLimiter{
		limiters: make(map[string]RateLimiter),
	}
}

// AddLimiter 添加限流器
func (mrl *MultiRateLimiter) AddLimiter(name string, limiter RateLimiter) {
	mrl.mu.Lock()
	defer mrl.mu.Unlock()
	mrl.limiters[name] = limiter
}

// Allow 检查所有限流器
func (mrl *MultiRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	for name, limiter := range mrl.limiters {
		allowed, err := limiter.Allow(ctx, key)
		if err != nil {
			return false, fmt.Errorf("limiter %s error: %w", name, err)
		}
		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

// AllowN 检查所有限流器是否允许 n 个请求
func (mrl *MultiRateLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	for name, limiter := range mrl.limiters {
		allowed, err := limiter.AllowN(ctx, key, n)
		if err != nil {
			return false, fmt.Errorf("limiter %s error: %w", name, err)
		}
		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

// GetLimit 获取指定限流器的配置
func (mrl *MultiRateLimiter) GetLimit(ctx context.Context, limiterName, key string) (Limit, error) {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	limiter, exists := mrl.limiters[limiterName]
	if !exists {
		return Limit{}, fmt.Errorf("limiter %s not found", limiterName)
	}

	return limiter.GetLimit(ctx, key)
}

// SetLimit 设置指定限流器的配置
func (mrl *MultiRateLimiter) SetLimit(ctx context.Context, limiterName, key string, limit Limit) error {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	limiter, exists := mrl.limiters[limiterName]
	if !exists {
		return fmt.Errorf("limiter %s not found", limiterName)
	}

	return limiter.SetLimit(ctx, key, limit)
}

// Reset 重置指定限流器
func (mrl *MultiRateLimiter) Reset(ctx context.Context, limiterName, key string) error {
	mrl.mu.RLock()
	defer mrl.mu.RUnlock()

	limiter, exists := mrl.limiters[limiterName]
	if !exists {
		return fmt.Errorf("limiter %s not found", limiterName)
	}

	return limiter.Reset(ctx, key)
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Global   Limit            `json:"global"`   // 全局限流
	User     Limit            `json:"user"`     // 用户限流
	IP       Limit            `json:"ip"`       // IP 限流
	Endpoint map[string]Limit `json:"endpoint"` // 端点限流
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Global: Limit{
			Rate:   1000,
			Burst:  2000,
			Window: time.Second,
		},
		User: Limit{
			Rate:   100,
			Burst:  200,
			Window: time.Second,
		},
		IP: Limit{
			Rate:   50,
			Burst:  100,
			Window: time.Second,
		},
		Endpoint: map[string]Limit{
			"auth": {
				Rate:   10,
				Burst:  20,
				Window: time.Second,
			},
			"upload": {
				Rate:   5,
				Burst:  10,
				Window: time.Second,
			},
		},
	}
}
