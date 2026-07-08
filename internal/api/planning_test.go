package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
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
}
