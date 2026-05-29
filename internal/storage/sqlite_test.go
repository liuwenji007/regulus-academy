package storage

import (
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	_, tree, err := store.CreateDomain("Go 并发")
	if err != nil {
		t.Fatalf("CreateDomain: %v", err)
	}
	if tree.DomainName != "Go 并发" {
		t.Errorf("期望领域名 Go 并发，得到 %s", tree.DomainName)
	}
	if len(tree.Layers) != 3 {
		t.Errorf("期望 3 层，得到 %d", len(tree.Layers))
	}

	got, err := store.GetDomainTree(tree.DomainID)
	if err != nil {
		t.Fatalf("GetDomainTree: %v", err)
	}
	if got.DomainID != tree.DomainID {
		t.Errorf("领域 ID 不匹配")
	}
}

func TestProgressAndSession(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, tree, _ := store.CreateDomain("测试")
	sess, err := store.CreateSession(DefaultUserID, tree.DomainID, "go-concurrency", "goroutine_basics", "explain", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = store.UpsertProgress(UserProgress{
		UserID:   DefaultUserID,
		DomainID: tree.DomainID,
		NodeKey:  "goroutine_basics",
		Layer:    "entry",
		Status:   "in_progress",
		Mastery:  0.5,
	})
	if err != nil {
		t.Fatal(err)
	}

	list, err := store.ListProgress(DefaultUserID, tree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("期望 1 条进度，得到 %d", len(list))
	}

	_, err = store.AddMessage(sess.ID, "user", "你好")
	if err != nil {
		t.Fatal(err)
	}
	msgs, err := store.ListMessages(sess.ID)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("消息数量错误: %v, len=%d", err, len(msgs))
	}
}
