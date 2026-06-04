package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
)

const (
	SourceSkillPack  = "skill_pack"
	SourceGenerated  = "generated"
)

// IntentResult 用户学习意图分析结果
type IntentResult struct {
	Slug          string   `json:"slug"`
	DisplayName   string   `json:"displayName"`
	Confidence    float64  `json:"confidence"`
	Reason        string   `json:"reason"`
	Source        string   `json:"source"`
	ScopeBreadth  string   `json:"scopeBreadth"` // narrow | moderate | broad
	RootSlug      string   `json:"rootSlug,omitempty"`
	FocusSlug     string   `json:"focusSlug,omitempty"`
	FocusLabel    string   `json:"focusLabel,omitempty"`
	FocusNodeKeys []string `json:"focusNodeKeys,omitempty"`
}

// ParseIntent 理解用户想学什么，并判断是否可走 Skill 包快路径
func (r *Registry) ParseIntent(ctx context.Context, client llm.Provider, userInput string) (IntentResult, error) {
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "domain.intent", Input: userInput,
	})
	defer endTrace()

	input := strings.TrimSpace(userInput)
	if input == "" {
		return IntentResult{}, fmt.Errorf("输入不能为空")
	}

	if slug, ok := r.MatchDomain(input); ok {
		return r.intentFromSlug(slug, input, 1, "与已有 Skill 包直接匹配"), nil
	}

	if !client.Configured() {
		slug := Slugify(input)
		if r.HasSkillPack(slug) {
			return r.intentFromSlug(slug, input, 0.8, "根据输入推断主题"), nil
		}
		return IntentResult{}, fmt.Errorf("未配置 LLM，无法生成新课程")
	}

	var out intentLLMOutput
	msgs := []llm.Message{
		{Role: "system", Content: "你是 Regulus Academy 的学习意图分析器。根据用户第一句话，理解其想学的主题。只输出 JSON。"},
		{Role: "user", Content: buildParseIntentPrompt(input, r.optionalSkillList())},
	}
	ctx = observability.WithGeneration(ctx, "domain.intent")
	if err := client.ChatJSON(ctx, msgs, 0.2, &out); err != nil {
		return IntentResult{}, fmt.Errorf("意图分析失败: %w", err)
	}

	result := IntentResult{
		Slug:         Slugify(strings.TrimSpace(out.Slug)),
		DisplayName:  strings.TrimSpace(out.DisplayName),
		Confidence:   out.Confidence,
		Reason:       strings.TrimSpace(out.Reason),
		ScopeBreadth: normalizeScope(out.ScopeBreadth),
	}
	if result.DisplayName == "" {
		result.DisplayName = input
	}
	if result.Slug == "" {
		result.Slug = Slugify(result.DisplayName)
	}
	if result.Slug == "" {
		result.Slug = Slugify(input)
	}

	if meta, ok := r.FindDomainBySlug(result.Slug); ok {
		result.Source = SourceSkillPack
		if meta.Name != "" {
			result.DisplayName = meta.Name
		}
		if result.Reason == "" {
			result.Reason = "匹配已有 Skill 包"
		}
	} else {
		result.Source = SourceGenerated
		if result.Reason == "" {
			result.Reason = fmt.Sprintf("将为你生成「%s」学习路径", result.DisplayName)
		}
	}
	return result, nil
}

func (r *Registry) intentFromSlug(slug, input string, confidence float64, reason string) IntentResult {
	meta, _ := r.FindDomainBySlug(slug)
	name := meta.Name
	if name == "" {
		name = input
	}
	return IntentResult{
		Slug:         slug,
		DisplayName:  name,
		Confidence:   confidence,
		Reason:       reason,
		Source:       SourceSkillPack,
		ScopeBreadth: ScopeModerate,
	}
}

func (r *Registry) optionalSkillList() []DomainMeta {
	list, err := r.ListDomains()
	if err != nil {
		return nil
	}
	return list
}

func (r *Registry) HasSkillPack(slug string) bool {
	_, ok := r.FindDomainBySlug(slug)
	return ok
}

type intentLLMOutput struct {
	Slug         string  `json:"slug"`
	DisplayName  string  `json:"displayName"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
	ScopeBreadth string  `json:"scopeBreadth"`
}

func buildParseIntentPrompt(userInput string, skills []DomainMeta) string {
	var b strings.Builder
	b.WriteString("用户输入：")
	b.WriteString(userInput)
	b.WriteString("\n\n")
	if len(skills) > 0 {
		b.WriteString("仓库中已有的 Skill 包（若用户想学的主题与其中一门相同，slug 请用对应值）：\n")
		for _, d := range skills {
			b.WriteString(fmt.Sprintf("- slug=%q 名称=%q\n", d.Slug, d.Name))
		}
		b.WriteString("\n")
	}
	b.WriteString(`请输出 JSON：
{
  "slug": "kebab-case 英文标识，如 rust、go-concurrency、agent-basics",
  "displayName": "用户想学的主题（中文，简短）",
  "confidence": 0.0到1.0,
  "reason": "一句话说明理解到的学习意图",
  "scopeBreadth": "narrow | moderate | broad"
}

规则：
- 理解用户真实想学的内容，不要强行限制在已有 Skill 列表
- slug 用小写英文与连字符，便于存储
- displayName 用中文
- scopeBreadth 评估主题广度：
  - narrow：聚焦子话题（如「Go channel」「React useEffect」）
  - moderate：中等范围（如「Go 语言」「前端工程化」）
  - broad：宽泛领域（如「Rust」「分布式系统」「Agent 开发」）`)
	return b.String()
}

var slugSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify 将主题转为 kebab-case slug
func Slugify(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		if r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
		} else if unicode.Is(unicode.Han, r) {
			b.WriteByte('-')
		} else {
			b.WriteByte('-')
		}
	}
	out := slugSanitizer.ReplaceAllString(b.String(), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		out = slugSanitizer.ReplaceAllString(s, "-")
		out = strings.Trim(out, "-")
	}
	return out
}

// IntentResultJSON 调试用
func IntentResultJSON(r IntentResult) string {
	b, _ := json.Marshal(r)
	return string(b)
}
