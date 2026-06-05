package domain

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/regulus-academy/regulus-academy/internal/llm"
)

const sampleTreeJSON = `{
  "domain": "Rust",
  "slug": "rust",
  "description": "系统学习 Rust，能独立开发可靠的后端服务",
  "modules": [
    { "key": "syntax_ownership", "label": "语法与所有权", "goal": "掌握基础语法与内存模型", "nodes": ["rust_basics", "ownership", "borrowing"] },
    { "key": "types_abstraction", "label": "类型与抽象", "goal": "结构体、枚举与 trait", "nodes": ["structs", "enums", "traits"] },
    { "key": "advanced_topics", "label": "进阶机制", "goal": "生命周期与异步", "nodes": ["lifetimes", "async_rust"] }
  ],
  "layers": {
    "entry": {
      "label": "入门", "time": "约 4～6 小时", "goal": "掌握基础语法与所有权，能读懂常见 Rust 代码并建立语言知识框架",
      "nodes": [{"key": "rust_basics", "title": "Rust 基础"}, {"key": "ownership", "title": "所有权"}, {"key": "borrowing", "title": "借用与引用"}]
    },
    "intermediate": {
      "label": "熟悉", "time": "约 12～18 小时", "goal": "能编写结构化的 Rust 程序，处理日常开发中的大多数场景",
      "nodes": [{"key": "structs", "title": "结构体"}, {"key": "enums", "title": "枚举与模式匹配"}, {"key": "traits", "title": "Trait 与泛型"}]
    },
    "advanced": {
      "label": "精通", "time": "约 20～30 小时", "goal": "能处理生命周期、异步与性能等复杂问题，应对绝大多数工程场景",
      "nodes": [{"key": "lifetimes", "title": "生命周期"}, {"key": "async_rust", "title": "异步 Rust"}]
    }
  },
  "nodes": [
    {"key":"rust_basics","node":"Rust 基础","layer":"入门","core_concepts":["变量与类型"],"common_mistakes":["混淆 mut"],"boundaries":["不讲 async"],"exercise_ideas":["解释 mut 的作用"]},
    {"key":"ownership","node":"所有权","layer":"入门","core_concepts":["所有权规则"],"common_mistakes":["双重可变借用"],"boundaries":["不讲生命周期细节"],"exercise_ideas":["判断能否编译"]},
    {"key":"borrowing","node":"借用与引用","layer":"入门","core_concepts":["不可变/可变借用"],"common_mistakes":["悬垂引用"],"boundaries":["不讲生命周期标注"],"exercise_ideas":["修复借用错误"]},
    {"key":"structs","node":"结构体","layer":"熟悉","core_concepts":["struct 定义"],"common_mistakes":["字段可见性"],"boundaries":["不讲 trait 对象"],"exercise_ideas":["定义一个 Point"]},
    {"key":"enums","node":"枚举","layer":"熟悉","core_concepts":["enum 与 match"],"common_mistakes":["遗漏分支"],"boundaries":["不讲 GADT"],"exercise_ideas":["用 match 处理 Option"]},
    {"key":"traits","node":"Trait 与泛型","layer":"熟悉","core_concepts":["trait 定义与实现"],"common_mistakes":["孤儿规则"],"boundaries":["不讲关联类型深入"],"exercise_ideas":["为类型实现 Display"]},
    {"key":"lifetimes","node":"生命周期","layer":"精通","core_concepts":["生命周期标注"],"common_mistakes":["不必要的 'a"],"boundaries":["不讲 HRTB"],"exercise_ideas":["标注函数签名"]},
    {"key":"async_rust","node":"异步 Rust","layer":"精通","core_concepts":["async/await"],"common_mistakes":["阻塞 runtime"],"boundaries":["不讲 tokio 源码"],"exercise_ideas":["写一个 async fn"]}
  ]
}`

func TestBuildTreePrompt_regeneratePreserveKeys(t *testing.T) {
	prompt := buildTreePrompt(IntentResult{Slug: "agent", DisplayName: "Agent", ScopeBreadth: ScopeBroad}, "Agent", "", []string{"agent_basics", "agent_tools"})
	if !strings.Contains(prompt, "重建课程") || !strings.Contains(prompt, "agent_basics") {
		t.Fatalf("regenerate prompt should list preserve keys: %s", prompt)
	}
}

func TestBuildTreePrompt_noTemplateGoalPhrases(t *testing.T) {
	prompt := buildTreePrompt(IntentResult{Slug: "agent", DisplayName: "Agent 原理", ScopeBreadth: ScopeBroad}, "Agent 原理", "", nil)
	for _, bad := range []string{"体现「看懂+知识框架」", "体现「能应用+常见场景」", "体现「高难度+绝大多数复杂场景」"} {
		if strings.Contains(prompt, bad) {
			t.Fatalf("prompt should not prime model with template goal %q", bad)
		}
	}
	if !strings.Contains(prompt, layerDefaults["entry"].Goal) {
		t.Fatal("prompt should include concrete entry goal example")
	}
}

func TestBuildTreePrompt_noTemplateTimePlaceholders(t *testing.T) {
	prompt := buildTreePrompt(IntentResult{Slug: "agent", DisplayName: "Agent 原理", ScopeBreadth: ScopeBroad}, "Agent 原理", "", nil)
	for _, bad := range []string{"按主题估算", "~2 小时", "~8 小时", "~20 小时"} {
		if strings.Contains(prompt, bad) {
			t.Fatalf("prompt should not prime model with template time %q", bad)
		}
	}
	if !strings.Contains(prompt, estimateLayerTime("entry", 3)) {
		t.Fatal("prompt should include concrete time hints")
	}
}

func TestBuildTreePromptMarshaledContainsMarkers(t *testing.T) {
	chdirRepo(t)
	prompt := buildTreePrompt(IntentResult{Slug: "go", DisplayName: "Go 语言", ScopeBreadth: ScopeBroad}, "Go 语言", "", nil)
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 知识树设计师。只输出 JSON。"},
		{Role: "user", Content: prompt},
	}
	body, err := json.Marshal(map[string]any{"messages": msgs})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "exercise_ideas") {
		t.Fatalf("missing exercise_ideas in marshaled body prefix=%q", s[:min(400, len(s))])
	}
}

func TestTreeBuilderBuildViaHTTPMock(t *testing.T) {
	chdirRepo(t)
	t.Setenv("REGULUS_TREE_CRITIQUE", "0")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "exercise_ideas") {
			t.Errorf("request missing exercise_ideas")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":` + strconv.Quote(sampleTreeJSON) + `}}]}`))
	}))
	defer srv.Close()

	client := llm.NewClient("test-key", srv.URL)
	builder := NewTreeBuilder(NewRegistry())
	intent := IntentResult{Slug: "rust", DisplayName: "Rust", Source: SourceGenerated, ScopeBreadth: ScopeModerate}
	_, _, err := builder.Build(context.Background(), client, intent, "rust", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestTreeBuilderBuild(t *testing.T) {
	mock := &mockLLM{jsonReply: sampleTreeJSON}
	builder := NewTreeBuilder(NewRegistry())
	intent := IntentResult{Slug: "rust", DisplayName: "Rust", Source: SourceGenerated, ScopeBreadth: ScopeModerate}
	tree, nodes, err := builder.Build(context.Background(), mock, intent, "rust", "后端开发，熟悉 Python")
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Layers) != 3 {
		t.Fatalf("layers=%d", len(tree.Layers))
	}
	if len(nodes) != 8 {
		t.Fatalf("nodes=%d", len(nodes))
	}
	if len(tree.Modules) != 3 {
		t.Fatalf("modules=%d", len(tree.Modules))
	}
	if tree.Layers[0].Goal == "" || tree.Layers[0].Time == "" {
		t.Fatal("层目标与时间不应为空")
	}
}

func TestValidateBuildOutputRejectsMissingNode(t *testing.T) {
	var out buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &out); err != nil {
		t.Fatal(err)
	}
	out.Nodes = out.Nodes[:1]
	_, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate})
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
	_, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate})
	if err == nil {
		t.Fatal("应拒绝缺少层级")
	}
}

func TestValidateBuildOutputAutoCorrectsGenericTime(t *testing.T) {
	var out buildTreeOutput
	if err := json.Unmarshal([]byte(sampleTreeJSON), &out); err != nil {
		t.Fatal(err)
	}
	entryNodes := out.Layers["entry"].Nodes
	out.Layers["entry"] = TreeLayerDef{
		Label: "入门", Time: "~2 小时", Goal: "看懂基础",
		Nodes: entryNodes,
	}
	out.Layers["advanced"] = TreeLayerDef{
		Label: "精通", Time: "约20小时", Goal: "高难度",
		Nodes: out.Layers["advanced"].Nodes,
	}
	tree, _, err := validateBuildOutput(out, IntentResult{DisplayName: "Rust", ScopeBreadth: ScopeModerate})
	if err != nil {
		t.Fatal(err)
	}
	if isGenericTime(tree.Layers[0].Time) || isGenericTime(tree.Layers[2].Time) {
		t.Fatalf("应已修正模板时间: entry=%q advanced=%q", tree.Layers[0].Time, tree.Layers[2].Time)
	}
	wantEntry := estimateLayerTime("entry", len(entryNodes))
	if tree.Layers[0].Time != wantEntry {
		t.Fatalf("entry time=%q want=%q", tree.Layers[0].Time, wantEntry)
	}
}

func TestEstimateLayerTime(t *testing.T) {
	got := estimateLayerTime("advanced", 2)
	if !strings.Contains(got, "小时") || isGenericTime(got) {
		t.Fatalf("estimate: %q", got)
	}
}

func TestNodeCountBoundsByScope(t *testing.T) {
	min, max := nodeCountBounds(ScopeNarrow)
	if min != 5 || max != 9 {
		t.Fatalf("narrow bounds=%d-%d", min, max)
	}
	min, max = nodeCountBounds(ScopeBroad)
	if min != 12 || max != 20 {
		t.Fatalf("broad bounds=%d-%d", min, max)
	}
}

func TestIsGenericTime(t *testing.T) {
	if !isGenericTime("~2 小时") {
		t.Fatal("应识别旧模板时间")
	}
	if isGenericTime("约 4～6 小时") {
		t.Fatal("不应误判实际估算时间")
	}
}
