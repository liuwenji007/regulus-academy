package domain

import (
	"context"
	"encoding/json"
	"testing"
)

const sampleTreeJSON = `{
  "domain": "Rust",
  "slug": "rust",
  "description": "Rust 编程",
  "layers": {
    "entry": {
      "label": "入门", "time": "~2 小时", "goal": "了解基础",
      "nodes": [{"key": "rust_basics", "title": "Rust 基础"}, {"key": "ownership", "title": "所有权"}]
    },
    "intermediate": {
      "label": "熟悉", "time": "~8 小时", "goal": "能写小程序",
      "nodes": [{"key": "structs", "title": "结构体"}, {"key": "enums", "title": "枚举"}]
    },
    "advanced": {
      "label": "精通", "time": "~20 小时", "goal": "理解底层",
      "nodes": [{"key": "lifetimes", "title": "生命周期"}, {"key": "async_rust", "title": "异步 Rust"}]
    }
  },
  "nodes": [
    {"key":"rust_basics","node":"Rust 基础","layer":"入门","core_concepts":["变量与类型"],"common_mistakes":["混淆 mut"],"boundaries":["不讲 async"],"exercise_ideas":["解释 mut 的作用"]},
    {"key":"ownership","node":"所有权","layer":"入门","core_concepts":["所有权规则"],"common_mistakes":["双重可变借用"],"boundaries":["不讲生命周期细节"],"exercise_ideas":["判断能否编译"]},
    {"key":"structs","node":"结构体","layer":"熟悉","core_concepts":["struct 定义"],"common_mistakes":["字段可见性"],"boundaries":["不讲 trait"],"exercise_ideas":["定义一个 Point"]},
    {"key":"enums","node":"枚举","layer":"熟悉","core_concepts":["enum 与 match"],"common_mistakes":["遗漏分支"],"boundaries":["不讲 GADT"],"exercise_ideas":["用 match 处理 Option"]},
    {"key":"lifetimes","node":"生命周期","layer":"精通","core_concepts":["生命周期标注"],"common_mistakes":["不必要的 'a"],"boundaries":["不讲 HRTB"],"exercise_ideas":["标注函数签名"]},
    {"key":"async_rust","node":"异步 Rust","layer":"精通","core_concepts":["async/await"],"common_mistakes":["阻塞 runtime"],"boundaries":["不讲 tokio 源码"],"exercise_ideas":["写一个 async fn"]}
  ]
}`

func TestTreeBuilderBuild(t *testing.T) {
	mock := &mockLLM{jsonReply: sampleTreeJSON}
	builder := NewTreeBuilder(NewRegistry())
	intent := IntentResult{Slug: "rust", DisplayName: "Rust", Source: SourceGenerated}
	tree, nodes, err := builder.Build(context.Background(), mock, intent, "rust")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Layers) != 3 {
		t.Fatalf("layers=%d", len(tree.Layers))
	}
	if len(nodes) != 6 {
		t.Fatalf("nodes=%d", len(nodes))
	}
}

func TestValidateBuildOutputRejectsMissingNode(t *testing.T) {
	var out buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &out); err != nil {
		t.Fatal(err)
	}
	out.Nodes = out.Nodes[:1]
	_, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust"})
	if err == nil {
		t.Fatal("应拒绝缺少节点边界")
	}
}

func TestValidateBuildOutputRejectsTooFewLayers(t *testing.T) {
	var out buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &out); err != nil {
		t.Fatal(err)
	}
	delete(out.Layers, "advanced")
	_, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust"})
	if err == nil {
		t.Fatal("应拒绝缺少层级")
	}
}
