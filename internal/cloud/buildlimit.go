package cloud

// BuildLimiter 限制全局并行建课 goroutine 数量
type BuildLimiter struct {
	sem chan struct{}
}

func NewBuildLimiter(max int) *BuildLimiter {
	if max <= 0 {
		max = 3
	}
	return &BuildLimiter{sem: make(chan struct{}, max)}
}

func (l *BuildLimiter) TryAcquire() bool {
	if l == nil {
		return true
	}
	select {
	case l.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l *BuildLimiter) Release() {
	if l == nil {
		return
	}
	select {
	case <-l.sem:
	default:
	}
}
