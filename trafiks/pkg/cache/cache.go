//go:generate mockgen -source=cache.go -destination=../../tests/mocks/cache.go -package=mocks

package cache

import (
	"context"
	"time"
)

// Cache interface for proxy caching
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// CacheKey generates a cache key from request details
func CacheKey(method, path, query string, bodyHash string) string {
	if bodyHash != "" {
		return method + ":" + path + ":" + query + ":" + bodyHash
	}
	return method + ":" + path + ":" + query
}
