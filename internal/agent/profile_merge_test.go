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

func TestRefineUserProfile(t *testing.T) {
	chdirCoachRepo(t)
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	user, err := store.CreateUser("测试用户")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpdateUserProfileSummary(user.ID, "【背景】产品经理\n【进展】会 Python")

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"【背景】产品经理\\n【进展】会 Python，正在学 Go\"}"}}]}`))
	}))
	defer mock.Close()

	coach, err := NewCoach(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	summary, err := coach.RefineUserProfile(context.Background(), user.ID, "最近在学 Go 并发")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "Go") {
		t.Fatalf("summary=%q", summary)
	}
	u, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.ProfileSummary != summary {
		t.Fatalf("db profile=%q", u.ProfileSummary)
	}
}
