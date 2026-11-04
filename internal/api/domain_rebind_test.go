package api

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRebindAndInvalidateTermCardsClearsCache(t *testing.T) {
	store, h, _ := setupTestServerStore(t, false, nil)

	tree := &storage.KnowledgeTree{
		DomainName: "术语缓存课",
		Layers: []storage.TreeLayer{{
			Key: "intro", Label: "入门", Time: "1h", Goal: "入门",
			Nodes: []storage.TreeNode{{Key: "n1", Title: "节点一"}},
		}},
	}
	dom, _, err := store.CreateDomainFromTree(storage.DefaultUserID, "术语缓存课", "term-rebind", "", tree, "{}", storage.DomainSourceGenerated, true, "")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.UpsertTermCard(&storage.TermCard{
		UserID:         storage.DefaultUserID,
		DomainID:       dom.ID,
		NodeKey:        "n1",
		NormalizedTerm: "goroutine",
		OriginalText:   "goroutine",
		CardJSON:       `{"term":"goroutine","definition":"旧定义"}`,
	}); err != nil {
		t.Fatal(err)
	}

	cached, err := store.GetTermCard(storage.DefaultUserID, dom.ID, "goroutine")
	if err != nil || cached == nil {
		t.Fatalf("expected term card before rebind, err=%v card=%v", err, cached)
	}

	h.rebindAndInvalidateTermCards(storage.DefaultUserID, dom.ID, "测试接地失败 domain=%s: %v")

	after, err := store.GetTermCard(storage.DefaultUserID, dom.ID, "goroutine")
	if err != nil {
		t.Fatal(err)
	}
	if after != nil {
		t.Fatalf("expected term card cache cleared after rebind, got %+v", after)
	}
}
