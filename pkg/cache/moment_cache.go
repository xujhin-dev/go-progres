package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// MomentCache Moment专用缓存
type MomentCache struct {
	client *redis.Client
}

// NewMomentCache 创建Moment缓存
func NewMomentCache(addr, password string) (*MomentCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &MomentCache{client: rdb}, nil
}

// Set 设置缓存
func (mc *MomentCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal cache value: %w", err)
	}

	return mc.client.Set(ctx, key, jsonData, expiration).Err()
}

// Get 获取缓存
func (mc *MomentCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := mc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		return fmt.Errorf("get cache: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("unmarshal cache value: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (mc *MomentCache) Delete(ctx context.Context, key string) error {
	return mc.client.Del(ctx, key).Err()
}

// DeletePattern 删除匹配模式的缓存
func (mc *MomentCache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := mc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("get cache keys: %w", err)
	}

	if len(keys) > 0 {
		return mc.client.Del(ctx, keys...).Err()
	}

	return nil
}

// 缓存键生成
func momentFeedKey(page int) string {
	return fmt.Sprintf("moment:feed:page:%d", page)
}

func momentDetailKey(momentID string) string {
	return fmt.Sprintf("moment:detail:%s", momentID)
}

func userMomentsKey(userID string) string {
	return fmt.Sprintf("user:moments:%s", userID)
}

func topicsKey() string {
	return "moment:topics:list"
}

// SetMomentFeed 设置动态列表缓存
func (mc *MomentCache) SetMomentFeed(ctx context.Context, page int, posts interface{}) error {
	key := momentFeedKey(page)
	return mc.Set(ctx, key, posts, 5*time.Minute)
}

// GetMomentFeed 获取动态列表缓存
func (mc *MomentCache) GetMomentFeed(ctx context.Context, page int, dest interface{}) error {
	key := momentFeedKey(page)
	return mc.Get(ctx, key, dest)
}

// SetMomentDetail 设置动态详情缓存
func (mc *MomentCache) SetMomentDetail(ctx context.Context, momentID string, post interface{}) error {
	key := momentDetailKey(momentID)
	return mc.Set(ctx, key, post, 10*time.Minute)
}

// GetMomentDetail 获取动态详情缓存
func (mc *MomentCache) GetMomentDetail(ctx context.Context, momentID string, dest interface{}) error {
	key := momentDetailKey(momentID)
	return mc.Get(ctx, key, dest)
}

// SetUserMoments 设置用户动态缓存
func (mc *MomentCache) SetUserMoments(ctx context.Context, userID string, posts interface{}) error {
	key := userMomentsKey(userID)
	return mc.Set(ctx, key, posts, 5*time.Minute)
}

// GetUserMoments 获取用户动态缓存
func (mc *MomentCache) GetUserMoments(ctx context.Context, userID string, dest interface{}) error {
	key := userMomentsKey(userID)
	return mc.Get(ctx, key, dest)
}

// SetTopics 设置话题缓存
func (mc *MomentCache) SetTopics(ctx context.Context, topics interface{}) error {
	key := topicsKey()
	return mc.Set(ctx, key, topics, 30*time.Minute)
}

// GetTopics 获取话题缓存
func (mc *MomentCache) GetTopics(ctx context.Context, dest interface{}) error {
	key := topicsKey()
	return mc.Get(ctx, key, dest)
}

// InvalidateUserCache 清除用户相关缓存
func (mc *MomentCache) InvalidateUserCache(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("user:moments:%s*", userID)
	return mc.DeletePattern(ctx, pattern)
}

// InvalidateMomentCache 清除动态相关缓存
func (mc *MomentCache) InvalidateMomentCache(ctx context.Context, momentID string) error {
	// 清除动态详情缓存
	detailKey := momentDetailKey(momentID)
	
	// 清除动态列表缓存（前几页）
	var keys []string
	for page := 1; page <= 3; page++ {
		feedKey := momentFeedKey(page)
		keys = append(keys, feedKey)
	}
	
	keys = append(keys, detailKey)
	
	for _, key := range keys {
		if err := mc.Delete(ctx, key); err != nil {
			return err
		}
	}
	
	return nil
}

// InvalidateTopicCache 清除话题相关缓存
func (mc *MomentCache) InvalidateTopicCache(ctx context.Context) error {
	topicKey := topicsKey()
	return mc.Delete(ctx, topicKey)
}

// Health 检查Redis健康状态
func (mc *MomentCache) Health(ctx context.Context) error {
	return mc.client.Ping(ctx).Err()
}
