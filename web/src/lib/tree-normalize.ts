import type { KnowledgeTree, TreeLayer, TreeModule } from './api'

/** 兼容旧数据 / 不完整 JSON，避免渲染阶段 TypeError */
export function normalizeKnowledgeTree(
  raw: KnowledgeTree,
  domainId: string,
  fallbackName?: string
): KnowledgeTree {
  const layers: TreeLayer[] = Array.isArray(raw.layers)
    ? raw.layers.map((layer) => ({
        key: layer?.key ?? '',
        label: layer?.label ?? '',
        time: layer?.time ?? '',
        goal: layer?.goal ?? '',
        nodes: Array.isArray(layer?.nodes)
          ? layer.nodes
              .filter((n) => n && typeof n.key === 'string')
              .map((n) => ({
                key: n.key,
                title: n.title ?? n.key,
              }))
          : [],
      }))
    : []

  const modules: TreeModule[] | undefined = Array.isArray(raw.modules)
    ? raw.modules
        .filter((m) => m && typeof m.key === 'string' && Array.isArray(m.nodes))
        .map((m, i) => ({
          key: m.key,
          label: m.label ?? m.key,
          goal: m.goal,
          order: m.order ?? i + 1,
          nodes: m.nodes.filter((k) => typeof k === 'string' && k.length > 0),
        }))
        .filter((m) => m.nodes.length > 0)
    : undefined

  return {
    domainId: raw.domainId || domainId,
    domainName: raw.domainName || fallbackName || '课程',
    layers,
    modules: modules?.length ? modules : undefined,
  }
}

export interface ResolvedGraphModule extends TreeModule {
  /** 是否为按 progress layer 降级的临时模块 */
  derivedFromLayers?: boolean
}

/** 图谱用模块列表；无 modules 时按入门/熟悉/精通层降级 */
export function resolveGraphModules(tree: KnowledgeTree): {
  modules: ResolvedGraphModule[]
  isDerived: boolean
} {
  if (tree.modules?.length) {
    return {
      modules: tree.modules.map((m) => ({ ...m })),
      isDerived: false,
    }
  }

  const derived: ResolvedGraphModule[] = []
  for (const layer of tree.layers) {
    if (!layer.nodes.length) continue
    derived.push({
      key: layer.key || `layer-${derived.length}`,
      label: layer.label || layer.key || '模块',
      goal: layer.goal,
      order: derived.length + 1,
      nodes: layer.nodes.map((n) => n.key),
      derivedFromLayers: true,
    })
  }
  return { modules: derived, isDerived: derived.length > 0 }
}

/** 节点 key → 所在 layer key */
export function nodeLayerKeyMap(tree: KnowledgeTree): Map<string, string> {
  const map = new Map<string, string>()
  for (const layer of tree.layers) {
    for (const node of layer.nodes) {
      map.set(node.key, layer.key)
    }
  }
  return map
}

/** 节点 key → 标题 */
export function nodeTitleMap(tree: KnowledgeTree): Map<string, string> {
  const map = new Map<string, string>()
  for (const layer of tree.layers) {
    for (const node of layer.nodes) {
      map.set(node.key, node.title)
    }
  }
  return map
}
