package channel

import (
	"sync"
	"time"
)

// PlatformHealthSnapshot 平台运行时健康状态
type PlatformHealthSnapshot struct {
	Connected   bool       `json:"connected"`
	LastEventAt *time.Time `json:"lastEventAt,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type platformHealthState struct {
	connected   bool
	lastEventAt *time.Time
	lastError   string
	updatedAt   time.Time
}

var (
	healthMu sync.RWMutex
	health   = map[string]*platformHealthState{}
)

func ensureHealth(platform string) *platformHealthState {
	if health[platform] == nil {
		health[platform] = &platformHealthState{}
	}
	return health[platform]
}

// SetPlatformConnected 更新平台连接状态
func SetPlatformConnected(platform string, connected bool) {
	healthMu.Lock()
	defer healthMu.Unlock()
	h := ensureHealth(platform)
	h.connected = connected
	h.updatedAt = time.Now().UTC()
}

// RecordPlatformEvent 记录收到平台事件
func RecordPlatformEvent(platform string) {
	healthMu.Lock()
	defer healthMu.Unlock()
	h := ensureHealth(platform)
	now := time.Now().UTC()
	h.lastEventAt = &now
	h.updatedAt = now
}

// RecordPlatformError 记录平台错误
func RecordPlatformError(platform string, errMsg string) {
	healthMu.Lock()
	defer healthMu.Unlock()
	h := ensureHealth(platform)
	h.lastError = errMsg
	h.updatedAt = time.Now().UTC()
}

// GetPlatformHealth 获取单个平台健康快照
func GetPlatformHealth(platform string) PlatformHealthSnapshot {
	healthMu.RLock()
	defer healthMu.RUnlock()
	h := health[platform]
	if h == nil {
		return PlatformHealthSnapshot{UpdatedAt: time.Now().UTC()}
	}
	return snapshotFrom(h)
}

// AllPlatformHealth 返回全部平台健康快照
func AllPlatformHealth() map[string]PlatformHealthSnapshot {
	healthMu.RLock()
	defer healthMu.RUnlock()
	out := make(map[string]PlatformHealthSnapshot, len(health))
	for k, h := range health {
		out[k] = snapshotFrom(h)
	}
	return out
}

func snapshotFrom(h *platformHealthState) PlatformHealthSnapshot {
	var last *time.Time
	if h.lastEventAt != nil {
		t := *h.lastEventAt
		last = &t
	}
	return PlatformHealthSnapshot{
		Connected:   h.connected,
		LastEventAt: last,
		LastError:   h.lastError,
		UpdatedAt:   h.updatedAt,
	}
}
