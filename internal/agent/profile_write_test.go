package agent

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestWriteUserProfile_preservesPreference(t *testing.T) {
	store, err := storage.Open(t.TempDir() + "/write-profile.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	user, err := store.CreateUser("偏好保留")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteGlobalProfile(user.ID, "前端工程师", "补并发", "短答优先，少讲废话"); err != nil {
		t.Fatal(err)
	}

	if err := WriteUserProfile(store, user.ID, "【背景】后端转前端\n【目标】搞定 Kubernetes"); err != nil {
		t.Fatal(err)
	}
	u, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.ProfilePreference != "短答优先，少讲废话" {
		t.Fatalf("preference overwritten: %q", u.ProfilePreference)
	}
	if u.ProfileBackground == "" || u.ProfileGoal == "" {
		t.Fatalf("background/goal should update: bg=%q goal=%q", u.ProfileBackground, u.ProfileGoal)
	}
}

func TestWriteUserProfile_emptySummaryKeepsPreference(t *testing.T) {
	store, err := storage.Open(t.TempDir() + "/write-profile-empty.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	user, err := store.CreateUser("清空背景")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.WriteGlobalProfile(user.ID, "旧背景", "旧目标", "喜欢选择题"); err != nil {
		t.Fatal(err)
	}
	if err := WriteUserProfile(store, user.ID, ""); err != nil {
		t.Fatal(err)
	}
	u, err := store.GetUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if u.ProfilePreference != "喜欢选择题" {
		t.Fatalf("preference should survive empty summary write: %q", u.ProfilePreference)
	}
	if u.ProfileBackground != "" || u.ProfileGoal != "" {
		t.Fatalf("bg/goal should clear: bg=%q goal=%q", u.ProfileBackground, u.ProfileGoal)
	}
}
