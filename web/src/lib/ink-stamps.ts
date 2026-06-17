/** 宣纸主题 · 节点图章 PNG（/public/graph/*.png，256×256） */

import { drawOrganicInkBlot, type InkDepth } from './ink-blot'

export type InkStampRole = 'domain' | 'module' | 'topic'

/** 学完：墨迹 */
export type PaperNodeVisual = 'lit' | 'empty' | 'progress'

const STAMP_PX = 256

/** 课程圆满主节点：dot_1～dot_3 */
export const DOMAIN_STAMP_IDS = [1, 2, 3] as const

/** 模块学完：dot_5～dot_8 */
export const MODULE_STAMP_IDS = [5, 6, 7, 8] as const

/** 子节点学完：dot_10～dot_16 */
export const TOPIC_STAMP_IDS = [10, 11, 12, 13, 14, 15, 16] as const

/** 未学完主节点：empty_1～empty_2 */
export const DOMAIN_EMPTY_IDS = [1, 2] as const

/** 未学完模块：empty_3 */
export const MODULE_EMPTY_IDS = [3] as const

/** 未学子节点：empty_4 */
export const TOPIC_EMPTY_IDS = [4] as const

/** 模块进行中：pre_1 */
export const MODULE_PROGRESS_IDS = [1] as const

/** 子节点进行中：pre_2 */
export const TOPIC_PROGRESS_IDS = [2] as const

const LIT_ASSET_KEYS: readonly string[] = [
  ...DOMAIN_STAMP_IDS.map((n) => `dot_${n}`),
  ...MODULE_STAMP_IDS.map((n) => `dot_${n}`),
  ...TOPIC_STAMP_IDS.map((n) => `dot_${n}`),
]

const ALL_ASSET_KEYS: readonly string[] = [
  ...LIT_ASSET_KEYS,
  ...DOMAIN_EMPTY_IDS.map((n) => `empty_${n}`),
  ...MODULE_EMPTY_IDS.map((n) => `empty_${n}`),
  ...TOPIC_EMPTY_IDS.map((n) => `empty_${n}`),
  ...MODULE_PROGRESS_IDS.map((n) => `pre_${n}`),
  ...TOPIC_PROGRESS_IDS.map((n) => `pre_${n}`),
]

const stampSources = new Map<string, ImageBitmap | HTMLImageElement>()
let preloadPromise: Promise<void> | null = null

function assetUrl(key: string): string {
  return `/graph/${key}.png`
}

async function loadOne(key: string): Promise<void> {
  const img = new Image()
  img.decoding = 'async'
  img.src = assetUrl(key)
  await img.decode()
  if (typeof createImageBitmap === 'function') {
    try {
      stampSources.set(key, await createImageBitmap(img))
      return
    } catch {
      /* fallback to Image */
    }
  }
  stampSources.set(key, img)
}

/** 预加载全部图章（宣纸图谱 mount 时调用） */
export function preloadInkStamps(): Promise<void> {
  if (!preloadPromise) {
    preloadPromise = Promise.all(ALL_ASSET_KEYS.map((key) => loadOne(key))).then(() => undefined)
  }
  return preloadPromise
}

export function inkStampsReady(): boolean {
  return ALL_ASSET_KEYS.every((key) => stampSources.has(key))
}

/** 学完墨迹：按层级在 dot 池中选取 */
export function pickInkStampId(seed: number, role: InkStampRole): number {
  if (role === 'domain') {
    return DOMAIN_STAMP_IDS[((seed % DOMAIN_STAMP_IDS.length) + DOMAIN_STAMP_IDS.length) % DOMAIN_STAMP_IDS.length]!
  }
  if (role === 'module') {
    return MODULE_STAMP_IDS[((seed % MODULE_STAMP_IDS.length) + MODULE_STAMP_IDS.length) % MODULE_STAMP_IDS.length]!
  }
  return TOPIC_STAMP_IDS[((seed % TOPIC_STAMP_IDS.length) + TOPIC_STAMP_IDS.length) % TOPIC_STAMP_IDS.length]!
}

/** 按节点状态选取图章资源名（不含 .png） */
export function pickPaperStampKey(seed: number, role: InkStampRole, visual: PaperNodeVisual): string {
  if (visual === 'lit') {
    return `dot_${pickInkStampId(seed, role)}`
  }
  if (visual === 'progress') {
    if (role === 'module') {
      const id = MODULE_PROGRESS_IDS[seed % MODULE_PROGRESS_IDS.length]!
      return `pre_${id}`
    }
    const id = TOPIC_PROGRESS_IDS[seed % TOPIC_PROGRESS_IDS.length]!
    return `pre_${id}`
  }
  if (role === 'domain') {
    const id = DOMAIN_EMPTY_IDS[((seed % DOMAIN_EMPTY_IDS.length) + DOMAIN_EMPTY_IDS.length) % DOMAIN_EMPTY_IDS.length]!
    return `empty_${id}`
  }
  if (role === 'module') {
    return `empty_${MODULE_EMPTY_IDS[0]}`
  }
  return `empty_${TOPIC_EMPTY_IDS[0]}`
}

const FALLBACK_DEPTH: Record<InkStampRole, InkDepth> = {
  domain: 'dark',
  module: 'mid',
  topic: 'light',
}

/** PNG 四周留白，放大绘制以贴合 vis 圆点视觉外径 */
const STAMP_DRAW_SCALE: Record<InkStampRole, number> = {
  domain: 1.45,
  module: 1.75,
  topic: 1.6,
}

/**
 * 未学 / 进行中图章与 dot 的墨迹外径对齐（p90 半径比）。
 * 模块空心圈 PNG 外圈本就偏大，不宜再按面积放大。
 * 主节点空心墨迹偏淡，保留适度面积补偿。
 */
const STAMP_VISUAL_ALIGN: Record<PaperNodeVisual, Partial<Record<InkStampRole, number>>> = {
  lit: {},
  empty: {
    domain: 1.45,
    module: 0.74,
    topic: 0.92,
  },
  progress: {
    module: 0.7,
    topic: 0.64,
  },
}

/** 画布绘制倍率：学完 / 未学 / 进行中 */
export function stampDrawScale(role: InkStampRole, visual: PaperNodeVisual): number {
  const align = STAMP_VISUAL_ALIGN[visual][role] ?? 1
  return STAMP_DRAW_SCALE[role] * align
}

/**
 * 绘制宣纸节点图章（学完 / 未学 / 进行中）。
 * @param radius 世界坐标下半径（与 paperInkDrawRadius 一致）
 */
export function drawPaperStamp(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  seed: number,
  role: InkStampRole,
  visual: PaperNodeVisual,
  alpha = 1,
  viewScale = 1
): void {
  if (alpha <= 0 || radius <= 0) return

  const key = pickPaperStampKey(seed, role, visual)
  const source = stampSources.get(key)
  if (!source) {
    if (visual === 'lit') {
      drawOrganicInkBlot(ctx, x, y, radius, seed, FALLBACK_DEPTH[role], alpha, viewScale)
    }
    return
  }

  const drawR = radius * stampDrawScale(role, visual)
  const size = drawR * 2
  const rot = visual === 'lit' ? (((seed >> 8) % 360) * Math.PI) / 180 : 0

  ctx.save()
  if (alpha < 1) ctx.globalAlpha = alpha
  ctx.translate(x, y)
  if (rot) ctx.rotate(rot)
  ctx.drawImage(source, -drawR, -drawR, size, size)
  ctx.restore()
}

/** @deprecated 使用 drawPaperStamp(..., 'lit') */
export function drawInkStamp(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  seed: number,
  role: InkStampRole,
  alpha = 1,
  viewScale = 1
): void {
  drawPaperStamp(ctx, x, y, radius, seed, role, 'lit', alpha, viewScale)
}

export { STAMP_PX, ALL_ASSET_KEYS as INK_STAMP_ASSETS }
