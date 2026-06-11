package cloud

import (
	"sync"
	"time"
)

// LastSeenThrottler 同一用户 5 分钟内最多写一次 last_seen_at
type LastSeenThrottler struct {
	mu    sync.Mutex
	seen  map[string]time.Time
	gap   time.Duration
}

func NewLastSeenThrottler() *LastSeenThrottler {
	return &LastSeenThrottler{
		seen: make(map[string]time.Time),
		gap:  5 * time.Minute,
	}
}

func (t *LastSeenThrottler) ShouldTouch(userID string) bool {
	if t == nil {
		return true
	}
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	if last, ok := t.seen[userID]; ok && now.Sub(last) < t.gap {
		return false
	}
	t.seen[userID] = now
	return true
}
