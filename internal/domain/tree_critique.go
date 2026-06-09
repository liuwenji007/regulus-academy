package domain

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const maxBuildAttempts = 2

// TreeCritiqueEnabled 默认开启；设 REGULUS_TREE_CRITIQUE=0|false|no 可关闭建树 critique。
func TreeCritiqueEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_TREE_CRITIQUE")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// treeCritiqueOutput LLM 单轮质检结果。
type treeCritiqueOutput struct {
	Pass     bool   `json:"pass"`
	Severity string `json:"severity"`
	Feedback string `json:"feedback"`
}

var recommendedLayerBounds = map[string]struct{ min, max int }{
	"entry":        {2, 5},
	"intermediate": {2, 6},
	"advanced":     {1, 5},
}

// collectTreeQualityIssues 程序化质量建议（供日志与 critique 输入，不阻断落库）。
func collectTreeQualityIssues(tree *storage.KnowledgeTree, nodes map[string]NodeSpec, intent IntentResult) []string {
	var issues []string
	totalNodes := countTreeNodes(tree)
	minTotal, maxTotal := nodeCountBounds(intent.ScopeBreadth)
	if totalNodes > 0 && (totalNodes < minTotal || totalNodes > maxTotal) {
		issues = append(issues, fmt.Sprintf(
			"节点总数 %d，建议 %d-%d（主题广度 %s）",
			totalNodes, minTotal, maxTotal, normalizeScope(intent.ScopeBreadth),
		))
	}
	if tree != nil {
		for _, layer := range tree.Layers {
			b, ok := recommendedLayerBounds[layer.Key]
			if !ok {
				continue
			}
			n := len(layer.Nodes)
			if n < b.min {
				issues = append(issues, fmt.Sprintf("层级 %s 建议至少 %d 个节点，当前 %d", layer.Label, b.min, n))
			}
			if n > b.max {
				issues = append(issues, fmt.Sprintf("层级 %s 建议最多 %d 个节点，当前 %d", layer.Label, b.max, n))
			}
		}
		if len(tree.Modules) > 0 {
			minM, maxM := moduleCountBounds(intent.ScopeBreadth)
			if len(tree.Modules) < minM || len(tree.Modules) > maxM {
				issues = append(issues, fmt.Sprintf("主题模块数 %d，建议 %d-%d", len(tree.Modules), minM, maxM))
			}
		}
	}
	seenConcept := map[string]string{}
	for key, spec := range nodes {
		if strings.TrimSpace(spec.Node) == "" {
			issues = append(issues, fmt.Sprintf("节点 %s 缺少标题", key))
		}
		if len(spec.Boundaries) == 0 {
			issues = append(issues, fmt.Sprintf("节点 %s 缺少 boundaries", key))
		}
		if len(spec.CommonMistakes) == 0 {
			issues = append(issues, fmt.Sprintf("节点 %s 缺少 common_mistakes", key))
		}
		minIdeas := minExerciseIdeasRequired(len(spec.CoreConcepts))
		if minIdeas > 0 && len(spec.ExerciseIdeas) < minIdeas {
			issues = append(issues, fmt.Sprintf("节点 %s 的 exercise_ideas 不足（需至少 %d 条，当前 %d 条）", key, minIdeas, len(spec.ExerciseIdeas)))
		}
		for _, c := range spec.CoreConcepts {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if other, ok := seenConcept[c]; ok && other != key {
				issues = append(issues, fmt.Sprintf("core_concept %q 在节点 %s 与 %s 重复", c, other, key))
			} else {
				seenConcept[c] = key
			}
		}
		if len(spec.CoreConcepts) > 0 && len(spec.TeachingBeats) == 0 {
			issues = append(issues, fmt.Sprintf("节点 %s 缺少 teaching_beats（将使用 fallback）", key))
		}
	}
	if totalNodes > 0 && totalNodes <= 8 {
		issues = append(issues, fmt.Sprintf("节点数 %d（≤8），请确认相邻节点 boundaries 已区分职责", totalNodes))
	}
	issues = append(issues, collectInvalidRequires(nodes)...)
	return issues
}

func collectInvalidRequires(nodes map[string]NodeSpec) []string {
	var issues []string
	for key, spec := range nodes {
		for _, req := range spec.Requires {
			req = strings.TrimSpace(req)
			if req == "" {
				continue
			}
			if _, ok := nodes[req]; !ok {
				issues = append(issues, fmt.Sprintf("节点 %s 的 requires 引用不存在的前置 %q", key, req))
			}
		}
	}
	return issues
}

func logTreeQualityIssues(issues []string) {
	for _, issue := range issues {
		log.Printf("建树提示: %s", issue)
	}
}

const treeCritiqueSystemPrompt = "你是 Regulus Academy 知识树质检员。用户消息已包含完整待检知识树（层内顺序与节点明细）及程序化检查建议，请直接基于该内容评估，不要要求用户再补充节点。只输出 JSON：pass（bool）、severity（ok|warn|fail）、feedback（中文，fail 时给出可执行的修正建议）。质量建议：节点总数/各层节点数/模块数偏离建议区间、exercise_ideas 不足、boundaries 缺失等——轻微偏离用 warn，严重偏离或结构重叠用 fail。exercise_ideas：core 仅 1 条时至少 1 条 idea；≥2 条时至少 2 条 idea。"

func buildTreeCritiqueUserMessage(
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	issues []string,
	intent IntentResult,
) string {
	var b strings.Builder
	b.WriteString("主题：")
	b.WriteString(intent.DisplayName)
	b.WriteString("\n领域广度：")
	b.WriteString(normalizeScope(intent.ScopeBreadth))
	b.WriteString("\n节点数：")
	fmt.Fprintf(&b, "%d\n", len(nodes))
	if len(issues) > 0 {
		b.WriteString("\n程序化检查发现：\n")
		for _, issue := range issues {
			b.WriteString("- ")
			b.WriteString(issue)
			b.WriteString("\n")
		}
	}
	if tree != nil {
		b.WriteString("\n三层目标摘要：\n")
		for _, layer := range tree.Layers {
			b.WriteString(layer.Label)
			b.WriteString("：")
			b.WriteString(layer.Goal)
			b.WriteString("\n")
		}
		appendTreeLayerOrder(&b, tree)
	}
	appendTreeNodeDetails(&b, tree, nodes)
	b.WriteString("\n请对照：覆盖学习目标 / 相邻节点是否重叠 / 难度梯度 / 节点规模是否合理。")
	return b.String()
}

func appendTreeLayerOrder(b *strings.Builder, tree *storage.KnowledgeTree) {
	if tree == nil || len(tree.Layers) == 0 {
		return
	}
	b.WriteString("\n【层内节点顺序】\n")
	for _, layer := range tree.Layers {
		fmt.Fprintf(b, "%s (%s):\n", layer.Label, layer.Key)
		for _, n := range layer.Nodes {
			fmt.Fprintf(b, "  - %s: %s\n", n.Key, n.Title)
		}
	}
}

func appendTreeNodeDetails(b *strings.Builder, tree *storage.KnowledgeTree, nodes map[string]NodeSpec) {
	if len(nodes) == 0 {
		return
	}
	b.WriteString("\n【节点明细】\n")
	written := map[string]struct{}{}
	writeNode := func(key string, spec NodeSpec) {
		if _, ok := written[key]; ok {
			return
		}
		written[key] = struct{}{}
		formatNodeSpecForCritique(b, key, spec)
	}
	if tree != nil {
		for _, layer := range tree.Layers {
			for _, n := range layer.Nodes {
				spec, ok := nodes[n.Key]
				if !ok {
					fmt.Fprintf(b, "\n### %s: %s\n（缺少 nodes 定义）\n", n.Key, n.Title)
					written[n.Key] = struct{}{}
					continue
				}
				writeNode(n.Key, spec)
			}
		}
	}
	for key, spec := range nodes {
		writeNode(key, spec)
	}
}

func formatNodeSpecForCritique(b *strings.Builder, key string, spec NodeSpec) {
	title := strings.TrimSpace(spec.Node)
	if title == "" {
		title = key
	}
	layer := strings.TrimSpace(spec.Layer)
	if layer == "" {
		layer = "—"
	}
	fmt.Fprintf(b, "\n### %s: %s（%s）\n", key, title, layer)
	writeCritiqueList(b, "core_concepts", spec.CoreConcepts)
	writeCritiqueList(b, "boundaries", spec.Boundaries)
	writeCritiqueList(b, "common_mistakes", spec.CommonMistakes)
	minIdeas := minExerciseIdeasRequired(len(spec.CoreConcepts))
	fmt.Fprintf(b, "exercise_ideas（%d 条，至少 %d）: %s\n", len(spec.ExerciseIdeas), minIdeas, joinCritiqueItems(spec.ExerciseIdeas))
	if len(spec.Requires) > 0 {
		writeCritiqueList(b, "requires", spec.Requires)
	}
	if len(spec.GradingHints) > 0 {
		writeCritiqueList(b, "grading_hints", spec.GradingHints)
	}
	beats := NormalizeTeachingBeats(&spec)
	fmt.Fprintf(b, "teaching_beats（%d 条，建议与 core 对齐）: %d 条\n", len(spec.CoreConcepts), len(beats))
}

func writeCritiqueList(b *strings.Builder, label string, items []string) {
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(joinCritiqueItems(items))
	b.WriteString("\n")
}

func joinCritiqueItems(items []string) string {
	if len(items) == 0 {
		return "（无）"
	}
	var parts []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			parts = append(parts, item)
		}
	}
	if len(parts) == 0 {
		return "（无）"
	}
	return strings.Join(parts, "；")
}

func critiqueTree(
	ctx context.Context,
	client llm.Provider,
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	issues []string,
	intent IntentResult,
) (treeCritiqueOutput, error) {
	userContent := buildTreeCritiqueUserMessage(tree, nodes, issues, intent)

	msgs := []llm.Message{
		{Role: "system", Content: treeCritiqueSystemPrompt},
		{Role: "user", Content: userContent},
	}
	ctx = observability.WithGeneration(ctx, "domain.critique_tree")
	var out treeCritiqueOutput
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return treeCritiqueOutput{}, err
	}
	out.Severity = strings.ToLower(strings.TrimSpace(out.Severity))
	if out.Severity == "" {
		if out.Pass {
			out.Severity = "ok"
		} else {
			out.Severity = "warn"
		}
	}
	return out, nil
}

// LogPreserveKeyHits 记录重建时旧 node key 在新树中的命中率（仅日志）。
func LogPreserveKeyHits(preserveKeys []string, tree *storage.KnowledgeTree) {
	if len(preserveKeys) == 0 || tree == nil {
		return
	}
	newKeys := make(map[string]struct{})
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			if k := strings.TrimSpace(n.Key); k != "" {
				newKeys[k] = struct{}{}
			}
		}
	}
	matched := 0
	for _, k := range preserveKeys {
		if _, ok := newKeys[strings.TrimSpace(k)]; ok {
			matched++
		}
	}
	log.Printf("重建 preserveKeys 命中 %d/%d", matched, len(preserveKeys))
}

func countTreeNodes(tree *storage.KnowledgeTree) int {
	if tree == nil {
		return 0
	}
	n := 0
	for _, layer := range tree.Layers {
		n += len(layer.Nodes)
	}
	return n
}
