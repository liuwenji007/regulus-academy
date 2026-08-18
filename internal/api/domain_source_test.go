package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestGetDomainSourceMaterialAPI(t *testing.T) {
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	tree := &storage.KnowledgeTree{
		DomainName: "导入课",
		Layers: []storage.TreeLayer{{
			Key: "intro", Label: "入门", Time: "2h", Goal: "入门",
			Nodes: []storage.TreeNode{{Key: "n1", Title: "节点一"}},
		}},
	}
	dom, _, err := store.CreateDomainFromTree(storage.DefaultUserID, "导入课", "src-mat", "", tree, "{}", storage.DomainSourceGenerated, true, "")
	if err != nil {
		t.Fatal(err)
	}

	miss, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/source-material")
	if err != nil {
		t.Fatal(err)
	}
	defer miss.Body.Close()
	if miss.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404 before save, got %d", miss.StatusCode)
	}

	if err := store.SaveDomainSourceMaterial(dom.ID, storage.DomainSourceMaterial{
		Kind: "pdf", Label: "demo.pdf", Text: "抽出的 Palantir 正文", PageCount: 2,
	}); err != nil {
		t.Fatal(err)
	}

	treeResp, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/tree")
	if err != nil {
		t.Fatal(err)
	}
	defer treeResp.Body.Close()
	if treeResp.StatusCode != http.StatusOK {
		t.Fatalf("tree status=%d", treeResp.StatusCode)
	}
	var treeBody struct {
		HasSourceMaterial bool `json:"hasSourceMaterial"`
	}
	if err := json.NewDecoder(treeResp.Body).Decode(&treeBody); err != nil {
		t.Fatal(err)
	}
	if !treeBody.HasSourceMaterial {
		t.Fatal("tree should flag hasSourceMaterial")
	}

	got, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/source-material")
	if err != nil {
		t.Fatal(err)
	}
	defer got.Body.Close()
	if got.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", got.StatusCode)
	}
	var body storage.DomainSourceMaterial
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Text != "抽出的 Palantir 正文" || body.Kind != "pdf" {
		t.Fatalf("body=%+v", body)
	}
}
