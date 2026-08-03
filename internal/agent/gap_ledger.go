package agent

import (
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

// GapLedger 认知缺口账本写入（旁路 / 错题 / 跳级共用）
type GapLedger struct {
	store    *storage.Store
	registry *domain.Registry
}

// NewGapLedger 创建账本
func NewGapLedger(store *storage.Store) *GapLedger {
	return &GapLedger{store: store, registry: domain.NewRegistry()}
}

// RecordConcepts 批量写入缺口；写入时尽量填 matched_*，供学完节点时精确关闭。
// nodeKey 表示「发现位置」；matched_* 表示「应去补的节点」。
func (g *GapLedger) RecordConcepts(
	userID, domainID, nodeKey, source, reason string,
	concepts []string,
) {
	if g == nil || g.store == nil {
		return
	}
	domainID = strings.TrimSpace(domainID)
	nodeKey = strings.TrimSpace(nodeKey)
	for _, c := range NormalizeConceptList(concepts) {
		matchedDomain, matchedNode := g.resolveMatch(userID, domainID, nodeKey, source, c)
		_, _ = g.store.UpsertKnowledgeGap(&storage.KnowledgeGap{
			UserID:          userID,
			DomainID:        domainID,
			NodeKey:         nodeKey,
			Concept:         c,
			Source:          source,
			Reason:          reason,
			MatchedDomainID: matchedDomain,
			MatchedNodeKey:  matchedNode,
		})
	}
}

func (g *GapLedger) resolveMatch(userID, domainID, nodeKey, source, concept string) (matchedDomain, matchedNode string) {
	if domainID == "" {
		return "", ""
	}
	if key, _ := MatchConceptToNode(g.store, g.registry, userID, domainID, concept); key != "" {
		return domainID, key
	}
	// 错题 / 跳级缺口默认挂在当前节点（概念属于本课本节）
	switch source {
	case storage.GapSourceMistake, storage.GapSourceCoachGap:
		if nodeKey != "" {
			return domainID, nodeKey
		}
	}
	// 旁路前置：无匹配则留空，学完发现节点时不会误关
	return "", ""
}

// RecordFromTermCard 从术语卡 prerequisites 写入（不把划词术语本身记为缺口）
func (g *GapLedger) RecordFromTermCard(
	userID, domainID, nodeKey string,
	card *TermCardPayload,
) {
	if card == nil {
		return
	}
	reason := fmt.Sprintf("查询「%s」时推断缺少前置", strings.TrimSpace(card.Term))
	g.RecordConcepts(userID, domainID, nodeKey, storage.GapSourceAsideLookup, reason, card.Prerequisites)
}

// MatchConceptToNode 在课程树中用 core_concepts 文本匹配缺口概念
func MatchConceptToNode(
	store *storage.Store,
	registry *domain.Registry,
	userID, domainID, concept string,
) (nodeKey, nodeTitle string) {
	if store == nil || concept == "" {
		return "", ""
	}
	tree, err := store.GetDomainTree(userID, domainID)
	if err != nil || tree == nil {
		return "", ""
	}
	dom, _ := store.GetDomain(userID, domainID)
	slug := ""
	if dom != nil {
		slug = dom.Slug
	}
	if registry == nil {
		registry = domain.NewRegistry()
	}
	bestKey := ""
	bestTitle := ""
	bestScore := 0
	for _, layer := range tree.Layers {
		for _, n := range layer.Nodes {
			node, err := registry.GetNode(store, domainID, slug, n.Key)
			if err != nil || node == nil {
				// 退化为标题匹配
				if ConceptsLooselyMatch(concept, n.Title) {
					return n.Key, n.Title
				}
				continue
			}
			score := 0
			for _, cc := range node.CoreConcepts {
				if ConceptsLooselyMatch(concept, cc) {
					score += 3
				}
			}
			if ConceptsLooselyMatch(concept, node.Node) || ConceptsLooselyMatch(concept, n.Title) {
				score += 2
			}
			if score > bestScore {
				bestScore = score
				bestKey = n.Key
				bestTitle = n.Title
				if bestTitle == "" {
					bestTitle = node.Node
				}
			}
		}
	}
	if bestScore > 0 {
		return bestKey, bestTitle
	}
	return "", ""
}
