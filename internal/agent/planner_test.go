package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestPlannerIntakeTurn(t *testing.T) {
	chdirCoachRepo(t)
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"reply\":\"听起来事情不少。本周能投入多少时间？\",\"ready_to_plan\":false}"}}]}`))
	}))
	defer mock.Close()

	planner, err := NewPlanner(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	out, err := planner.IntakeTurn(context.Background(), user.ID, nil, "工作项目 deadline 紧，Go 课程也拖了")
	if err != nil {
		t.Fatal(err)
	}
	if out.Reply == "" {
		t.Fatal("empty reply")
	}
	if out.ReadyToPlan {
		t.Fatal("should not be ready yet")
	}
}

func TestPlannerIntakeExplicitPlan(t *testing.T) {
	chdirCoachRepo(t)
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"reply\":\"好的\",\"ready_to_plan\":false}"}}]}`))
	}))
	defer mock.Close()

	planner, err := NewPlanner(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	out, err := planner.IntakeTurn(context.Background(), user.ID, nil, "帮我整理出行动方案")
	if err != nil {
		t.Fatal(err)
	}
	if !out.ReadyToPlan {
		t.Fatal("explicit plan request should set ready_to_plan")
	}
}

func TestPlannerIntakeFallbackError(t *testing.T) {
	chdirCoachRepo(t)
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}

	calls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls < 5 {
			// intake ChatPromptJSON 路径：无效 JSON，触发「解析 JSON 失败」
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"not-json"}}]}`))
			return
		}
		// fallback 纯文本路径失败
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`fallback unavailable`))
	}))
	defer mock.Close()

	planner, err := NewPlanner(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	_, err = planner.IntakeTurn(context.Background(), user.ID, nil, "事情很多")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "解析 JSON 失败") {
		t.Fatalf("should return fallback error, got stale intake error: %v", err)
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected fallback HTTP error, got: %v", err)
	}
}

func TestPlannerSynthesize(t *testing.T) {
	chdirCoachRepo(t)
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := `{"situation_summary":"事务过多","matrix":{"important_urgent":[{"title":"交报告","next_step":"写大纲"}],"important_not_urgent":[],"quick_wins":[],"defer_or_drop":[]},"action_plan":{"today":[{"title":"写报告大纲","minutes":15,"kind":"task"}],"this_week":[]},"learning_focus":[{"area":"Go","rationale":"继续推进","suggested_minutes":15}],"mindset_note":"先做一件小事"}`
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconvQuote(body) + `}}]}`))
	}))
	defer mock.Close()

	planner, err := NewPlanner(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	plan, reply, err := planner.Synthesize(context.Background(), user.ID, nil, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if plan.SituationSummary == "" {
		t.Fatal("missing summary")
	}
	if len(plan.ActionPlan.Today) == 0 {
		t.Fatal("expected today actions")
	}
	if plan.Focus == nil || plan.Focus.NorthStar == "" {
		t.Fatalf("expected focus after normalize: %+v", plan.Focus)
	}
	if plan.UIState == nil || plan.UIState.NorthStarPinned {
		t.Fatal("fresh plan should not be pinned")
	}
	if reply == "" {
		t.Fatal("empty reply")
	}
}

func strconvQuote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func TestPlanningStorageCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreatePlanningSession(user.ID, "intake")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddPlanningMessage(sess.ID, "assistant", storage.PlanningOpenMessage); err != nil {
		t.Fatal(err)
	}

	active, err := store.FindActivePlanningSession(user.ID)
	if err != nil || active == nil || active.ID != sess.ID {
		t.Fatalf("active=%v err=%v", active, err)
	}

	sess.Phase = "plan_ready"
	sess.PlanJSON = `{"situation_summary":"ok"}`
	if err := store.UpdatePlanningSession(sess); err != nil {
		t.Fatal(err)
	}

	msgs, err := store.ListPlanningMessages(sess.ID)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("msgs=%d err=%v", len(msgs), err)
	}
}
