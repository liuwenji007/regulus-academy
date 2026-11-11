package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestPatchDomainFieldAndListFields(t *testing.T) {
	store, _, ts := setupTestServerStore(t, false, nil)
	defer ts.Close()

	dom, _, err := store.CreateDomainFromTree(storage.DefaultUserID, "Go 语言", "go", "", &storage.KnowledgeTree{
		DomainName: "Go 语言",
		Layers:     []storage.TreeLayer{{Key: "l", Label: "入门", Nodes: []storage.TreeNode{{Key: "a", Title: "变量"}}}},
	}, "{}", storage.DomainSourceGenerated, true, "")
	if err != nil {
		t.Fatal(err)
	}

	body := bytes.NewBufferString(`{"field":"编程语言"}`)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/domain/"+dom.ID+"/field", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status %d", resp.StatusCode)
	}

	got, err := store.GetDomain(storage.DefaultUserID, dom.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field != "编程语言" || got.FieldSource != "user" {
		t.Fatalf("stored %+v", got)
	}

	listResp, err := http.Get(ts.URL + "/api/domain-fields")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var payload struct {
		Presets []string `json:"presets"`
		Fields  []string `json:"fields"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Presets) != len(domain.PresetFields) {
		t.Fatalf("presets %+v", payload.Presets)
	}
	found := false
	for _, field := range payload.Fields {
		if field == "编程语言" {
			found = true
		}
	}
	if !found {
		t.Fatalf("fields %+v", payload.Fields)
	}
}

func TestRegenerateDomainPreservesUserField(t *testing.T) {
	chdirToRepo(t)
	mock := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body := readBody(r)
		if strings.Contains(body, "课程分类器") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"field\":\"编程语言\",\"nearest\":\"编程语言\",\"confidence\":0.95,\"reason\":\"mock\"}"}}]}`))
			return
		}
		if strings.Contains(body, "知识树设计师") {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleRustTreeJSON) + `}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"slug\":\"rust\",\"displayName\":\"Rust\",\"confidence\":0.9,\"reason\":\"用户想学 Rust\",\"scopeBreadth\":\"moderate\"}"}}]}`))
	}
	store, _, ts := setupTestServerWithHandler(t, true, mock)
	defer ts.Close()

	result := buildDomainResult(t, ts.URL, map[string]string{"name": "rust"})
	tree := decodeBuildTreeFromResult(t, result)
	if err := store.UpdateDomainField(storage.DefaultUserID, tree.DomainID, "人文与通识", "user"); err != nil {
		t.Fatal(err)
	}

	regBody, _ := json.Marshal(map[string]string{"confirmName": tree.DomainName})
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/domain/"+tree.DomainID+"/regenerate", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("regenerate status=%d body=%s", resp.StatusCode, string(b))
	}
	var regResult map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&regResult); err != nil {
		t.Fatal(err)
	}
	newTree := decodeBuildTreeFromMap(t, regResult)
	if newTree.DomainID == tree.DomainID {
		t.Fatal("expected new domain id after regenerate")
	}
	got, err := store.GetDomain(storage.DefaultUserID, newTree.DomainID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Field != "人文与通识" || got.FieldSource != "user" {
		t.Fatalf("field not preserved after regenerate: field=%q source=%q", got.Field, got.FieldSource)
	}
}
