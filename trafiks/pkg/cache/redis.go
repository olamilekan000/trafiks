//go:generate mockgen -source=redis.go -destination=../../tests/mocks/redis.go -package=mocks

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

type RedisClient interface {
	Cache
	Ping(ctx context.Context) (string, error)
	Connect(ctx context.Context, conf RedisConf) error
	Close() error
	GetClient() *redis.Client
}

type redisClient struct {
	client *redis.Client
	conf   RedisConf
}

func NewRedisClient() RedisClient {
	return &redisClient{}
}

func (r *redisClient) Connect(ctx context.Context, conf RedisConf) error {
	r.conf = conf

	r.client = redis.NewClient(&redis.Options{
		Addr:     conf.Host,
		Username: conf.Username,
		Password: conf.Password,
		DB:       conf.Db,
	})

	_, err := r.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %q", err)
	}

	return nil
}

func (r *redisClient) Ping(ctx context.Context) (string, error) {
	return r.client.Ping(ctx).Result()
}

func (r *redisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}

	return nil
}

func (r *redisClient) Get(ctx context.Context, key string) ([]byte, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (r *redisClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *redisClient) Delete(ctx context.Context, key string) error {
	if r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return r.client.Del(ctx, key).Err()
}

func (r *redisClient) Clear(ctx context.Context) error {
	if r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	// Use SCAN to find all keys matching the cache key pattern
	// Cache keys follow the pattern: method:path:query or method:path:query:hash
	// Pattern "*:*:*" matches cache keys (at least 2 colons)
	pattern := "*:*:*"
	var cursor uint64 = 0

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning cache keys: %w", err)
		}

		// Delete found keys in batches
		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("error deleting cache keys: %w", err)
			}
		}

		// Continue scanning if there are more keys
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

func (r *redisClient) GetClient() *redis.Client {
	return r.client
}

func CacheKey(method, url, query string) string {
	if query != "" {
		return method + ":" + url + ":" + query
	}
	return method + ":" + url
}

type RedisConf struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Db       int    `json:"db"`
}
