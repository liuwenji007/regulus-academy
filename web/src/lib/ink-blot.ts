/** 宣纸主题 · 有机墨迹绘制（canvas 程序化，等价于 SVG feTurbulence 位移的轻量实现） */

export type InkDepth = 'dark' | 'mid' | 'light'

type Rgb = readonly [number, number, number]

const INK: Record<InkDepth, { core: Rgb; bleed: Rgb }> = {
  dark: { core: [20, 18, 16], bleed: [48, 44, 40] },
  mid: { core: [76, 70, 64], bleed: [108, 102, 94] },
  light: { core: [128, 120, 110], bleed: [158, 150, 140] },
}

/** 确定性极坐标噪声：同一 seed 始终相同轮廓 */
function radiusAtAngle(seed: number, angle: number, baseR: number, wobble: number): number {
  const s = seed * 0.00017
  const n =
    1 +
    wobble *
      (0.13 * Math.sin(angle * 2.3 + s * 19.7) +
        0.09 * Math.sin(angle * 4.1 - s * 27.3) +
        0.06 * Math.sin(angle * 6.7 + s * 33.1) +
        0.04 * Math.sin(angle * 9.2 - s * 43.9))
  return baseR * Math.max(0.58, n)
}

function buildContour(
  cx: number,
  cy: number,
  baseR: number,
  seed: number,
  segments: number,
  wobble: number
): Array<{ x: number; y: number }> {
  const pts: Array<{ x: number; y: number }> = []
  for (let i = 0; i < segments; i++) {
    const a = (i / segments) * Math.PI * 2
    const r = radiusAtAngle(seed, a, baseR, wobble)
    pts.push({ x: cx + Math.cos(a) * r, y: cy + Math.sin(a) * r })
  }
  return pts
}

/** Catmull-Rom → cubic bezier 闭合平滑路径 */
function traceSmoothClosedPath(ctx: CanvasRenderingContext2D, pts: Array<{ x: number; y: number }>): void {
  const n = pts.length
  if (n < 3) return
  const at = (i: number) => pts[(i + n) % n]!
  ctx.beginPath()
  ctx.moveTo(at(0).x, at(0).y)
  for (let i = 0; i < n; i++) {
    const p0 = at(i - 1)
    const p1 = at(i)
    const p2 = at(i + 1)
    const p3 = at(i + 2)
    ctx.bezierCurveTo(
      p1.x + (p2.x - p0.x) / 6,
      p1.y + (p2.y - p0.y) / 6,
      p2.x - (p3.x - p1.x) / 6,
      p2.y - (p3.y - p1.y) / 6,
      p2.x,
      p2.y
    )
  }
  ctx.closePath()
}

function fillOrganicLayer(
  ctx: CanvasRenderingContext2D,
  cx: number,
  cy: number,
  baseR: number,
  seed: number,
  segments: number,
  wobble: number,
  rgb: Rgb,
  layerAlpha: number,
  viewScale: number,
  softBleed: boolean
): void {
  if (layerAlpha <= 0 || baseR <= 0) return
  const pts = buildContour(cx, cy, baseR, seed, segments, wobble)
  const outer = baseR * 1.1
  const g = ctx.createRadialGradient(cx, cy, 0, cx, cy, outer)
  g.addColorStop(0, `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${layerAlpha.toFixed(3)})`)
  g.addColorStop(0.42, `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${(layerAlpha * 0.62).toFixed(3)})`)
  g.addColorStop(0.72, `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${(layerAlpha * 0.18).toFixed(3)})`)
  g.addColorStop(1, `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, 0)`)

  const pad = outer * 1.15
  ctx.save()
  traceSmoothClosedPath(ctx, pts)
  ctx.clip()
  ctx.fillStyle = g
  ctx.fillRect(cx - pad, cy - pad, pad * 2, pad * 2)

  if (softBleed && viewScale > 0.06) {
    const blurPx = Math.min(5, Math.max(0.8, baseR * viewScale * 0.07))
    ctx.filter = `blur(${blurPx / Math.max(viewScale, 0.001)}px)`
    ctx.globalAlpha = 0.55
    ctx.fillStyle = g
    ctx.fillRect(cx - pad, cy - pad, pad * 2, pad * 2)
  }
  ctx.restore()
}

/** 节点墨点：三层同心有机轮廓，放大后边缘自然渗化 */
export function drawOrganicInkBlot(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  seed: number,
  depth: InkDepth,
  alpha = 1,
  viewScale = 1
): void {
  if (alpha <= 0 || radius <= 0) return
  const c = INK[depth]
  const segs = 36

  fillOrganicLayer(ctx, x, y, radius * 1.28, seed + 7, segs, 0.38, c.bleed, 0.1 * alpha, viewScale, true)
  fillOrganicLayer(ctx, x, y, radius * 1.02, seed + 19, segs, 0.24, c.bleed, 0.32 * alpha, viewScale, false)
  fillOrganicLayer(ctx, x, y, radius * 0.82, seed, segs, 0.17, c.core, 0.9 * alpha, viewScale, false)
}

/** 大片墨晕：低对比、大半径、轮廓更松散 */
export function drawOrganicInkWash(
  ctx: CanvasRenderingContext2D,
  cx: number,
  cy: number,
  spread: number,
  seed: number,
  alpha: number,
  intensity = 1,
  viewScale = 1
): void {
  if (alpha <= 0 || spread <= 0) return
  const bleed: Rgb = [52, 48, 44]
  const washA = 0.14 * intensity * alpha
  fillOrganicLayer(ctx, cx, cy, spread * 1.05, seed, 28, 0.42, bleed, washA, viewScale, true)
  fillOrganicLayer(ctx, cx, cy, spread * 0.72, seed + 31, 24, 0.32, bleed, washA * 0.55, viewScale, false)
}

/** 极小墨点（领域旁散落） */
export function drawOrganicInkSpeckle(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  seed: number,
  alpha: number,
  viewScale = 1
): void {
  if (alpha <= 0 || radius <= 0) return
  fillOrganicLayer(ctx, x, y, radius, seed, 16, 0.28, [41, 37, 33], alpha, viewScale, viewScale > 0.1)
}
