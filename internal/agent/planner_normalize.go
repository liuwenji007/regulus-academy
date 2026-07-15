package agent

import "strings"

const (
	maxPlanningTodayItems = 3
	maxPlanningClearFirst = 3
)

// NormalizePlanningResult 合成后归一：校验匹配 ID、限制今日学习、补齐 focus/clear_first、重置 ui_state 钉住态。
func (p *Planner) NormalizePlanningResult(userID string, plan *PlanningResult) {
	if plan == nil {
		return
	}
	p.hydratePlanningShape(userID, plan)
	resetUIStateAfterSynthesize(plan)
}

// HydrateLegacyPlan 读取旧 plan_json 时补齐 focus/clear_first，不重置用户钉住与勾选。
func (p *Planner) HydrateLegacyPlan(userID string, plan *PlanningResult) {
	if plan == nil {
		return
	}
	p.hydratePlanningShape(userID, plan)
	if plan.UIState == nil {
		plan.UIState = &PlanningUIState{Checked: map[string]bool{}}
	} else if plan.UIState.Checked == nil {
		plan.UIState.Checked = map[string]bool{}
	}
}

func (p *Planner) hydratePlanningShape(userID string, plan *PlanningResult) {
	p.sanitizeLearningFocus(userID, plan)
	p.sanitizeTodayLearningMatch(userID, plan)
	limitTodayActions(plan)
	ensureClearFirst(plan)
	ensureFocus(plan)
	syncLearningFocusFromToday(plan)
}

func (p *Planner) sanitizeTodayLearningMatch(userID string, plan *PlanningResult) {
	if plan.Focus == nil || plan.Focus.TodayLearning == nil {
		return
	}
	tl := plan.Focus.TodayLearning
	if tl.MatchedDomainID == "" && tl.MatchedNodeKey == "" {
		return
	}
	lf := PlanningLearningFocus{
		MatchedDomainID:  tl.MatchedDomainID,
		MatchedNodeKey:   tl.MatchedNodeKey,
		MatchedNodeTitle: tl.MatchedNodeTitle,
	}
	tmp := &PlanningResult{LearningFocus: []PlanningLearningFocus{lf}}
	p.sanitizeLearningFocus(userID, tmp)
	if len(tmp.LearningFocus) == 0 {
		tl.MatchedDomainID = ""
		tl.MatchedNodeKey = ""
		tl.MatchedNodeTitle = ""
		return
	}
	out := tmp.LearningFocus[0]
	tl.MatchedDomainID = out.MatchedDomainID
	tl.MatchedNodeKey = out.MatchedNodeKey
	tl.MatchedNodeTitle = out.MatchedNodeTitle
}

func limitTodayActions(plan *PlanningResult) {
	today := plan.ActionPlan.Today
	if len(today) == 0 {
		return
	}
	var learning *PlanningActionItem
	var tasks []PlanningActionItem
	for i := range today {
		item := today[i]
		if strings.EqualFold(strings.TrimSpace(item.Kind), "learning") {
			if learning == nil {
				cp := item
				cp.Kind = "learning"
				learning = &cp
			}
			continue
		}
		item.Kind = "task"
		tasks = append(tasks, item)
	}
	out := make([]PlanningActionItem, 0, maxPlanningTodayItems)
	if learning != nil {
		out = append(out, *learning)
	}
	for _, t := range tasks {
		if len(out) >= maxPlanningTodayItems {
			break
		}
		out = append(out, t)
	}
	plan.ActionPlan.Today = out
}

func ensureClearFirst(plan *PlanningResult) {
	if len(plan.ClearFirst) > 0 {
		if len(plan.ClearFirst) > maxPlanningClearFirst {
			plan.ClearFirst = plan.ClearFirst[:maxPlanningClearFirst]
		}
		return
	}
	for _, item := range plan.Matrix.QuickWins {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		plan.ClearFirst = append(plan.ClearFirst, PlanningClearItem{
			Title:    title,
			NextStep: strings.TrimSpace(item.NextStep),
			Minutes:  item.Minutes,
		})
		if len(plan.ClearFirst) >= maxPlanningClearFirst {
			break
		}
	}
}

func ensureFocus(plan *PlanningResult) {
	if plan.Focus == nil {
		plan.Focus = &PlanningFocus{}
	}
	f := plan.Focus
	if strings.TrimSpace(f.NorthStar) == "" {
		f.NorthStar = strings.TrimSpace(plan.SituationSummary)
	}
	if strings.TrimSpace(f.NorthStar) == "" {
		for _, item := range plan.Matrix.ImportantNotUrgent {
			if t := strings.TrimSpace(item.Title); t != "" {
				f.NorthStar = t
				break
			}
		}
	}
	if strings.TrimSpace(f.NorthStar) == "" {
		f.NorthStar = "恢复可执行的学习节奏"
	}
	if strings.TrimSpace(f.WeekWedge) == "" {
		if len(plan.LearningFocus) > 0 {
			f.WeekWedge = strings.TrimSpace(plan.LearningFocus[0].Area)
		}
		if f.WeekWedge == "" {
			for _, item := range plan.Matrix.ImportantNotUrgent {
				if t := strings.TrimSpace(item.Title); t != "" {
					f.WeekWedge = t
					break
				}
			}
		}
	}
	if f.TodayLearning == nil || strings.TrimSpace(f.TodayLearning.Title) == "" {
		f.TodayLearning = deriveTodayLearning(plan)
	} else if f.TodayLearning.Minutes <= 0 {
		f.TodayLearning.Minutes = 15
	}
}

func deriveTodayLearning(plan *PlanningResult) *PlanningFocusTodayLearning {
	for _, item := range plan.ActionPlan.Today {
		if strings.EqualFold(strings.TrimSpace(item.Kind), "learning") {
			tl := &PlanningFocusTodayLearning{
				Title:   strings.TrimSpace(item.Title),
				Minutes: item.Minutes,
			}
			if tl.Minutes <= 0 {
				tl.Minutes = 15
			}
			if len(plan.LearningFocus) > 0 {
				lf := plan.LearningFocus[0]
				tl.MatchedDomainID = lf.MatchedDomainID
				tl.MatchedNodeKey = lf.MatchedNodeKey
				tl.MatchedNodeTitle = lf.MatchedNodeTitle
			}
			return tl
		}
	}
	if len(plan.LearningFocus) > 0 {
		lf := plan.LearningFocus[0]
		mins := lf.SuggestedMinutes
		if mins <= 0 {
			mins = 15
		}
		title := strings.TrimSpace(lf.MatchedNodeTitle)
		if title == "" {
			title = strings.TrimSpace(lf.Area)
		}
		if title == "" {
			return nil
		}
		return &PlanningFocusTodayLearning{
			Title:            title,
			Minutes:          mins,
			MatchedDomainID:  lf.MatchedDomainID,
			MatchedNodeKey:   lf.MatchedNodeKey,
			MatchedNodeTitle: lf.MatchedNodeTitle,
		}
	}
	return nil
}

func syncLearningFocusFromToday(plan *PlanningResult) {
	if plan.Focus == nil || plan.Focus.TodayLearning == nil {
		return
	}
	tl := plan.Focus.TodayLearning
	title := strings.TrimSpace(tl.Title)
	if title == "" {
		return
	}
	mins := tl.Minutes
	if mins <= 0 {
		mins = 15
	}
	lf := PlanningLearningFocus{
		Area:             title,
		Rationale:        "今日学习聚焦",
		SuggestedMinutes: mins,
		MatchedDomainID:  tl.MatchedDomainID,
		MatchedNodeKey:   tl.MatchedNodeKey,
		MatchedNodeTitle: tl.MatchedNodeTitle,
	}
	if len(plan.LearningFocus) == 0 {
		plan.LearningFocus = []PlanningLearningFocus{lf}
	} else {
		plan.LearningFocus[0] = lf
		if len(plan.LearningFocus) > 1 {
			plan.LearningFocus = plan.LearningFocus[:1]
		}
	}
	// 保证 today 中有对应 learning 项（供勾选键 today:i）
	hasLearning := false
	for i := range plan.ActionPlan.Today {
		if strings.EqualFold(plan.ActionPlan.Today[i].Kind, "learning") {
			plan.ActionPlan.Today[i].Title = title
			plan.ActionPlan.Today[i].Minutes = mins
			hasLearning = true
			break
		}
	}
	if !hasLearning {
		plan.ActionPlan.Today = append([]PlanningActionItem{{
			Title: title, Minutes: mins, Kind: "learning",
		}}, plan.ActionPlan.Today...)
		if len(plan.ActionPlan.Today) > maxPlanningTodayItems {
			plan.ActionPlan.Today = plan.ActionPlan.Today[:maxPlanningTodayItems]
		}
	}
}

func resetUIStateAfterSynthesize(plan *PlanningResult) {
	plan.UIState = &PlanningUIState{
		NorthStarPinned: false,
		Checked:         map[string]bool{},
	}
}

// ApplyPlanningFocusPatch 合并钉住/勾选等 UI 状态，不重跑 LLM。
func ApplyPlanningFocusPatch(plan *PlanningResult, patch PlanningFocusPatch) {
	if plan == nil {
		return
	}
	if plan.Focus == nil {
		plan.Focus = &PlanningFocus{}
	}
	if plan.UIState == nil {
		plan.UIState = &PlanningUIState{Checked: map[string]bool{}}
	}
	if plan.UIState.Checked == nil {
		plan.UIState.Checked = map[string]bool{}
	}
	if patch.NorthStar != nil {
		if s := strings.TrimSpace(*patch.NorthStar); s != "" {
			plan.Focus.NorthStar = s
		}
	}
	if patch.NorthStarPinned != nil {
		plan.UIState.NorthStarPinned = *patch.NorthStarPinned
	}
	for k, v := range patch.Checked {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if v {
			plan.UIState.Checked[key] = true
		} else {
			delete(plan.UIState.Checked, key)
		}
	}
}

// PlanningFocusPatch 轻量更新北星与勾选状态
type PlanningFocusPatch struct {
	NorthStarPinned *bool           `json:"north_star_pinned,omitempty"`
	NorthStar       *string         `json:"north_star,omitempty"`
	Checked         map[string]bool `json:"checked,omitempty"`
}
