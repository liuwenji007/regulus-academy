package cloud

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipBucket struct {
	count   int
	resetAt time.Time
}

// RateLimiter 每 IP 每分钟请求上限
type RateLimiter struct {
	limit   int
	mu      sync.Mutex
	buckets map[string]*ipBucket
}

func NewRateLimiter(perMinute int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return &RateLimiter{limit: perMinute, buckets: make(map[string]*ipBucket)}
}

func (rl *RateLimiter) Allow(ip string) bool {
	if rl == nil {
		return true
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.buckets[ip]
	if !ok || now.After(b.resetAt) {
		rl.buckets[ip] = &ipBucket{count: 1, resetAt: now.Add(time.Minute)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

// Middleware 返回 cloud 模式 IP 限流中间件
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if !rl.Allow(clientIP(r)) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"请求过于频繁，请稍后再试"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
