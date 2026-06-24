package cloud

import (
	"sync"
	"time"
)

type userCreateBucket struct {
	count   int
	resetAt time.Time
}

// UserCreateLimiter 每 IP 每日创建学习角色上限
type UserCreateLimiter struct {
	limit   int
	mu      sync.Mutex
	buckets map[string]*userCreateBucket
}

func NewUserCreateLimiter(perDay int) *UserCreateLimiter {
	if perDay <= 0 {
		perDay = 5
	}
	return &UserCreateLimiter{limit: perDay, buckets: make(map[string]*userCreateBucket)}
}

func (l *UserCreateLimiter) Allow(ip string) bool {
	if l == nil {
		return true
	}
	ip = normalizeIP(ip)
	now := time.Now().UTC()
	dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[ip]
	if !ok || now.After(b.resetAt) {
		l.buckets[ip] = &userCreateBucket{count: 1, resetAt: dayEnd}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

func normalizeIP(ip string) string {
	if ip == "" {
		return "unknown"
	}
	return ip
}
