package domain

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestGetNodeFromDB(t *testing.T) {
	chdirRepo(t)
	store, err := storage.Open(filepath.Join(t.TempDir(), "nodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	reg := NewRegistry()
	tree, nodes, err := reg.LoadTreeAndNodes("go-concurrency")
	if err != nil {
		t.Fatal(err)
	}
	nodesJSON, _ := json.Marshal(nodes)
	_, tree, err = store.CreateDomainFromTree(storage.DefaultUserID, "Go 并发", "go-concurrency-test", tree, string(nodesJSON), storage.DomainSourceGenerated, false)
	if err != nil {
		t.Fatal(err)
	}

	spec, err := reg.GetNode(store, tree.DomainID, "nonexistent-slug", "goroutine_basics")
	if err != nil {
		t.Fatal(err)
	}
	if spec.Key != "goroutine_basics" && spec.Node == "" {
		t.Fatalf("got %+v", spec)
	}
}
