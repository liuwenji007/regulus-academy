package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// AsideStreamEvent 旁路 SSE 事件
type AsideStreamEvent struct {
	Type    string `json:"type"` // delta | message | error | card
	Text    string `json:"text,omitempty"`
	Content string `json:"content,omitempty"`
	Code    string `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
	Card    any    `json:"card,omitempty"`
}

// AsideService 旁路学习服务
type AsideService struct {
	store  *storage.Store
	aside  *agent.Aside
	ledger *agent.GapLedger
	llm    llm.Provider
}

// NewAsideService 创建
func NewAsideService(store *storage.Store, aside *agent.Aside, llmClient llm.Provider) *AsideService {
	return &AsideService{
		store:  store,
		aside:  aside,
		ledger: agent.NewGapLedger(store),
		llm:    llmClient,
	}
}

// SetLLM 热更新（主 LLM；旁路可能另有轻量 profile，由调用方通过 context 注入）
func (s *AsideService) SetLLM(client llm.Provider) {
	s.llm = client
	if s.aside != nil {
		s.aside.SetLLM(client)
	}
}

// Aside 返回底层 Agent
func (s *AsideService) Aside() *agent.Aside {
	return s.aside
}

// ExplainRequest 术语解释请求
type ExplainRequest struct {
	DomainID       string `json:"domainId"`
	NodeKey        string `json:"nodeKey"`
	CoachSessionID string `json:"coachSessionId"`
	AnchorText     string `json:"anchorText"`
	Intent         string `json:"intent"`
}

// ExplainResult 解释结果
type ExplainResult struct {
	Cached   bool                   `json:"cached"`
	Card     *agent.TermCardPayload `json:"card"`
	Markdown string                 `json:"markdown"`
	HitCount int                    `json:"hitCount"`
}

func normalizeAsideAnchor(anchor string) (norm string, err error) {
	anchor = strings.TrimSpace(anchor)
	if anchor == "" {
		return "", fmt.Errorf("划词原文不能为空")
	}
	norm = agent.NormalizeConcept(anchor)
	if norm == "" {
		norm = strings.ToLower(anchor)
	}
	return norm, nil
}

// ExplainCached 仅读缓存；未命中返回 (nil, nil)。命中时只抬术语卡 hit_count，不写缺口账本。
func (s *AsideService) ExplainCached(userID string, req ExplainRequest) (*ExplainResult, error) {
	anchor := strings.TrimSpace(req.AnchorText)
	norm, err := normalizeAsideAnchor(anchor)
	if err != nil {
		return nil, err
	}
	domainID := strings.TrimSpace(req.DomainID)
	cached, err := s.store.GetTermCard(userID, domainID, norm)
	if err != nil {
		return nil, err
	}
	if cached == nil {
		return nil, nil
	}
	card := agent.ParseTermCardJSON(cached.CardJSON)
	if card == nil {
		return nil, nil
	}
	updated, _ := s.store.UpsertTermCard(&storage.TermCard{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		NormalizedTerm: norm,
		OriginalText:   anchor,
		CardJSON:       cached.CardJSON,
	})
	hit := 1
	if updated != nil {
		hit = updated.HitCount
	}
	return &ExplainResult{
		Cached:   true,
		Card:     card,
		Markdown: agent.FormatTermCardMarkdown(card),
		HitCount: hit,
	}, nil
}

// ExplainGenerate 生成术语卡并记账（调用方已注入 LLM / 校验额度）
func (s *AsideService) ExplainGenerate(ctx context.Context, userID string, req ExplainRequest) (*ExplainResult, error) {
	anchor := strings.TrimSpace(req.AnchorText)
	norm, err := normalizeAsideAnchor(anchor)
	if err != nil {
		return nil, err
	}
	domainID := strings.TrimSpace(req.DomainID)

	ac := s.buildContext(userID, req.DomainID, req.NodeKey, req.CoachSessionID, anchor, req.Intent, "")
	card, err := s.aside.ExplainTerm(ctx, ac)
	if err != nil {
		return nil, err
	}

	cardJSON := agent.TermCardToJSON(card)
	stored, err := s.store.UpsertTermCard(&storage.TermCard{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		NormalizedTerm: norm,
		OriginalText:   anchor,
		CardJSON:       cardJSON,
	})
	if err != nil {
		return nil, err
	}

	md := agent.FormatTermCardMarkdown(card)
	_, _ = s.store.AddAsideMessage(&storage.AsideMessage{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		CoachSessionID: req.CoachSessionID,
		Role:           "user",
		Content:        fmt.Sprintf("[%s] %s", intentLabel(req.Intent), anchor),
		AnchorText:     anchor,
		Intent:         req.Intent,
	})
	_, _ = s.store.AddAsideMessage(&storage.AsideMessage{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		CoachSessionID: req.CoachSessionID,
		Role:           "assistant",
		Content:        md,
		AnchorText:     anchor,
		Intent:         req.Intent,
	})

	go s.ledger.RecordFromTermCard(userID, domainID, req.NodeKey, card)

	hit := 1
	if stored != nil {
		hit = stored.HitCount
	}
	return &ExplainResult{
		Cached:   false,
		Card:     card,
		Markdown: md,
		HitCount: hit,
	}, nil
}

// Explain 兼容：先缓存后生成
func (s *AsideService) Explain(ctx context.Context, userID string, req ExplainRequest) (*ExplainResult, error) {
	if out, err := s.ExplainCached(userID, req); err != nil {
		return nil, err
	} else if out != nil {
		return out, nil
	}
	return s.ExplainGenerate(ctx, userID, req)
}

// AskRequest 自由问答
type AskRequest struct {
	DomainID       string `json:"domainId"`
	NodeKey        string `json:"nodeKey"`
	CoachSessionID string `json:"coachSessionId"`
	AnchorText     string `json:"anchorText"`
	Question       string `json:"question"`
}

// AskStream 流式问答：先生成再落库，避免当前 user turn 进历史重复。
func (s *AsideService) AskStream(
	ctx context.Context,
	userID string,
	req AskRequest,
	emit func(AsideStreamEvent),
) (string, error) {
	q := strings.TrimSpace(req.Question)
	if q == "" {
		return "", fmt.Errorf("问题不能为空")
	}
	domainID := strings.TrimSpace(req.DomainID)

	ac := s.buildContext(userID, req.DomainID, req.NodeKey, req.CoachSessionID, req.AnchorText, agent.AsideIntentAsk, q)
	full, err := s.aside.Ask(ctx, ac, func(delta string) {
		if emit != nil && delta != "" {
			emit(AsideStreamEvent{Type: "delta", Text: delta})
		}
	})
	if err != nil {
		return "", err
	}

	_, _ = s.store.AddAsideMessage(&storage.AsideMessage{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		CoachSessionID: req.CoachSessionID,
		Role:           "user",
		Content:        q,
		AnchorText:     req.AnchorText,
		Intent:         agent.AsideIntentAsk,
	})
	_, _ = s.store.AddAsideMessage(&storage.AsideMessage{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        req.NodeKey,
		CoachSessionID: req.CoachSessionID,
		Role:           "assistant",
		Content:        full,
		AnchorText:     req.AnchorText,
		Intent:         agent.AsideIntentAsk,
	})

	go s.extractAskGaps(userID, domainID, req.NodeKey, full)

	if emit != nil {
		emit(AsideStreamEvent{Type: "message", Content: full})
	}
	return full, nil
}

// extractAskGaps 仅从回答中「可能需要先了解」类行抽取短词；不把整句问题落库。
func (s *AsideService) extractAskGaps(userID, domainID, nodeKey, answer string) {
	concepts := extractPrerequisiteConcepts(answer)
	if len(concepts) == 0 {
		return
	}
	s.ledger.RecordConcepts(userID, domainID, nodeKey, storage.GapSourceAsideLookup,
		"助教自由问答", concepts)
}

func extractPrerequisiteConcepts(answer string) []string {
	var concepts []string
	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.Contains(line, "可能需要") && !strings.Contains(line, "先了解") {
			continue
		}
		// 去掉标签前缀，只保留列表部分
		for _, sep := range []string{"可能需要先了解", "可能需要了解", "可能需要", "先了解"} {
			if i := strings.Index(line, sep); i >= 0 {
				line = line[i+len(sep):]
				break
			}
		}
		line = strings.TrimLeft(line, " ：:*-—>")
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '、' || r == ',' || r == '；' || r == ';' || r == '|'
		})
		for _, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.Trim(p, "*#>`-—：:。.")
			if p == "" || isGapNoiseToken(p) {
				continue
			}
			if c := agent.NormalizeConcept(p); c != "" && utf8.RuneCountInString(c) >= 2 && utf8.RuneCountInString(c) <= 16 {
				concepts = append(concepts, c)
			}
		}
	}
	return agent.NormalizeConceptList(concepts)
}

func isGapNoiseToken(p string) bool {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case "可能需要", "先了解", "前置", "前置知识", "概念", "知识", "以下", "如下",
		"the", "a", "an", "and", "or", "of", "to", "in", "on":
		return true
	}
	return false
}

// ListTerms 术语本
func (s *AsideService) ListTerms(userID, domainID string) ([]map[string]any, error) {
	cards, err := s.store.ListTermCards(userID, domainID, 100)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(cards))
	for _, c := range cards {
		item := map[string]any{
			"id":             c.ID,
			"domainId":       c.DomainID,
			"nodeKey":        c.NodeKey,
			"normalizedTerm": c.NormalizedTerm,
			"originalText":   c.OriginalText,
			"hitCount":       c.HitCount,
			"lastHitAt":      c.LastHitAt,
		}
		if card := agent.ParseTermCardJSON(c.CardJSON); card != nil {
			item["card"] = card
			item["oneLiner"] = card.OneLiner
			item["term"] = card.Term
		} else {
			item["term"] = c.NormalizedTerm
		}
		out = append(out, item)
	}
	return out, nil
}

// ListMessages 旁路历史
func (s *AsideService) ListMessages(userID, domainID string) ([]storage.AsideMessage, error) {
	return s.store.ListAsideMessages(userID, domainID, 80)
}

// ListGaps / ResolveGap 代理到 storage
func (s *AsideService) ListGaps(userID, domainID string) ([]storage.KnowledgeGap, error) {
	return s.store.ListOpenKnowledgeGaps(userID, domainID, 50)
}

func (s *AsideService) ResolveGap(userID string, id int64) error {
	return s.store.ResolveKnowledgeGap(userID, id)
}

func (s *AsideService) buildContext(
	userID, domainID, nodeKey, coachSessionID, anchor, intent, question string,
) agent.AsideContext {
	ac := agent.AsideContext{
		UserID:         userID,
		DomainID:       domainID,
		NodeKey:        nodeKey,
		CoachSessionID: coachSessionID,
		AnchorText:     anchor,
		Intent:         intent,
		Question:       question,
	}
	if domainID != "" {
		if tree, err := s.store.GetDomainTree(userID, domainID); err == nil && tree != nil {
			ac.DomainName = tree.DomainName
			if nodeKey != "" {
				ac.NodeTitle = domain.NodeTitle(tree, nodeKey)
			}
		}
	}
	if coachSessionID != "" {
		sess, err := s.store.GetSession(coachSessionID)
		if err == nil && sess != nil && sess.UserID == userID {
			if domainID == "" || sess.DomainID == "" || sess.DomainID == domainID {
				if msgs, err := s.store.ListMessages(coachSessionID); err == nil {
					start := 0
					if len(msgs) > 6 {
						start = len(msgs) - 6
					}
					for _, m := range msgs[start:] {
						ac.RecentCoach = append(ac.RecentCoach, llm.Message{Role: m.Role, Content: m.Content})
					}
				}
			}
		}
	}
	return ac
}

func intentLabel(intent string) string {
	switch strings.TrimSpace(intent) {
	case agent.AsideIntentReading:
		return "怎么读"
	case agent.AsideIntentExpand:
		return "展开讲"
	default:
		return "这是什么"
	}
}
