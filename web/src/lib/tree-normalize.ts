import type { KnowledgeTree, TreeLayer } from './api'

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

  return {
    domainId: raw.domainId || domainId,
    domainName: raw.domainName || fallbackName || '课程',
    layers,
  }
}
