package storage

import (
	"path/filepath"
	"testing"
)

func TestResolveKnowledgeGapsByNode_OnlyMatched(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "gaps.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	uid := DefaultUserID
	domainID := "dom1"

	// 在节点 A 发现的前置缺口，matched 指向 B
	prereq, err := store.UpsertKnowledgeGap(&KnowledgeGap{
		UserID: uid, DomainID: domainID, NodeKey: "node-a",
		Concept: "事务", Source: GapSourceAsideLookup,
		MatchedDomainID: domainID, MatchedNodeKey: "node-b",
		Reason: "前置",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 在节点 A 发现但尚无匹配的前置（matched 空）
	orphan, err := store.UpsertKnowledgeGap(&KnowledgeGap{
		UserID: uid, DomainID: domainID, NodeKey: "node-a",
		Concept: "2PC", Source: GapSourceAsideLookup, Reason: "无匹配",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 错题挂在 A（matched=A）
	mistake, err := store.UpsertKnowledgeGap(&KnowledgeGap{
		UserID: uid, DomainID: domainID, NodeKey: "node-a",
		Concept: "幂等", Source: GapSourceMistake,
		MatchedDomainID: domainID, MatchedNodeKey: "node-a",
		Reason: "答错",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.ResolveKnowledgeGapsByNode(uid, domainID, "node-a"); err != nil {
		t.Fatal(err)
	}

	open, err := store.ListOpenKnowledgeGaps(uid, domainID, 20)
	if err != nil {
		t.Fatal(err)
	}
	openIDs := map[int64]bool{}
	for _, g := range open {
		openIDs[g.ID] = true
	}

	if openIDs[mistake.ID] {
		t.Error("学完 A 应关闭 matched=A 的错题缺口")
	}
	if !openIDs[prereq.ID] {
		t.Error("学完 A 不应关闭 matched=B 的前置缺口")
	}
	if !openIDs[orphan.ID] {
		t.Error("学完 A 不应关闭 matched 为空的发现缺口")
	}

	if err := store.ResolveKnowledgeGapsByNode(uid, domainID, "node-b"); err != nil {
		t.Fatal(err)
	}
	open, err = store.ListOpenKnowledgeGaps(uid, domainID, 20)
	if err != nil {
		t.Fatal(err)
	}
	still := false
	for _, g := range open {
		if g.ID == prereq.ID {
			still = true
		}
	}
	if still {
		t.Error("学完 B 应关闭 matched=B 的前置缺口")
	}
}

func TestUpsertTermCardAtomic(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "terms.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c1, err := store.UpsertTermCard(&TermCard{
		UserID: DefaultUserID, DomainID: "d", NormalizedTerm: "rpc",
		OriginalText: "RPC", CardJSON: `{"term":"RPC"}`,
	})
	if err != nil || c1 == nil || c1.HitCount != 1 {
		t.Fatalf("first insert: %+v err=%v", c1, err)
	}
	c2, err := store.UpsertTermCard(&TermCard{
		UserID: DefaultUserID, DomainID: "d", NormalizedTerm: "rpc",
		OriginalText: "RPC", CardJSON: `{"term":"RPC"}`,
	})
	if err != nil || c2 == nil || c2.HitCount != 2 {
		t.Fatalf("second hit: %+v err=%v", c2, err)
	}
}
