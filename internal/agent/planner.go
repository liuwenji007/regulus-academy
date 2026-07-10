package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/observability"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const (
	planningIntakeTimeout     = 60 * time.Second
	planningSynthesizeTimeout = 120 * time.Second
	maxPlanningProfileRunes   = 500
	maxPlanningCourses        = 8
)

const planningResultSchemaCompact = `{
  "situation_summary": "string",
  "matrix": {
    "important_urgent": [{"title","why?","minutes?","next_step?"}],
    "important_not_urgent": [...],
    "quick_wins": [...],
    "defer_or_drop": [{"title","reason"}]
  },
  "action_plan": {
    "today": [{"title","minutes","kind":"task|learning","reason?"}],
    "this_week": [...]
  },
  "learning_focus": [{"area","rationale","suggested_minutes","matched_domain_id?","matched_node_key?","matched_node_title?"}],
  "mindset_note": "string"
}`

// PlanningMatrixItem 四象限条目
type PlanningMatrixItem struct {
	Title    string `json:"title"`
	Why      string `json:"why,omitempty"`
	Minutes  int    `json:"minutes,omitempty"`
	NextStep string `json:"next_step,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// PlanningActionItem 行动项
type PlanningActionItem struct {
	Title   string `json:"title"`
	Minutes int    `json:"minutes"`
	Kind    string `json:"kind"`
	Reason  string `json:"reason,omitempty"`
}

// PlanningLearningFocus 学习聚焦
type PlanningLearningFocus struct {
	Area             string `json:"area"`
	Rationale        string `json:"rationale"`
	SuggestedMinutes int    `json:"suggested_minutes"`
	MatchedDomainID  string `json:"matched_domain_id,omitempty"`
	MatchedNodeKey   string `json:"matched_node_key,omitempty"`
	MatchedNodeTitle string `json:"matched_node_title,omitempty"`
}

// PlanningMatrix 四象限
type PlanningMatrix struct {
	ImportantUrgent    []PlanningMatrixItem `json:"important_urgent"`
	ImportantNotUrgent []PlanningMatrixItem `json:"important_not_urgent"`
	QuickWins          []PlanningMatrixItem `json:"quick_wins"`
	DeferOrDrop        []PlanningMatrixItem `json:"defer_or_drop"`
}

// PlanningActionPlan 行动方案
type PlanningActionPlan struct {
	Today    []PlanningActionItem `json:"today"`
	ThisWeek []PlanningActionItem `json:"this_week"`
}

// PlanningResult 结构化规划
type PlanningResult struct {
	SituationSummary string                  `json:"situation_summary"`
	Matrix           PlanningMatrix          `json:"matrix"`
	ActionPlan       PlanningActionPlan      `json:"action_plan"`
	LearningFocus    []PlanningLearningFocus `json:"learning_focus"`
	MindsetNote      string                  `json:"mindset_note"`
}

// PlanningTurnOutput intake 阶段 LLM 输出
type PlanningTurnOutput struct {
	Reply         string `json:"reply"`
	ReadyToPlan   bool   `json:"ready_to_plan"`
	FollowUpHint  string `json:"follow_up_hint,omitempty"`
}

// Planner 行动助手 Agent
type Planner struct {
	store    *storage.Store
	llm      atomic.Value // llm.Provider
	core     string
	intake   string
	synth    string
}

// NewPlanner 创建 Planner
func NewPlanner(store *storage.Store, llmClient llm.Provider) (*Planner, error) {
	core, err := domain.LoadPrompt("planner_core")
	if err != nil {
		return nil, err
	}
	intake, err := domain.LoadPrompt("planner_intake")
	if err != nil {
		return nil, err
	}
	synth, err := domain.LoadPrompt("planner_synthesize")
	if err != nil {
		return nil, err
	}
	p := &Planner{store: store, core: core, intake: intake, synth: synth}
	p.llm.Store(llmClient)
	return p, nil
}

func (p *Planner) llmClient(ctx context.Context) llm.Provider {
	base := p.defaultLLM()
	return llm.ProviderFromContext(ctx, base)
}

func (p *Planner) defaultLLM() llm.Provider {
	if v := p.llm.Load(); v != nil {
		return v.(llm.Provider)
	}
	return nil
}

// SetLLM 热更新 LLM
func (p *Planner) SetLLM(client llm.Provider) {
	if client != nil {
		p.llm.Store(client)
	}
}

// IntakeTurn intake 阶段处理用户消息
func (p *Planner) IntakeTurn(ctx context.Context, userID string, history []llm.Message, userMsg string) (*PlanningTurnOutput, error) {
	if !p.llmClient(ctx).Configured() {
		return nil, fmt.Errorf("未配置 LLM API Key")
	}
	out, err := p.intakeTurnOnce(ctx, userID, history, userMsg, false)
	if err == nil {
		if wantsExplicitPlan(userMsg) {
			out.ReadyToPlan = true
		}
		return out, nil
	}
	if isPlanningRetryable(err) {
		if out2, err2 := p.intakeTurnOnce(ctx, userID, trimPlanningHistory(history), userMsg, true); err2 == nil {
			if wantsExplicitPlan(userMsg) {
				out2.ReadyToPlan = true
			}
			return out2, nil
		}
	}
	reply, fbErr := p.intakeFallbackPlain(ctx, userID, history, userMsg)
	if fbErr != nil {
		return nil, fbErr
	}
	ready := wantsExplicitPlan(userMsg)
	return &PlanningTurnOutput{Reply: reply, ReadyToPlan: ready}, nil
}

func (p *Planner) intakeTurnOnce(ctx context.Context, userID string, history []llm.Message, userMsg string, compact bool) (*PlanningTurnOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, planningIntakeTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "planning.intake", UserID: userID,
	})
	defer endTrace()

	profile, courseCtx, _ := p.buildUserContext(userID)
	if compact {
		profile = truncateRunes(profile, maxPlanningProfileRunes/2)
		courseCtx = ""
	}
	msgs := p.buildIntakeMessages(history, profile, courseCtx, userMsg)
	schema := `{"reply":"给用户的中文回复","ready_to_plan":false}`
	if full, err := domain.LoadSchema("planning_turn.json"); err == nil && full != "" {
		schema = full
	}
	msgs[0].Content += "\n\n【输出格式】仅输出 JSON，不要 markdown 代码块：\n" + schema
	ctx = observability.WithGeneration(ctx, "planning.intake")

	var out PlanningTurnOutput
	if err := llm.ChatPromptJSON(ctx, p.llmClient(ctx), msgs, 0.3, &out); err != nil {
		return nil, err
	}
	out.Reply = strings.TrimSpace(out.Reply)
	if out.Reply == "" {
		return nil, fmt.Errorf("模型未返回有效回复")
	}
	return &out, nil
}

func (p *Planner) intakeFallbackPlain(ctx context.Context, userID string, history []llm.Message, userMsg string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, planningIntakeTimeout)
	defer cancel()
	ctx = observability.WithGeneration(ctx, "planning.intake_fallback")

	profile, _, _ := p.buildUserContext(userID)
	system := p.core + "\n\n" + p.intake + "\n\n本轮模型 JSON 输出失败，请直接用中文回复用户（不要 JSON）。"
	var userParts []string
	if profile != "" {
		userParts = append(userParts, "【学生画像】\n"+truncateRunes(profile, maxPlanningProfileRunes))
	}
	userParts = append(userParts, "【用户】\n"+strings.TrimSpace(userMsg))
	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimPlanningHistory(history)...)
	msgs = append(msgs, llm.Message{Role: "user", Content: strings.Join(userParts, "\n\n")})
	reply, err := p.llmClient(ctx).ChatWithTemp(ctx, msgs, 0.5)
	if err != nil {
		return "", err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("模型未返回有效回复")
	}
	return reply, nil
}

// PlanReadyChat 已有规划后的轻量对话（不重新生成完整方案）
func (p *Planner) PlanReadyChat(ctx context.Context, userID string, history []llm.Message, userMsg string, existing *PlanningResult) (string, error) {
	if !p.llmClient(ctx).Configured() {
		return "", fmt.Errorf("未配置 LLM API Key")
	}
	ctx, cancel := context.WithTimeout(ctx, planningIntakeTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "planning.plan_ready_chat", UserID: userID,
	})
	defer endTrace()
	ctx = observability.WithGeneration(ctx, "planning.plan_ready_chat")

	summary := ""
	if existing != nil {
		summary = strings.TrimSpace(existing.SituationSummary)
	}
	system := p.core + "\n\n用户右侧已有行动方案清单。本轮只做简短对话：倾听补充、回答疑问、或提示可说「更新规划 / 精简今日行动」来调整方案。不要重新输出完整规划或 JSON。"
	var userParts []string
	if summary != "" {
		userParts = append(userParts, "【当前方案摘要】\n"+summary)
	}
	userParts = append(userParts, "【用户】\n"+strings.TrimSpace(userMsg))
	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimPlanningHistory(history)...)
	msgs = append(msgs, llm.Message{Role: "user", Content: strings.Join(userParts, "\n\n")})
	reply, err := p.llmClient(ctx).ChatWithTemp(ctx, msgs, 0.5)
	if err != nil {
		return "", err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "", fmt.Errorf("模型未返回有效回复")
	}
	return reply, nil
}

// Synthesize 生成或更新结构化规划
func (p *Planner) Synthesize(ctx context.Context, userID string, history []llm.Message, existingPlan *PlanningResult, refineInstruction string) (*PlanningResult, string, error) {
	if !p.llmClient(ctx).Configured() {
		return nil, "", fmt.Errorf("未配置 LLM API Key")
	}
	out, reply, err := p.synthesizeOnce(ctx, userID, history, existingPlan, refineInstruction, false)
	if err == nil {
		return out, reply, nil
	}
	if !isPlanningRetryable(err) {
		return nil, "", err
	}
	trimmed := trimPlanningHistory(history)
	if len(trimmed) > 8 {
		trimmed = trimmed[len(trimmed)-8:]
	}
	var compactExisting *PlanningResult
	if existingPlan != nil {
		compactExisting = &PlanningResult{SituationSummary: existingPlan.SituationSummary}
	}
	return p.synthesizeOnce(ctx, userID, trimmed, compactExisting, refineInstruction, true)
}

func (p *Planner) synthesizeOnce(ctx context.Context, userID string, history []llm.Message, existingPlan *PlanningResult, refineInstruction string, compact bool) (*PlanningResult, string, error) {
	ctx, cancel := context.WithTimeout(ctx, planningSynthesizeTimeout)
	defer cancel()
	ctx, endTrace := observability.Trace(ctx, observability.TraceMeta{
		Name: "planning.synthesize", UserID: userID,
	})
	defer endTrace()

	profile, courseCtx, activeCoach := p.buildUserContext(userID)
	if compact {
		profile = truncateRunes(profile, maxPlanningProfileRunes/2)
		courseCtx = truncateRunes(courseCtx, 1200)
	}
	msgs := p.buildSynthesizeMessages(history, profile, courseCtx, activeCoach, existingPlan, refineInstruction)
	msgs[0].Content += "\n\n【输出格式】仅输出 JSON，不要 markdown 代码块：\n" + planningResultSchemaCompact
	ctx = observability.WithGeneration(ctx, "planning.synthesize")

	var out PlanningResult
	if err := llm.ChatPromptJSON(ctx, p.llmClient(ctx), msgs, 0.2, &out); err != nil {
		if compact {
			return nil, "", fmt.Errorf("规划生成失败，请稍后重试或缩短描述：%w", err)
		}
		return nil, "", err
	}
	p.sanitizeLearningFocus(userID, &out)

	summary := strings.TrimSpace(out.SituationSummary)
	reply := summary
	if reply == "" {
		reply = "规划已整理好，请看下方行动清单。"
	} else {
		reply = "整理好了：" + reply + "\n\n详细行动方案见下方清单；有需要可以继续跟我说怎么调整。"
	}
	return &out, reply, nil
}

func isPlanningRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "空内容") ||
		strings.Contains(msg, "空结果") ||
		strings.Contains(msg, "解析 JSON 失败")
}

func (p *Planner) buildIntakeMessages(history []llm.Message, profile, courseCtx, userMsg string) []llm.Message {
	system := p.core + "\n\n" + p.intake
	var userParts []string
	if ctx := joinContextParts(profile, courseCtx, ""); ctx != "" {
		userParts = append(userParts, "【上下文】\n"+ctx)
	}
	userParts = append(userParts, "【任务】\n倾听用户现状，必要时追问；判断是否 ready_to_plan。")
	if msg := strings.TrimSpace(userMsg); msg != "" {
		userParts = append(userParts, "【用户】\n"+msg)
	}
	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimPlanningHistory(history)...)
	msgs = append(msgs, llm.Message{Role: "user", Content: strings.Join(userParts, "\n\n")})
	return msgs
}

func (p *Planner) buildSynthesizeMessages(history []llm.Message, profile, courseCtx, activeCoach string, existing *PlanningResult, refine string) []llm.Message {
	system := p.core + "\n\n" + p.synth
	var userParts []string
	if ctx := joinContextParts(profile, courseCtx, activeCoach); ctx != "" {
		userParts = append(userParts, "【上下文】\n"+ctx)
	}
	if existing != nil {
		b, _ := json.Marshal(existing)
		userParts = append(userParts, "【已有规划】\n"+string(b))
	}
	instr := "根据对话与上下文输出完整规划 JSON。"
	if refine := strings.TrimSpace(refine); refine != "" {
		instr = "用户在已有规划基础上要求调整：" + refine
	}
	userParts = append(userParts, "【任务】\n"+instr)
	msgs := []llm.Message{{Role: "system", Content: system}}
	msgs = append(msgs, trimPlanningHistory(history)...)
	msgs = append(msgs, llm.Message{Role: "user", Content: strings.Join(userParts, "\n\n")})
	return msgs
}

func (p *Planner) buildUserContext(userID string) (profile, courseCtx, activeCoach string) {
	if built, err := p.store.ComposeForBuild(userID); err == nil {
		profile = truncateRunes(strings.TrimSpace(built), maxPlanningProfileRunes)
	}
	var b strings.Builder
	if domains, err := p.store.ListDomainSummaries(userID); err == nil && len(domains) > 0 {
		b.WriteString("【用户课程】\n")
		limit := len(domains)
		if limit > maxPlanningCourses {
			limit = maxPlanningCourses
		}
		for i := 0; i < limit; i++ {
			d := domains[i]
			pct := 0
			if d.NodeTotal > 0 {
				pct = d.Completed * 100 / d.NodeTotal
			}
			fmt.Fprintf(&b, "- id=%s name=%s progress=%d/%d (%d%%)\n", d.ID, d.Name, d.Completed, d.NodeTotal, pct)
			if progress, err := p.store.ListProgress(userID, d.ID); err == nil {
				for _, pr := range progress {
					if pr.Status == "in_progress" {
						tree, _ := p.store.GetDomainTree(userID, d.ID)
						title := pr.NodeKey
						if tree != nil {
							title = domain.NodeTitle(tree, pr.NodeKey)
						}
						fmt.Fprintf(&b, "  · 进行中节点: key=%s title=%s layer=%s\n", pr.NodeKey, title, pr.Layer)
					}
				}
				completed := domain.CompletedKeysFromProgress(progress)
				if tree, err := p.store.GetDomainTree(userID, d.ID); err == nil && tree != nil {
					if key, layer, title, ok := firstUncompletedNode(tree, completed); ok {
						fmt.Fprintf(&b, "  · 推荐下一节点: key=%s title=%s layer=%s\n", key, title, layer)
					}
				}
			}
		}
		if len(domains) > maxPlanningCourses {
			fmt.Fprintf(&b, "… 另有 %d 门课程未列出\n", len(domains)-maxPlanningCourses)
		}
		courseCtx = strings.TrimSpace(b.String())
	}
	return profile, courseCtx, activeCoach
}

func firstUncompletedNode(tree *storage.KnowledgeTree, completed map[string]bool) (key, layer, title string, ok bool) {
	if tree == nil {
		return "", "", "", false
	}
	for _, ly := range tree.Layers {
		for _, n := range ly.Nodes {
			if !completed[n.Key] {
				return n.Key, ly.Key, n.Title, true
			}
		}
	}
	return "", "", "", false
}

func (p *Planner) sanitizeLearningFocus(userID string, plan *PlanningResult) {
	if plan == nil {
		return
	}
	validDomains := map[string]bool{}
	if domains, err := p.store.ListDomainSummaries(userID); err == nil {
		for _, d := range domains {
			validDomains[d.ID] = true
		}
	}
	for i := range plan.LearningFocus {
		lf := &plan.LearningFocus[i]
		if lf.MatchedDomainID != "" && !validDomains[lf.MatchedDomainID] {
			lf.MatchedDomainID = ""
			lf.MatchedNodeKey = ""
			lf.MatchedNodeTitle = ""
			continue
		}
		if lf.MatchedDomainID != "" && lf.MatchedNodeKey != "" {
			tree, err := p.store.GetDomainTree(userID, lf.MatchedDomainID)
			if err != nil || tree == nil {
				lf.MatchedDomainID = ""
				lf.MatchedNodeKey = ""
				lf.MatchedNodeTitle = ""
				continue
			}
			title := domain.NodeTitle(tree, lf.MatchedNodeKey)
			if title == lf.MatchedNodeKey && !nodeExists(tree, lf.MatchedNodeKey) {
				lf.MatchedDomainID = ""
				lf.MatchedNodeKey = ""
				lf.MatchedNodeTitle = ""
			} else {
				lf.MatchedNodeTitle = title
			}
		}
	}
}

func nodeExists(tree *storage.KnowledgeTree, key string) bool {
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			if n.Key == key {
				return true
			}
		}
	}
	return false
}

func joinContextParts(profile, courseCtx, activeCoach string) string {
	var parts []string
	if profile != "" {
		parts = append(parts, "【学生画像】"+profile)
	}
	if courseCtx != "" {
		parts = append(parts, courseCtx)
	}
	if activeCoach != "" {
		parts = append(parts, activeCoach)
	}
	return strings.Join(parts, "\n")
}

func trimPlanningHistory(h []llm.Message) []llm.Message {
	const max = 16
	if len(h) <= max {
		return h
	}
	return h[len(h)-max:]
}

func wantsExplicitPlan(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"帮我整理", "帮我理清", "出方案", "生成规划", "整理一下", "给我方案",
		"行动方案", "开始规划", "可以整理了", "整理吧", "帮我整理出",
	}
	for _, t := range triggers {
		if strings.Contains(msg, t) {
			return true
		}
	}
	return false
}

// WantsExplicitPlan 用户明确要求生成/整理规划
func WantsExplicitPlan(msg string) bool {
	return wantsExplicitPlan(msg)
}

func wantsPlanUpdate(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))
	triggers := []string{
		"更新规划", "重新整理", "重新规划", "调整规划", "改一下规划", "修改规划",
		"精简今日", "减到", "加到", "改成", "改为", "调整", "更新方案", "重做",
	}
	for _, t := range triggers {
		if strings.Contains(msg, t) {
			return true
		}
	}
	return false
}

// WantsPlanUpdate 用户要求调整已有规划
func WantsPlanUpdate(msg string) bool {
	return wantsPlanUpdate(msg)
}

// HistoryFromPlanningMessages 将存储消息转为 LLM 历史
func HistoryFromPlanningMessages(msgs []storage.PlanningMessage) []llm.Message {
	out := make([]llm.Message, 0, len(msgs))
	for _, m := range msgs {
		role := m.Role
		if role != "user" && role != "assistant" {
			continue
		}
		out = append(out, llm.Message{Role: role, Content: m.Content})
	}
	return out
}

// ParsePlanningResult 解析 plan_json
func ParsePlanningResult(raw string) (*PlanningResult, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var plan PlanningResult
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// MarshalPlanningResult 序列化规划
func MarshalPlanningResult(plan *PlanningResult) (string, error) {
	if plan == nil {
		return "{}", nil
	}
	b, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
