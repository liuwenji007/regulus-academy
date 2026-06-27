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
	Before    NodePatchFields `json:"before"`
	After     NodePatchFields `json:"after"`
	Summary   string          `json:"summary"`
}

// OptimizePatch 优化补丁集合。
type OptimizePatch struct {
	DomainID        string              `json:"domainId"`
	BaseTreeVersion int                 `json:"baseTreeVersion"`
	Patches         []OptimizePatchItem `json:"patches"`
}

type optimizeNodeLLMOutput struct {
	Boundaries     []string      `json:"boundaries,omitempty"`
	CommonMistakes []string      `json:"common_mistakes,omitempty"`
	ExerciseIdeas  []string      `json:"exercise_ideas,omitempty"`
	GradingHints   []string      `json:"grading_hints,omitempty"`
	TeachingBeats  []ConceptBeat `json:"teaching_beats,omitempty"`
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
				return nil, fmt.Errorf("节点 %s 优化失败: %w", nodeKey, err)
			}
			afterSpec = mergeEnrichedSpec(afterSpec, enriched)
		}

		after := extractPatchFields(&afterSpec)
		if patchFieldsEqual(before, after) {
			continue
		}
		summary := summarizePatch(nodeFindings)
		patches = append(patches, OptimizePatchItem{
			ID:        fmt.Sprintf("patch:%s", nodeKey),
			FindingID: firstFindingID(nodeFindings),
			NodeKey:   nodeKey,
			Before:    before,
			After:     after,
			Summary:   summary,
		})
	}

	return &OptimizePatch{
		DomainID:        domainID,
		BaseTreeVersion: treeVersion,
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
- teaching_beats 需与 core_concepts 对齐，参考标杆格式（concept、must_teach、context_type）
- boundaries 写清「不讲什么」
- 只输出需要补全/改进的字段：boundaries、common_mistakes、exercise_ideas、grading_hints、teaching_beats
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
	if len(enriched.Boundaries) > 0 {
		base.Boundaries = enriched.Boundaries
	}
	if len(enriched.CommonMistakes) > 0 {
		base.CommonMistakes = enriched.CommonMistakes
	}
	if len(enriched.ExerciseIdeas) > 0 {
		base.ExerciseIdeas = enriched.ExerciseIdeas
	}
	if len(enriched.GradingHints) > 0 {
		base.GradingHints = enriched.GradingHints
	}
	if len(enriched.TeachingBeats) > 0 {
		base.TeachingBeats = enriched.TeachingBeats
	}
	return base
}

func patchFieldsEqual(a, b NodePatchFields) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func summarizePatch(findings []Finding) string {
	if len(findings) == 1 {
		return findings[0].Suggestion
	}
	return fmt.Sprintf("批量优化 %d 项（%s）", len(findings), findings[0].NodeKey)
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
