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

func TestValidateExtendOutputRejectsDuplicateKey(t *testing.T) {
	var out extendTreeOutput
	if err := json.Unmarshal([]byte(sampleExtendJSON), &out); err != nil {
		t.Fatal(err)
	}
	out.Layers["advanced"].Nodes[0].Key = "async_rust"
	_, _, _, err := validateExtendOutput([]string{"async_rust"}, out, IntentResult{ScopeBreadth: ScopeModerate})
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
