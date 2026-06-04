package agent

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// CoachTask Coach 调用场景
type CoachTask string

const (
	TaskBegin        CoachTask = "begin"
	TaskExplainQA    CoachTask = "explain_qa"
	TaskRealWorld    CoachTask = "real_world"
	TaskExercise     CoachTask = "exercise"
	TaskGrade        CoachTask = "grade"
	TaskMasteryCheck CoachTask = "mastery_check"
	TaskReview       CoachTask = "review"
	TaskCompletedQA    CoachTask = "completed_qa"
	TaskProfileRefresh CoachTask = "profile_refresh"
)

// GenerationName Langfuse / OTel generation 名
func (t CoachTask) GenerationName() string {
	switch t {
	case TaskBegin:
		return "coach.begin"
	case TaskExplainQA:
		return "coach.explain_qa"
	case TaskRealWorld:
		return "coach.real_world"
	case TaskExercise:
		return "coach.exercise"
	case TaskGrade:
		return "coach.grade"
	case TaskMasteryCheck:
		return "coach.mastery_check"
	case TaskReview:
		return "coach.review"
	case TaskCompletedQA:
		return "coach.completed_qa"
	case TaskProfileRefresh:
		return "coach.profile_refresh"
	default:
		return "coach.unknown"
	}
}

// Prompter 拼装消息
type Prompter struct {
	core     string
	phases   map[CoachTask]string
}

// NewPrompter 创建 Prompter
func NewPrompter() (*Prompter, error) {
	core, err := domain.LoadPrompt("core")
	if err != nil {
		return nil, err
	}
	loadPhase := func(name string) (string, error) {
		return domain.LoadPrompt(name)
	}
	explain, err := loadPhase("phase_explain")
	if err != nil {
		return nil, err
	}
	exercise, err := loadPhase("phase_exercise")
	if err != nil {
		return nil, err
	}
	grade, err := loadPhase("phase_grade")
	if err != nil {
		return nil, err
	}
	mastery, err := loadPhase("phase_mastery")
	if err != nil {
		return nil, err
	}
	review, err := loadPhase("phase_review")
	if err != nil {
		return nil, err
	}
	profileRefresh, err := loadPhase("phase_profile_refresh")
	if err != nil {
		return nil, err
	}
	phases := map[CoachTask]string{
		TaskBegin:            explain,
		TaskExplainQA:        explain,
		TaskRealWorld:        explain,
		TaskReview:           review,
		TaskCompletedQA:      explain,
		TaskExercise:         exercise,
		TaskGrade:            grade,
		TaskMasteryCheck:     mastery,
		TaskProfileRefresh:   profileRefresh,
	}
	return &Prompter{core: core, phases: phases}, nil
}

// PromptInput 动态上下文
type PromptInput struct {
	DomainName          string
	Node                *domain.NodeSpec
	NodeKey             string
	Layer               string
	Progress            []storage.UserProgress
	Reinforce           *string
	Phase               string
	Exercise            *storage.ExerciseContext
	History             []llm.Message
	RecentMistakes      []string
	UserProfile         string
	PendingPrereqTitles []string
	TaskInstruction     string
	UserMessage         string
}

// BuildMessages 构建 LLM 消息列表
func (p *Prompter) BuildMessages(in PromptInput, task CoachTask, schemaJSON string) []llm.Message {
	system := p.core
	if phase := p.phases[task]; phase != "" {
		system += "\n\n" + phase
	}
	if schemaJSON != "" {
		system += "\n\n【输出格式】仅输出 JSON，不要 markdown 代码块：\n" + schemaJSON
	}

	var userParts []string
	if ctx := buildContext(in, task); ctx != "" {
		userParts = append(userParts, "【上下文】\n"+ctx)
	}
	if instr := strings.TrimSpace(in.TaskInstruction); instr != "" {
		userParts = append(userParts, "【任务】\n"+instr)
	}
	if msg := strings.TrimSpace(in.UserMessage); msg != "" {
		userParts = append(userParts, "【用户】\n"+msg)
	}

	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimHistoryForTask(in.History, task)...)
	if len(userParts) > 0 {
		msgs = append(msgs, llm.Message{Role: "user", Content: strings.Join(userParts, "\n\n")})
	}
	return msgs
}

func buildContext(in PromptInput, task CoachTask) string {
	var b strings.Builder
	fmt.Fprintf(&b, "【领域】%s\n", in.DomainName)

	if in.Node != nil {
		fmt.Fprintf(&b, "【当前节点】%s（%s）\n", in.Node.Node, in.Layer)
		if len(in.Node.CoreConcepts) > 0 {
			b.WriteString("【本节点】核心：")
			b.WriteString(strings.Join(in.Node.CoreConcepts, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.CommonMistakes) > 0 && includeMistakes(task) {
			b.WriteString("易混：")
			b.WriteString(strings.Join(in.Node.CommonMistakes, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.Boundaries) > 0 && includeBoundaries(task) {
			b.WriteString("后续节点再学：")
			b.WriteString(strings.Join(in.Node.Boundaries, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.ExerciseIdeas) > 0 && task == TaskExercise {
			b.WriteString("【出题参考】")
			b.WriteString(strings.Join(in.Node.ExerciseIdeas, "；"))
			b.WriteString("\n")
		}
		if len(in.Node.GradingHints) > 0 && includeGradingHints(task) {
			b.WriteString("【评分要点】")
			b.WriteString(strings.Join(in.Node.GradingHints, "；"))
			b.WriteString("\n")
		}
	}

	if len(in.RecentMistakes) > 0 && includeRecentMistakes(task) {
		fmt.Fprintf(&b, "【本次薄弱】%s\n", strings.Join(in.RecentMistakes, "；"))
	}

	if summary := progressSummary(in.Progress, in.NodeKey); summary != "" && includeProgress(task) {
		fmt.Fprintf(&b, "【进度】已完成：%s\n", summary)
	}

	if in.Reinforce != nil && *in.Reinforce != "" && task == TaskExercise {
		fmt.Fprintf(&b, "【可选巩固】%s（仅出题时使用，勿向用户提及）\n", *in.Reinforce)
	}

	if strings.TrimSpace(in.UserProfile) != "" && includeProfile(task) {
		fmt.Fprintf(&b, "【学生画像】%s\n", strings.TrimSpace(in.UserProfile))
	}

	if len(in.PendingPrereqTitles) > 0 && includePrereqs(task) {
		fmt.Fprintf(&b, "【前置未完成】用户尚未点亮：%s。开场先用 1～2 句补必要背景，再进入本节点；勿指责或阻止学习。\n",
			strings.Join(in.PendingPrereqTitles, "、"))
	}

	fmt.Fprintf(&b, "【当前阶段】%s\n", in.Phase)

	if in.Exercise != nil && in.Exercise.Question != "" && includeExercise(task) {
		fmt.Fprintf(&b, "【当前练习题】%s\n", in.Exercise.Question)
		if in.Exercise.AnswerFormat != "" {
			fmt.Fprintf(&b, "【作答方式】%s\n", in.Exercise.AnswerFormat)
		}
	}
	return b.String()
}

func includeMistakes(task CoachTask) bool {
	switch task {
	case TaskGrade, TaskMasteryCheck, TaskExplainQA, TaskReview, TaskRealWorld, TaskCompletedQA, TaskBegin:
		return true
	default:
		return false
	}
}

func includeBoundaries(task CoachTask) bool {
	switch task {
	case TaskGrade:
		return false
	default:
		return true
	}
}

func includeGradingHints(task CoachTask) bool {
	return task == TaskGrade || task == TaskMasteryCheck
}

func includeRecentMistakes(task CoachTask) bool {
	switch task {
	case TaskExplainQA, TaskReview, TaskGrade, TaskMasteryCheck, TaskRealWorld, TaskCompletedQA, TaskProfileRefresh:
		return true
	default:
		return false
	}
}

func includeProgress(task CoachTask) bool {
	switch task {
	case TaskBegin, TaskMasteryCheck:
		return true
	default:
		return false
	}
}

func includeProfile(task CoachTask) bool {
	switch task {
	case TaskBegin, TaskExplainQA, TaskReview, TaskMasteryCheck, TaskRealWorld, TaskCompletedQA, TaskProfileRefresh:
		return true
	default:
		return false
	}
}

func includePrereqs(task CoachTask) bool {
	return task == TaskBegin
}

func includeExercise(task CoachTask) bool {
	return task == TaskGrade || task == TaskMasteryCheck
}

func progressSummary(progress []storage.UserProgress, currentKey string) string {
	if len(progress) == 0 {
		return ""
	}
	doneSet := make(map[string]struct{})
	var doneOrder []string
	for _, pr := range progress {
		if pr.Status == "completed" {
			if _, ok := doneSet[pr.NodeKey]; !ok {
				doneSet[pr.NodeKey] = struct{}{}
				doneOrder = append(doneOrder, pr.NodeKey)
			}
		}
	}
	if len(doneOrder) == 0 {
		return ""
	}
	if currentKey == "" || len(doneOrder) <= 4 {
		return strings.Join(doneOrder, ", ")
	}
	// 当前节点若在已完成列表中：前后各保留邻近 key；否则展示最近完成的节点
	idx := -1
	for i, k := range doneOrder {
		if k == currentKey {
			idx = i
			break
		}
	}
	if idx < 0 {
		const tail = 4
		if len(doneOrder) <= tail {
			return strings.Join(doneOrder, ", ")
		}
		return strings.Join(doneOrder[len(doneOrder)-tail:], ", ")
	}
	start := idx - 2
	if start < 0 {
		start = 0
	}
	end := idx + 3
	if end > len(doneOrder) {
		end = len(doneOrder)
	}
	return strings.Join(doneOrder[start:end], ", ")
}

func trimHistoryForTask(h []llm.Message, task CoachTask) []llm.Message {
	max := historyLimit(task)
	if len(h) <= max {
		return h
	}
	return h[len(h)-max:]
}

func historyLimit(task CoachTask) int {
	switch task {
	case TaskMasteryCheck, TaskProfileRefresh:
		return 12
	case TaskExercise, TaskRealWorld, TaskGrade:
		return 4
	default:
		return 8
	}
}
