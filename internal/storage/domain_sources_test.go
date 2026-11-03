package storage

import (
	"path/filepath"
	"testing"
)

func TestDomainSourceRoundTripAndDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, tree, err := store.CreateDomain("内部课")
	if err != nil {
		t.Fatal(err)
	}
	src, err := store.InsertDomainSource(&DomainSource{
		UserID:   DefaultUserID,
		DomainID: tree.DomainID,
		Kind:     "text",
		Filename: "wiki.txt",
		Text:     "计费阈值默认为 42。超过上限必须拒绝。",
	}, []string{"计费阈值默认为 42。", "超过上限必须拒绝。"})
	if err != nil {
		t.Fatal(err)
	}
	meta, err := store.ListDomainSourceMeta(DefaultUserID, tree.DomainID)
	if err != nil || len(meta) != 1 || meta[0].ChunkCount != 2 {
		t.Fatalf("meta=%+v err=%v", meta, err)
	}
	chunks, err := store.ListSourceChunks(DefaultUserID, tree.DomainID)
	if err != nil || len(chunks) != 2 || chunks[0].Filename != "wiki.txt" {
		t.Fatalf("chunks=%+v err=%v", chunks, err)
	}
	if err := store.ApplyChunkBindings(DefaultUserID, tree.DomainID, map[string]string{chunks[0].ID: "quota"}); err != nil {
		t.Fatal(err)
	}
	chunks, _ = store.ListSourceChunks(DefaultUserID, tree.DomainID)
	if chunks[0].NodeKey != "quota" || chunks[1].NodeKey != "" {
		t.Fatalf("bindings=%+v", chunks)
	}

	_, tree2, err := store.CreateDomain("内部课二")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.MoveDomainSources(DefaultUserID, tree.DomainID, tree2.DomainID); err != nil {
		t.Fatal(err)
	}
	left, _ := store.ListDomainSourceMeta(DefaultUserID, tree.DomainID)
	moved, _ := store.ListDomainSourceMeta(DefaultUserID, tree2.DomainID)
	if len(left) != 0 || len(moved) != 1 || moved[0].ID != src.ID {
		t.Fatalf("left=%d moved=%+v", len(left), moved)
	}
	movedChunks, _ := store.ListSourceChunks(DefaultUserID, tree2.DomainID)
	for _, c := range movedChunks {
		if c.NodeKey != "" {
			t.Fatalf("node key not cleared: %+v", c)
		}
	}
	if err := store.DeleteDomain(DefaultUserID, tree2.DomainID); err != nil {
		t.Fatal(err)
	}
	after, _ := store.ListDomainSourceMeta(DefaultUserID, tree2.DomainID)
	if len(after) != 0 {
		t.Fatalf("orphans=%+v", after)
	}
}
