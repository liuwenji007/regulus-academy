package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// CourseOptimizeLLMEnabled 默认开启；REGULUS_COURSE_OPTIMIZE_LLM=0 关闭。
func CourseOptimizeLLMEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_COURSE_OPTIMIZE_LLM")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// NodePatchFields 节点补丁字段（白名单合并）。
type NodePatchFields struct {
	Boundaries     []string      `json:"boundaries,omitempty"`
	CommonMistakes []string      `json:"common_mistakes,omitempty"`
	ExerciseIdeas  []string      `json:"exercise_ideas,omitempty"`
	GradingHints   []string      `json:"grading_hints,omitempty"`
	TeachingBeats  []ConceptBeat `json:"teaching_beats,omitempty"`
	Requires       []string      `json:"requires,omitempty"`
}

// OptimizePatchItem 单条优化补丁。
type OptimizePatchItem struct {
	ID        string          `json:"id"`
	FindingID string          `json:"findingId"`
	NodeKey   string          `json:"nodeKey"`
	NodeTitle string          `json:"nodeTitle,omitempty"`
	Before    NodePatchFields `json:"before"`
	After     NodePatchFields `json:"after"`
	Summary   string          `json:"summary"`
	Benefits  []string        `json:"benefits,omitempty"`
}

// OptimizePatch 优化补丁集合。
type OptimizePatch struct {
	DomainID        string              `json:"domainId"`
	BaseTreeVersion int                 `json:"baseTreeVersion"`
	Headline        string              `json:"headline,omitempty"`
	Patches         []OptimizePatchItem `json:"patches"`
}

type optimizeNodeLLMOutput struct {
	Boundaries     FlexStringList `json:"boundaries,omitempty"`
	CommonMistakes FlexStringList `json:"common_mistakes,omitempty"`
	ExerciseIdeas  FlexStringList `json:"exercise_ideas,omitempty"`
	GradingHints   FlexStringList `json:"grading_hints,omitempty"`
	TeachingBeats  []ConceptBeat  `json:"teaching_beats,omitempty"`
}

// BuildOptimizePatch 根据 findings 生成优化补丁（不修改原 nodes）。
func BuildOptimizePatch(
	ctx context.Context,
	client llm.Provider,
	domainID string,
	treeVersion int,
	tree *storage.KnowledgeTree,
	nodes map[string]NodeSpec,
	findings []Finding,
) (*OptimizePatch, error) {
	if len(findings) == 0 {
		return &OptimizePatch{DomainID: domainID, BaseTreeVersion: treeVersion, Patches: nil}, nil
	}

	// Group by node for enrich_node; fix_requires handled programmatically.
	byNode := map[string][]Finding{}
	for _, f := range findings {
		if !f.AutoFixable {
			continue
		}
		if f.FixKind == FixKindFixRequires && f.NodeKey != "" {
			byNode[f.NodeKey] = append(byNode[f.NodeKey], f)
			continue
		}
		if f.FixKind == FixKindEnrichNode && f.NodeKey != "" {
			byNode[f.NodeKey] = append(byNode[f.NodeKey], f)
		}
	}

	var patches []OptimizePatchItem
	var nodeErrors []string
	for nodeKey, nodeFindings := range byNode {
		spec, ok := nodes[nodeKey]
		if !ok {
			continue
		}
		before := extractPatchFields(&spec)

		// Programmatic fix_requires first.
		afterSpec := spec
		for _, f := range nodeFindings {
			if f.FixKind == FixKindFixRequires && f.Code == CodeInvalidRequires {
				afterSpec.Requires = filterValidRequires(afterSpec.Requires, nodes)
			}
		}

		needsLLM := false
		for _, f := range nodeFindings {
			if f.FixKind == FixKindEnrichNode {
				needsLLM = true
				break
			}
		}

		if needsLLM {
			if !CourseOptimizeLLMEnabled() || client == nil || !client.Configured() {
				continue
			}
			enriched, err := optimizeNodeWithLLM(ctx, client, tree, nodeKey, afterSpec, nodeFindings)
			if err != nil {
				nodeErrors = append(nodeErrors, fmt.Sprintf("%s: %v", nodeKey, err))
				continue
			}
			afterSpec = mergeEnrichedSpec(afterSpec, enriched)
		}

		after := extractPatchFields(&afterSpec)
		if patchFieldsEqual(before, after) {
			continue
		}
		benefits := describePatchBenefits(before, after, nodeFindings)
		summary := summarizePatch(benefits, nodeFindings)
		title := strings.TrimSpace(afterSpec.Node)
		if title == "" && tree != nil {
			title = NodeTitle(tree, nodeKey)
		}
		if title == "" {
			title = nodeKey
		}
		patches = append(patches, OptimizePatchItem{
			ID:        fmt.Sprintf("patch:%s", nodeKey),
			FindingID: firstFindingID(nodeFindings),
			NodeKey:   nodeKey,
			NodeTitle: title,
			Before:    before,
			After:     after,
			Summary:   summary,
			Benefits:  benefits,
		})
	}

	if len(patches) == 0 && len(nodeErrors) > 0 {
		return nil, fmt.Errorf("节点优化失败: %s", strings.Join(nodeErrors, "; "))
	}

	return &OptimizePatch{
		DomainID:        domainID,
		BaseTreeVersion: treeVersion,
		Headline:        buildOptimizeHeadline(patches),
		Patches:         patches,
	}, nil
}

func extractPatchFields(spec *NodeSpec) NodePatchFields {
	if spec == nil {
		return NodePatchFields{}
	}
	return NodePatchFields{
		Boundaries:     append([]string{}, spec.Boundaries...),
		CommonMistakes: append([]string{}, spec.CommonMistakes...),
		ExerciseIdeas:  append([]string{}, spec.ExerciseIdeas...),
		GradingHints:   append([]string{}, spec.GradingHints...),
		TeachingBeats:  append([]ConceptBeat{}, spec.TeachingBeats...),
		Requires:       append([]string{}, spec.Requires...),
	}
}

func filterValidRequires(requires []string, nodes map[string]NodeSpec) []string {
	var out []string
	for _, req := range requires {
		req = strings.TrimSpace(req)
		if req == "" {
			continue
		}
		if _, ok := nodes[req]; ok {
			out = append(out, req)
		}
	}
	return out
}

func optimizeNodeWithLLM(
	ctx context.Context,
	client llm.Provider,
	tree *storage.KnowledgeTree,
	nodeKey string,
	spec NodeSpec,
	findings []Finding,
) (optimizeNodeLLMOutput, error) {
	var issues []string
	for _, f := range findings {
		if f.FixKind == FixKindEnrichNode {
			issues = append(issues, f.Suggestion+": "+f.Message)
		}
	}
	specJSON, _ := json.Marshal(spec)
	prompt := fmt.Sprintf(`## 任务
请优化以下节点的教学内容字段，只输出 JSON（字段名用 snake_case）。

## 节点 key
%s

## 当前节点定义
%s

## 待修复项
%s

## 要求
- 不得修改 key、layer、requires 的节点引用（requires 仅删除无效项，勿新增不存在的前置）
- teaching_beats 需与 core_concepts 对齐，参考标杆格式（concept、must_teach、context_type）；每个 beat 的 must_teach 至少 2 条
- boundaries 写清「不讲什么」
- 只输出需要补全/改进的字段：boundaries、common_mistakes、exercise_ideas、grading_hints、teaching_beats
- boundaries、common_mistakes、exercise_ideas、grading_hints 必须是字符串数组（如 ["要点1","要点2"]），不要输出 JSON 对象
- 数组元素之间只用英文逗号 , 分隔；禁止 and、以及、中文顿号充当分隔符。正确示例：["不讲 A","不讲 B"]
`, nodeKey, string(specJSON), strings.Join(issues, "\n- "))

	if tree != nil {
		prompt += "\n## 课程名\n" + tree.DomainName
	}

	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 节点边界编辑。只输出 JSON，不得编造与节点无关的内容。"},
		{Role: "user", Content: prompt},
	}
	ctx = observability.WithGeneration(ctx, "domain.optimize_node")
	var out optimizeNodeLLMOutput
	if err := client.ChatJSON(ctx, msgs, 0.3, &out); err != nil {
		return optimizeNodeLLMOutput{}, err
	}
	return out, nil
}

func mergeEnrichedSpec(base NodeSpec, enriched optimizeNodeLLMOutput) NodeSpec {
	if hints := enriched.Boundaries.strings(); len(hints) > 0 {
		base.Boundaries = hints
	}
	if hints := enriched.CommonMistakes.strings(); len(hints) > 0 {
		base.CommonMistakes = hints
	}
	if hints := enriched.ExerciseIdeas.strings(); len(hints) > 0 {
		base.ExerciseIdeas = hints
	}
	if hints := enriched.GradingHints.strings(); len(hints) > 0 {
		base.GradingHints = hints
	}
	if len(enriched.TeachingBeats) > 0 {
		base.TeachingBeats = EnsureConceptBeatMustTeachMin(enriched.TeachingBeats, MinMustTeachItems, &base)
	}
	return base
}

func patchFieldsEqual(a, b NodePatchFields) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func summarizePatch(benefits []string, findings []Finding) string {
	if len(benefits) > 0 {
		return benefits[0]
	}
	if len(findings) == 1 {
		return humanizeSuggestion(findings[0].Suggestion)
	}
	if len(findings) > 1 {
		return fmt.Sprintf("补强节点教学内容（%d 项）", len(findings))
	}
	return "补强节点教学内容"
}

func humanizeSuggestion(s string) string {
	s = strings.TrimSpace(s)
	repl := map[string]string{
		"补充 must_teach 要点":     "补齐必讲要点，讲解更完整",
		"按 core_concepts 补全教学节拍": "补齐教学节拍，讲解与练习更对齐",
		"对齐 Go 并发域标杆":         "补齐教学节拍，讲解与练习更对齐",
		"补充练习题思路":            "丰富练习题思路，练习更贴近考点",
		"补充评分要点":             "补充批改要点，反馈更一致",
		"删除无效前置":             "修正无效前置依赖",
	}
	for k, v := range repl {
		if strings.Contains(s, k) {
			return v
		}
	}
	return s
}

// describePatchBenefits 把字段变更翻译成用户可理解的收益说明。
func describePatchBenefits(before, after NodePatchFields, findings []Finding) []string {
	var benefits []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, x := range benefits {
			if x == s {
				return
			}
		}
		benefits = append(benefits, s)
	}

	if len(after.TeachingBeats) > 0 && !teachingBeatsEqual(before.TeachingBeats, after.TeachingBeats) {
		add("补齐讲解节拍与必讲要点，学这一节时教练讲得更聚焦、少跑偏")
	}
	if len(after.Boundaries) > 0 && !stringSlicesEqual(before.Boundaries, after.Boundaries) {
		add("划清本节不讲什么，减少重复内容与跑题")
	}
	if len(after.CommonMistakes) > 0 && !stringSlicesEqual(before.CommonMistakes, after.CommonMistakes) {
		add("补充常见误区，练习与点评会专门盯易错点")
	}
	if len(after.ExerciseIdeas) > 0 && !stringSlicesEqual(before.ExerciseIdeas, after.ExerciseIdeas) {
		add("丰富练习题思路，后续出题更贴考点")
	}
	if len(after.GradingHints) > 0 && !stringSlicesEqual(before.GradingHints, after.GradingHints) {
		add("补充批改要点，答对/答错时的反馈更稳定")
	}
	if after.Requires != nil && !stringSlicesEqual(before.Requires, after.Requires) {
		add("修正前置依赖，学习路径更顺")
	}

	if len(benefits) == 0 {
		for _, f := range findings {
			add(humanizeSuggestion(f.Suggestion))
		}
	}
	return benefits
}

func teachingBeatsEqual(a, b []ConceptBeat) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func buildOptimizeHeadline(patches []OptimizePatchItem) string {
	n := len(patches)
	if n == 0 {
		return ""
	}
	kinds := map[string]bool{}
	for _, p := range patches {
		for _, b := range p.Benefits {
			switch {
			case strings.Contains(b, "讲解") || strings.Contains(b, "节拍") || strings.Contains(b, "必讲"):
				kinds["讲解"] = true
			case strings.Contains(b, "练习") || strings.Contains(b, "出题"):
				kinds["练习"] = true
			case strings.Contains(b, "批改") || strings.Contains(b, "反馈"):
				kinds["批改"] = true
			case strings.Contains(b, "误区"):
				kinds["易错点"] = true
			case strings.Contains(b, "前置"):
				kinds["路径"] = true
			case strings.Contains(b, "边界") || strings.Contains(b, "跑题"):
				kinds["边界"] = true
			}
		}
	}
	order := []string{"讲解", "练习", "批改", "易错点", "边界", "路径"}
	var parts []string
	for _, k := range order {
		if kinds[k] {
			parts = append(parts, k)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("将对 %d 个知识点补强教学内容，不影响已有进度。", n)
	}
	return fmt.Sprintf("将对 %d 个知识点提升%s，写入后教练讲解、出题与反馈会更到位；不改节点、不影响已有进度。",
		n, strings.Join(parts, " / "))
}

func firstFindingID(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}
	return findings[0].ID
}

// ApplyOptimizePatches 将选中补丁合并进 nodes map。
func ApplyOptimizePatches(nodes map[string]NodeSpec, patches []OptimizePatchItem, patchIDs []string) (map[string]NodeSpec, error) {
	if len(patches) == 0 {
		return nodes, nil
	}
	want := map[string]struct{}{}
	if len(patchIDs) > 0 {
		for _, id := range patchIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				want[id] = struct{}{}
			}
		}
	}
	out := make(map[string]NodeSpec, len(nodes))
	for k, v := range nodes {
		out[k] = v
	}
	for _, p := range patches {
		if len(want) > 0 {
			if _, ok := want[p.ID]; !ok {
				if _, ok2 := want[p.FindingID]; !ok2 {
					continue
				}
			}
		}
		spec, ok := out[p.NodeKey]
		if !ok {
			return nil, fmt.Errorf("节点 %s 不存在", p.NodeKey)
		}
		spec = applyPatchFields(spec, p.After)
		out[p.NodeKey] = spec
	}
	return out, nil
}

func applyPatchFields(spec NodeSpec, patch NodePatchFields) NodeSpec {
	if len(patch.Boundaries) > 0 {
		spec.Boundaries = append([]string{}, patch.Boundaries...)
	}
	if len(patch.CommonMistakes) > 0 {
		spec.CommonMistakes = append([]string{}, patch.CommonMistakes...)
	}
	if len(patch.ExerciseIdeas) > 0 {
		spec.ExerciseIdeas = append([]string{}, patch.ExerciseIdeas...)
	}
	if len(patch.GradingHints) > 0 {
		spec.GradingHints = append([]string{}, patch.GradingHints...)
	}
	if len(patch.TeachingBeats) > 0 {
		spec.TeachingBeats = append([]ConceptBeat{}, patch.TeachingBeats...)
	}
	if patch.Requires != nil {
		spec.Requires = append([]string{}, patch.Requires...)
	}
	return spec
}
