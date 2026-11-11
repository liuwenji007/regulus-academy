package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

// PresetFields 预设领域。模型优先归入这些名字，前端旋臂槽位也按这个顺序固定。
var PresetFields = []string{
	"编程语言",
	"AI 与数据",
	"工程与架构",
	"运维与基础设施",
	"语言与沟通",
	"人文与通识",
}

const (
	// MaxDomainFields 一个用户最多同时存在的领域数（预设加自定义）
	MaxDomainFields = 8
	// FieldCreateBelow 置信度低于该值、且名额未满时，才允许新建领域
	FieldCreateBelow = 0.6
)

// FieldClass 领域归类结果
type FieldClass struct {
	Field      string
	Confidence float64
	Reason     string
	Created    bool
}

type fieldLLMOutput struct {
	Field      string  `json:"field"`
	Nearest    string  `json:"nearest"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// AllowedFields 预设领域在前，再接该用户已经用过、且不在预设里的领域
func AllowedFields(existing []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range PresetFields {
		add(name)
	}
	for _, name := range existing {
		add(name)
	}
	return out
}

// ResolveFieldChoice 把模型输出收束到允许的领域列表。
// 命中已有领域就用它；只有置信度低于 FieldCreateBelow 且总数少于 MaxDomainFields 时才接受新名字。
func ResolveFieldChoice(field, nearest string, confidence float64, allowed []string) (chosen string, created bool) {
	field = strings.TrimSpace(field)
	nearest = strings.TrimSpace(nearest)
	contains := func(name string) bool {
		for _, item := range allowed {
			if item == name {
				return true
			}
		}
		return false
	}
	if field != "" && contains(field) {
		return field, false
	}
	if field != "" && confidence < FieldCreateBelow && len(allowed) < MaxDomainFields {
		return field, true
	}
	if contains(nearest) {
		return nearest, false
	}
	if len(allowed) > 0 {
		return allowed[0], false
	}
	return PresetFields[0], false
}

// ClassifyDomainField 让模型把一门课归入领域。client 未配置时归入第一个预设领域。
func ClassifyDomainField(
	ctx context.Context,
	client llm.Provider,
	name, slug string,
	moduleLabels []string,
	existing []string,
) (*FieldClass, error) {
	allowed := AllowedFields(existing)
	if client == nil || !client.Configured() {
		return &FieldClass{Field: PresetFields[0], Reason: "未配置模型，归入默认领域"}, nil
	}
	ctx, endSpan := observability.StartChildSpan(ctx, "domain.field", observability.TraceMeta{Input: name})
	defer endSpan()

	var out fieldLLMOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 的课程分类器。把一门课归入一个学习领域。只输出 JSON。"},
		{Role: "user", Content: fieldPrompt(name, slug, moduleLabels, allowed)},
	}
	ctx = observability.WithGeneration(ctx, "domain.field")
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return nil, fmt.Errorf("领域归类失败: %w", err)
	}
	chosen, created := ResolveFieldChoice(out.Field, out.Nearest, out.Confidence, allowed)
	return &FieldClass{
		Field:      chosen,
		Confidence: out.Confidence,
		Reason:     strings.TrimSpace(out.Reason),
		Created:    created,
	}, nil
}

func fieldPrompt(name, slug string, moduleLabels, allowed []string) string {
	labels := strings.Join(moduleLabels, "、")
	if labels == "" {
		labels = "（无）"
	}
	var lines []string
	for _, name := range allowed {
		lines = append(lines, "- "+name)
	}
	return fmt.Sprintf(`课程名：%q
slug：%q
模块：%s

已有领域（优先从中选择）：
%s

输出 JSON：
{"field":"编程语言","nearest":"编程语言","confidence":0.9,"reason":"这是一门编程语言课"}

规则：
- field 尽量使用已有领域中的原文
- 只有现有领域都不合适时，field 才用一个新的简短中文名（2～6 个字），并把 confidence 设到 0.6 以下
- nearest 必须是已有领域之一，表示最接近的归类
- 不要发明与已有领域同义的新名字`, name, slug, labels, strings.Join(lines, "\n"))
}
