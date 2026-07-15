package agent

import (
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestNormalizePlanningResult_limitsLearningAndBuildsFocus(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.CreateUser("归一")
	if err != nil {
		t.Fatal(err)
	}
	p := &Planner{store: store}

	plan := &PlanningResult{
		SituationSummary: "事务过多学不动",
		Matrix: PlanningMatrix{
			QuickWins: []PlanningMatrixItem{
				{Title: "回一封邮件", NextStep: "打开邮箱", Minutes: 10},
				{Title: "约牙医", Minutes: 5},
			},
		},
		ActionPlan: PlanningActionPlan{
			Today: []PlanningActionItem{
				{Title: "学 A", Minutes: 15, Kind: "learning"},
				{Title: "学 B", Minutes: 15, Kind: "learning"},
				{Title: "回邮件", Minutes: 10, Kind: "task"},
				{Title: "多余", Minutes: 5, Kind: "task"},
			},
		},
		LearningFocus: []PlanningLearningFocus{
			{Area: "Go 并发", Rationale: "续练", SuggestedMinutes: 15},
		},
		MindsetNote: "先做一件",
	}
	p.NormalizePlanningResult(user.ID, plan)

	if plan.Focus == nil || plan.Focus.NorthStar == "" {
		t.Fatalf("focus missing: %+v", plan.Focus)
	}
	if plan.Focus.NorthStar != "事务过多学不动" {
		t.Fatalf("north_star=%q", plan.Focus.NorthStar)
	}
	if plan.UIState == nil || plan.UIState.NorthStarPinned {
		t.Fatal("synthesize should leave north star unpinned")
	}
	learningCount := 0
	for _, item := range plan.ActionPlan.Today {
		if item.Kind == "learning" {
			learningCount++
		}
	}
	if learningCount != 1 {
		t.Fatalf("learning count=%d today=%+v", learningCount, plan.ActionPlan.Today)
	}
	if len(plan.ActionPlan.Today) > maxPlanningTodayItems {
		t.Fatalf("today len=%d", len(plan.ActionPlan.Today))
	}
	if len(plan.ClearFirst) == 0 {
		t.Fatal("expected clear_first from quick_wins")
	}
	if plan.Focus.TodayLearning == nil || plan.Focus.TodayLearning.Title == "" {
		t.Fatal("expected today_learning")
	}
	if plan.Focus.TodayLearning.MatchedDomainID != "" {
		t.Fatal("fabricated domain id should stay empty")
	}
}

func TestApplyPlanningFocusPatch(t *testing.T) {
	plan := &PlanningResult{
		SituationSummary: "s",
		Focus:            &PlanningFocus{NorthStar: "旧北星"},
		UIState:          &PlanningUIState{NorthStarPinned: false, Checked: map[string]bool{}},
	}
	pinned := true
	star := "新北星"
	ApplyPlanningFocusPatch(plan, PlanningFocusPatch{
		NorthStarPinned: &pinned,
		NorthStar:       &star,
		Checked:         map[string]bool{"clear:0": true, "today:1": false},
	})
	if !plan.UIState.NorthStarPinned {
		t.Fatal("expected pinned")
	}
	if plan.Focus.NorthStar != "新北星" {
		t.Fatalf("north_star=%q", plan.Focus.NorthStar)
	}
	if !plan.UIState.Checked["clear:0"] {
		t.Fatal("expected clear:0 checked")
	}
	if _, ok := plan.UIState.Checked["today:1"]; ok {
		t.Fatal("unchecked key should be removed")
	}
}

func TestHydrateLegacyPlan_preservesPin(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	user, err := store.CreateUser("hydrate")
	if err != nil {
		t.Fatal(err)
	}
	p := &Planner{store: store}
	plan := &PlanningResult{
		SituationSummary: "恢复节奏",
		ActionPlan: PlanningActionPlan{
			Today: []PlanningActionItem{{Title: "节点练", Minutes: 15, Kind: "learning"}},
		},
		UIState: &PlanningUIState{
			NorthStarPinned: true,
			Checked:         map[string]bool{"today:0": true},
		},
	}
	p.HydrateLegacyPlan(user.ID, plan)
	if plan.Focus == nil || plan.Focus.NorthStar != "恢复节奏" {
		t.Fatalf("focus=%+v", plan.Focus)
	}
	if !plan.UIState.NorthStarPinned || !plan.UIState.Checked["today:0"] {
		t.Fatalf("ui_state mutated: %+v", plan.UIState)
	}
}
