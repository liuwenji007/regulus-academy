package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const (
	distillChunkSize    = 4000
	distillChunkOverlap = 400
)

// DistillSection 蒸馏大纲章节
type DistillSection struct {
	Heading  string   `json:"heading"`
	Points   []string `json:"points"`
	Concepts []string `json:"concepts"`
}

// DistillOutline 材料结构化大纲
type DistillOutline struct {
	Title         string           `json:"title"`
	Sections      []DistillSection `json:"sections"`
	SuggestedSlug string           `json:"suggestedSlug,omitempty"`
	ScopeBreadth  string           `json:"scopeBreadth,omitempty"`
}

type distillMapOutput struct {
	Points   []string `json:"points"`
	Concepts []string `json:"concepts"`
}

// Distill 将长文本 map-reduce 压成结构化大纲
func Distill(ctx context.Context, client llm.Provider, text string) (*DistillOutline, error) {
	if !client.Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法蒸馏材料")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("材料正文为空")
	}

	chunks := chunkText(text, distillChunkSize, distillChunkOverlap)
	var mapped []distillMapOutput
	for i, chunk := range chunks {
		ReportBuildProgress(ctx, "distill", fmt.Sprintf("正在分析材料片段 %d/%d…", i+1, len(chunks)))
		out, err := distillMapChunk(ctx, client, chunk)
		if err != nil {
			return nil, err
		}
		mapped = append(mapped, out)
	}

	ReportBuildProgress(ctx, "distill", "正在合并材料大纲…")
	return distillReduce(ctx, client, mapped)
}

func chunkText(text string, size, overlap int) []string {
	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}
	var chunks []string
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end >= len(runes) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

func distillMapChunk(ctx context.Context, client llm.Provider, chunk string) (distillMapOutput, error) {
	prompt := `从以下材料片段提取学习要点。只输出 JSON：
{"points":["要点1","要点2"],"concepts":["概念1","概念2"]}

要求：
- points：3～8 条，概括片段中的知识要点
- concepts：1～5 个核心概念名词
- 忠实于材料，不要编造片段外内容

材料片段：
` + chunk

	var out distillMapOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是学习材料分析助手。只输出 JSON。"},
		{Role: "user", Content: prompt},
	}
	ctx = observability.WithGeneration(ctx, "domain.distill_map")
	if err := client.ChatJSON(ctx, msgs, 0.1, &out); err != nil {
		return distillMapOutput{}, fmt.Errorf("材料片段分析失败: %w", err)
	}
	return out, nil
}

func distillReduce(ctx context.Context, client llm.Provider, mapped []distillMapOutput) (*DistillOutline, error) {
	var b strings.Builder
	b.WriteString("各片段要点汇总（JSON 数组）：\n")
	raw, _ := json.Marshal(mapped)
	b.Write(raw)
	b.WriteString(`

请合并为固定 schema 的 JSON：
{
  "title": "材料主题",
  "sections": [{"heading": "章节名", "points": ["..."], "concepts": ["..."]}],
  "suggestedSlug": "optional-english-slug",
  "scopeBreadth": "narrow|moderate|broad"
}

要求：
- sections 3～8 个，按材料逻辑分章
- 合并重复概念，保留材料中的核心知识脉络
- boundaries 类信息可写入 points（标明材料未展开部分）
- 不要捏造材料外内容`)

	var out DistillOutline
	msgs := []llm.Message{
		{Role: "system", Content: "你是学习材料大纲整理助手。只输出 JSON。"},
		{Role: "user", Content: b.String()},
	}
	ctx = observability.WithGeneration(ctx, "domain.distill_reduce")
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return nil, fmt.Errorf("材料大纲合并失败: %w", err)
	}
	if strings.TrimSpace(out.Title) == "" {
		return nil, fmt.Errorf("蒸馏结果缺少标题")
	}
	if len(out.Sections) == 0 {
		return nil, fmt.Errorf("蒸馏结果缺少章节")
	}
	out.ScopeBreadth = normalizeScope(out.ScopeBreadth)
	return &out, nil
}

// FormatRefOutline 将蒸馏大纲格式化为建树 prompt 段落
func FormatRefOutline(outline *DistillOutline) string {
	if outline == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("【参考材料大纲】（建树须覆盖其中核心概念，boundaries 标明材料未展开部分；勿捏造材料外内容）\n")
	b.WriteString("主题：")
	b.WriteString(outline.Title)
	b.WriteString("\n")
	if outline.ScopeBreadth != "" {
		b.WriteString("材料广度：")
		b.WriteString(outline.ScopeBreadth)
		b.WriteString("\n")
	}
	for _, sec := range outline.Sections {
		b.WriteString("\n## ")
		b.WriteString(sec.Heading)
		b.WriteString("\n")
		if len(sec.Concepts) > 0 {
			b.WriteString("概念：")
			b.WriteString(strings.Join(sec.Concepts, "、"))
			b.WriteString("\n")
		}
		for _, p := range sec.Points {
			b.WriteString("- ")
			b.WriteString(p)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}
