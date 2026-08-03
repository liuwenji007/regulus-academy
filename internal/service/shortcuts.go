package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/regulus-academy/regulus-academy/internal/agent"
	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

const maxSidebarRecommendations = 2

// LearningShortcuts 侧栏「上一节 + 今日推荐」
type LearningShortcuts struct {
	LastLesson      *LastLessonShortcut      `json:"lastLesson"`
	Recommendations []ShortcutRecommendation `json:"recommendations"`
	HasCourses      bool                     `json:"hasCourses"`
}

// LastLessonShortcut 上一节学的课
type LastLessonShortcut struct {
	DomainID     string    `json:"domainId"`
	DomainName   string    `json:"domainName"`
	NodeKey      string    `json:"nodeKey"`
	NodeTitle    string    `json:"nodeTitle"`
	SessionID    string    `json:"sessionId"`
	Phase        string    `json:"phase"`
	Status       string    `json:"status"`
	LastActiveAt time.Time `json:"lastActiveAt"`
	CanResume    bool      `json:"canResume"`
}

// ShortcutRecommendation 今日推荐条目
type ShortcutRecommendation struct {
	Source     string `json:"source"` // planning | progress | gap
	DomainID   string `json:"domainId"`
	DomainName string `json:"domainName"`
	Title      string `json:"title,omitempty"`
	NodeKey    string `json:"nodeKey,omitempty"`
	NodeTitle  string `json:"nodeTitle,omitempty"`
	Minutes    int    `json:"minutes,omitempty"`
	Completed  int    `json:"completed"`
	NodeTotal  int    `json:"nodeTotal"`
	SessionID  string `json:"sessionId,omitempty"`
	CanResume  bool   `json:"canResume"`
	Reason     string `json:"reason,omitempty"` // 可解释推荐理由
}

// ShortcutsService 侧栏学习快捷入口
type ShortcutsService struct {
	store   *storage.Store
	planner *agent.Planner
}

// NewShortcutsService 创建 ShortcutsService
func NewShortcutsService(store *storage.Store, planner *agent.Planner) *ShortcutsService {
	return &ShortcutsService{store: store, planner: planner}
}

// GetLearningShortcuts 组装侧栏上一节与今日推荐
func (s *ShortcutsService) GetLearningShortcuts(userID string) (*LearningShortcuts, error) {
	out := &LearningShortcuts{
		Recommendations: []ShortcutRecommendation{},
	}
	domains, err := s.store.ListDomainSummaries(userID)
	if err != nil {
		return nil, err
	}
	out.HasCourses = len(domains) > 0
	byID := make(map[string]storage.DomainSummary, len(domains))
	for _, d := range domains {
		byID[d.ID] = d
	}

	lastDomainID := ""
	if last, err := s.buildLastLesson(userID, byID); err != nil {
		return nil, err
	} else if last != nil {
		out.LastLesson = last
		lastDomainID = last.DomainID
	}

	used := map[string]bool{}
	if lastDomainID != "" {
		used[lastDomainID] = true
	}

	if rec := s.planningRecommendation(userID, byID); rec != nil {
		out.Recommendations = append(out.Recommendations, *rec)
		used[rec.DomainID] = true
	}

	if rec := s.gapRecommendation(userID, byID, used); rec != nil {
		out.Recommendations = append(out.Recommendations, *rec)
		used[rec.DomainID] = true
	}

	activity, _ := s.store.DomainLastActivity(userID)
	for _, rec := range progressRecommendations(domains, used, activity, maxSidebarRecommendations-len(out.Recommendations)) {
		out.Recommendations = append(out.Recommendations, rec)
		used[rec.DomainID] = true
	}

	return out, nil
}

func (s *ShortcutsService) buildLastLesson(userID string, byID map[string]storage.DomainSummary) (*LastLessonShortcut, error) {
	access, err := s.store.FindLastDomainAccess(userID)
	if err != nil {
		return nil, err
	}
	sess, err := s.store.FindLastStudiedSession(userID)
	if err != nil {
		return nil, err
	}

	useAccess := false
	if access != nil && !access.AccessedAt.IsZero() {
		if sess == nil {
			useAccess = true
		} else {
			sessAt := sess.UpdatedAt
			if sessAt.IsZero() {
				sessAt = sess.CreatedAt
			}
			useAccess = !access.AccessedAt.Before(sessAt)
		}
	}
	if useAccess {
		return s.lastLessonFromDomain(userID, access.DomainID, access.NodeKey, access.AccessedAt, byID)
	}
	if sess == nil {
		return nil, nil
	}
	return s.lastLessonFromSession(userID, sess, byID)
}

func (s *ShortcutsService) lastLessonFromDomain(
	userID, domainID, preferNodeKey string,
	accessedAt time.Time,
	byID map[string]storage.DomainSummary,
) (*LastLessonShortcut, error) {
	domName := domainNameFromMap(userID, domainID, byID, s.store)
	var sess *storage.Session
	if preferNodeKey != "" {
		if found, err := s.store.FindLatestSession(userID, domainID, preferNodeKey); err == nil {
			sess = found
		}
	}
	if sess == nil {
		if found, err := s.store.FindLatestSessionInDomain(userID, domainID); err == nil {
			sess = found
		}
	}
	if sess != nil {
		out := s.sessionToLastLesson(userID, sess, byID)
		out.DomainName = domName
		if accessedAt.After(out.LastActiveAt) {
			out.LastActiveAt = accessedAt
		}
		return out, nil
	}
	nodeKey := preferNodeKey
	nodeTitle := ""
	if nodeKey != "" {
		if tree, err := s.store.GetDomainTree(userID, domainID); err == nil && tree != nil {
			nodeTitle = domain.NodeTitle(tree, nodeKey)
		}
	}
	return &LastLessonShortcut{
		DomainID:     domainID,
		DomainName:   domName,
		NodeKey:      nodeKey,
		NodeTitle:    nodeTitle,
		LastActiveAt: accessedAt,
		CanResume:    false,
	}, nil
}

func (s *ShortcutsService) lastLessonFromSession(
	userID string,
	sess *storage.Session,
	byID map[string]storage.DomainSummary,
) (*LastLessonShortcut, error) {
	return s.sessionToLastLesson(userID, sess, byID), nil
}

func (s *ShortcutsService) sessionToLastLesson(
	userID string,
	sess *storage.Session,
	byID map[string]storage.DomainSummary,
) *LastLessonShortcut {
	domName := domainNameFromMap(userID, sess.DomainID, byID, s.store)
	nodeTitle := sess.NodeKey
	if tree, err := s.store.GetDomainTree(userID, sess.DomainID); err == nil && tree != nil {
		if t := domain.NodeTitle(tree, sess.NodeKey); t != "" {
			nodeTitle = t
		}
	}
	canResume := sess.Status == "active" && sess.Phase != "completed"
	lastAt := sess.UpdatedAt
	if lastAt.IsZero() {
		lastAt = sess.CreatedAt
	}
	return &LastLessonShortcut{
		DomainID:     sess.DomainID,
		DomainName:   domName,
		NodeKey:      sess.NodeKey,
		NodeTitle:    nodeTitle,
		SessionID:    sess.ID,
		Phase:        sess.Phase,
		Status:       sess.Status,
		LastActiveAt: lastAt,
		CanResume:    canResume,
	}
}

func domainNameFromMap(userID, domainID string, byID map[string]storage.DomainSummary, store *storage.Store) string {
	if d, ok := byID[domainID]; ok {
		return d.Name
	}
	if dom, err := store.GetDomain(userID, domainID); err == nil && dom != nil {
		return dom.Name
	}
	return ""
}

func (s *ShortcutsService) planningRecommendation(userID string, byID map[string]storage.DomainSummary) *ShortcutRecommendation {
	ps, err := s.store.FindActivePlanningSession(userID)
	if err != nil || ps == nil {
		return nil
	}
	plan, err := agent.ParsePlanningResult(ps.PlanJSON)
	if err != nil || plan == nil {
		return nil
	}
	if s.planner != nil {
		s.planner.HydrateLegacyPlan(userID, plan)
	}
	if plan.Focus == nil || plan.Focus.TodayLearning == nil {
		return nil
	}
	tl := plan.Focus.TodayLearning
	domainID := strings.TrimSpace(tl.MatchedDomainID)
	if domainID == "" {
		return nil
	}
	d, ok := byID[domainID]
	if !ok {
		return nil
	}
	rec := &ShortcutRecommendation{
		Source:     "planning",
		DomainID:   d.ID,
		DomainName: d.Name,
		Title:      strings.TrimSpace(tl.Title),
		NodeKey:    strings.TrimSpace(tl.MatchedNodeKey),
		NodeTitle:  strings.TrimSpace(tl.MatchedNodeTitle),
		Minutes:    tl.Minutes,
		Completed:  d.Completed,
		NodeTotal:  d.NodeTotal,
	}
	if rec.Minutes <= 0 {
		rec.Minutes = 15
	}
	if rec.NodeKey != "" {
		if active, err := s.store.FindActiveSession(userID, domainID, rec.NodeKey); err == nil && active != nil {
			rec.SessionID = active.ID
			rec.CanResume = true
		}
		if rec.NodeTitle == "" {
			if tree, err := s.store.GetDomainTree(userID, domainID); err == nil && tree != nil {
				rec.NodeTitle = domain.NodeTitle(tree, rec.NodeKey)
			}
		}
	}
	return rec
}

func (s *ShortcutsService) gapRecommendation(
	userID string,
	byID map[string]storage.DomainSummary,
	used map[string]bool,
) *ShortcutRecommendation {
	gaps, err := s.store.ListOpenKnowledgeGaps(userID, "", 15)
	if err != nil || len(gaps) == 0 {
		return nil
	}
	registry := domain.NewRegistry()
	for _, g := range gaps {
		domainID := strings.TrimSpace(g.MatchedDomainID)
		nodeKey := strings.TrimSpace(g.MatchedNodeKey)
		nodeTitle := ""

		if domainID == "" {
			domainID = strings.TrimSpace(g.DomainID)
		}
		if domainID == "" {
			continue
		}
		if used[domainID] {
			continue
		}
		d, ok := byID[domainID]
		if !ok {
			continue
		}
		if d.NodeTotal > 0 && d.Completed >= d.NodeTotal {
			continue
		}

		if nodeKey == "" {
			matchedKey, matchedTitle := agent.MatchConceptToNode(s.store, registry, userID, domainID, g.Concept)
			if matchedKey != "" {
				nodeKey = matchedKey
				nodeTitle = matchedTitle
				_ = s.store.UpdateKnowledgeGapMatch(userID, g.ID, domainID, matchedKey)
			}
		} else if nodeTitle == "" {
			if tree, err := s.store.GetDomainTree(userID, domainID); err == nil && tree != nil {
				nodeTitle = domain.NodeTitle(tree, nodeKey)
			}
		}
		if nodeKey == "" {
			// 无节点映射时仍可推荐该课程
			nodeTitle = ""
		}

		reason := strings.TrimSpace(g.Reason)
		if reason == "" {
			switch g.Source {
			case storage.GapSourceMistake:
				reason = fmt.Sprintf("你在练习中错过「%s」相关题（%d 次）", g.Concept, g.HitCount)
			case storage.GapSourceCoachGap:
				reason = fmt.Sprintf("掌握检测发现「%s」还不稳", g.Concept)
			default:
				reason = fmt.Sprintf("你查过「%s」%d 次，建议先补这块", g.Concept, g.HitCount)
			}
		}

		rec := &ShortcutRecommendation{
			Source:     "gap",
			DomainID:   d.ID,
			DomainName: d.Name,
			Title:      "补知识缺口",
			NodeKey:    nodeKey,
			NodeTitle:  nodeTitle,
			Completed:  d.Completed,
			NodeTotal:  d.NodeTotal,
			Reason:     reason,
			Minutes:    15,
		}
		if nodeKey != "" {
			if active, err := s.store.FindActiveSession(userID, domainID, nodeKey); err == nil && active != nil {
				rec.SessionID = active.ID
				rec.CanResume = true
			}
		}
		return rec
	}
	return nil
}

func progressRecommendations(
	domains []storage.DomainSummary,
	used map[string]bool,
	activity map[string]time.Time,
	limit int,
) []ShortcutRecommendation {
	if limit <= 0 {
		return nil
	}
	type cand struct {
		d      storage.DomainSummary
		started bool
		at     time.Time
	}
	var list []cand
	for _, d := range domains {
		if used[d.ID] {
			continue
		}
		if d.NodeTotal <= 0 || d.Completed >= d.NodeTotal {
			continue
		}
		c := cand{d: d, started: d.Completed > 0}
		if at, ok := activity[d.ID]; ok {
			c.at = at
		} else {
			c.at = d.CreatedAt
		}
		list = append(list, c)
	}
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].started != list[j].started {
			return list[i].started
		}
		if !list[i].at.Equal(list[j].at) {
			return list[i].at.After(list[j].at)
		}
		return list[i].d.CreatedAt.After(list[j].d.CreatedAt)
	})
	out := make([]ShortcutRecommendation, 0, limit)
	for _, c := range list {
		if len(out) >= limit {
			break
		}
		out = append(out, ShortcutRecommendation{
			Source:     "progress",
			DomainID:   c.d.ID,
			DomainName: c.d.Name,
			Completed:  c.d.Completed,
			NodeTotal:  c.d.NodeTotal,
		})
	}
	return out
}
