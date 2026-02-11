package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"
	"user_crud_jwt/internal/pkg/config"

	"github.com/redis/go-redis/v9"
)

// CacheService 缓存服务接口
type CacheService interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetWithTTL(ctx context.Context, key string, dest interface{}) (time.Duration, error)
	SetWithTTL(ctx context.Context, key string, value interface{}) error
	InvalidatePattern(ctx context.Context, pattern string) error
	GetMultiple(ctx context.Context, keys []string, dest interface{}) error
}

// RedisCache Redis 缓存实现
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache 创建 Redis 缓存服务
func NewRedisCache(client *redis.Client) CacheService {
	prefix := "go-progres:"
	if config.GlobalConfig.Server.Mode == "test" {
		prefix = "test:" + prefix
	}
	return &RedisCache{
		client: client,
		prefix: prefix,
	}
}

// getKey 获取完整的缓存键
func (c *RedisCache) getKey(key string) string {
	return c.prefix + key
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.getKey(key)
	val, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		return fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("cache unmarshal error: %w", err)
	}

	return nil
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := c.getKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	if err := c.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("cache set error: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.getKey(key)
	return c.client.Del(ctx, fullKey).Err()
}

// Exists 检查缓存是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.getKey(key)
	result, err := c.client.Exists(ctx, fullKey).Result()
	return result > 0, err
}

// GetWithTTL 获取缓存并返回剩余时间
func (c *RedisCache) GetWithTTL(ctx context.Context, key string, dest interface{}) (time.Duration, error) {
	fullKey := c.getKey(key)

	// 使用管道获取值和TTL
	pipe := c.client.Pipeline()
	getCmd := pipe.Get(ctx, fullKey)
	ttlCmd := pipe.TTL(ctx, fullKey)

	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("cache pipeline error: %w", err)
	}

	val, err := getCmd.Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("cache miss")
		}
		return 0, fmt.Errorf("cache get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return 0, fmt.Errorf("cache unmarshal error: %w", err)
	}

	ttl, _ := ttlCmd.Result()
	if ttl < 0 {
		ttl = 0
	}

	return ttl, nil
}

// SetWithTTL 设置缓存并使用默认TTL
func (c *RedisCache) SetWithTTL(ctx context.Context, key string, value interface{}) error {
	// 默认TTL为1小时
	return c.Set(ctx, key, value, time.Hour)
}

// InvalidatePattern 根据模式批量删除缓存
func (c *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	fullPattern := c.prefix + pattern
	keys, err := c.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		return fmt.Errorf("cache keys error: %w", err)
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

// GetMultiple 批量获取缓存
func (c *RedisCache) GetMultiple(ctx context.Context, keys []string, dest interface{}) error {
	if len(keys) == 0 {
		return nil
	}

	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.getKey(key)
	}

	vals, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return fmt.Errorf("cache mget error: %w", err)
	}

	// 将结果转换为JSON数组
	results := make([]interface{}, len(vals))
	for i, val := range vals {
		if val != nil {
			var v interface{}
			if err := json.Unmarshal([]byte(val.(string)), &v); err != nil {
				return fmt.Errorf("cache unmarshal error at index %d: %w", i, err)
			}
			results[i] = v
		}
	}

	// 将结果序列化到目标
	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

// MemoryCache 内存缓存实现（用于开发/测试）
type MemoryCache struct {
	data map[string]*cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache() CacheService {
	return &MemoryCache{
		data: make(map[string]*cacheItem),
	}
}

func (c *MemoryCache) getKey(key string) string {
	return "mem:" + key
}

func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullKey := c.getKey(key)
	item, exists := c.data[fullKey]
	if !exists || time.Now().After(item.expiration) {
		return fmt.Errorf("cache miss")
	}

	data, err := json.Marshal(item.value)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullKey := c.getKey(key)
	c.data[fullKey] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}

	// 清理过期项
	c.cleanup()
	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, c.getKey(key))
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullKey := c.getKey(key)
	item, exists := c.data[fullKey]
	if !exists {
		return false, nil
	}

	if time.Now().After(item.expiration) {
		delete(c.data, fullKey)
		return false, nil
	}

	return true, nil
}

func (c *MemoryCache) GetWithTTL(ctx context.Context, key string, dest interface{}) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fullKey := c.getKey(key)
	item, exists := c.data[fullKey]
	if !exists {
		return 0, fmt.Errorf("cache miss")
	}

	if time.Now().After(item.expiration) {
		delete(c.data, fullKey)
		return 0, fmt.Errorf("cache miss")
	}

	ttl := time.Until(item.expiration)

	data, err := json.Marshal(item.value)
	if err != nil {
		return 0, fmt.Errorf("cache marshal error: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return 0, fmt.Errorf("cache unmarshal error: %w", err)
	}

	return ttl, nil
}

func (c *MemoryCache) SetWithTTL(ctx context.Context, key string, value interface{}) error {
	return c.Set(ctx, key, value, time.Hour)
}

func (c *MemoryCache) InvalidatePattern(ctx context.Context, pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	fullPattern := c.getKey(pattern)
	for key := range c.data {
		if matched, _ := filepath.Match(fullPattern, key); matched {
			delete(c.data, key)
		}
	}

	return nil
}

func (c *MemoryCache) GetMultiple(ctx context.Context, keys []string, dest interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make([]interface{}, len(keys))
	for i, key := range keys {
		fullKey := c.getKey(key)
		if item, exists := c.data[fullKey]; exists && !time.Now().After(item.expiration) {
			results[i] = item.value
		}
	}

	data, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("cache marshal error: %w", err)
	}

	return json.Unmarshal(data, dest)
}

func (c *MemoryCache) cleanup() {
	now := time.Now()
	for key, item := range c.data {
		if now.After(item.expiration) {
			delete(c.data, key)
		}
	}
}
