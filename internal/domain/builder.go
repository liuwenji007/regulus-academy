package domain

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// TreeBuilder LLM 动态建树
type TreeBuilder struct {
	registry *Registry
}

// NewTreeBuilder 创建建树器
func NewTreeBuilder(r *Registry) *TreeBuilder {
	return &TreeBuilder{registry: r}
}

type buildTreeOutput struct {
	Domain      string                  `json:"domain"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	Modules     []TreeModuleDef         `json:"modules"`
	Layers      map[string]TreeLayerDef `json:"layers"`
	Nodes       []NodeSpec              `json:"nodes"`
}

const (
	ScopeNarrow   = "narrow"
	ScopeModerate = "moderate"
	ScopeBroad    = "broad"
)

var layerDefaults = map[string]struct {
	Label string
	Goal  string
}{
	"entry": {
		Label: "入门",
		Goal:  "快速掌握核心概念，能看懂相关代码与文档，建立该领域的知识框架",
	},
	"intermediate": {
		Label: "熟悉",
		Goal:  "能在真实项目中动手应用，独立应对大多数常见场景",
	},
	"advanced": {
		Label: "精通",
		Goal:  "能排查疑难问题、做架构取舍，覆盖绝大多数复杂场景",
	},
}

// Build 根据意图 LLM 生成知识树与节点边界；profile 为可选学生画像。
func (b *TreeBuilder) Build(ctx context.Context, client llm.Provider, intent IntentResult, userInput, profile string) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	return b.build(ctx, client, intent, userInput, profile, nil)
}

// BuildRegenerate 重建课程：在 prompt 中提示尽量复用旧 node key，便于进度迁移。
func (b *TreeBuilder) BuildRegenerate(ctx context.Context, client llm.Provider, intent IntentResult, userInput, profile string, preserveKeys []string) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	return b.build(ctx, client, intent, userInput, profile, preserveKeys)
}

func (b *TreeBuilder) build(ctx context.Context, client llm.Provider, intent IntentResult, userInput, profile string, preserveKeys []string) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	if !client.Configured() {
		return nil, nil, fmt.Errorf("未配置 LLM，无法生成知识树")
	}

	basePrompt := buildTreePrompt(intent, userInput, profile, preserveKeys)
	ReportBuildProgress(ctx, "build_tree", "正在生成知识树…")
	tree, nodes, err := b.generateAndValidate(ctx, client, intent, basePrompt, "")
	if err != nil {
		return nil, nil, err
	}

	issues := collectTreeQualityIssues(tree, nodes, intent)
	if len(issues) > 0 {
		logTreeQualityIssues(issues)
	}

	if !TreeCritiqueEnabled() {
		return tree, nodes, nil
	}

	ReportBuildProgress(ctx, "critique", "正在质检知识树…")
	critique, cerr := critiqueTree(ctx, client, tree, nodes, issues, intent)
	if cerr != nil {
		log.Printf("建树 critique 跳过: %v", cerr)
		return tree, nodes, nil
	}
	if critique.Severity != "fail" {
		return tree, nodes, nil
	}

	feedback := strings.TrimSpace(critique.Feedback)
	if feedback == "" {
		return tree, nodes, nil
	}
	log.Printf("建树 critique 不合格，尝试按反馈重生成: %s", feedback)
	ReportBuildProgress(ctx, "build_tree", "正在按质检反馈优化知识树…")
	tree2, nodes2, err2 := b.generateAndValidate(ctx, client, intent, basePrompt, feedback)
	if err2 != nil {
		log.Printf("建树 critique 重生成失败，保留初版: %v", err2)
		return tree, nodes, nil
	}
	return tree2, nodes2, nil
}

func (b *TreeBuilder) generateAndValidate(
	ctx context.Context,
	client llm.Provider,
	intent IntentResult,
	basePrompt, extraNote string,
) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	var lastErr error
	for attempt := 0; attempt < maxBuildAttempts; attempt++ {
		prompt := basePrompt
		if attempt > 0 && lastErr != nil {
			ReportBuildProgress(ctx, "build_tree", "知识树结构需修正，正在重新生成…")
			prompt += "\n\n上次输出未通过校验：" + lastErr.Error() + "。请修正 JSON 后重新输出。"
		}
		if extraNote != "" {
			prompt += "\n\n质检反馈（请据此修正知识树）：" + extraNote
		}

		var out buildTreeOutput
		msgs := []llm.Message{
			{Role: "system", Content: "你是 Regulus Academy 知识树设计师。根据具体领域为在职开发者设计可执行的三层渐进式学习路径。只输出 JSON。"},
			{Role: "user", Content: prompt},
		}
		genCtx := observability.WithGeneration(ctx, "domain.build_tree")
		if err := client.ChatJSON(genCtx, msgs, 0.4, &out); err != nil {
			return nil, nil, fmt.Errorf("知识树生成失败: %w", err)
		}

		tree, nodes, err := validateBuildOutput(out, intent)
		if err != nil {
			lastErr = err
			log.Printf("建树校验未通过（第 %d/%d 次 LLM 输出）: %v", attempt+1, maxBuildAttempts, err)
			continue
		}
		tree.DomainName = intent.DisplayName
		return tree, nodes, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未知校验错误")
	}
	return nil, nil, fmt.Errorf("知识树校验失败（已重试 %d 次）: %w", maxBuildAttempts-1, lastErr)
}

func nodeCountBounds(scope string) (minTotal, maxTotal int) {
	switch normalizeScope(scope) {
	case ScopeNarrow:
		return 5, 9
	case ScopeBroad:
		return 12, 20
	default:
		return 8, 14
	}
}

func normalizeScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ScopeNarrow, "small", "focused":
		return ScopeNarrow
	case ScopeBroad, "large", "wide":
		return ScopeBroad
	default:
		return ScopeModerate
	}
}

func buildTreePrompt(intent IntentResult, userInput, profile string, preserveKeys []string) string {
	core, _ := LoadPrompt("core")
	scope := normalizeScope(intent.ScopeBreadth)
	minTotal, maxTotal := nodeCountBounds(scope)

	var b strings.Builder
	b.WriteString("用户原话：")
	b.WriteString(userInput)
	b.WriteString("\n主题：")
	b.WriteString(intent.DisplayName)
	b.WriteString("\nslug：")
	b.WriteString(intent.Slug)
	b.WriteString("\n领域广度评估：")
	b.WriteString(scope)
	b.WriteString("（")
	b.WriteString(scopeBreadthHint(scope))
	b.WriteString("）\n")
	if intent.Reason != "" {
		b.WriteString("学习意图：")
		b.WriteString(intent.Reason)
		b.WriteString("\n")
	}
	profile = strings.TrimSpace(profile)
	if profile != "" {
		b.WriteString("\n【学生画像】（建树时参考，勿编造画像外事实）\n")
		b.WriteString(profile)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if core != "" {
		b.WriteString("学习方式参考：\n")
		b.WriteString(core)
		b.WriteString("\n\n")
	}
	if len(preserveKeys) > 0 {
		b.WriteString("## 重建课程（保留学习进度）\n\n")
		b.WriteString("- 下列 node key 来自用户旧版课程；概念仍对应时**必须复用相同 key**\n")
		b.WriteString("- 仅当概念合并或拆分时可改 key，并确保新节点标题能体现原概念\n")
		b.WriteString("- 旧 key 列表：")
		b.WriteString(strings.Join(preserveKeys, "、"))
		b.WriteString("\n\n")
	}

	b.WriteString(`## 三层定位（必须体现在各层 goal 中，可结合本主题改写）

- **入门**：快速掌握基础知识，能看懂代码/文档/讨论，建立该领域的知识框架（不是浅尝辄止，而是「看得懂地图」）
- **熟悉**：可以开始动手应用，能独立完成大多数日常/常见场景下的任务
- **精通**：能解决高难度与边界问题，在绝大多数复杂场景下仍能做出正确判断

`)

	entryTimeHint := estimateLayerTime("entry", 3)
	interTimeHint := estimateLayerTime("intermediate", 4)
	advTimeHint := estimateLayerTime("advanced", 2)

	b.WriteString(`## 时间与规模

- **time 按本主题各层实际节点数估算**（每节点约 40～55 分钟，含讲解、练习与消化；精通层可略长）
- 用自然中文区间填写，节点多则加长、少则缩短
- 下方 JSON 示例中的 time、goal 仅演示格式；须结合本主题与各层 nodes 数量分别填写，勿照抄示例原文
- 参考起点（节点数变化时请按比例改写，勿三层抄同一组数字）：`)
	fmt.Fprintf(&b, "entry %s，intermediate %s，advanced %s\n", entryTimeHint, interTimeHint, advTimeHint)
	fmt.Fprintf(&b, "- 本主题建议总节点数：%d～%d 个，按领域实际拆分，不要凑数\n\n", minTotal, maxTotal)

	b.WriteString(`## 主题模块 modules（与 layers 独立）

- **modules** = 知识结构分块（如「基础语法」「方法与接口」「泛型」「并发」），供知识图谱聚类展示
- **layers** = 学习进度深度（入门/熟悉/精通），供课程列表与学习路径使用
- 每个节点 key 必须恰好出现在一个 module 的 nodes 数组中
- module 的 label 用中文主题名，禁止复用「入门」「熟悉」「精通」
- 模块数量：窄主题 2～3 个，中等 3～5 个，宽主题 4～6 个

## JSON 结构

{
  "domain": "中文领域名",
  "slug": "与上文 slug 一致",
  "description": "一句话描述学完能做什么",
  "modules": [
    { "key": "basics", "label": "基础语法", "goal": "可选，一句话说明本模块覆盖什么", "nodes": ["node_key_a", "node_key_b"] },
    { "key": "concurrency", "label": "并发", "goal": "...", "nodes": ["node_key_c"] }
  ],
  "layers": {
`)
	fmt.Fprintf(&b, `    "entry": { "label": "入门", "time": "%s", "goal": "%s", "nodes": [{"key": "snake_case", "title": "节点中文名"}] },
    "intermediate": { "label": "熟悉", "time": "%s", "goal": "%s", "nodes": [{"key": "another_key", "title": "另一节点"}] },
    "advanced": { "label": "精通", "time": "%s", "goal": "%s", "nodes": [{"key": "advanced_key", "title": "进阶节点"}] }
`, entryTimeHint, layerDefaults["entry"].Goal,
		interTimeHint, layerDefaults["intermediate"].Goal,
		advTimeHint, layerDefaults["advanced"].Goal)
	b.WriteString(`  },
  "nodes": [
    {
      "key": "与 layers 中 key 一致",
      "node": "节点中文名",
      "layer": "入门/熟悉/精通 之一",
      "core_concepts": ["..."],
      "common_mistakes": ["..."],
      "boundaries": ["本节点不讲什么"],
      "exercise_ideas": ["可出的练习题方向"],
      "grading_hints": ["可选，批改时对照的评分要点短语"]
    }
  ]
}

## 硬性约束

- 必须包含 modules 数组，且每个节点 key 恰好归属 1 个 module
- 必须包含 entry、intermediate、advanced 三层
- 入门层 2～5 节点，熟悉层 2～6 节点，精通层 1～5 节点（窄主题偏少，宽主题偏多）
- 节点按由浅入深排列；boundaries 标明不越界，避免层与层之间内容重叠
- 当总节点数 ≤ 8：相邻节点 core_concepts 互不重复；boundaries 写明「由哪一节点负责」以免题面重叠
- 每个节点：exercise_ideas 条数 ≥ min(2, core_concepts 条数)（1 个 concept 至少 1 条；≥2 个 concept 至少 2 条），且每条 idea 尽量对应不同 concept
- 每个 layers 中的 key 必须在 nodes 数组中有完整边界定义
- key 用 snake_case 英文`)
	return b.String()
}

func scopeBreadthHint(scope string) string {
	switch scope {
	case ScopeNarrow:
		return "聚焦子话题，如「Go channel」「React Hooks」"
	case ScopeBroad:
		return "宽泛领域，如「Rust 语言」「分布式系统」「Agent 开发」"
	default:
		return "中等范围主题，如「Go 语言」「前端工程化」"
	}
}

func validateBuildOutput(out buildTreeOutput, intent IntentResult) (*storage.KnowledgeTree, map[string]NodeSpec, error) {
	order := []string{"entry", "intermediate", "advanced"}
	tree := &storage.KnowledgeTree{DomainName: out.Domain}
	if tree.DomainName == "" {
		tree.DomainName = intent.DisplayName
	}

	nodeKeys := map[string]struct{}{}
	for _, layerKey := range order {
		layer, ok := out.Layers[layerKey]
		if !ok {
			return nil, nil, fmt.Errorf("缺少层级 %s", layerKey)
		}
		nodes := make([]storage.TreeNode, len(layer.Nodes))
		for i, n := range layer.Nodes {
			if n.Key == "" {
				return nil, nil, fmt.Errorf("层级 %s 存在空 key", layerKey)
			}
			nodeKeys[n.Key] = struct{}{}
			nodes[i] = storage.TreeNode{Key: n.Key, Title: n.Title}
		}
		if len(nodes) == 0 {
			return nil, nil, fmt.Errorf("层级 %s 不能有 0 个节点", layerKey)
		}

		def := layerDefaults[layerKey]
		label := strings.TrimSpace(layer.Label)
		if label == "" {
			label = def.Label
		}
		goal := strings.TrimSpace(layer.Goal)
		if goal == "" {
			goal = def.Goal
		}
		timeEst := strings.TrimSpace(layer.Time)
		if timeEst == "" {
			return nil, nil, fmt.Errorf("层级 %s 缺少 time 估算", layerKey)
		}
		if isGenericTime(timeEst) {
			fixed := estimateLayerTime(layerKey, len(nodes))
			log.Printf("建树提示: 层级 %s 的 time %q 为模板值，已自动修正为 %q", layerKey, timeEst, fixed)
			timeEst = fixed
		}

		tree.Layers = append(tree.Layers, storage.TreeLayer{
			Key: layerKey, Label: label, Time: timeEst, Goal: goal, Nodes: nodes,
		})
	}

	if len(tree.Layers) != 3 {
		return nil, nil, fmt.Errorf("需要 3 层知识树")
	}

	nodes := make(map[string]NodeSpec, len(out.Nodes))
	for _, spec := range out.Nodes {
		if spec.Key == "" {
			continue
		}
		if _, ok := nodeKeys[spec.Key]; !ok {
			return nil, nil, fmt.Errorf("节点 %s 不在知识树中", spec.Key)
		}
		if len(spec.CoreConcepts) == 0 {
			return nil, nil, fmt.Errorf("节点 %s 缺少 core_concepts", spec.Key)
		}
		nodes[spec.Key] = spec
	}
	for key := range nodeKeys {
		if _, ok := nodes[key]; !ok {
			return nil, nil, fmt.Errorf("缺少节点边界定义: %s", key)
		}
	}

	MergeNodeRequires(tree, nodes)

	modules, err := validateModules(out.Modules, nodeKeys, intent.ScopeBreadth)
	if err != nil {
		return nil, nil, err
	}
	tree.Modules = modules

	return tree, nodes, nil
}

// isGenericTime 检测是否仍在使用旧版固定时间模板或 prompt 占位符。
func isGenericTime(timeEst string) bool {
	t := strings.ReplaceAll(strings.TrimSpace(timeEst), " ", "")
	generic := []string{
		"~2小时", "~8小时", "~20小时",
		"约2小时", "约8小时", "约20小时",
		"按主题估算",
	}
	for _, g := range generic {
		if strings.EqualFold(t, g) {
			return true
		}
	}
	return false
}

// estimateLayerTime 按该层节点数估算学习时长（每节点约 40～55 分钟含练习与消化）。
func estimateLayerTime(layerKey string, nodeCount int) string {
	if nodeCount < 1 {
		nodeCount = 1
	}
	minsLow := nodeCount * 40
	minsHigh := nodeCount * 55
	switch layerKey {
	case "intermediate":
		minsLow = minsLow * 5 / 4
		minsHigh = minsHigh * 5 / 4
	case "advanced":
		minsLow = minsLow * 3 / 2
		minsHigh = minsHigh * 3 / 2
	}
	lowH := (minsLow + 30) / 60
	if lowH < 1 {
		lowH = 1
	}
	highH := (minsHigh + 59) / 60
	if highH <= lowH {
		highH = lowH + 1
	}
	return fmt.Sprintf("约 %d～%d 小时", lowH, highH)
}
