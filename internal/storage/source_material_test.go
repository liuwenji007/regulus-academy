package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDomainSourceMaterial_roundTripAndCopy(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "src-mat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	dom, tree, err := store.CreateDomainFromTree(
		DefaultUserID, "导入课", "import-demo", "", SampleTree("x", "导入课"), "{}", DomainSourceGenerated, true, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	has, err := store.HasDomainSourceMaterial(dom.ID)
	if err != nil || has {
		t.Fatalf("expected no material, has=%v err=%v", has, err)
	}

	if err := store.SaveDomainSourceMaterial(dom.ID, DomainSourceMaterial{
		Kind: "pdf", Label: "Palantir FDE.pdf", Text: "FDE 案例正文", PageCount: 3,
	}); err != nil {
		t.Fatal(err)
	}
	has, err = store.HasDomainSourceMaterial(dom.ID)
	if err != nil || !has {
		t.Fatalf("expected material, has=%v err=%v", has, err)
	}
	got, err := store.GetDomainSourceMaterial(DefaultUserID, dom.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind != "pdf" || got.Label != "Palantir FDE.pdf" || got.Text != "FDE 案例正文" || got.PageCount != 3 {
		t.Fatalf("got %+v", got)
	}
	if got.CharCount != len([]rune("FDE 案例正文")) {
		t.Fatalf("charCount=%d", got.CharCount)
	}

	dom2, _, err := store.CreateDomainFromTree(
		DefaultUserID, "导入课2", "import-demo-2", "", SampleTree("y", "导入课2"), "{}", DomainSourceGenerated, true, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CopyDomainSourceMaterial(dom.ID, dom2.ID); err != nil {
		t.Fatal(err)
	}
	copied, err := store.GetDomainSourceMaterial(DefaultUserID, dom2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copied.Text != got.Text {
		t.Fatalf("copied text=%q", copied.Text)
	}

	if err := store.DeleteDomain(DefaultUserID, tree.DomainID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetDomainSourceMaterial(DefaultUserID, tree.DomainID); err == nil {
		t.Fatal("material should be gone with domain")
	}
}

func TestGetDomainSourceMaterial_missing(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "src-mat-miss.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	dom, _, err := store.CreateDomainFromTree(
		DefaultUserID, "空课", "empty-src", "", SampleTree("z", "空课"), "{}", DomainSourceGenerated, true, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.GetDomainSourceMaterial(DefaultUserID, dom.ID)
	if err == nil || !strings.Contains(err.Error(), "没有导入原文") {
		t.Fatalf("want missing error, got %v", err)
	}
}
