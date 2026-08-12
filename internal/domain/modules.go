package domain

import (
	"fmt"
	"sort"
	"strings"

	"github.com/regulus-academy/regulus-academy/internal/storage"
)

var forbiddenModuleLabels = map[string]struct{}{
	"入门": {}, "熟悉": {}, "精通": {},
	"entry": {}, "intermediate": {}, "advanced": {},
}

// reservedLayerModuleKeys 与 layers 层 key 相同，不能作为 module key（LLM 常误用 advanced）
var reservedLayerModuleKeys = map[string]struct{}{
	"entry": {}, "intermediate": {}, "advanced": {},
}

func sanitizeReservedModuleKey(key string, used map[string]struct{}) string {
	k := strings.TrimSpace(key)
	if _, reserved := reservedLayerModuleKeys[strings.ToLower(k)]; reserved {
		k = "mod_" + strings.ToLower(k)
	}
	base := k
	for suffix := 2; ; suffix++ {
		if _, exists := used[k]; !exists {
			used[k] = struct{}{}
			return k
		}
		k = fmt.Sprintf("%s_%d", base, suffix)
	}
}

const (
	moduleCountHardMin = 2
	moduleCountHardMax = 8
)

// moduleCountBounds 按领域广度给出建议模块数（仅用于 prompt / critique，不阻断建课）
func moduleCountBounds(scope string) (min, max int) {
	switch normalizeScope(scope) {
	case ScopeNarrow:
		return 2, 3
	case ScopeBroad:
		return 3, 6
	default:
		return 3, 5
	}
}

// repairModules 确定性修补 modules 一致性问题（幽灵引用、双归属、漏挂），避免因此整树重试 LLM。
// 不放宽模块数上下限、缺 label、进度层 label 等硬约束。
func repairModules(modules []TreeModuleDef, nodeKeys map[string]struct{}) (repaired []TreeModuleDef, notes []string) {
	if len(modules) == 0 {
		return modules, nil
	}

	type draft struct {
		def   TreeModuleDef
		nodes []string
	}
	drafts := make([]draft, 0, len(modules))
	assigned := map[string]string{}

	for i, m := range modules {
		cp := m
		cp.Key = strings.TrimSpace(m.Key)
		cp.Label = strings.TrimSpace(m.Label)
		cp.Goal = strings.TrimSpace(m.Goal)
		if cp.Order == 0 {
			cp.Order = i + 1
		}
		var kept []string
		for _, nk := range m.Nodes {
			nk = strings.TrimSpace(nk)
			if nk == "" {
				continue
			}
			if _, ok := nodeKeys[nk]; !ok {
				notes = append(notes, fmt.Sprintf("移除幽灵节点 %s（模块 %s）", nk, cp.Key))
				continue
			}
			if prev, dup := assigned[nk]; dup {
				notes = append(notes, fmt.Sprintf("节点 %s 重复归属，保留模块 %s，移除自 %s", nk, prev, cp.Key))
				continue
			}
			assigned[nk] = cp.Key
			kept = append(kept, nk)
		}
		cp.Nodes = kept
		if len(kept) == 0 {
			notes = append(notes, fmt.Sprintf("丢弃空模块 %s", cp.Key))
			continue
		}
		drafts = append(drafts, draft{def: cp, nodes: kept})
	}

	var orphans []string
	for nk := range nodeKeys {
		if _, ok := assigned[nk]; !ok {
			orphans = append(orphans, nk)
		}
	}
	sort.Strings(orphans)

	for _, nk := range orphans {
		if len(drafts) == 0 {
			break
		}
		best := 0
		for i := 1; i < len(drafts); i++ {
			if len(drafts[i].nodes) < len(drafts[best].nodes) {
				best = i
				continue
			}
			if len(drafts[i].nodes) == len(drafts[best].nodes) && drafts[i].def.Order < drafts[best].def.Order {
				best = i
			}
		}
		drafts[best].nodes = append(drafts[best].nodes, nk)
		drafts[best].def.Nodes = drafts[best].nodes
		assigned[nk] = drafts[best].def.Key
		notes = append(notes, fmt.Sprintf("漏挂节点 %s 归入模块 %s", nk, drafts[best].def.Key))
	}

	repaired = make([]TreeModuleDef, len(drafts))
	for i, d := range drafts {
		repaired[i] = d.def
		repaired[i].Nodes = append([]string(nil), d.nodes...)
	}
	return repaired, notes
}

func validateModules(modules []TreeModuleDef, nodeKeys map[string]struct{}) ([]storage.TreeModule, error) {
	if len(modules) < moduleCountHardMin || len(modules) > moduleCountHardMax {
		return nil, fmt.Errorf("主题模块数量应在 %d-%d 之间，得到 %d", moduleCountHardMin, moduleCountHardMax, len(modules))
	}

	assigned := map[string]string{}
	usedModuleKeys := map[string]struct{}{}
	out := make([]storage.TreeModule, 0, len(modules))
	for i, m := range modules {
		rawKey := strings.TrimSpace(m.Key)
		label := strings.TrimSpace(m.Label)
		if rawKey == "" {
			return nil, fmt.Errorf("模块 %d 缺少 key", i+1)
		}
		key := sanitizeReservedModuleKey(rawKey, usedModuleKeys)
		if label == "" {
			return nil, fmt.Errorf("模块 %s 缺少 label", key)
		}
		if _, forbidden := forbiddenModuleLabels[strings.ToLower(label)]; forbidden {
			return nil, fmt.Errorf("模块 %s 的 label 不能使用进度层名称「%s」，请用主题名（如基础、并发）", key, label)
		}
		if len(m.Nodes) == 0 {
			return nil, fmt.Errorf("模块 %s 至少包含 1 个节点", key)
		}

		order := m.Order
		if order == 0 {
			order = i + 1
		}
		mod := storage.TreeModule{
			Key: key, Label: label,
			Goal: strings.TrimSpace(m.Goal), Order: order,
			Nodes: make([]string, 0, len(m.Nodes)),
		}
		for _, nk := range m.Nodes {
			nk = strings.TrimSpace(nk)
			if nk == "" {
				continue
			}
			if _, ok := nodeKeys[nk]; !ok {
				return nil, fmt.Errorf("模块 %s 引用了不存在的节点 %s", key, nk)
			}
			if prev, dup := assigned[nk]; dup {
				return nil, fmt.Errorf("节点 %s 同时归属模块 %s 与 %s", nk, prev, key)
			}
			assigned[nk] = key
			mod.Nodes = append(mod.Nodes, nk)
		}
		if len(mod.Nodes) == 0 {
			return nil, fmt.Errorf("模块 %s 至少包含 1 个有效节点", key)
		}
		out = append(out, mod)
	}

	for nk := range nodeKeys {
		if _, ok := assigned[nk]; !ok {
			return nil, fmt.Errorf("节点 %s 未分配到任何主题模块", nk)
		}
	}
	return out, nil
}

func filterModulesForTree(tree *storage.KnowledgeTree, selectedSet map[string]struct{}) []storage.TreeModule {
	if tree == nil || len(tree.Modules) == 0 {
		return nil
	}
	out := make([]storage.TreeModule, 0, len(tree.Modules))
	for _, m := range tree.Modules {
		var kept []string
		for _, k := range m.Nodes {
			if _, ok := selectedSet[k]; ok {
				kept = append(kept, k)
			}
		}
		if len(kept) > 0 {
			copied := m
			copied.Nodes = kept
			out = append(out, copied)
		}
	}
	return out
}

func treeModulesFromFile(defs []TreeModuleDef) []storage.TreeModule {
	if len(defs) == 0 {
		return nil
	}
	out := make([]storage.TreeModule, len(defs))
	for i, d := range defs {
		order := d.Order
		if order == 0 {
			order = i + 1
		}
		out[i] = storage.TreeModule{
			Key: d.Key, Label: d.Label, Goal: d.Goal, Order: order,
			Nodes: append([]string(nil), d.Nodes...),
		}
	}
	return out
}
