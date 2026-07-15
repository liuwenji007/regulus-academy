package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestLearningShortcutsAPI(t *testing.T) {
	chdirToRepo(t)
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	user, err := store.CreateUser("侧栏测试")
	if err != nil {
		t.Fatal(err)
	}

	_, tree, err := store.CreateDomainFromTree(user.ID, "快捷课", "shortcut-course", "", &storage.KnowledgeTree{
		DomainName: "快捷课",
		Layers: []storage.TreeLayer{{
			Key: "entry", Label: "入门", Time: "1w", Goal: "g",
			Nodes: []storage.TreeNode{{Key: "n1", Title: "节点一"}},
		}},
	}, "{}", "test", true, "")
	if err != nil {
		t.Fatal(err)
	}

	sess, err := store.CreateSession(user.ID, tree.DomainID, "shortcut-course", "n1", "exercise", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = store.AddMessage(sess.ID, "user", "hello")

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/learning/shortcuts", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-User-Id", user.ID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var body struct {
		LastLesson *struct {
			DomainID  string `json:"domainId"`
			SessionID string `json:"sessionId"`
			CanResume bool   `json:"canResume"`
			NodeTitle string `json:"nodeTitle"`
		} `json:"lastLesson"`
		Recommendations []any `json:"recommendations"`
		HasCourses      bool  `json:"hasCourses"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !body.HasCourses {
		t.Fatal("hasCourses")
	}
	if body.LastLesson == nil || body.LastLesson.DomainID != tree.DomainID {
		t.Fatalf("lastLesson=%+v", body.LastLesson)
	}
	if body.LastLesson.SessionID != sess.ID || !body.LastLesson.CanResume {
		t.Fatalf("lastLesson resume=%+v", body.LastLesson)
	}
	if body.LastLesson.NodeTitle != "节点一" {
		t.Fatalf("nodeTitle=%q", body.LastLesson.NodeTitle)
	}
}
