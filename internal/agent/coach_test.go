package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

type mockProvider struct {
	replies []string
	calls   int
}

func (m *mockProvider) Configured() bool { return true }
func (m *mockProvider) Name() string     { return "mock" }
func (m *mockProvider) Model() string    { return "mock" }

func (m *mockProvider) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	return m.ChatWithTemp(ctx, messages, 0.6)
}

func (m *mockProvider) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	if m.calls >= len(m.replies) {
		return "ok", nil
	}
	r := m.replies[m.calls]
	m.calls++
	return r, nil
}

func (m *mockProvider) ChatStream(ctx context.Context, messages []llm.Message, temp float64, onDelta func(string)) (string, error) {
	return llm.StreamViaChat(ctx, m.ChatWithTemp, messages, temp, onDelta)
}

func (m *mockProvider) ChatJSON(ctx context.Context, messages []llm.Message, temp float64, dest any) error {
	raw, err := m.ChatWithTemp(ctx, messages, temp)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), dest)
}

func (m *mockProvider) Ping(ctx context.Context) error { return nil }

type recordingMock struct {
	mockProvider
	lastMessages []llm.Message
}

func (m *recordingMock) ChatWithTemp(ctx context.Context, messages []llm.Message, temp float64) (string, error) {
	m.lastMessages = append([]llm.Message(nil), messages...)
	return m.mockProvider.ChatWithTemp(ctx, messages, temp)
}

func (m *recordingMock) lastUserContent() string {
	for i := len(m.lastMessages) - 1; i >= 0; i-- {
		if m.lastMessages[i].Role == "user" {
			return m.lastMessages[i].Content
		}
	}
	return ""
}

func setupCoachRecording(t *testing.T, replies ...string) (*Coach, *storage.Store, *storage.Session, *recordingMock) {
	t.Helper()
	t.Setenv("LANGFUSE_ENABLED", "false")
	chdirToRepo(t)
	store, err := storage.Open(filepath.Join(t.TempDir(), "coach_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	rec := &recordingMock{mockProvider: mockProvider{replies: replies}}
	coach, err := NewCoach(store, rec)
	if err != nil {
		t.Fatal(err)
	}

	reg := domain.NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	return coach, store, sess, rec
}

func chdirToRepo(t *testing.T) {
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
			return
		}
	}
	t.Fatal("找不到 regulus-coach 目录")
}

func setupCoach(t *testing.T, replies ...string) (*Coach, *storage.Store, *storage.Session) {
	t.Helper()
	t.Setenv("LANGFUSE_ENABLED", "false")
	chdirToRepo(t)
	store, err := storage.Open(filepath.Join(t.TempDir(), "coach_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	coach, err := NewCoach(store, &mockProvider{replies: replies})
	if err != nil {
		t.Fatal(err)
	}

	reg := domain.NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency", "go", tree, string(nodesJSON), storage.DomainSourceSkillPack, false, "")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(storage.DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", &storage.SessionContext{DomainSlug: "go-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	return coach, store, sess
}

func TestHandleMessageExerciseBackToExplain(t *testing.T) {
	coach, store, sess := setupCoach(t, "我们重新讲一下")
	sctx := storage.ParseSessionContext(sess)
	sctx.Exercise = &storage.ExerciseContext{
		Question:     "哪一学派强调自我实现？",
		AnswerFormat: "choice",
		Choices:      []string{"行为主义", "人本主义"},
	}
	_ = storage.SaveSessionContext(sess, sctx)
	sess.Phase = "exercise"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "不懂")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Exercise == nil {
		t.Fatal("expected exercise meta")
	}
}

func TestExerciseBackToExplainInjectsCurrentQuestion(t *testing.T) {
	coach, store, sess, rec := setupCoachRecording(t, "针对人本主义讲解")
	sctx := storage.ParseSessionContext(sess)
	sctx.Exercise = &storage.ExerciseContext{
		Question:     "哪一学派强调自我实现与人本潜能？",
		AnswerFormat: "choice",
		Choices:      []string{"行为主义", "人本主义"},
	}
	_ = storage.SaveSessionContext(sess, sctx)
	sess.Phase = "exercise"
	_ = store.UpdateSession(sess)

	_, err := coach.HandleMessage(context.Background(), sess, "不懂，回讲解")
	if err != nil {
		t.Fatal(err)
	}
	if rec.calls != 1 {
		t.Fatalf("calls=%d", rec.calls)
	}
	last := rec.lastUserContent()
	if !strings.Contains(last, "【当前练习题】哪一学派强调自我实现与人本潜能？") {
		t.Fatalf("prompt missing current exercise: %s", last)
	}
	if !strings.Contains(last, "不要回头讲已答对的上一题") {
		t.Fatalf("prompt missing focus instruction: %s", last)
	}
}

func TestHandleMessageStartExerciseCodeFill(t *testing.T) {
	exerciseJSON := `{"question":"写一个 goroutine","question_type":"code_fill","answer_format":"json","reinforced_concepts":["goroutine 是 Go 的轻量级并发执行单元"]}`
	coach, store, sess := setupCoach(t, exerciseJSON)

	result, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Content == "" {
		t.Fatal("期望有题目内容")
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("源码补全应规范为 text, meta=%+v", result.Exercise)
	}
	sctx := storage.ParseSessionContext(sess)
	if sctx.Exercise == nil || sctx.Exercise.AnswerFormat != "text" {
		t.Fatalf("stored exercise=%+v", sctx.Exercise)
	}
	if len(sctx.TestedConcepts) != 0 {
		t.Fatalf("出题后不应计入 TestedConcepts，got=%v", sctx.TestedConcepts)
	}
	_ = store
}

func TestGradeInvalidJSONKeepsExercise(t *testing.T) {
	exerciseJSON := `{"question":"补全 docker-compose 片段","question_type":"code_fill","answer_format":"json","reinforced_concepts":["depends_on 与 volumes"]}`
	coach, store, sess := setupCoach(t, exerciseJSON)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "depends_on: db volumes: ./data")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("invalid json should stay in exercise, phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "json" {
		t.Fatalf("exercise meta=%+v", result.Exercise)
	}
	if !strings.Contains(result.Content, "JSON") {
		t.Fatalf("expected format hint, got %q", result.Content)
	}
	final, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Phase != "exercise" {
		t.Fatalf("session phase=%s", final.Phase)
	}
}

func TestReviewGradeRetryPersistsExercisePhase(t *testing.T) {
	exerciseJSON := `{"question":"说明区别","question_type":"short_answer","answer_format":"text","reinforced_concepts":["轻量级"]}`
	gradeWrong := `{"passed":false,"feedback":"再想想栈大小","mistake_concepts":["轻量级"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradeWrong)

	sctx := storage.ParseSessionContext(sess)
	sctx.LastExercise = &storage.ExerciseContext{
		Question:     "说明区别",
		QuestionType: "short_answer",
		AnswerFormat: "text",
	}
	_ = storage.SaveSessionContext(sess, sctx)
	sess.Phase = "review"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "都是并发")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("response phase=%s", result.Phase)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Phase != "exercise" {
		t.Fatalf("stored session phase=%s", reloaded.Phase)
	}
}

func TestReviewGradeRetry(t *testing.T) {
	exerciseJSON := `{"question":"说明区别","question_type":"short_answer","answer_format":"text","reinforced_concepts":["轻量级"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradePass)

	sctx := storage.ParseSessionContext(sess)
	sctx.LastExercise = &storage.ExerciseContext{
		Question:     "说明区别",
		QuestionType: "short_answer",
		AnswerFormat: "text",
	}
	_ = storage.SaveSessionContext(sess, sctx)
	sess.Phase = "review"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "goroutine 栈更小")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" && result.Phase != "completed" && result.Phase != "review" {
		t.Fatalf("unexpected phase=%s", result.Phase)
	}
}

func TestGradeWrongKeepsExerciseComposer(t *testing.T) {
	exerciseJSON := `{"question":"说明 goroutine 与线程的区别","question_type":"short_answer","answer_format":"text","reinforced_concepts":["与操作系统线程的区别：更小的栈、由 Go runtime 调度"]}`
	gradeWrong := `{"passed":false,"feedback":"还没讲到栈大小差异，再想想。","mistake_concepts":["与操作系统线程的区别"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradeWrong)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "都是并发")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("wrong answer should stay in exercise, phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("exercise meta=%+v", result.Exercise)
	}
	final, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Phase != "exercise" {
		t.Fatalf("session phase=%s", final.Phase)
	}
	sctx := storage.ParseSessionContext(final)
	if sctx.Exercise == nil || sctx.Exercise.AnswerFormat != "text" {
		t.Fatalf("stored exercise=%+v", sctx.Exercise)
	}
	if sctx.Exercise.WrongAttempts != 1 {
		t.Fatalf("wrongAttempts=%d want 1", sctx.Exercise.WrongAttempts)
	}
}

func TestGradeSecondWrongSwapsSimilarExercise(t *testing.T) {
	exerciseJSON := `{"question":"说明 goroutine 与线程的区别","question_type":"short_answer","answer_format":"text","reinforced_concepts":["轻量级"]}`
	gradeWrong1 := `{"passed":false,"feedback":"还没提到栈大小。","mistake_concepts":["轻量级"]}`
	gradeWrong2 := `{"passed":false,"feedback":"关键差是栈更小、由 runtime 调度。下面换题巩固。","mistake_concepts":["轻量级"]}`
	exercise2 := `{"question":"为什么说 goroutine 比线程更轻？","question_type":"short_answer","answer_format":"text","reinforced_concepts":["轻量级"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradeWrong1, gradeWrong2, exercise2)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	first, err := coach.HandleMessage(context.Background(), reloaded, "都是并发")
	if err != nil {
		t.Fatal(err)
	}
	if first.Phase != "exercise" || !strings.Contains(first.Content, "栈") {
		t.Fatalf("first miss content=%q phase=%s", first.Content, first.Phase)
	}
	after1, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	sctx1 := storage.ParseSessionContext(after1)
	if sctx1.Exercise == nil || sctx1.Exercise.WrongAttempts != 1 {
		t.Fatalf("after first miss: %+v", sctx1.Exercise)
	}
	q1 := sctx1.Exercise.Question

	second, err := coach.HandleMessage(context.Background(), after1, "还是一样")
	if err != nil {
		t.Fatal(err)
	}
	if second.Phase != "exercise" {
		t.Fatalf("second miss phase=%s", second.Phase)
	}
	if !strings.Contains(second.Content, "---") {
		t.Fatalf("expected feedback+new question separator: %q", second.Content)
	}
	if !strings.Contains(second.Content, "为什么说 goroutine") {
		t.Fatalf("expected new similar question: %q", second.Content)
	}
	after2, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	sctx2 := storage.ParseSessionContext(after2)
	if sctx2.Exercise == nil || sctx2.Exercise.Question == q1 {
		t.Fatalf("should swap to new exercise, got %+v", sctx2.Exercise)
	}
	if sctx2.Exercise.WrongAttempts != 0 {
		t.Fatalf("new exercise wrongAttempts=%d", sctx2.Exercise.WrongAttempts)
	}
}

func TestGradeTaskInstruction(t *testing.T) {
	pass := gradeTaskInstruction(0, &ChoiceGradeVerdict{Passed: true})
	if strings.Contains(pass, "答错") {
		t.Fatalf("pass path should not mention wrong attempt: %s", pass)
	}
	first := gradeTaskInstruction(0, &ChoiceGradeVerdict{Passed: false})
	if !strings.Contains(first, "禁止写出标准答案") {
		t.Fatalf("first miss: %s", first)
	}
	second := gradeTaskInstruction(1, nil)
	if !strings.Contains(second, "第 2 次") || !strings.Contains(second, "换一道相似题") {
		t.Fatalf("second miss: %s", second)
	}
}

func TestNewExerciseAfterSwapDoesNotForcePriorFormat(t *testing.T) {
	exerciseJSON := `{"question":"补全 compose","question_type":"code_fill","answer_format":"json","reinforced_concepts":["networks"]}`
	gradeWrong := `{"passed":false,"feedback":"networks 配置不对","mistake_concepts":["networks"]}`
	exerciseText := `{"question":"docker-compose up 后台运行加什么参数","question_type":"short_answer","answer_format":"text","reinforced_concepts":["networks"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradeWrong, exerciseText)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = coach.HandleMessage(context.Background(), reloaded, `{"networks":{"app-net":{"driver":"bridge"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	afterWrong, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), afterWrong, "再来一道")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("swap should allow text format, meta=%+v", result.Exercise)
	}
}

func TestGradePassedRecordsTestedConcepts(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "1")
	exerciseJSON := `{"question":"说明 goroutine","question_type":"short_answer","answer_format":"text","reinforced_concepts":["goroutine 是 Go 的轻量级并发执行单元"]}`
	exerciseJSON2 := `{"question":"goroutine 与线程区别","question_type":"short_answer","answer_format":"choice","choices":["A","B"],"choice_mode":"single","correct_choice":"A","reinforced_concepts":["与操作系统线程的区别：更小的栈、由 Go runtime 调度"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradePass, exerciseJSON2)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storage.ParseSessionContext(reloaded).TestedConcepts) != 0 {
		t.Fatal("出题后 TestedConcepts 仍应为空")
	}

	result, err := coach.HandleMessage(context.Background(), reloaded, "轻量级并发单元")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("答对且仍有待考查概念应自动下一题，phase=%s", result.Phase)
	}
	if result.Exercise == nil {
		t.Fatal("自动下一题应带 exercise meta")
	}
	final, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	tested := storage.ParseSessionContext(final).TestedConcepts
	if len(tested) != 1 || !strings.Contains(tested[0], "轻量级") {
		t.Fatalf("答对后应记录概念，got=%v", tested)
	}
}

func TestGradeChoiceOverridesLLMPassed(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	exerciseJSON := `{"question":"关于 Hook，以下说法哪些正确？","question_type":"short_answer","answer_format":"choice","choices":["只有 1、2 正确","只有 1、2、4 正确","全部正确"],"choice_mode":"single","correct_choice":"B","reinforced_concepts":["Hook 事件"]}`
	gradeWrong := `{"passed":false,"feedback":"你对第 3 条判断错了","mistake_concepts":["Hook 事件"]}`
	exerciseJSON2 := `{"question":"第二题","question_type":"short_answer","answer_format":"text","reinforced_concepts":["与操作系统线程的区别：更小的栈、由 Go runtime 调度"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradeWrong, exerciseJSON2)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "B")
	if err != nil {
		t.Fatal(err)
	}
	// 程序判分正确且仍有待考查概念：自动下一题，不因 LLM passed=false 卡在错题态
	if result.Phase != "exercise" {
		t.Fatalf("expected exercise after correct choice, phase=%s content=%q", result.Phase, result.Content)
	}
}

func TestMasterySkipDeferPreservesTestedConcepts(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "1")
	exerciseJSON := `{"question":"说明 goroutine","question_type":"short_answer","answer_format":"text","reinforced_concepts":["goroutine 是 Go 的轻量级并发执行单元"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	readyDefer := `{"ready":true,"feedback":"整体不错","gap_concepts":[]}`
	exerciseJSON2 := `{"question":"第二题","question_type":"short_answer","answer_format":"text","reinforced_concepts":["与操作系统线程的区别：更小的栈、由 Go runtime 调度"]}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradePass, readyDefer, exerciseJSON2)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = coach.HandleMessage(context.Background(), reloaded, "轻量级并发")
	if err != nil {
		t.Fatal(err)
	}
	afterGrade, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storage.ParseSessionContext(afterGrade).TestedConcepts) != 1 {
		t.Fatal("答对后应已记录概念")
	}

	_, err = coach.HandleMessage(context.Background(), afterGrade, "我已经掌握了，下一节")
	if err != nil {
		t.Fatal(err)
	}
	afterMastery, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	sctx := storage.ParseSessionContext(afterMastery)
	if len(sctx.TestedConcepts) != 1 {
		t.Fatalf("掌握度评估延迟完成时不应清空 TestedConcepts，got=%v", sctx.TestedConcepts)
	}
	if afterMastery.Phase != "exercise" {
		t.Fatalf("延迟完成时应自动连题，phase=%s", afterMastery.Phase)
	}
}

func TestEvaluateMasterySkipNotReadyThenForce(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	notReady := `{"ready":false,"feedback":"依赖顺序还没讲清","gap_concepts":["任务依赖排序","调试设备前置条件"]}`
	coach, store, sess := setupCoach(t, notReady)

	sess.Phase = "review"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "我已经掌握了，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "review" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.NodeCompleted {
		t.Fatal("不应直接完成")
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	sctx := storage.ParseSessionContext(reloaded)
	if !sctx.SkipMasteryWarned {
		t.Fatal("应标记已提醒且已写入数据库")
	}

	result, err = coach.HandleMessage(context.Background(), reloaded, "我已经掌握了，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("应强制完成 result=%+v", result)
	}
	mistakes, err := store.ListMistakesForReinforce(storage.DefaultUserID, sess.DomainID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mistakes) == 0 {
		t.Fatal("应记录易错概念")
	}
}

func TestEntryLayerSkipsApplyExercise(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	exerciseJSON := `{"question":"说明 goroutine","question_type":"short_answer","answer_format":"text","reinforced_concepts":["goroutine 是 Go 的轻量级并发执行单元"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	coach, store, sess := setupCoach(t, exerciseJSON, gradePass)

	_, err := coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "轻量级并发单元")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("入门层答对概念题后应直接完成，phase=%s completed=%v", result.Phase, result.NodeCompleted)
	}
}

func TestGradeRequiresApplyBeforeComplete(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	exerciseJSON := `{"question":"说明 channel","question_type":"short_answer","answer_format":"text","reinforced_concepts":["无缓冲 channel 的同步特性"]}`
	applyJSON := `{"question":"补全代码","question_type":"code_fill","answer_format":"json","reinforced_concepts":["带缓冲 channel 的容量与阻塞条件"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	coach, store, base := setupCoach(t, exerciseJSON, gradePass, applyJSON)
	sess, err := store.CreateSession(
		storage.DefaultUserID,
		base.DomainID,
		"go-concurrency",
		"channel",
		"explain",
		&storage.SessionContext{DomainSlug: "go-concurrency"},
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "轻量级并发单元")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("答对概念题后应自动出 apply 题，phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("应出 text apply 题: %+v", result.Exercise)
	}
	sctx := storage.ParseSessionContext(reloaded)
	if sctx.ApplyExercisePassed {
		t.Fatal("尚未通过 apply 题")
	}
}

func TestMasterySkipReadyChainsApplyExercise(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "0")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	ready := `{"ready":true,"feedback":"掌握不错，可以进入下一节","gap_concepts":[]}`
	applyJSON := `{"question":"补全代码","question_type":"code_fill","answer_format":"json","reinforced_concepts":["带缓冲 channel 的容量与阻塞条件"]}`
	coach, store, base := setupCoach(t, ready, applyJSON)
	sess, err := store.CreateSession(
		storage.DefaultUserID,
		base.DomainID,
		"go-concurrency",
		"channel",
		"review",
		&storage.SessionContext{
			DomainSlug:          "go-concurrency",
			TestedConcepts:      []string{"无缓冲 channel 的同步特性"},
			ApplyExercisePassed: false,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := coach.HandleMessage(context.Background(), sess, "已经掌握，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("mastery ready 但缺 apply 时应自动出题，phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("应出 text apply 题: %+v", result.Exercise)
	}
}

func TestEvaluateMasterySkipReady(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "1")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "0")
	ready := `{"ready":true,"feedback":"掌握不错，可以进入下一节","gap_concepts":[]}`
	coach, _, sess := setupCoach(t, ready)

	result, err := coach.HandleMessage(context.Background(), sess, "已经掌握，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("应直接完成 result=%+v", result)
	}
}

func TestGradeApplyDeferWaivedByReadiness(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "1")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	exerciseJSON := `{"question":"说明 channel","question_type":"short_answer","answer_format":"text","reinforced_concepts":["无缓冲 channel 的同步特性"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	ready := `{"ready":true,"feedback":"对话中已体现应用能力，可以点亮","gap_concepts":[]}`
	coach, store, base := setupCoach(t, exerciseJSON, gradePass, ready)
	sess, err := store.CreateSession(
		storage.DefaultUserID,
		base.DomainID,
		"go-concurrency",
		"channel",
		"explain",
		&storage.SessionContext{DomainSlug: "go-concurrency"},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "轻量级并发单元")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("readiness 应豁免 apply defer，phase=%s completed=%v", result.Phase, result.NodeCompleted)
	}
}

func TestGradeReadinessNotReadyChainsApply(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "1")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	exerciseJSON := `{"question":"说明 channel","question_type":"short_answer","answer_format":"text","reinforced_concepts":["无缓冲 channel 的同步特性"]}`
	gradePass := `{"passed":true,"feedback":"很好"}`
	notReady := `{"ready":false,"feedback":"还需应用练习","gap_concepts":["带缓冲 channel 的容量与阻塞条件"]}`
	applyJSON := `{"question":"补全代码","question_type":"code_fill","answer_format":"json","reinforced_concepts":["带缓冲 channel 的容量与阻塞条件"]}`
	coach, store, base := setupCoach(t, exerciseJSON, gradePass, notReady, applyJSON)
	sess, err := store.CreateSession(
		storage.DefaultUserID,
		base.DomainID,
		"go-concurrency",
		"channel",
		"explain",
		&storage.SessionContext{DomainSlug: "go-concurrency"},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = coach.HandleMessage(context.Background(), sess, "开始练习")
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), reloaded, "轻量级并发单元")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "exercise" {
		t.Fatalf("readiness 未达标应连 apply 题，phase=%s", result.Phase)
	}
	if result.Exercise == nil || result.Exercise.AnswerFormat != "text" {
		t.Fatalf("应出 apply 题: %+v", result.Exercise)
	}
}

func TestMasterySkipApplyDeferWaivedByReadiness(t *testing.T) {
	t.Setenv("REGULUS_LLM_COMPLETION_CHECK", "1")
	t.Setenv("REGULUS_STRICT_CONCEPT_COVERAGE", "0")
	t.Setenv("REGULUS_REQUIRE_APPLY_EXERCISE", "1")
	ready := `{"ready":true,"feedback":"可以进入下一节","gap_concepts":[]}`
	coach, store, base := setupCoach(t, ready)
	sess, err := store.CreateSession(
		storage.DefaultUserID,
		base.DomainID,
		"go-concurrency",
		"channel",
		"review",
		&storage.SessionContext{
			DomainSlug:          "go-concurrency",
			TestedConcepts:      []string{"无缓冲 channel 的同步特性"},
			ApplyExercisePassed: false,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := coach.HandleMessage(context.Background(), sess, "已经掌握，下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.Phase != "completed" || !result.NodeCompleted {
		t.Fatalf("申请掌握且 readiness 放行应点亮，phase=%s", result.Phase)
	}
	_ = store
}

func TestStartNextBlockedFromExplainPhase(t *testing.T) {
	coach, _, sess := setupCoach(t)
	if sess.Phase != "explain" {
		t.Fatalf("phase=%s want explain", sess.Phase)
	}

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.NextSessionID != "" {
		t.Fatalf("未完成节点不应直接切节: %+v", result)
	}
	if result.Phase != "explain" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Content == "" || !containsAll(result.Content, "尚未完成", "已经掌握") {
		t.Fatalf("应提示先完成或申请掌握: %q", result.Content)
	}
}

func TestBlockStartNextBeforeCompleted(t *testing.T) {
	coach, _, sess := setupCoach(t)
	sess.Phase = "review"
	_ = coach.store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if result.NextSessionID != "" {
		t.Fatalf("未完成节点不应直接切节: %+v", result)
	}
	if result.Phase != "review" {
		t.Fatalf("phase=%s", result.Phase)
	}
	if result.Content == "" || !containsAll(result.Content, "尚未完成", "已经掌握") {
		t.Fatalf("应提示先完成或申请掌握: %q", result.Content)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestStartNextNodeAfterCompleted(t *testing.T) {
	beginReply := "这是下一节开场讲解"
	coach, store, sess := setupCoach(t, beginReply)
	sess.Phase = "completed"
	sess.Status = "completed"
	_ = store.UpdateSession(sess)

	result, err := coach.HandleMessage(context.Background(), sess, "下一节")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Content, "尚未完成") {
		t.Fatalf("completed 阶段不应拦截切节: %q", result.Content)
	}
	if result.Phase != "explain" || result.NextSessionID == "" {
		t.Fatalf("应进入下一节 result=%+v", result)
	}
	if result.NextNodeKey != "first_goroutine" {
		t.Fatalf("nextNodeKey=%q", result.NextNodeKey)
	}
	newSess, err := store.GetSession(result.NextSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if newSess.NodeKey != "first_goroutine" || newSess.Phase != "explain" {
		t.Fatalf("new session=%+v", newSess)
	}
}
