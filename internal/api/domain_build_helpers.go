package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/domain"
	"github.com/regulus-academy/regulus-academy/internal/llm"
	"github.com/regulus-academy/regulus-academy/internal/storage"
)

func buildActionFromRequest(action string, force bool) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if action == "merge" || action == "separate" {
		return action
	}
	if force {
		return "separate"
	}
	return ""
}

func relatedBuildResponse(rel *domain.DomainRelation, intent domain.IntentResult) map[string]any {
	return map[string]any{
		"status":         "related",
		"relation":       rel.Kind,
		"message":        rel.Message,
		"existingDomain": rel.ExistingDomain,
		"intent":         intent,
	}
}

func (h *Handler) attachCourseLinks(result map[string]any, uid string, tree *storage.KnowledgeTree) map[string]any {
	if tree == nil || tree.DomainID == "" {
		return result
	}
	dom, err := h.store.GetDomain(uid, tree.DomainID)
	if err != nil {
		return result
	}
	all, err := h.store.ListDomainSummaries(uid)
	if err != nil {
		return result
	}
	current := storage.DomainSummary{
		ID: dom.ID, Name: dom.Name, Slug: dom.Slug, ParentSlug: dom.ParentSlug, Source: dom.Source,
	}
	var parentTree *storage.KnowledgeTree
	if dom.ParentSlug == "" {
		parentTree = tree
	}
	links := h.registry.ResolveCourseLinks(all, current, parentTree, h.domainDerivationResolver())
	if links.Parent == nil && len(links.Derivations) == 0 {
		return result
	}
	result["courseLinks"] = links
	return result
}

func (h *Handler) mergeExistingSubtopicDomain(
	ctx context.Context,
	uid, name string,
	intent domain.IntentResult,
	rel *domain.DomainRelation,
	profile string,
	llmClient llm.Provider,
) (map[string]any, error) {
	if !llmClient.Configured() {
		return nil, fmt.Errorf("未配置 LLM，无法合并生成根课程")
	}
	childID := rel.ExistingDomain.ID
	childSlug := strings.TrimSpace(rel.ExistingDomain.Slug)
	childTree, err := h.registry.ResolveTree(h.store, uid, childID)
	if err != nil {
		return nil, err
	}
	childNodeSpecs, err := h.registry.LoadDomainNodes(h.store, childID, childSlug)
	if err != nil {
		childNodeSpecs = map[string]domain.NodeSpec{}
	}

	rootSlug := intent.Slug
	if root := domain.TopicRoot(rootSlug); root != "" {
		rootSlug = root
	}
	rootIntent := intent
	rootIntent.Slug = rootSlug
	if dn := domain.RootDisplayName(rootSlug); dn != "" {
		rootIntent.DisplayName = dn
	} else if rootIntent.DisplayName == "" {
		rootIntent.DisplayName = name
	}
	rootIntent.ScopeBreadth = domain.ScopeBroad

	builder := domain.NewTreeBuilder(h.registry)
	tree, nodes, err := builder.Build(ctx, llmClient, rootIntent, name, profile)
	if err != nil {
		return nil, err
	}
	focusKeys := domain.MergeDomainIntoTree(tree, nodes, childTree, childNodeSpecs)
	nodesJSON, err := marshalNodesJSON(nodes)
	if err != nil {
		return nil, err
	}
	displayName := rootIntent.DisplayName
	domain.ReportBuildProgress(ctx, "saving", "正在保存课程…")
	_, tree, err = h.store.CreateDomainFromTree(uid, displayName, rootSlug, "", tree, nodesJSON, storage.DomainSourceGenerated, true, "")
	if err != nil {
		return nil, err
	}
	newID := tree.DomainID
	validKeys := nodeKeySet(focusKeys)
	oldTree := childTree
	migrateRes, err := h.store.MigrateProgressByNodeKey(uid, childID, newID, validKeys, oldTree, tree)
	if err != nil {
		_ = h.store.DeleteDomain(uid, newID)
		return nil, err
	}
	if _, err := h.store.MigrateSessionsByNodeKey(uid, childID, newID, rootSlug, validKeys, oldTree, tree); err != nil {
		_ = h.store.DeleteDomain(uid, newID)
		return nil, fmt.Errorf("进度已迁移但会话迁移失败: %w", err)
	}
	if err := h.store.DeleteDomain(uid, childID); err != nil {
		return nil, fmt.Errorf("合并完成但删除旧子课程失败: %w", err)
	}
	focusLabel := rel.ExistingDomain.Name
	msg := fmt.Sprintf("已合并「%s」到「%s」", focusLabel, displayName)
	if migrateRes.Migrated > 0 {
		msg += fmt.Sprintf("，已保留 %d 个已掌握节点", migrateRes.Migrated)
	}
	out := h.treeBuildResponse(intent, tree, focusKeys, focusLabel, true, msg, true, false)
	return h.attachCourseLinks(out, uid, tree), nil
}

func (h *Handler) domainDerivationResolver() domain.DerivationResolver {
	return func(domainID string) *domain.DerivationDef {
		raw, err := h.store.GetDomainDerivationJSON(domainID)
		if err != nil || strings.TrimSpace(raw) == "" {
			return nil
		}
		var d domain.DerivationDef
		if json.Unmarshal([]byte(raw), &d) != nil {
			return nil
		}
		if len(d.ParentAnchorKeywords) == 0 && strings.TrimSpace(d.JumpLabel) == "" {
			return nil
		}
		return &d
	}
}

func (h *Handler) resolveIntentParentRelation(
	ctx context.Context,
	uid, userInput string,
	intent domain.IntentResult,
	existing []storage.DomainSummary,
	llmClient llm.Provider,
) (domain.IntentResult, string) {
	if !domain.ShouldResolveGeneratedRelation(intent) {
		return intent, ""
	}
	if len(existing) == 0 {
		if list, err := h.store.ListDomainSummaries(uid); err == nil {
			existing = list
		}
	}
	candidates := domain.CandidateParentDomains(existing, intent)
	if len(candidates) == 0 {
		return intent, ""
	}
	domain.ReportBuildProgress(ctx, "relation", "正在分析课程关联…")
	rel, err := domain.ResolveGeneratedCourseRelation(ctx, llmClient, userInput, intent, candidates)
	if err != nil || rel == nil {
		return intent, ""
	}
	intent.ParentSlug = rel.ParentSlug
	intent.IsSubtopic = true
	return intent, domain.DerivationFromRelation(intent, rel)
}
