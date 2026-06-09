package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func chdirToCoachRoot(t *testing.T) *Prompter {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			if err := os.Chdir(d); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(wd) })
			break
		}
	}
	p, err := NewPrompter()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func sampleInput() PromptInput {
	return PromptInput{
		DomainName: "Go 并发",
		Node: &domain.NodeSpec{
			Node:           "channel 通信",
			Key:            "channel",
			Layer:          "熟悉",
			CoreConcepts:   []string{"无缓冲 channel"},
			CommonMistakes: []string{"忘记 close"},
			Boundaries:     []string{"不讲 select"},
			ExerciseIdeas:  []string{"deadlock 题"},
			GradingHints:   []string{"应提到阻塞条件"},
		},
		NodeKey:         "channel",
		Layer:           "熟悉",
		Phase:           "explain",
		TaskInstruction: "请讲解。",
		UserProfile:     "有 Python 基础",
	}
}

func TestBuildMessages_SystemStableContextInUser(t *testing.T) {
	p := chdirToCoachRoot(t)
	in := sampleInput()
	msgs := p.BuildMessages(in, TaskExplainQA, "")

	if len(msgs) < 2 {
		t.Fatalf("expected system + user, got %d messages", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Fatal("first message should be system")
	}
	if strings.Contains(msgs[0].Content, "【领域】") {
		t.Fatal("system should not contain dynamic context")
	}
	if !strings.Contains(msgs[len(msgs)-1].Content, "【领域】") {
		t.Fatal("user message should contain context")
	}
	if !strings.Contains(msgs[len(msgs)-1].Content, "【任务】") {
		t.Fatal("user message should contain task instruction")
	}
}

func TestBuildContext_TaskExerciseIncludesExerciseIdeas(t *testing.T) {
	in := sampleInput()
	ctx := buildContext(in, TaskExercise)
	if !strings.Contains(ctx, "【出题参考】") {
		t.Fatal("exercise task should include exercise ideas")
	}
	if strings.Contains(ctx, "【学生画像】") {
		t.Fatal("exercise task should not include profile")
	}
}

func TestBuildContext_TaskExplainOmitsExerciseIdeas(t *testing.T) {
	in := sampleInput()
	ctx := buildContext(in, TaskExplainQA)
	if strings.Contains(ctx, "【出题参考】") {
		t.Fatal("explain task should omit exercise ideas")
	}
	if !strings.Contains(ctx, "【学生画像】") {
		t.Fatal("explain task should include profile")
	}
}

func TestBuildContext_TaskGradeIncludesGradingHints(t *testing.T) {
	in := sampleInput()
	ctx := buildContext(in, TaskGrade)
	if !strings.Contains(ctx, "【评分要点】") {
		t.Fatal("grade task should include grading hints")
	}
	if strings.Contains(ctx, "后续节点再学") {
		t.Fatal("grade task should omit boundaries")
	}
}

func TestBuildContext_TaskGradeIncludesChoiceVerdict(t *testing.T) {
	in := sampleInput()
	in.Exercise = &storage.ExerciseContext{
		Question:     "以下哪项？",
		AnswerFormat: "choice",
		Choices:      []string{"a", "b"},
		ChoiceMode:   "single",
	}
	in.ChoiceGradeVerdict = &ChoiceGradeVerdict{
		Passed:         true,
		UserLetters:    []rune{'B'},
		CorrectLetters: []rune{'B'},
	}
	ctx := buildContext(in, TaskGrade)
	if !strings.Contains(ctx, "【系统判定】") || !strings.Contains(ctx, "判定：正确") {
		t.Fatalf("grade context should include choice verdict: %q", ctx)
	}
}

func TestBuildContext_TaskExplainOmitsGradingHints(t *testing.T) {
	in := sampleInput()
	ctx := buildContext(in, TaskExplainQA)
	if strings.Contains(ctx, "【评分要点】") {
		t.Fatal("explain task should omit grading hints")
	}
}

func TestBuildContext_ExplainedAndTeachingBeats(t *testing.T) {
	in := sampleInput()
	in.ExplainedConcepts = []string{"无缓冲 channel 的同步特性"}
	in.Node.TeachingBeats = []domain.ConceptBeat{{
		Concept:     "无缓冲 channel 的同步特性",
		MustTeach:   []string{"同步握手"},
		ContextType: "workplace",
	}}
	ctx := buildContext(in, TaskExercise)
	if !strings.Contains(ctx, "【已深讲】") || !strings.Contains(ctx, "【教学节拍】") {
		t.Fatalf("context: %s", ctx)
	}
}

func TestNewPrompterLoadsDeepenPhase(t *testing.T) {
	p := chdirToCoachRoot(t)
	if !strings.Contains(p.phases[TaskDeepen], "递进深讲") {
		t.Fatal("TaskDeepen should use phase_deepen.md")
	}
}

func TestNewPrompterLoadsReviewPhase(t *testing.T) {
	p := chdirToCoachRoot(t)
	if !strings.Contains(p.phases[TaskReview], "巩固答疑") {
		t.Fatal("TaskReview should use phase_review.md")
	}
	if !strings.Contains(p.phases[TaskProfileRefresh], "节末学生画像") {
		t.Fatal("TaskProfileRefresh should use phase_profile_refresh.md")
	}
}

func TestHistoryLimitByTask(t *testing.T) {
	if historyLimit(TaskMasteryCheck) != 12 {
		t.Fatal("mastery should use 12")
	}
	if historyLimit(TaskProfileRefresh) != 12 {
		t.Fatal("profile refresh should use 12")
	}
	if historyLimit(TaskGrade) != 4 {
		t.Fatal("grade should use 4")
	}
	if historyLimit(TaskExplainQA) != 8 {
		t.Fatal("explain should use 8")
	}
}

func TestTrimHistoryForTask(t *testing.T) {
	h := make([]llm.Message, 20)
	for i := range h {
		h[i] = llm.Message{Role: "user", Content: "x"}
	}
	trimmed := trimHistoryForTask(h, TaskGrade)
	if len(trimmed) != 4 {
		t.Fatalf("grade history should trim to 4, got %d", len(trimmed))
	}
}

func TestProgressSummaryTruncates(t *testing.T) {
	keys := []string{"a", "b", "c", "d", "e", "f", "g"}
	var progress []storage.UserProgress
	for _, k := range keys {
		progress = append(progress, storage.UserProgress{NodeKey: k, Status: "completed"})
	}
	summary := progressSummary(progress, "e")
	if strings.Contains(summary, "a") {
		t.Fatal("summary should truncate distant completed keys")
	}
	if !strings.Contains(summary, "e") {
		t.Fatal("summary should include current node key")
	}
}

func TestBuildContext_ConceptCoverageFacts(t *testing.T) {
	in := sampleInput()
	in.Node.CoreConcepts = []string{"概念甲", "概念乙", "概念丙"}
	in.TestedConcepts = []string{"概念甲"}
	ctx := buildContext(in, TaskExercise)
	if !strings.Contains(ctx, "【本会话已考查】") || !strings.Contains(ctx, "概念甲") {
		t.Fatal("should include tested concepts")
	}
	if !strings.Contains(ctx, "【待考查】") || !strings.Contains(ctx, "概念乙") {
		t.Fatal("should include uncovered concepts")
	}
}

func TestProgressSummaryCurrentKeyNotCompleted(t *testing.T) {
	keys := []string{"a", "b", "c", "d", "e", "f", "g"}
	var progress []storage.UserProgress
	for _, k := range keys {
		progress = append(progress, storage.UserProgress{NodeKey: k, Status: "completed"})
	}
	// 正在学 h（未完成），应展示最近 4 个已完成节点，而非列表开头 a,b,c
	summary := progressSummary(progress, "h")
	if strings.Contains(summary, "a") || strings.Contains(summary, "b") {
		t.Fatalf("不应返回最早节点，got %q", summary)
	}
	for _, want := range []string{"d", "e", "f", "g"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("应包含最近完成节点 %s，got %q", want, summary)
		}
	}
}
