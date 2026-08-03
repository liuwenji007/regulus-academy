package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// TermCardPayload 结构化术语卡片（LLM JSON 输出）
type TermCardPayload struct {
	Term            string   `json:"term"`
	OriginalText    string   `json:"originalText"`
	IPA             string   `json:"ipa"`
	ReadingCn       string   `json:"readingCn"`
	OneLiner        string   `json:"oneLiner"`
	Explanation     string   `json:"explanation"`
	Analogy         string   `json:"analogy"`
	RelationToLesson string  `json:"relationToLesson"`
	Prerequisites   []string `json:"prerequisites"`
	ConfidenceHint  string   `json:"confidenceHint"`
	RedirectHint    string   `json:"redirectHint"`
}

// AsideExplainIntent 划词意图
const (
	AsideIntentWhat    = "what"    // 这是什么
	AsideIntentReading = "reading" // 怎么读
	AsideIntentExpand  = "expand"  // 展开讲
	AsideIntentAsk     = "ask"     // 自由问答
)

// AsideContext 旁路请求上下文（只读引用主线）
type AsideContext struct {
	UserID         string
	DomainID       string
	DomainName     string
	NodeKey        string
	NodeTitle      string
	CoachSessionID string
	AnchorText     string
	Intent         string
	Question       string
	RecentCoach    []llm.Message // 主线最近消息（只读）
}

// Aside 旁路学习助手 Agent（不写 sessions / user_progress / mistakes）
type Aside struct {
	store    *storage.Store
	llm      atomic.Value // llm.Provider
	registry *domain.Registry
}

// NewAside 创建旁路 Agent
func NewAside(store *storage.Store, llmClient llm.Provider) *Aside {
	a := &Aside{
		store:    store,
		registry: domain.NewRegistry(),
	}
	a.llm.Store(llmClient)
	return a
}

func (a *Aside) llmClient(ctx context.Context) llm.Provider {
	base := a.defaultLLM()
	return llm.ProviderFromContext(ctx, base)
}

func (a *Aside) defaultLLM() llm.Provider {
	if v := a.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
}

// SetLLM 热更新
func (a *Aside) SetLLM(client llm.Provider) {
	if client != nil {
		a.llm.Store(client)
	}
}

const asideSystemPrompt = `你是划词助教，不是主线教练。
职责：帮助学生理解课堂上遇到的术语、读音、翻译，或简短发散；不要接手完整教学。
硬性约束：
1. 解释简洁、可挂接到当前课；末尾用一两句把学生导回主线课程。
2. 不要布置作业、不要考试、不要推进课程进度。
3. 若涉及读音，给出 IPA 与中文近似读法。
4. 推断学生可能缺少的前置概念，写入 prerequisites（短词列表）。`

const termCardSchema = `{
  "term": "规范中文或常用术语名",
  "originalText": "用户划词原文",
  "ipa": "国际音标，非英文可空",
  "readingCn": "中文近似读法，可空",
  "oneLiner": "一句话定义",
  "explanation": "2-5 句 Markdown 解释",
  "analogy": "一个生活/工程类比",
  "relationToLesson": "与当前课的关系",
  "prerequisites": ["可能缺少的前置概念"],
  "confidenceHint": "beginner|intermediate|advanced",
  "redirectHint": "导回主线的一句话"
}`

// ExplainTerm 结构化术语解释（ChatJSON）
func (a *Aside) ExplainTerm(ctx context.Context, ac AsideContext) (*TermCardPayload, error) {
	client := a.llmClient(ctx)
	if !client.Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	anchor := strings.TrimSpace(ac.AnchorText)
	if anchor == "" {
		return nil, fmt.Errorf("划词原文不能为空")
	}
	intent := strings.TrimSpace(ac.Intent)
	if intent == "" {
		intent = AsideIntentWhat
	}

	focus := "给出完整术语卡片：定义、类比、与本课关系、前置知识。"
	switch intent {
	case AsideIntentReading:
		focus = "重点给出读音（IPA + 中文近似）与一句话含义；explanation 可短；仍要填 prerequisites。"
	case AsideIntentExpand:
		focus = "展开讲清楚原理与常见误区，仍保持简洁；末尾导回主线。"
	}

	var b strings.Builder
	b.WriteString("【任务】为划词生成术语卡片 JSON。\n")
	fmt.Fprintf(&b, "【意图】%s — %s\n", intent, focus)
	fmt.Fprintf(&b, "【划词】%s\n", anchor)
	if ac.DomainName != "" {
		fmt.Fprintf(&b, "【课程】%s\n", ac.DomainName)
	}
	if ac.NodeTitle != "" {
		fmt.Fprintf(&b, "【当前节点】%s\n", ac.NodeTitle)
	}
	if node := a.loadNode(ac); node != nil && len(node.CoreConcepts) > 0 {
		fmt.Fprintf(&b, "【本节点核心概念】%s\n", strings.Join(node.CoreConcepts, "；"))
	}
	if len(ac.RecentCoach) > 0 {
		b.WriteString("【主线近期对话摘录】\n")
		for _, m := range ac.RecentCoach {
			role := m.Role
			content := m.Content
			if len([]rune(content)) > 200 {
				content = string([]rune(content)[:200]) + "…"
			}
			fmt.Fprintf(&b, "- %s: %s\n", role, content)
		}
	}

	msgs := []llm.Message{
		{Role: "system", Content: asideSystemPrompt + "\n\n【输出格式】仅输出 JSON，不要 markdown 代码块：\n" + termCardSchema},
		{Role: "user", Content: b.String()},
	}

	var out TermCardPayload
	if err := client.ChatJSON(ctx, msgs, 0.3, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.OriginalText) == "" {
		out.OriginalText = anchor
	}
	if strings.TrimSpace(out.Term) == "" {
		out.Term = anchor
	}
	out.Prerequisites = NormalizeConceptList(out.Prerequisites)
	if strings.TrimSpace(out.RedirectHint) == "" {
		out.RedirectHint = "先记住这个要点，回到左侧主线继续学，系统学完会更扎实。"
	}
	return &out, nil
}

// Ask 自由问答（流式）
func (a *Aside) Ask(ctx context.Context, ac AsideContext, onDelta func(string)) (string, error) {
	client := a.llmClient(ctx)
	if !client.Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}
	q := strings.TrimSpace(ac.Question)
	if q == "" && strings.TrimSpace(ac.AnchorText) != "" {
		q = "请解释：" + strings.TrimSpace(ac.AnchorText)
	}
	if q == "" {
		return "", fmt.Errorf("问题不能为空")
	}

	var b strings.Builder
	b.WriteString("用简洁 Markdown 回答。末尾一句导回主线课程。\n")
	if ac.DomainName != "" {
		fmt.Fprintf(&b, "【课程】%s\n", ac.DomainName)
	}
	if ac.NodeTitle != "" {
		fmt.Fprintf(&b, "【当前节点】%s\n", ac.NodeTitle)
	}
	if anchor := strings.TrimSpace(ac.AnchorText); anchor != "" {
		fmt.Fprintf(&b, "【划词】%s\n", anchor)
	}
	fmt.Fprintf(&b, "【问题】%s\n", q)

	msgs := []llm.Message{{Role: "system", Content: asideSystemPrompt}}
	// 仅按当前课拉历史，避免 domain 为空时跨课串台；当前问句由调用方落库前传入，不在历史里。
	if strings.TrimSpace(ac.DomainID) != "" {
		history, _ := a.store.ListAsideMessages(ac.UserID, ac.DomainID, 12)
		for _, m := range history {
			if m.Role == "user" || m.Role == "assistant" {
				msgs = append(msgs, llm.Message{Role: m.Role, Content: m.Content})
			}
		}
	}
	msgs = append(msgs, llm.Message{Role: "user", Content: b.String()})

	return client.ChatStream(ctx, msgs, 0.5, onDelta)
}

func (a *Aside) loadNode(ac AsideContext) *domain.NodeSpec {
	if ac.DomainID == "" || ac.NodeKey == "" {
		return nil
	}
	slug := ""
	if dom, err := a.store.GetDomain(ac.UserID, ac.DomainID); err == nil && dom != nil {
		slug = dom.Slug
	}
	node, err := a.registry.GetNode(a.store, ac.DomainID, slug, ac.NodeKey)
	if err != nil {
		return nil
	}
	return node
}

// FormatTermCardMarkdown 将卡片格式化为可读 Markdown（给面板展示）
func FormatTermCardMarkdown(card *TermCardPayload) string {
	if card == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", strings.TrimSpace(card.Term))
	if card.OneLiner != "" {
		fmt.Fprintf(&b, "**%s**\n\n", card.OneLiner)
	}
	if card.IPA != "" || card.ReadingCn != "" {
		b.WriteString("**读音** ")
		if card.IPA != "" {
			fmt.Fprintf(&b, `%s `, card.IPA)
		}
		if card.ReadingCn != "" {
			fmt.Fprintf(&b, "（%s）", card.ReadingCn)
		}
		b.WriteString("\n\n")
	}
	if card.Explanation != "" {
		b.WriteString(card.Explanation)
		b.WriteString("\n\n")
	}
	if card.Analogy != "" {
		fmt.Fprintf(&b, "> 类比：%s\n\n", card.Analogy)
	}
	if card.RelationToLesson != "" {
		fmt.Fprintf(&b, "**与本课** %s\n\n", card.RelationToLesson)
	}
	if len(card.Prerequisites) > 0 {
		fmt.Fprintf(&b, "**可能需要先了解** %s\n\n", strings.Join(card.Prerequisites, "、"))
	}
	if card.RedirectHint != "" {
		fmt.Fprintf(&b, "—\n*%s*", card.RedirectHint)
	}
	return b.String()
}

// TermCardToJSON 序列化卡片
func TermCardToJSON(card *TermCardPayload) string {
	if card == nil {
		return "{}"
	}
	b, err := json.Marshal(card)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ParseTermCardJSON 反序列化
func ParseTermCardJSON(raw string) *TermCardPayload {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var c TermCardPayload
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil
	}
	return &c
}
