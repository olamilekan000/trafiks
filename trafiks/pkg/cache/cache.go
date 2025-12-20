package cache

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CacheControl represents parsed Cache-Control header directives
type CacheControl struct {
	MaxAge         int  // max-age value in seconds, -1 if not present
	SMaxAge        int  // s-maxage value in seconds, -1 if not present
	NoCache        bool // no-cache directive present
	NoStore        bool // no-store directive present
	Private        bool // private directive present
	Public         bool // public directive present
	MustRevalidate bool // must-revalidate directive present
}

// ParseCacheControl parses Cache-Control header value
func ParseCacheControl(headerValue string) *CacheControl {
	cc := &CacheControl{
		MaxAge:  -1,
		SMaxAge: -1,
	}

	if headerValue == "" {
		return cc
	}

	directives := strings.Split(headerValue, ",")
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		parts := strings.SplitN(directive, "=", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))

		switch key {
		case "max-age":
			if len(parts) == 2 {
				if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					cc.MaxAge = val
				}
			}
		case "s-maxage":
			if len(parts) == 2 {
				if val, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					cc.SMaxAge = val
				}
			}
		case "no-cache":
			cc.NoCache = true
		case "no-store":
			cc.NoStore = true
		case "private":
			cc.Private = true
		case "public":
			cc.Public = true
		case "must-revalidate":
			cc.MustRevalidate = true
		}
	}

	return cc
}

func IsCacheable(resp *http.Response) bool {
	status := resp.StatusCode
	cacheableStatusCodes := map[int]bool{
		200: true,
		203: true,
		204: true,
		206: true,
		300: true,
		301: true,
		404: true,
		405: true,
		410: true,
		414: true,
		501: true,
	}

	if !cacheableStatusCodes[status] {
		return false
	}

	cacheControl := ParseCacheControl(resp.Header.Get("Cache-Control"))
	return !cacheControl.NoStore
}

// GetCacheTTL calculates TTL from Cache-Control or uses default if not present
func GetCacheTTL(resp *http.Response, defaultTTL time.Duration) time.Duration {
	cacheControl := ParseCacheControl(resp.Header.Get("Cache-Control"))

	if cacheControl.SMaxAge > 0 {
		return time.Duration(cacheControl.SMaxAge) * time.Second
	}

	if cacheControl.MaxAge > 0 {
		return time.Duration(cacheControl.MaxAge) * time.Second
	}

	if expires := resp.Header.Get("Expires"); expires != "" {
		if expTime, err := http.ParseTime(expires); err == nil {
			ttl := time.Until(expTime)
			if ttl > 0 {
				return ttl
			}
		}
	}

	return defaultTTL
}

// CacheEntry represents a cached HTTP response
type CacheEntry struct {
	Body        []byte `json:"body"`
	ContentType string `json:"content_type"`
	StatusCode  int    `json:"status_code"`
}

func (e *CacheEntry) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func DecodeCacheEntry(data []byte) (*CacheEntry, error) {
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}
