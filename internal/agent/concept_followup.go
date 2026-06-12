package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const followUpDeepenThreshold = 2

// MatchConceptInMessage 从用户消息中匹配最相关的核心概念（命中得分最高者优先）。
func MatchConceptInMessage(msg string, concepts []string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" || len(concepts) == 0 {
		return ""
	}
	best := ""
	bestScore := 0
	for _, c := range concepts {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		score := conceptMatchScore(msg, c)
		if score == 0 {
			continue
		}
		if score > bestScore || (score == bestScore && len([]rune(c)) > len([]rune(best))) {
			bestScore = score
			best = c
		}
	}
	return best
}

func conceptTokens(concept string) []string {
	replacer := strings.NewReplacer("：", " ", "、", " ", "，", " ", ",", " ", "；", " ", "（", " ", "）", " ", "的", " ")
	normalized := replacer.Replace(strings.TrimSpace(concept))
	var out []string
	for _, p := range strings.Fields(normalized) {
		p = strings.TrimSpace(p)
		if len([]rune(p)) < 2 {
			continue
		}
		out = append(out, p)
		for _, prefix := range []string{"与", "和"} {
			if strings.HasPrefix(p, prefix) && len([]rune(p)) > len([]rune(prefix))+1 {
				out = append(out, strings.TrimPrefix(p, prefix))
			}
		}
	}
	return out
}

func conceptMatchScore(msg, concept string) int {
	if conceptMatches(concept, msg) {
		return len([]rune(concept)) + 1000
	}
	total := 0
	for _, tok := range conceptTokens(concept) {
		if len([]rune(tok)) < 2 || (len([]rune(tok)) < 4 && isASCIIToken(tok)) {
			continue
		}
		if strings.Contains(msg, tok) {
			total += len([]rune(tok))
		}
	}
	return total
}

func isASCIIToken(tok string) bool {
	for _, r := range tok {
		if r > 127 {
			return false
		}
	}
	return tok != ""
}

func conceptAlreadyDeepened(concept string, deepened []string) bool {
	for _, d := range deepened {
		if conceptMatches(concept, d) {
			return true
		}
	}
	return false
}

func recordConceptFollowUp(sctx *storage.SessionContext, concept string) int {
	if sctx.ConceptFollowUps == nil {
		sctx.ConceptFollowUps = make(map[string]int)
	}
	sctx.ConceptFollowUps[concept]++
	return sctx.ConceptFollowUps[concept]
}

// DeepenedConcepts 记录追问深讲已触发（门禁）；ExplainedConcepts 供出题 prompt 使用，二者分工不同。
func shouldDeepenOnFollowUp(sctx *storage.SessionContext, concept string) bool {
	if concept == "" || conceptAlreadyDeepened(concept, sctx.DeepenedConcepts) {
		return false
	}
	return sctx.ConceptFollowUps[concept] >= followUpDeepenThreshold
}

func markConceptDeepened(sctx *storage.SessionContext, core []string, concept string) {
	norm := NormalizeToCoreConcept(concept, core)
	if norm == "" {
		norm = strings.TrimSpace(concept)
	}
	if norm == "" {
		return
	}
	if conceptAlreadyDeepened(norm, sctx.DeepenedConcepts) {
		return
	}
	sctx.DeepenedConcepts = append(sctx.DeepenedConcepts, norm)
}

// maybeDeepenOnFollowUp 同一概念被追问达到阈值时触发递进深讲。
func (c *Coach) maybeDeepenOnFollowUp(
	ctx context.Context,
	sess *storage.Session,
	sctx *storage.SessionContext,
	userMsg string,
) (*MessageResult, bool) {
	userMsg = strings.TrimSpace(userMsg)
	if userMsg == "" {
		return nil, false
	}
	node, err := c.registry.GetNode(c.store, sess.DomainID, sess.DomainSlug, sess.NodeKey)
	if err != nil || node == nil {
		return nil, false
	}
	candidates := append([]string{}, node.CoreConcepts...)
	if sess.Phase == "review" && len(sctx.RecentMistakes) > 0 {
		candidates = append(sctx.RecentMistakes, candidates...)
	}
	matched := MatchConceptInMessage(userMsg, candidates)
	if matched == "" {
		return nil, false
	}
	if norm := NormalizeToCoreConcept(matched, node.CoreConcepts); norm != "" {
		matched = norm
	}
	recordConceptFollowUp(sctx, matched)
	if !shouldDeepenOnFollowUp(sctx, matched) {
		_ = storage.SaveSessionContext(sess, *sctx)
		_ = c.store.UpdateSession(sess)
		return nil, false
	}
	content, err := c.deepenConcept(ctx, sess, sctx, matched, userMsg)
	if err != nil {
		return nil, false
	}
	markConceptDeepened(sctx, node.CoreConcepts, matched)
	_ = storage.SaveSessionContext(sess, *sctx)
	_ = c.store.UpdateSession(sess)
	content = fmt.Sprintf("你反复问到「%s」，我展开讲一下：\n\n%s", matched, content)
	return &MessageResult{Role: "assistant", Content: content, Phase: sess.Phase}, true
}
