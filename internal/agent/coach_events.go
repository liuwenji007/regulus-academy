package agent

import "context"

// CoachStreamStage 流式阶段（驱动前端真实进度文案）
type CoachStreamStage string

const (
	StageThinking CoachStreamStage = "thinking"
	StageGrading  CoachStreamStage = "grading"
	StageMastery  CoachStreamStage = "mastery"
	StageExercise CoachStreamStage = "exercise"
)

// CoachStreamEvent 教练流式事件（经 context sink 下发）
type CoachStreamEvent struct {
	Type  string           `json:"type"` // stage | delta
	Stage CoachStreamStage `json:"stage,omitempty"`
	Text  string           `json:"text,omitempty"`
}

// CoachEventSink 流式事件回调；nil 表示非流式路径
type CoachEventSink func(CoachStreamEvent)

type coachEventSinkKey struct{}

// WithCoachEventSink 注入流式事件 sink（Web SSE）；IM/旧端点不注入。
func WithCoachEventSink(ctx context.Context, sink CoachEventSink) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, coachEventSinkKey{}, sink)
}

// CoachEventSinkFromContext 取出 sink；无则返回 nil
func CoachEventSinkFromContext(ctx context.Context) CoachEventSink {
	if ctx == nil {
		return nil
	}
	if sink, ok := ctx.Value(coachEventSinkKey{}).(CoachEventSink); ok {
		return sink
	}
	return nil
}

func emitStage(ctx context.Context, stage CoachStreamStage) {
	if sink := CoachEventSinkFromContext(ctx); sink != nil {
		sink(CoachStreamEvent{Type: "stage", Stage: stage})
	}
}

func emitDelta(ctx context.Context, text string) {
	if text == "" {
		return
	}
	if sink := CoachEventSinkFromContext(ctx); sink != nil {
		sink(CoachStreamEvent{Type: "delta", Text: text})
	}
}
