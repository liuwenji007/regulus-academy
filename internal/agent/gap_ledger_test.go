package agent

import (
	"path/filepath"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestRecordFromTermCard_OnlyPrerequisites(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ledger := NewGapLedger(store)
	card := &TermCardPayload{
		Term:          "RPC",
		OriginalText:  "RPC",
		Prerequisites: []string{"网络", "序列化"},
	}
	ledger.RecordFromTermCard(storage.DefaultUserID, "dom", "node-a", card)

	gaps, err := store.ListOpenKnowledgeGaps(storage.DefaultUserID, "dom", 20)
	if err != nil {
		t.Fatal(err)
	}
	concepts := map[string]bool{}
	for _, g := range gaps {
		concepts[g.Concept] = true
		if g.NodeKey != "node-a" {
			t.Errorf("discovery node_key want node-a, got %s", g.NodeKey)
		}
	}
	if concepts["rpc"] || concepts["RPC"] {
		t.Error("划词术语本身不应记为缺口")
	}
	if !concepts["网络"] || !concepts["序列化"] {
		t.Fatalf("应记录前置: %v", concepts)
	}
}
