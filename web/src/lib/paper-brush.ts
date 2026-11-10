/** 宣纸描红 · 毛笔圆相：课程与模块用 /public/graph 下的笔刷圆圈，按描红配色染色 */

export type BrushRole = 'domain' | 'module'

const BRUSH_KEYS: Record<BrushRole, readonly string[]> = {
  domain: ['empty_1'],
  module: ['empty_3', 'empty_2'],
}

/** PNG 四周留白，按圆圈外径换算的绘制倍率 */
const PAD = 1.08

const sources = new Map<string, HTMLImageElement>()
const tinted = new Map<string, HTMLCanvasElement>()
let preload: Promise<void> | null = null

export function preloadBrushes(): Promise<void> {
  if (!preload) {
    const keys = [...new Set(Object.values(BRUSH_KEYS).flat())]
    preload = Promise.all(
      keys.map(async (key) => {
        const img = new Image()
        img.decoding = 'async'
        img.src = `/graph/${key}.png`
        await img.decode()
        sources.set(key, img)
      })
    ).then(() => undefined)
  }
  return preload
}

export function brushesReady(): boolean {
  return Object.values(BRUSH_KEYS).every((keys) => keys.every((key) => sources.has(key)))
}

function tintedBrush(key: string, rgb: string): HTMLCanvasElement | null {
  const cacheKey = `${key}|${rgb}`
  const hit = tinted.get(cacheKey)
  if (hit) return hit
  const img = sources.get(key)
  if (!img) return null
  const c = document.createElement('canvas')
  c.width = img.naturalWidth
  c.height = img.naturalHeight
  const g = c.getContext('2d')!
  g.drawImage(img, 0, 0)
  g.globalCompositeOperation = 'source-in'
  g.fillStyle = `rgb(${rgb})`
  g.fillRect(0, 0, c.width, c.height)
  tinted.set(cacheKey, c)
  return c
}

/**
 * 画一枚毛笔圆圈；sweep < 1 时只露出从 12 点钟方向顺时针的一段，像一笔写到一半。
 * @returns 是否已画出（资源未加载时返回 false，调用方回退矢量）
 */
export function drawBrushRing(
  ctx: CanvasRenderingContext2D,
  opts: { x: number; y: number; radius: number; role: BrushRole; seed: number; rgb: string; alpha: number; sweep?: number }
): boolean {
  const keys = BRUSH_KEYS[opts.role]
  const key = keys[Math.abs(opts.seed) % keys.length]!
  const tex = tintedBrush(key, opts.rgb)
  if (!tex) return false
  const sweep = Math.max(0, Math.min(1, opts.sweep ?? 1))
  if (sweep <= 0) return true
  const half = opts.radius * PAD
  ctx.save()
  ctx.translate(opts.x, opts.y)
  ctx.rotate(((Math.abs(opts.seed) % 5) - 2) * 0.05)
  if (sweep < 1) {
    ctx.beginPath()
    ctx.moveTo(0, 0)
    ctx.arc(0, 0, half * 1.05, -Math.PI / 2, -Math.PI / 2 + sweep * Math.PI * 2)
    ctx.closePath()
    ctx.clip()
  }
  ctx.globalAlpha = opts.alpha
  ctx.drawImage(tex, -half, -half, half * 2, half * 2)
  ctx.restore()
  return true
}
