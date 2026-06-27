package domain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const CourseAuditReportVersion = 1

// Audit dimension keys.
const (
	DimensionStructure          = "structure"
	DimensionNodeCompleteness   = "nodeCompleteness"
	DimensionTeachingAlignment  = "teachingAlignment"
	DimensionPrerequisites      = "prerequisites"
)

// Finding severity.
const (
	SeverityFail = "fail"
	SeverityWarn = "warn"
	SeverityInfo = "info"
)

// Finding codes.
const (
	CodeNodeCountOutOfRange    = "NODE_COUNT_OUT_OF_RANGE"
	CodeLayerNodeCountSkew     = "LAYER_NODE_COUNT_SKEW"
	CodeModuleCountSkew        = "MODULE_COUNT_SKEW"
	CodeDuplicateCoreConcept   = "DUPLICATE_CORE_CONCEPT"
	CodeBoundariesOverlapHint  = "BOUNDARIES_OVERLAP_HINT"
	CodeMissingTitle           = "MISSING_TITLE"
	CodeMissingBoundaries      = "MISSING_BOUNDARIES"
	CodeMissingCommonMistakes  = "MISSING_COMMON_MISTAKES"
	CodeInsufficientExercise   = "INSUFFICIENT_EXERCISE_IDEAS"
	CodeMissingGradingHints    = "MISSING_GRADING_HINTS"
	CodeMissingTeachingBeats   = "MISSING_TEACHING_BEATS"
	CodeBeatConceptMismatch    = "BEAT_CONCEPT_MISMATCH"
	CodeBeatMustTeachThin      = "BEAT_MUST_TEACH_THIN"
	CodeInvalidRequires        = "INVALID_REQUIRES"
)

// Fix kinds for optimize.
const (
	FixKindEnrichNode  = "enrich_node"
	FixKindFixRequires = "fix_requires"
	FixKindReorderHint = "reorder_hint"
	FixKindManualOnly  = "manual_only"
)

// CourseAuditReport 课程体检报告。
type CourseAuditReport struct {
	Version     int                    `json:"version"`
	DomainID    string                 `json:"domainId"`
	DomainName  string                 `json:"domainName"`
	Source      string                 `json:"source"`
	TreeVersion int                    `json:"treeVersion"`
	AuditedAt   string                 `json:"auditedAt"`
	Summary     AuditSummary           `json:"summary"`
	Dimensions  map[string]AuditDimension `json:"dimensions"`
	Findings    []Finding              `json:"findings"`
	LLMCritique *LLMCritiqueSummary    `json:"llmCritique,omitempty"`
}

type AuditSummary struct {
	Score      int    `json:"score"`
	Grade      string `json:"grade"`
	FailCount  int    `json:"failCount"`
	WarnCount  int    `json:"warnCount"`
	InfoCount  int    `json:"infoCount"`
	Headline   string `json:"headline"`
}

type AuditDimension struct {
	Score         int `json:"score"`
	FindingCount  int `json:"findingCount"`
}

type Finding struct {
	ID          string `json:"id"`
	Severity    string `json:"severity"`
	Dimension   string `json:"dimension"`
	NodeKey     string `json:"nodeKey,omitempty"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	AutoFixable bool   `json:"autoFixable"`
	FixKind     string `json:"fixKind"`
}

type LLMCritiqueSummary struct {
	Severity string `json:"severity"`
	Feedback string `json:"feedback"`
}

// CourseAuditLLMEnabled 默认开启 LLM 整树评语；REGULUS_COURSE_AUDIT_LLM=0 关闭。
func CourseAuditLLMEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("REGULUS_COURSE_AUDIT_LLM")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

// AuditCourseInput 体检入参。
type AuditCourseInput struct {
	DomainID    string
	DomainName  string
	Source      string
	TreeVersion int
	Tree        *storage.KnowledgeTree
	Nodes       map[string]NodeSpec
	ScopeBreadth string
}

// AuditCourse 生成课程体检报告（规则 + 可选 LLM critique）。
func AuditCourse(ctx context.Context, client llm.Provider, in AuditCourseInput) (*CourseAuditReport, error) {
	intent := IntentResult{
		DisplayName:  in.DomainName,
		ScopeBreadth: in.ScopeBreadth,
	}
	if intent.ScopeBreadth == "" && in.Tree != nil {
		intent.ScopeBreadth = InferScopeFromTree(in.Tree)
	}
	if intent.ScopeBreadth == "" {
		intent.ScopeBreadth = ScopeModerate
	}

	findings := collectStructuredFindings(in.Tree, in.Nodes, intent)

	var llmCritique *LLMCritiqueSummary
	if CourseAuditLLMEnabled() && client != nil && client.Configured() {
		issueStrings := findingsToIssueStrings(findings)
		out, err := critiqueTree(ctx, client, in.Tree, in.Nodes, issueStrings, intent)
		if err == nil {
			llmCritique = &LLMCritiqueSummary{
				Severity: out.Severity,
				Feedback: strings.TrimSpace(out.Feedback),
			}
		}
	}

	summary, dimensions := scoreAuditFindings(findings)
	report := &CourseAuditReport{
		Version:     CourseAuditReportVersion,
		DomainID:    in.DomainID,
		DomainName:  in.DomainName,
		Source:      in.Source,
		TreeVersion: in.TreeVersion,
		AuditedAt:   time.Now().UTC().Format(time.RFC3339),
		Summary:     summary,
		Dimensions:  dimensions,
		Findings:    findings,
		LLMCritique: llmCritique,
	}
	return report, nil
}

func collectStructuredFindings(tree *storage.KnowledgeTree, nodes map[string]NodeSpec, intent IntentResult) []Finding {
	var findings []Finding
	totalNodes := countTreeNodes(tree)
	minTotal, maxTotal := nodeCountBounds(intent.ScopeBreadth)

	if totalNodes > 0 && (totalNodes < minTotal || totalNodes > maxTotal) {
		findings = append(findings, Finding{
			ID:          "structure:node_count",
			Severity:    SeverityWarn,
			Dimension:   DimensionStructure,
			Code:        CodeNodeCountOutOfRange,
			Message:     fmt.Sprintf("节点总数 %d，建议 %d-%d（主题广度 %s）", totalNodes, minTotal, maxTotal, normalizeScope(intent.ScopeBreadth)),
			Suggestion:  "调整各层节点数量或重新生成课程大纲",
			AutoFixable: false,
			FixKind:     FixKindManualOnly,
		})
	}

	if tree != nil {
		for _, layer := range tree.Layers {
			b, ok := recommendedLayerBounds[layer.Key]
			if !ok {
				continue
			}
			n := len(layer.Nodes)
			if n < b.min {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("structure:layer:%s:low", layer.Key),
					Severity:    SeverityWarn,
					Dimension:   DimensionStructure,
					Code:        CodeLayerNodeCountSkew,
					Message:     fmt.Sprintf("层级 %s 建议至少 %d 个节点，当前 %d", layer.Label, b.min, n),
					Suggestion:  "在对应层级补充节点或合并相邻主题",
					AutoFixable: false,
					FixKind:     FixKindManualOnly,
				})
			}
			if n > b.max {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("structure:layer:%s:high", layer.Key),
					Severity:    SeverityWarn,
					Dimension:   DimensionStructure,
					Code:        CodeLayerNodeCountSkew,
					Message:     fmt.Sprintf("层级 %s 建议最多 %d 个节点，当前 %d", layer.Label, b.max, n),
					Suggestion:  "拆分过粗的层级或移除重复主题",
					AutoFixable: false,
					FixKind:     FixKindManualOnly,
				})
			}
		}
		if len(tree.Modules) > 0 {
			minM, maxM := moduleCountBounds(intent.ScopeBreadth)
			if len(tree.Modules) < minM || len(tree.Modules) > maxM {
				findings = append(findings, Finding{
					ID:          "structure:module_count",
					Severity:    SeverityWarn,
					Dimension:   DimensionStructure,
					Code:        CodeModuleCountSkew,
					Message:     fmt.Sprintf("主题模块数 %d，建议 %d-%d", len(tree.Modules), minM, maxM),
					Suggestion:  "按主题重新划分 module",
					AutoFixable: false,
					FixKind:     FixKindManualOnly,
				})
			}
		}
	}

	if totalNodes > 0 && totalNodes <= 8 {
		findings = append(findings, Finding{
			ID:          "structure:boundaries_hint",
			Severity:    SeverityInfo,
			Dimension:   DimensionStructure,
			Code:        CodeBoundariesOverlapHint,
			Message:     fmt.Sprintf("节点数 %d（≤8），请确认相邻节点 boundaries 已区分职责", totalNodes),
			Suggestion:  "检查各节点「不讲什么」是否互斥，避免讲解跑题",
			AutoFixable: false,
			FixKind:     FixKindManualOnly,
		})
	}

	seenConcept := map[string]string{}
	for key, spec := range nodes {
		if strings.TrimSpace(spec.Node) == "" {
			findings = append(findings, nodeFinding(key, SeverityWarn, DimensionNodeCompleteness, CodeMissingTitle,
				fmt.Sprintf("节点 %s 缺少标题", key), "补充节点中文标题", false, FixKindManualOnly))
		}
		if len(spec.Boundaries) == 0 {
			findings = append(findings, nodeFinding(key, SeverityWarn, DimensionNodeCompleteness, CodeMissingBoundaries,
				fmt.Sprintf("节点 %s 缺少 boundaries", key), "补充「不讲什么」边界，防止教练跑题", true, FixKindEnrichNode))
		}
		if len(spec.CommonMistakes) == 0 {
			findings = append(findings, nodeFinding(key, SeverityWarn, DimensionNodeCompleteness, CodeMissingCommonMistakes,
				fmt.Sprintf("节点 %s 缺少 common_mistakes", key), "补充常见误区，便于出题与批改", true, FixKindEnrichNode))
		}
		minIdeas := minExerciseIdeasRequired(len(spec.CoreConcepts))
		if minIdeas > 0 && len(spec.ExerciseIdeas) < minIdeas {
			findings = append(findings, nodeFinding(key, SeverityWarn, DimensionNodeCompleteness, CodeInsufficientExercise,
				fmt.Sprintf("节点 %s 的 exercise_ideas 不足（需至少 %d 条，当前 %d 条）", key, minIdeas, len(spec.ExerciseIdeas)),
				"补充练习题思路", true, FixKindEnrichNode))
		}
		if len(spec.CoreConcepts) > 0 && len(spec.GradingHints) == 0 {
			findings = append(findings, nodeFinding(key, SeverityInfo, DimensionNodeCompleteness, CodeMissingGradingHints,
				fmt.Sprintf("节点 %s 缺少 grading_hints", key), "补充评分要点，提升批改一致性", true, FixKindEnrichNode))
		}

		for _, c := range spec.CoreConcepts {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if other, ok := seenConcept[c]; ok && other != key {
				findings = append(findings, Finding{
					ID:          fmt.Sprintf("structure:dup_concept:%s", slugifyFindingID(c)),
					Severity:    SeverityWarn,
					Dimension:   DimensionStructure,
					Code:        CodeDuplicateCoreConcept,
					Message:     fmt.Sprintf("core_concept %q 在节点 %s 与 %s 重复", c, other, key),
					Suggestion:  "合并重复概念或调整节点边界",
					AutoFixable: false,
					FixKind:     FixKindManualOnly,
				})
			} else {
				seenConcept[c] = key
			}
		}

		if len(spec.CoreConcepts) > 0 && len(spec.TeachingBeats) == 0 {
			findings = append(findings, nodeFinding(key, SeverityWarn, DimensionTeachingAlignment, CodeMissingTeachingBeats,
				fmt.Sprintf("节点 %s 缺少 teaching_beats（将使用 fallback）", key),
				"按 core_concepts 补全教学节拍，对齐 Go 并发域标杆", true, FixKindEnrichNode))
		}

		for _, c := range spec.CoreConcepts {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			if BeatForConcept(&spec, c) == nil && len(spec.TeachingBeats) > 0 {
				findings = append(findings, nodeFinding(key, SeverityWarn, DimensionTeachingAlignment, CodeBeatConceptMismatch,
					fmt.Sprintf("节点 %s 的核心概念 %q 无对应 teaching_beat", key, c),
					"为该概念添加 teaching_beat 或调整 concept 表述", true, FixKindEnrichNode))
			}
		}
		for _, beat := range spec.TeachingBeats {
			if len(beat.MustTeach) < 2 {
				findings = append(findings, nodeFinding(key, SeverityInfo, DimensionTeachingAlignment, CodeBeatMustTeachThin,
					fmt.Sprintf("节点 %s 的概念 %q 的 must_teach 少于 2 条", key, beat.Concept),
					"补充 must_teach 要点", true, FixKindEnrichNode))
			}
		}

		for _, req := range spec.Requires {
			req = strings.TrimSpace(req)
			if req == "" {
				continue
			}
			if _, ok := nodes[req]; !ok {
				findings = append(findings, nodeFinding(key, SeverityFail, DimensionPrerequisites, CodeInvalidRequires,
					fmt.Sprintf("节点 %s 的 requires 引用不存在的前置 %q", key, req),
					"删除无效前置或补充对应节点", true, FixKindFixRequires))
			}
		}
	}

	return findings
}

func nodeFinding(nodeKey, severity, dimension, code, message, suggestion string, autoFix bool, fixKind string) Finding {
	return Finding{
		ID:          fmt.Sprintf("node:%s:%s", nodeKey, strings.ToLower(code)),
		Severity:    severity,
		Dimension:   dimension,
		NodeKey:     nodeKey,
		Code:        code,
		Message:     message,
		Suggestion:  suggestion,
		AutoFixable: autoFix,
		FixKind:     fixKind,
	}
}

func slugifyFindingID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 40 {
		s = s[:40]
	}
	return strings.ReplaceAll(s, " ", "_")
}

func findingsToIssueStrings(findings []Finding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.Message)
	}
	return out
}

func severityPenalty(sev string) int {
	switch sev {
	case SeverityFail:
		return 15
	case SeverityWarn:
		return 5
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func scoreAuditFindings(findings []Finding) (AuditSummary, map[string]AuditDimension) {
	dims := map[string]AuditDimension{
		DimensionStructure:         {Score: 100},
		DimensionNodeCompleteness:  {Score: 100},
		DimensionTeachingAlignment: {Score: 100},
		DimensionPrerequisites:     {Score: 100},
	}
	var fail, warn, info int
	for _, f := range findings {
		pen := severityPenalty(f.Severity)
		switch f.Severity {
		case SeverityFail:
			fail++
		case SeverityWarn:
			warn++
		case SeverityInfo:
			info++
		}
		d := dims[f.Dimension]
		d.Score -= pen
		if d.Score < 0 {
			d.Score = 0
		}
		d.FindingCount++
		dims[f.Dimension] = d
	}
	totalScore := 100
	for _, f := range findings {
		totalScore -= severityPenalty(f.Severity)
	}
	if totalScore < 0 {
		totalScore = 0
	}
	grade := "D"
	switch {
	case totalScore >= 90:
		grade = "A"
	case totalScore >= 75:
		grade = "B"
	case totalScore >= 60:
		grade = "C"
	}
	headline := buildAuditHeadline(findings, fail, warn)
	return AuditSummary{
		Score:     totalScore,
		Grade:     grade,
		FailCount: fail,
		WarnCount: warn,
		InfoCount: info,
		Headline:  headline,
	}, dims
}

func buildAuditHeadline(findings []Finding, fail, warn int) string {
	if len(findings) == 0 {
		return "未发现明显问题，课程结构良好"
	}
	var beatsMissing int
	for _, f := range findings {
		if f.Code == CodeMissingTeachingBeats {
			beatsMissing++
		}
	}
	if beatsMissing > 0 {
		return fmt.Sprintf("教考对齐不足，%d 个节点缺少 teaching_beats", beatsMissing)
	}
	if fail > 0 {
		return fmt.Sprintf("发现 %d 项严重问题，建议优先处理前置依赖与结构", fail)
	}
	if warn > 0 {
		return fmt.Sprintf("发现 %d 项待改进，可勾选自动优化项批量补全", warn)
	}
	return "仅有少量提示项，整体可继续使用"
}

// FindingsByIDs 按 ID 筛选 findings。
func FindingsByIDs(all []Finding, ids []string) []Finding {
	if len(ids) == 0 {
		return nil
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			want[id] = struct{}{}
		}
	}
	var out []Finding
	for _, f := range all {
		if _, ok := want[f.ID]; ok {
			out = append(out, f)
		}
	}
	return out
}

// CollectStructuredFindingsForOptimize 导出结构化 findings（供 optimize fallback）。
func CollectStructuredFindingsForOptimize(tree *storage.KnowledgeTree, nodes map[string]NodeSpec, intent IntentResult) []Finding {
	return collectStructuredFindings(tree, nodes, intent)
}
