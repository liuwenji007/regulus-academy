import type { UserProgress } from './api'

export type GraphModuleLike = { key: string; nodes: string[] }

function normalizeCompleted(status: string | undefined): boolean {
  return status === 'completed'
}

/** 模块内已完成节点占比 0..1 */
export function moduleCompletionRatio(
  mod: GraphModuleLike,
  progressMap: Map<string, UserProgress>,
  validKeys: Set<string>
): number {
  let total = 0
  let done = 0
  for (const nodeKey of mod.nodes) {
    if (!validKeys.has(nodeKey)) continue
    total++
    if (normalizeCompleted(progressMap.get(nodeKey)?.status)) done++
  }
  if (total === 0) return 0
  return done / total
}

/** 领域内已完成节点占比 0..1 */
export function domainCompletionRatio(
  modules: GraphModuleLike[],
  progressMap: Map<string, UserProgress>,
  validKeys: Set<string>
): number {
  let total = 0
  let done = 0
  for (const mod of modules) {
    for (const nodeKey of mod.nodes) {
      if (!validKeys.has(nodeKey)) continue
      total++
      if (normalizeCompleted(progressMap.get(nodeKey)?.status)) done++
    }
  }
  if (total === 0) return 0
  return done / total
}

/** 路径边不透明度：随模块完成率从 0.2 升到 0.85 */
export function pathEdgeOpacity(ratio: number): number {
  const r = Math.max(0, Math.min(1, ratio))
  return 0.2 + r * 0.65
}

/** pending 节点不透明度（压低以突出已点亮节点） */
export const PENDING_NODE_OPACITY = 0.32

/** 模块内推荐路径：前序节点已学完时边更亮 */
export function pathSegmentOpacity(prevCompleted: boolean, modRatio: number): number {
  const r = Math.max(0, Math.min(1, modRatio))
  if (prevCompleted) return 0.4 + r * 0.5
  return 0.1 + r * 0.18
}
