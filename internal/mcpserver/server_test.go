package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestMCPToolsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	tree := &storage.KnowledgeTree{
		DomainName: "MCP 课",
		Layers: []storage.TreeLayer{{
			Key: "intro", Label: "入门", Time: "1h", Goal: "入门",
			Nodes: []storage.TreeNode{{Key: "n1", Title: "第一节"}},
		}},
	}
	dom, _, err := store.CreateDomainFromTree(storage.DefaultUserID, "MCP 课", "mcp-demo", "", tree, "{}", storage.DomainSourceGenerated, true, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = store.UpsertNodeNote(storage.DefaultUserID, dom.ID, "n1", "笔记内容")
	_ = store.UpsertMistake(storage.DefaultUserID, dom.ID, "n1", "踩坑概念")

	srv := New(store, nil, Config{UserID: storage.DefaultUserID, WebBaseURL: "http://localhost:8080"})

	var in bytes.Buffer
	writeLine(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params":  map[string]any{},
	})
	writeLine(&in, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	writeLine(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})
	writeLine(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "get_notes",
			"arguments": map[string]any{"domainId": dom.ID, "nodeKey": "n1"},
		},
	})
	writeLine(&in, map[string]any{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "open_session_link",
			"arguments": map[string]any{"domainId": dom.ID, "nodeKey": "n1"},
		},
	})

	var out bytes.Buffer
	if err := srv.Run(context.Background(), &in, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected >=3 responses, got %d: %q", len(lines), out.String())
	}

	var listResp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatal(err)
	}
	if len(listResp.Result.Tools) != 5 {
		t.Fatalf("want 5 tools, got %d", len(listResp.Result.Tools))
	}

	var notesResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[2]), &notesResp); err != nil {
		t.Fatal(err)
	}
	if notesResp.Result.IsError || !strings.Contains(notesResp.Result.Content[0].Text, "笔记内容") {
		t.Fatalf("unexpected notes result: %+v", notesResp.Result)
	}

	var linkResp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(lines[3]), &linkResp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(linkResp.Result.Content[0].Text, "#/tree/"+dom.ID) {
		t.Fatalf("deep link missing, got %s", linkResp.Result.Content[0].Text)
	}
	if !strings.Contains(linkResp.Result.Content[0].Text, "node=n1") {
		t.Fatalf("nodeKey missing from deep link url, got %s", linkResp.Result.Content[0].Text)
	}
}

func writeLine(buf *bytes.Buffer, v any) {
	b, _ := json.Marshal(v)
	buf.Write(b)
	buf.WriteByte('\n')
}
