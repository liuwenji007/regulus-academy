package domain

import "context"

type buildProgressKey struct{}

// BuildProgressReporter 建树流程阶段上报（由 API 层实现并写入 job 表）
type BuildProgressReporter interface {
	ReportPhase(phase, message string)
}

// WithBuildProgress 在 context 中挂载进度上报器
func WithBuildProgress(ctx context.Context, r BuildProgressReporter) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, buildProgressKey{}, r)
}

// ReportBuildProgress 上报当前阶段（无 reporter 时为 no-op）
func ReportBuildProgress(ctx context.Context, phase, message string) {
	if ctx == nil {
		return
	}
	r, _ := ctx.Value(buildProgressKey{}).(BuildProgressReporter)
	if r == nil {
		return
	}
	r.ReportPhase(phase, message)
}
