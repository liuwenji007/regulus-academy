/*
 * @Date: 2026-06-06 21:12:21
 * @LastEditors: liuwenjie
 * @LastEditTime: 2026-06-07 21:38:22
 * @FilePath: /hermes-academy/web/src/lib/graph-lod.ts
 */
export type GraphLodLevel = 'galaxy' | 'constellation' | 'node'

/** 缩放阈值：scale 为 vis-network getScale() 返回值（越大越放大） */
export const LOD_GALAXY_MAX = 0.15
export const LOD_CONSTELLATION_MAX = 0.15

export function lodFromScale(scale: number, _multiDomain = true): GraphLodLevel {
  if (!Number.isFinite(scale) || scale <= 0) return 'galaxy'
  if (scale < LOD_GALAXY_MAX) return 'galaxy'
  if (scale < LOD_CONSTELLATION_MAX) return 'constellation'
  return 'node'
}

export function lodLabel(level: GraphLodLevel): string {
  switch (level) {
    case 'galaxy':
      return '全景'
    case 'constellation':
      return '星座'
    case 'node':
      return '节点'
  }
}
