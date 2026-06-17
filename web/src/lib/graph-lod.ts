/*
 * @Date: 2026-06-06 21:12:21
 * @LastEditors: liuwenjie
 * @LastEditTime: 2026-06-07 21:38:22
 * @FilePath: /hermes-academy/web/src/lib/graph-lod.ts
 */
import type { GraphCanvasTheme } from './graph-canvas-theme'

export type GraphLodLevel = 'galaxy' | 'constellation' | 'node'

/** 缩放阈值：scale 为 vis-network getScale() 返回值（越大越放大） */
export const SKY_LOD_GALAXY_MAX = 0.015
export const SKY_LOD_CONSTELLATION_MAX = 0.015
export const PAPER_LOD_GALAXY_MAX = 0.1
export const PAPER_LOD_CONSTELLATION_MAX = 0.15

/** 宣纸星座层子节点（主题）相对近景的尺寸比例 */
export const LOD_CONSTELLATION_TOPIC_SIZE_SCALE = 0.52

export function lodThresholds(theme: GraphCanvasTheme): { galaxyMax: number; constellationMax: number } {
  return theme === 'paper'
    ? { galaxyMax: PAPER_LOD_GALAXY_MAX, constellationMax: PAPER_LOD_CONSTELLATION_MAX }
    : { galaxyMax: SKY_LOD_GALAXY_MAX, constellationMax: SKY_LOD_CONSTELLATION_MAX }
}

export function topicSizeForLod(
  baseSize: number,
  level: GraphLodLevel,
  theme: GraphCanvasTheme = 'sky'
): number {
  if (theme === 'paper' && level === 'constellation') {
    return Math.max(6, Math.round(baseSize * LOD_CONSTELLATION_TOPIC_SIZE_SCALE))
  }
  return baseSize
}

export function topicLabelsVisible(level: GraphLodLevel, theme: GraphCanvasTheme = 'sky'): boolean {
  if (theme === 'paper') return level === 'node'
  return level === 'node'
}

export function lodFromScale(
  scale: number,
  multiDomain = true,
  theme: GraphCanvasTheme = 'sky'
): GraphLodLevel {
  if (!multiDomain) return 'node'
  if (!Number.isFinite(scale) || scale <= 0) return 'galaxy'
  const { galaxyMax, constellationMax } = lodThresholds(theme)
  if (scale < galaxyMax) return 'galaxy'
  if (scale < constellationMax) return 'constellation'
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
