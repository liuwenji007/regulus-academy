package domain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchAndLoadTree(t *testing.T) {
	wd, _ := os.Getwd()
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			_ = os.Chdir(d)
			break
		}
	}
	r := NewRegistry()
	slug, ok := r.MatchDomain("Go 并发")
	if !ok || slug != "go-concurrency" {
		t.Fatalf("match failed: %s %v", slug, ok)
	}
	tree, err := r.LoadTree(slug)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Layers) != 3 {
		t.Fatalf("layers=%d", len(tree.Layers))
	}
	node, err := r.LoadNode(slug, "channel")
	if err != nil {
		t.Fatal(err)
	}
	if node.Key != "channel" {
		t.Fatalf("node key=%s", node.Key)
	}
}

func TestLoadPrompt(t *testing.T) {
	wd, _ := os.Getwd()
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			_ = os.Chdir(d)
			break
		}
	}
	p, err := LoadPrompt("core")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) < 50 {
		t.Fatal("core prompt too short")
	}
}

func TestLoadProtocol(t *testing.T) {
	wd, _ := os.Getwd()
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "regulus-coach")); err == nil {
			_ = os.Chdir(d)
			break
		}
	}
	p, err := LoadProtocol()
	if err != nil {
		t.Fatal(err)
	}
	if len(p) < 50 {
		t.Fatal("protocol too short")
	}
}
