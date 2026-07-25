package main

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter interface {
	Allow(context.Context, string) (bool, int, error)
	Limit() int
	Window() time.Duration
}

type RedisRateLimiter struct {
	client redis.Scripter
	limit  int
	window time.Duration
}

var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

func NewRedisRateLimiter(client redis.Scripter, limit int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{client: client, limit: limit, window: window}
}

func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, int, error) {
	windowSeconds := max(1, int(l.window.Seconds()))
	count, err := rateLimitScript.Run(
		ctx,
		l.client,
		[]string{"rate_limit:shorten:" + key},
		windowSeconds,
	).Int()
	if err != nil {
		return false, 0, err
	}

	remaining := max(0, l.limit-count)
	return count <= l.limit, remaining, nil
}

func (l *RedisRateLimiter) Limit() int {
	return l.limit
}

func (l *RedisRateLimiter) Window() time.Duration {
	return l.window
}

func rateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()

		allowed, remaining, err := limiter.Allow(ctx, clientIP(c.Request))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "Rate limiter is unavailable"})
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(max(1, int(limiter.Window().Seconds()))))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"message": "Too many requests"})
			return
		}

		c.Next()
	}
}

func clientIP(request *http.Request) string {
	if forwardedIP := strings.TrimSpace(request.Header.Get("X-Real-IP")); forwardedIP != "" {
		return forwardedIP
	}

	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		return host
	}
	return request.RemoteAddr
}
