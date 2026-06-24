package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const relationConfidenceMin = 0.75

// GeneratedCourseRelation LLM 判断的生成课父子关联结果
type GeneratedCourseRelation struct {
	ParentSlug     string
	AnchorKeywords []string
	JumpLabel      string
	Confidence     float64
	Reason         string
}

type generatedRelationLLMOutput struct {
	ParentSlug     string   `json:"parentSlug"`
	AnchorKeywords []string `json:"anchorKeywords"`
	Confidence     float64  `json:"confidence"`
	Reason         string   `json:"reason"`
	JumpLabel      string   `json:"jumpLabel,omitempty"`
}

// CandidateParentDomains 筛选同主题族内可作为父课的已有课程（纯规则，无 LLM）
func CandidateParentDomains(all []storage.DomainSummary, intent IntentResult) []storage.DomainSummary {
	wantRoot := strings.TrimSpace(intent.TopicRoot)
	if wantRoot == "" {
		wantRoot = TopicRoot(intent.Slug)
	}
	if wantRoot == "" {
		return nil
	}
	newSlug := strings.ToLower(strings.TrimSpace(intent.Slug))

	var roots, withParent []storage.DomainSummary
	for _, d := range all {
		s := strings.ToLower(strings.TrimSpace(d.Slug))
		if s == "" || s == newSlug {
			continue
		}
		if TopicRoot(s) != wantRoot {
			continue
		}
		if strings.TrimSpace(d.ParentSlug) == "" {
			roots = append(roots, d)
		} else {
			withParent = append(withParent, d)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	sort.Slice(withParent, func(i, j int) bool { return withParent[i].Name < withParent[j].Name })
	return append(roots, withParent...)
}

// ResolveGeneratedCourseRelation 用 LLM 判断窄主题生成课应挂到哪个父课
func ResolveGeneratedCourseRelation(
	ctx context.Context,
	client llm.Provider,
	userInput string,
	intent IntentResult,
	candidates []storage.DomainSummary,
) (*GeneratedCourseRelation, error) {
	if client == nil || !client.Configured() || len(candidates) == 0 {
		return nil, nil
	}
	ctx, endSpan := observability.StartChildSpan(ctx, "domain.course_relation", observability.TraceMeta{
		Input: userInput,
	})
	defer endSpan()

	var out generatedRelationLLMOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 的课程关联分析器。判断用户新建的窄主题课程是否应作为已有课程的子话题。只输出 JSON。"},
		{Role: "user", Content: buildCourseRelationPrompt(userInput, intent, candidates)},
	}
	ctx = observability.WithGeneration(ctx, "domain.course_relation")
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return nil, fmt.Errorf("课程关联分析失败: %w", err)
	}

	parentSlug := strings.ToLower(strings.TrimSpace(out.ParentSlug))
	if parentSlug == "" || out.Confidence < relationConfidenceMin {
		return nil, nil
	}

	valid := false
	for _, c := range candidates {
		if strings.ToLower(strings.TrimSpace(c.Slug)) == parentSlug {
			valid = true
			break
		}
	}
	if !valid {
		return nil, nil
	}

	kw := make([]string, 0, len(out.AnchorKeywords))
	seen := map[string]struct{}{}
	for _, k := range out.AnchorKeywords {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		low := strings.ToLower(k)
		if _, ok := seen[low]; ok {
			continue
		}
		seen[low] = struct{}{}
		kw = append(kw, k)
	}

	return &GeneratedCourseRelation{
		ParentSlug:     parentSlug,
		AnchorKeywords: kw,
		JumpLabel:      strings.TrimSpace(out.JumpLabel),
		Confidence:     out.Confidence,
		Reason:         strings.TrimSpace(out.Reason),
	}, nil
}

// DerivationFromRelation 将关联结果转为落库的 derivation JSON
func DerivationFromRelation(intent IntentResult, rel *GeneratedCourseRelation) string {
	if rel == nil {
		return ""
	}
	label := rel.JumpLabel
	if label == "" {
		name := strings.TrimSpace(intent.DisplayName)
		if name == "" {
			name = intent.Slug
		}
		label = "深入学习 " + name
	}
	d := DerivationDef{
		ParentAnchorKeywords: rel.AnchorKeywords,
		JumpLabel:            label,
	}
	if len(d.ParentAnchorKeywords) == 0 && d.JumpLabel == "" {
		return ""
	}
	b, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	return string(b)
}

func buildCourseRelationPrompt(userInput string, intent IntentResult, candidates []storage.DomainSummary) string {
	type candidate struct {
		Slug      string `json:"slug"`
		Name      string `json:"name"`
		NodeTotal int    `json:"nodeTotal"`
	}
	var list []candidate
	for _, c := range candidates {
		list = append(list, candidate{Slug: c.Slug, Name: c.Name, NodeTotal: c.NodeTotal})
	}
	candJSON, _ := json.Marshal(list)
	return fmt.Sprintf(`用户原话：%q
新建课程：displayName=%q slug=%q scopeBreadth=%q topicRoot=%q

候选父课（只能从中选一个 slug，无法判断则 parentSlug 为空）：
%s

输出 JSON：
{
  "parentSlug": "go-language",
  "anchorKeywords": ["模块", "go mod", "依赖"],
  "confidence": 0.88,
  "reason": "包管理是 Go 语言的子话题",
  "jumpLabel": "深入学习 Go 包管理"
}

规则：
- parentSlug 必须是候选列表中的 slug 之一；若新建课应独立成根课则 parentSlug 为空、confidence 低于 0.75
- anchorKeywords：2～5 个词，用于在父课知识树节点标题中匹配插入位置（中文或英文）
- jumpLabel：父课页衍生跳转条文案，可省略`,
		userInput, intent.DisplayName, intent.Slug, normalizeScope(intent.ScopeBreadth), intent.TopicRoot, string(candJSON))
}

// ShouldResolveGeneratedRelation 是否应对生成课做 LLM 父子关联
func ShouldResolveGeneratedRelation(intent IntentResult) bool {
	if intent.Source != SourceGenerated || normalizeScope(intent.ScopeBreadth) != ScopeNarrow {
		return false
	}
	if strings.TrimSpace(intent.ParentSlug) != "" {
		return false
	}
	return true
}
