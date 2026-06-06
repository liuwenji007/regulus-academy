package domain

import (
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func TestExtendEligibilityThreshold(t *testing.T) {
	tree := &storage.KnowledgeTree{
		Layers: []storage.TreeLayer{
			{Key: "entry", Nodes: []storage.TreeNode{{Key: "a"}, {Key: "b"}, {Key: "c"}, {Key: "d"}, {Key: "e"}}},
			{Key: "intermediate", Nodes: []storage.TreeNode{{Key: "f"}, {Key: "g"}, {Key: "h"}, {Key: "i"}, {Key: "j"}}},
		},
	}
	progress := []storage.UserProgress{
		{Status: "completed"}, {Status: "completed"}, {Status: "completed"}, {Status: "completed"},
		{Status: "completed"}, {Status: "completed"}, {Status: "completed"}, {Status: "completed"},
		{Status: "in_progress"},
	}
	eligible, completed, total, _ := ExtendEligibility(tree, progress, 0.8)
	if total != 10 || completed != 8 {
		t.Fatalf("completed=%d total=%d", completed, total)
	}
	if !eligible {
		t.Fatal("8/10 应达标")
	}

	progress = progress[:7]
	eligible, _, _, reason := ExtendEligibility(tree, progress, 0.8)
	if eligible || reason == "" {
		t.Fatalf("7/10 不应达标 eligible=%v reason=%q", eligible, reason)
	}
}
