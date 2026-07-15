package service

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestGetLearningShortcuts_lastLessonByUpdatedAt(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, treeA, err := store.CreateDomain("课程A")
	if err != nil {
		t.Fatal(err)
	}
	_, treeB, err := store.CreateDomain("课程B")
	if err != nil {
		t.Fatal(err)
	}

	nodeA := treeA.Layers[0].Nodes[0].Key
	nodeB := treeB.Layers[0].Nodes[0].Key

	older, err := store.CreateSession(storage.DefaultUserID, treeA.DomainID, "", nodeA, "explain", nil)
	if err != nil {
		t.Fatal(err)
	}
	newer, err := store.CreateSession(storage.DefaultUserID, treeB.DomainID, "", nodeB, "exercise", nil)
	if err != nil {
		t.Fatal(err)
	}
	// 再碰一下旧课节点，但更新时间更早：先写 older.updated_at 为更早，newer 更新
	_, _ = store.AddMessage(older.ID, "user", "早些时候的消息")
	time.Sleep(10 * time.Millisecond)
	_, _ = store.AddMessage(newer.ID, "user", "最近的消息")

	svc := NewShortcutsService(store, nil)
	out, err := svc.GetLearningShortcuts(storage.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if out.LastLesson == nil {
		t.Fatal("expected lastLesson")
	}
	if out.LastLesson.DomainID != treeB.DomainID {
		t.Fatalf("last domain=%s want B %s", out.LastLesson.DomainID, treeB.DomainID)
	}
	if !out.LastLesson.CanResume {
		t.Fatal("active incomplete session should canResume")
	}
	if out.LastLesson.SessionID != newer.ID {
		t.Fatalf("session=%s want %s", out.LastLesson.SessionID, newer.ID)
	}
}

func TestGetLearningShortcuts_noSession(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, _, err = store.CreateDomain("空课")
	if err != nil {
		t.Fatal(err)
	}

	svc := NewShortcutsService(store, nil)
	out, err := svc.GetLearningShortcuts(storage.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if out.LastLesson != nil {
		t.Fatalf("expected nil lastLesson, got %+v", out.LastLesson)
	}
	if !out.HasCourses {
		t.Fatal("hasCourses should be true")
	}
}

func TestGetLearningShortcuts_planningFirstAndExcludeLast(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, treeLast, err := store.CreateDomain("上一节")
	if err != nil {
		t.Fatal(err)
	}
	_, treePlan, err := store.CreateDomain("今日规划课")
	if err != nil {
		t.Fatal(err)
	}
	_, treeProg, err := store.CreateDomain("进行中课")
	if err != nil {
		t.Fatal(err)
	}

	nodeLast := treeLast.Layers[0].Nodes[0].Key
	nodePlan := treePlan.Layers[0].Nodes[0].Key

	sess, err := store.CreateSession(storage.DefaultUserID, treeLast.DomainID, "", nodeLast, "explain", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(sess.ID, "user", "上次学到这")

	// 进行中：完成 1 节点，未学完
	progNode := treeProg.Layers[0].Nodes[0].Key
	_ = store.UpsertProgress(storage.UserProgress{
		UserID: storage.DefaultUserID, DomainID: treeProg.DomainID,
		NodeKey: progNode, Layer: treeProg.Layers[0].Key, Status: "completed",
	})

	planJSON, _ := json.Marshal(map[string]any{
		"situation_summary": "overload",
		"focus": map[string]any{
			"north_star": "稳住节奏",
			"today_learning": map[string]any{
				"title":              "练 15 分钟",
				"minutes":            15,
				"matched_domain_id":  treePlan.DomainID,
				"matched_node_key":   nodePlan,
				"matched_node_title": "规划节点",
			},
		},
		"matrix": map[string]any{
			"important_urgent":     []any{},
			"important_not_urgent": []any{},
			"quick_wins":           []any{},
			"defer_or_drop":        []any{},
		},
		"action_plan": map[string]any{
			"today":     []any{},
			"this_week": []any{},
		},
		"learning_focus": []any{},
		"mindset_note":   "",
	})
	ps, err := store.CreatePlanningSession(storage.DefaultUserID, "plan_ready")
	if err != nil {
		t.Fatal(err)
	}
	ps.PlanJSON = string(planJSON)
	if err := store.UpdatePlanningSession(ps); err != nil {
		t.Fatal(err)
	}

	svc := NewShortcutsService(store, nil)
	out, err := svc.GetLearningShortcuts(storage.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if out.LastLesson == nil || out.LastLesson.DomainID != treeLast.DomainID {
		t.Fatalf("lastLesson=%+v", out.LastLesson)
	}
	if len(out.Recommendations) < 1 {
		t.Fatalf("recs=%+v", out.Recommendations)
	}
	if out.Recommendations[0].Source != "planning" || out.Recommendations[0].DomainID != treePlan.DomainID {
		t.Fatalf("first rec=%+v", out.Recommendations[0])
	}
	if out.Recommendations[0].Title != "练 15 分钟" {
		t.Fatalf("title=%q", out.Recommendations[0].Title)
	}
	for _, r := range out.Recommendations {
		if r.DomainID == treeLast.DomainID {
			t.Fatalf("progress/recs should exclude last lesson domain: %+v", out.Recommendations)
		}
	}
	// 最多 2 条：planning + progress 进行中课
	if len(out.Recommendations) > 2 {
		t.Fatalf("len=%d", len(out.Recommendations))
	}
	foundProg := false
	for _, r := range out.Recommendations {
		if r.DomainID == treeProg.DomainID && r.Source == "progress" {
			foundProg = true
		}
	}
	if !foundProg {
		t.Fatalf("expected progress fill for in-progress course, got %+v", out.Recommendations)
	}
}

func TestGetLearningShortcuts_domainAccessBeatsStaleSession(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, treeOld, err := store.CreateDomain("旧课")
	if err != nil {
		t.Fatal(err)
	}
	_, treeNew, err := store.CreateDomain("新进入的课")
	if err != nil {
		t.Fatal(err)
	}
	nodeOld := treeOld.Layers[0].Nodes[0].Key
	sess, err := store.CreateSession(storage.DefaultUserID, treeOld.DomainID, "", nodeOld, "explain", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(sess.ID, "user", "很久以前")

	if err := store.TouchDomainAccess(storage.DefaultUserID, treeNew.DomainID, ""); err != nil {
		t.Fatal(err)
	}

	svc := NewShortcutsService(store, nil)
	out, err := svc.GetLearningShortcuts(storage.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if out.LastLesson == nil || out.LastLesson.DomainID != treeNew.DomainID {
		t.Fatalf("lastLesson should follow domain access, got %+v", out.LastLesson)
	}
	if out.LastLesson.CanResume {
		t.Fatal("tree-only visit without session should not canResume")
	}
}

func TestGetLearningShortcuts_completedNotRecommended(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, tree, err := store.CreateDomain("已完成课")
	if err != nil {
		t.Fatal(err)
	}
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			_ = store.UpsertProgress(storage.UserProgress{
				UserID: storage.DefaultUserID, DomainID: tree.DomainID,
				NodeKey: n.Key, Layer: layer.Key, Status: "completed",
			})
		}
	}

	svc := NewShortcutsService(store, nil)
	out, err := svc.GetLearningShortcuts(storage.DefaultUserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Recommendations) != 0 {
		t.Fatalf("completed course should not be recommended: %+v", out.Recommendations)
	}
}
