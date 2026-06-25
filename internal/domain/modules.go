package domain

import (
	"fmt"
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
