package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// Cache 缓存接口
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(addr string, password string, db int) Cache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCache{client: client}
}

// Get 从缓存中获取数据
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set 设置缓存数据
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, data, expiration).Err()
}

// Delete 删除缓存数据
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// MemoryCache 内存缓存实现（用于测试和本地开发）
type MemoryCache struct {
	data map[string]cacheItem
}

type cacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache() Cache {
	return &MemoryCache{
		data: make(map[string]cacheItem),
	}
}

// Get 从缓存中获取数据
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	item, exists := c.data[key]
	if !exists {
		return fmt.Errorf("key not found")
	}
	if time.Now().After(item.expiration) {
		delete(c.data, key)
		return fmt.Errorf("key expired")
	}
	return json.Unmarshal(item.value, dest)
}

// Set 设置缓存数据
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.data[key] = cacheItem{
		value:      data,
		expiration: time.Now().Add(expiration),
	}
	return nil
}

// Delete 删除缓存数据
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}
