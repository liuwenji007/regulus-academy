package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestInitProfileFromOnboarding(t *testing.T) {
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

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"产品经理，有 Python 基础，目标掌握 Go 并发。\"}"}}]}`))
	}))
	defer mock.Close()

	coach, err := NewCoach(store, llm.NewClient("test", mock.URL))
	if err != nil {
		t.Fatal(err)
	}

	summary, err := coach.InitProfileFromOnboarding(context.Background(), user.ID, "产品经理", "会 Python", "学 Go 并发")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "产品经理") {
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

func chdirCoachRepo(t *testing.T) {
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
