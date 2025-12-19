//go:generate mockgen -source=redis.go -destination=../../tests/mocks/redis.go -package=mocks

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConf struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
	Db       int    `json:"db"`
}

type RedisClient interface {
	Ping(ctx context.Context) (string, error)
	Connect(ctx context.Context, conf RedisConf) error
	Delete(ctx context.Context, key string) error
	Close() error
	// Cache interface methods for proxy caching
	GetCache(ctx context.Context, key string) ([]byte, error)
	SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error
	DeleteCache(ctx context.Context, key string) error
	ClearCache(ctx context.Context) error
	// GetClient returns the underlying redis.Client for advanced operations
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

func (r *redisClient) Set(ctx context.Context, key string, value interface{}, expiration *time.Duration) error {
	var exp time.Duration
	if expiration != nil {
		exp = *expiration
	}

	return r.client.Set(ctx, key, value, exp).Err()
}

func (r *redisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *redisClient) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *redisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}

	return nil
}

// Cache interface implementation methods
// These methods implement the Cache interface for proxy caching
// They work with []byte for HTTP response caching

func (r *redisClient) GetCache(ctx context.Context, key string) ([]byte, error) {
	val, err := r.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (r *redisClient) SetCache(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiration := &ttl
	return r.Set(ctx, key, value, expiration)
}

func (r *redisClient) DeleteCache(ctx context.Context, key string) error {
	return r.Delete(ctx, key)
}

func (r *redisClient) ClearCache(ctx context.Context) error {
	// For Redis, we'd need to implement a pattern-based delete
	// For now, this is a no-op
	return nil
}

// AsCache converts a RedisClient to Cache interface
func AsCache(client RedisClient) Cache {
	return &redisCacheAdapter{client: client}
}

// redisCacheAdapter adapts RedisClient to Cache interface
type redisCacheAdapter struct {
	client RedisClient
}

func (r *redisCacheAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	return r.client.GetCache(ctx, key)
}

func (r *redisCacheAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return r.client.SetCache(ctx, key, value, ttl)
}

func (r *redisCacheAdapter) Delete(ctx context.Context, key string) error {
	return r.client.DeleteCache(ctx, key)
}

func (r *redisCacheAdapter) Clear(ctx context.Context) error {
	return r.client.ClearCache(ctx)
}

func (r *redisClient) GetClient() *redis.Client {
	return r.client
}
