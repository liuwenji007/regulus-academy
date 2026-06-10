package domain

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const sampleExtendJSON = `{
  "layers": {
    "advanced": {
      "label": "精通", "time": "约 6 小时", "goal": "进阶实战",
      "nodes": [{"key":"rust_advanced_capstone","title":"Rust 实战项目"}]
    }
  },
  "nodes": [{
    "key":"rust_advanced_capstone","node":"Rust 实战项目","layer":"精通",
    "core_concepts":["项目结构"],"common_mistakes":["过度设计"],"boundaries":["不讲底层"],"exercise_ideas":["规划模块"]
  }],
  "modules": [{
    "key":"rust_capstone_mod","label":"实战进阶","nodes":["rust_advanced_capstone"]
  }]
}`

func TestValidateExtendOutputTruncatesExcessNodes(t *testing.T) {
	var out extendTreeOutput
	raw := `{
	  "layers": {
	    "advanced": {
	      "label": "精通", "nodes": [
	        {"key":"n1","title":"1"},{"key":"n2","title":"2"},{"key":"n3","title":"3"},
	        {"key":"n4","title":"4"},{"key":"n5","title":"5"},{"key":"n6","title":"6"}
	      ]
	    }
	  },
	  "nodes": [
	    {"key":"n1","node":"1","layer":"精通","core_concepts":["a"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]},
	    {"key":"n2","node":"2","layer":"精通","core_concepts":["b"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]},
	    {"key":"n3","node":"3","layer":"精通","core_concepts":["c"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]},
	    {"key":"n4","node":"4","layer":"精通","core_concepts":["d"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]},
	    {"key":"n5","node":"5","layer":"精通","core_concepts":["e"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]},
	    {"key":"n6","node":"6","layer":"精通","core_concepts":["f"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]}
	  ],
	  "modules": [{"key":"m","label":"进阶","nodes":["n1","n2","n3","n4","n5","n6"]}]
	}`
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	_, _, added, err := validateExtendOutput(nil, out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 5 {
		t.Fatalf("expected 5 kept, got %v", added)
	}
}

func TestValidateExtendOutputRejectsDuplicateKey(t *testing.T) {
	var out extendTreeOutput
	if err := json.Unmarshal([]byte(sampleExtendJSON), &out); err != nil {
		t.Fatal(err)
	}
	out.Layers["advanced"].Nodes[0].Key = "async_rust"
	_, _, _, err := validateExtendOutput([]string{"async_rust"}, out, 5)
	if err == nil {
		t.Fatal("应拒绝重复 key")
	}
}

func TestTreeBuilderExtend(t *testing.T) {
	t.Setenv("REGULUS_TREE_CRITIQUE", "0")
	tree := &storage.KnowledgeTree{
		DomainName: "Rust",
		Layers: []storage.TreeLayer{
			{Key: "entry", Label: "入门", Nodes: []storage.TreeNode{{Key: "rust_basics", Title: "基础"}}},
			{Key: "intermediate", Label: "熟悉", Nodes: []storage.TreeNode{{Key: "traits", Title: "Trait"}}},
			{Key: "advanced", Label: "精通", Nodes: []storage.TreeNode{{Key: "async_rust", Title: "异步"}}},
		},
		Modules: []storage.TreeModule{{Key: "core", Label: "核心", Nodes: []string{"rust_basics", "traits", "async_rust"}}},
	}
	nodes := map[string]NodeSpec{
		"rust_basics": {Key: "rust_basics", CoreConcepts: []string{"语法"}},
		"traits":      {Key: "traits", CoreConcepts: []string{"trait"}},
		"async_rust":  {Key: "async_rust", CoreConcepts: []string{"async"}},
	}
	mock := &mockLLM{jsonReply: sampleExtendJSON}
	builder := NewTreeBuilder(NewRegistry())
	result, err := builder.Extend(context.Background(), mock, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate}, tree, nodes, "", nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.AddedNodeKeys) != 1 || result.AddedNodeKeys[0] != "rust_advanced_capstone" {
		t.Fatalf("added=%v", result.AddedNodeKeys)
	}
}

func TestExtendNodeLimit(t *testing.T) {
	if got := extendNodeLimit(ScopeNarrow, 8, 10); got != 5 {
		t.Fatalf("narrow cap: got %d want 5", got)
	}
	if got := extendNodeLimit(ScopeModerate, 8, 10); got != 5 {
		t.Fatalf("moderate 10-node: got %d want 5", got)
	}
	if got := extendNodeLimit(ScopeBroad, 16, 20); got != 7 {
		t.Fatalf("broad large course: got %d want 7", got)
	}
	if got := extendNodeLimit(ScopeBroad, 20, 25); got != 8 {
		t.Fatalf("broad max: got %d want 8", got)
	}
}

func TestValidateExtendOutputAcceptsIntermediateLayer(t *testing.T) {
	var out extendTreeOutput
	raw := `{
	  "layers": {
	    "intermediate": {
	      "label": "熟悉",
	      "nodes": [{"key":"prod_debug","title":"生产排障"}]
	    }
	  },
	  "nodes": [{
	    "key":"prod_debug","node":"生产排障","layer":"熟悉",
	    "core_concepts":["日志"],"common_mistakes":[],"boundaries":[],"exercise_ideas":["x"]
	  }],
	  "modules": [{"key":"production","label":"生产实战","nodes":["prod_debug"]}]
	}`
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatal(err)
	}
	delta, _, added, err := validateExtendOutput([]string{"existing"}, out, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0] != "prod_debug" {
		t.Fatalf("added=%v", added)
	}
	if len(delta.Layers) != 1 || delta.Layers[0].Key != "intermediate" {
		t.Fatalf("layers=%+v", delta.Layers)
	}
}
