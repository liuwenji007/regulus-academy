package channel

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// navContext 规则/LLM 导航所需的轻量上下文
type navContext struct {
	UserID          string
	Courses         []storage.DomainSummary
	PendingDomainID string
	ActiveDomainID  string
	ActiveNodeKey   string
	HasActiveSession bool
	FlatNodes       []flatNode
}

func (r *Router) buildNavContext(userID string) navContext {
	ctx := navContext{UserID: userID}
	ctx.Courses, _ = r.store.ListDomainSummaries(userID)

	r.mu.Lock()
	ctx.PendingDomainID = r.pending[userID]
	r.mu.Unlock()

	if active, _ := r.store.GetChannelActiveNode(userID); active != nil {
		ctx.ActiveDomainID = active.DomainID
		ctx.ActiveNodeKey = active.NodeKey
	}
	if sess, _ := r.sessions.ActiveSessionForUser(userID); sess != nil {
		ctx.HasActiveSession = true
		if ctx.ActiveDomainID == "" {
			ctx.ActiveDomainID = sess.DomainID
		}
		if ctx.ActiveNodeKey == "" {
			ctx.ActiveNodeKey = sess.NodeKey
		}
	}

	domainID := ctx.PendingDomainID
	if domainID == "" {
		domainID = ctx.ActiveDomainID
	}
	if domainID != "" {
		if tree, _ := r.store.GetDomainTree(userID, domainID); tree != nil {
			ctx.FlatNodes = flattenNodes(tree)
		}
	}
	return ctx
}

func resolveCourseRef(list []storage.DomainSummary, ref string) (domainID string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	if n := parsePositiveInt(ref); n > 0 && n <= len(list) {
		return list[n-1].ID, true
	}
	lower := strings.ToLower(ref)
	for _, d := range list {
		if d.ID == ref || d.Slug == ref || d.Name == ref {
			return d.ID, true
		}
	}
	for _, d := range list {
		if strings.Contains(strings.ToLower(d.Name), lower) {
			return d.ID, true
		}
		if d.Slug != "" && strings.Contains(strings.ToLower(d.Slug), lower) {
			return d.ID, true
		}
	}
	return "", false
}

func resolveNodeRef(nodes []flatNode, ref string) (nodeKey, layer string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", "", false
	}
	if n := parsePositiveInt(ref); n > 0 && n <= len(nodes) {
		return nodes[n-1].Key, nodes[n-1].Layer, true
	}
	lower := strings.ToLower(ref)
	for _, nd := range nodes {
		if nd.Key == ref {
			return nd.Key, nd.Layer, true
		}
	}
	for _, nd := range nodes {
		if strings.Contains(strings.ToLower(nd.Title), lower) ||
			strings.Contains(strings.ToLower(nd.Key), lower) {
			return nd.Key, nd.Layer, true
		}
	}
	return "", "", false
}

func findCourseInText(list []storage.DomainSummary, text string) (domainID, remainder string, ok bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	bestLen := 0
	for _, d := range list {
		candidates := []string{d.Name}
		if d.Slug != "" {
			candidates = append(candidates, d.Slug)
		}
		for _, c := range candidates {
			cl := strings.ToLower(c)
			if cl == "" {
				continue
			}
			if idx := strings.Index(lower, cl); idx >= 0 && len([]rune(cl)) >= 3 && len(cl) > bestLen {
				if !courseMatchAtWordBoundary(lower, cl, idx) {
					continue
				}
				bestLen = len(cl)
				domainID = d.ID
				remainder = strings.TrimSpace(text[idx+len(c):])
			}
		}
	}
	return domainID, remainder, domainID != ""
}

func extractOrdinalNodeRef(text string) string {
	text = strings.TrimSpace(text)
	patterns := []string{"第一个节点", "第1个节点", "第一个", "第1个", "节点1", "节点 1"}
	for _, p := range patterns {
		if strings.Contains(text, p) {
			return "1"
		}
	}
	for i, r := range text {
		if r == '第' && i+2 < len([]rune(text)) {
			rs := []rune(text[i:])
			if len(rs) >= 3 && rs[2] == '个' {
				digit := rs[1]
				if digit >= '0' && digit <= '9' {
					return string(digit)
				}
			}
		}
	}
	return ""
}

func extractCourseOrdinal(text string) string {
	rs := []rune(strings.TrimSpace(text))
	for i := 0; i < len(rs); i++ {
		if rs[i] == '第' && i+2 < len(rs) && rs[i+2] == '门' {
			if unicode.IsDigit(rs[i+1]) {
				return string(rs[i+1])
			}
		}
	}
	return ""
}

func courseRefFromID(list []storage.DomainSummary, id string) string {
	for i, d := range list {
		if d.ID == id {
			return fmt.Sprintf("%d", i+1)
		}
	}
	for _, d := range list {
		if d.ID == id {
			if d.Slug != "" {
				return d.Slug
			}
			return d.Name
		}
	}
	return ""
}

// courseMatchAtWordBoundary 避免「一般go标准项目」里的 go 误匹配课程名
func courseMatchAtWordBoundary(lower, candidate string, idx int) bool {
	if len([]rune(candidate)) >= 6 {
		return true
	}
	beforeOK := idx == 0 || !isASCIILetter(rune(lower[idx-1]))
	afterIdx := idx + len(candidate)
	afterOK := afterIdx >= len(lower) || !isASCIILetter(rune(lower[afterIdx]))
	return beforeOK && afterOK
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func stripNavFiller(s string) string {
	s = strings.TrimSpace(s)
	for _, prefix := range []string{"的", "里", "中", "关于", "一下", "看看", "打开", "进入", "开始", "学", "学习"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
	}
	return s
}
