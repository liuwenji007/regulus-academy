package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/agent"
)

func TestPlanningStartAndGet(t *testing.T) {
	chdirToRepo(t)
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	user, err := store.CreateUser("规划")
	if err != nil {
		t.Fatal(err)
	}

	startBody, _ := json.Marshal(map[string]any{"forceNew": true})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/planning/start", bytes.NewReader(startBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status=%d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	sid, _ := out["sessionId"].(string)
	if sid == "" {
		t.Fatal("missing sessionId")
	}
	msgs, ok := out["messages"].([]any)
	if !ok || len(msgs) == 0 {
		t.Fatal("expected opening message")
	}

	req2, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/planning/"+sid, nil)
	req2.Header.Set("X-User-Id", user.ID)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("get status=%d", resp2.StatusCode)
	}

	req3, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/planning/active", nil)
	req3.Header.Set("X-User-Id", user.ID)
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatal(err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("active status=%d", resp3.StatusCode)
	}
	var active map[string]any
	if err := json.NewDecoder(resp3.Body).Decode(&active); err != nil {
		t.Fatal(err)
	}
	if active["sessionId"] != sid {
		t.Fatalf("active session=%v", active["sessionId"])
	}
}

func TestPlanningMessageWithMockLLM(t *testing.T) {
	chdirToRepo(t)
	store, _, ts := setupTestServerWithHandler(t, true, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		plan := `{"situation_summary":"聚焦交付","matrix":{"important_urgent":[{"title":"周报"}],"important_not_urgent":[],"quick_wins":[],"defer_or_drop":[]},"action_plan":{"today":[{"title":"写周报","minutes":15,"kind":"task"}],"this_week":[]},"learning_focus":[],"mindset_note":"开始"}`
		b, _ := json.Marshal(plan)
		payload := `{"choices":[{"message":{"role":"assistant","content":` + string(b) + `}}]}`
		_, _ = w.Write([]byte(payload))
	})
	defer ts.Close()

	user, err := store.CreateUser("规划消息")
	if err != nil {
		t.Fatal(err)
	}

	startBody, _ := json.Marshal(map[string]any{"forceNew": true})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/planning/start", bytes.NewReader(startBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var startOut map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&startOut); err != nil {
		t.Fatal(err)
	}
	sid, _ := startOut["sessionId"].(string)

	msgBody, _ := json.Marshal(map[string]string{
		"sessionId": sid,
		"content":   "项目 deadline 紧，帮我整理出行动方案",
	})
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/planning/message", bytes.NewReader(msgBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Id", user.ID)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("message status=%d", resp2.StatusCode)
	}
	var msgOut map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&msgOut); err != nil {
		t.Fatal(err)
	}
	if msgOut["phase"] != "plan_ready" {
		t.Fatalf("phase=%v", msgOut["phase"])
	}
	if msgOut["plan"] == nil {
		t.Fatal("expected plan in response")
	}
	planMap, ok := msgOut["plan"].(map[string]any)
	if !ok {
		t.Fatal("plan type")
	}
	focus, _ := planMap["focus"].(map[string]any)
	if focus == nil || focus["north_star"] == "" {
		t.Fatalf("expected focus.north_star, plan=%v", planMap)
	}
}

// intake 判定 ready_to_plan 后会再跑 synthesize；回归：响应必须带 plan（曾因 := 遮蔽导致前端右侧空白）
func TestPlanningIntakeReadyToPlanReturnsPlan(t *testing.T) {
	chdirToRepo(t)
	call := 0
	store, _, ts := setupTestServerWithHandler(t, true, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		call++
		var content string
		// intake 可能重试；合成也可能重试——凡是像 turn JSON 的给 intake，其余给 plan
		if call <= 2 {
			content = `{"reply":"信息够了，我来整理聚焦方案。","ready_to_plan":true}`
		} else {
			content = `{"situation_summary":"在 Python/Go/Rust 里先选一条线","focus":{"north_star":"本周把 Go 并发练完"},"clear_first":[{"title":"回掉阻塞邮件","minutes":15}],"matrix":{"important_urgent":[],"important_not_urgent":[],"quick_wins":[],"defer_or_drop":[]},"action_plan":{"today":[{"title":"读 channel 一小节","minutes":20,"kind":"learning"}],"this_week":[]},"learning_focus":[],"mindset_note":"一次只推一条线","user_reply":"整理好了，右侧是你的聚焦位。"}`
		}
		b, _ := json.Marshal(content)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + string(b) + `}}]}`))
	})
	defer ts.Close()

	user, err := store.CreateUser("intake就绪")
	if err != nil {
		t.Fatal(err)
	}
	startBody, _ := json.Marshal(map[string]any{"forceNew": true})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/planning/start", bytes.NewReader(startBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var startOut map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&startOut); err != nil {
		t.Fatal(err)
	}
	sid, _ := startOut["sessionId"].(string)

	// 避免命中 WantsExplicitPlan，走 intake → ready_to_plan → synthesize
	msgBody, _ := json.Marshal(map[string]string{
		"sessionId": sid,
		"content":   "最近 Python、Go、Rust 都想碰，有点乱，本周大概只能挤 5 小时",
	})
	req2, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/planning/message", bytes.NewReader(msgBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-User-Id", user.ID)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("message status=%d call=%d", resp2.StatusCode, call)
	}
	var msgOut map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&msgOut); err != nil {
		t.Fatal(err)
	}
	if msgOut["phase"] != "plan_ready" {
		t.Fatalf("phase=%v call=%d out=%v", msgOut["phase"], call, msgOut)
	}
	if msgOut["synthesized"] != true {
		t.Fatalf("synthesized=%v", msgOut["synthesized"])
	}
	if msgOut["plan"] == nil {
		t.Fatal("expected plan in message response (not only after refresh)")
	}
	planMap, _ := msgOut["plan"].(map[string]any)
	focus, _ := planMap["focus"].(map[string]any)
	if focus == nil || focus["north_star"] == "" {
		t.Fatalf("expected focus.north_star, plan=%v", planMap)
	}
}

func TestPatchPlanningFocus(t *testing.T) {
	chdirToRepo(t)
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	user, err := store.CreateUser("聚焦补丁")
	if err != nil {
		t.Fatal(err)
	}
	sess, err := store.CreatePlanningSession(user.ID, "plan_ready")
	if err != nil {
		t.Fatal(err)
	}
	plan := &agent.PlanningResult{
		SituationSummary: "过载",
		Focus:            &agent.PlanningFocus{NorthStar: "把 Go 并发练完"},
		ClearFirst:       []agent.PlanningClearItem{{Title: "回邮件", Minutes: 10}},
		ActionPlan: agent.PlanningActionPlan{
			Today: []agent.PlanningActionItem{{Title: "练习 channel", Minutes: 15, Kind: "learning"}},
		},
		MindsetNote: "一步",
		UIState:     &agent.PlanningUIState{NorthStarPinned: false, Checked: map[string]bool{}},
	}
	raw, err := agent.MarshalPlanningResult(plan)
	if err != nil {
		t.Fatal(err)
	}
	sess.PlanJSON = raw
	if err := store.UpdatePlanningSession(sess); err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"north_star_pinned": true,
		"checked":           map[string]bool{"clear:0": true},
	})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/planning/"+sess.ID+"/focus", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	planOut, _ := out["plan"].(map[string]any)
	ui, _ := planOut["ui_state"].(map[string]any)
	if ui["north_star_pinned"] != true {
		t.Fatalf("ui_state=%v", ui)
	}
	checked, _ := ui["checked"].(map[string]any)
	if checked["clear:0"] != true {
		t.Fatalf("checked=%v", checked)
	}
	focus, _ := planOut["focus"].(map[string]any)
	if focus["north_star"] != "把 Go 并发练完" {
		t.Fatalf("north_star should be preserved, got %v", focus["north_star"])
	}

	reloaded, err := store.GetPlanningSession(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := agent.ParsePlanningResult(reloaded.PlanJSON)
	if err != nil || persisted == nil || persisted.UIState == nil || !persisted.UIState.NorthStarPinned {
		t.Fatalf("not persisted: %+v err=%v", persisted, err)
	}
}
