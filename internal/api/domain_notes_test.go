package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestListDomainNotesAndMistakes(t *testing.T) {
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	tree := &storage.KnowledgeTree{
		DomainName: "测试课",
		Layers: []storage.TreeLayer{{
			Key: "intro", Label: "入门", Time: "2h", Goal: "入门",
			Nodes: []storage.TreeNode{{Key: "n1", Title: "节点一"}},
		}},
	}
	dom, _, err := store.CreateDomainFromTree(storage.DefaultUserID, "测试课", "test-notes", "", tree, "{}", storage.DomainSourceGenerated, true, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.UpsertNodeNote(storage.DefaultUserID, dom.ID, "n1", "## 核心理解\n我学会了"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertMistake(storage.DefaultUserID, dom.ID, "n1", "把 goroutine 当线程"); err != nil {
		t.Fatal(err)
	}

	notesResp, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/notes")
	if err != nil {
		t.Fatal(err)
	}
	defer notesResp.Body.Close()
	if notesResp.StatusCode != http.StatusOK {
		t.Fatalf("notes status = %d", notesResp.StatusCode)
	}
	var notesBody struct {
		DomainID string `json:"domainId"`
		Notes    []struct {
			NodeKey   string `json:"nodeKey"`
			ContentMd string `json:"contentMd"`
		} `json:"notes"`
	}
	if err := json.NewDecoder(notesResp.Body).Decode(&notesBody); err != nil {
		t.Fatal(err)
	}
	if notesBody.DomainID != dom.ID || len(notesBody.Notes) != 1 || notesBody.Notes[0].ContentMd == "" {
		t.Fatalf("unexpected notes body: %+v", notesBody)
	}

	singleResp, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/notes?nodeKey=n1")
	if err != nil {
		t.Fatal(err)
	}
	defer singleResp.Body.Close()
	if singleResp.StatusCode != http.StatusOK {
		t.Fatalf("single note status = %d", singleResp.StatusCode)
	}

	mistakesResp, err := http.Get(ts.URL + "/api/domain/" + dom.ID + "/mistakes")
	if err != nil {
		t.Fatal(err)
	}
	defer mistakesResp.Body.Close()
	if mistakesResp.StatusCode != http.StatusOK {
		t.Fatalf("mistakes status = %d", mistakesResp.StatusCode)
	}
	var mistakesBody struct {
		Mistakes []struct {
			NodeKey  string   `json:"nodeKey"`
			Concepts []string `json:"concepts"`
		} `json:"mistakes"`
	}
	if err := json.NewDecoder(mistakesResp.Body).Decode(&mistakesBody); err != nil {
		t.Fatal(err)
	}
	if len(mistakesBody.Mistakes) != 1 || len(mistakesBody.Mistakes[0].Concepts) != 1 {
		t.Fatalf("unexpected mistakes body: %+v", mistakesBody)
	}

	// 不存在的领域
	notFound, err := http.Get(ts.URL + "/api/domain/no-such-id/notes")
	if err != nil {
		t.Fatal(err)
	}
	defer notFound.Body.Close()
	if notFound.StatusCode != http.StatusNotFound {
		t.Fatalf("want 404, got %d", notFound.StatusCode)
	}

	_ = time.Now() // keep import stable if test helpers change
}
