import type { GraphLodLevel } from './graph-lod'
import { hashString } from './constellation-layout'
import { brushesReady, drawBrushRing } from './paper-brush'
import { GALAXY_ARMS, GALAXY_ARM_WIDTH, GALAXY_CORE, galaxyArmPoint } from './constellation-packing'

export type StarVisualState = 'root' | 'module' | 'done' | 'progress' | 'available' | 'recommended' | 'locked'

export interface DrawableStar {
  id: string
  role: 'domain' | 'module' | 'topic'
  label: string
  labelAngle: number
  state: StarVisualState
  domainId: string
  mastery?: number
}

export interface DrawableSegment {
  from: string
  to: string
  kind: 'spine' | 'limb' | 'parent' | 'prereq'
  tone: 'gold' | 'mid' | 'dim' | 'flow' | 'parent' | 'prereq'
}

export interface ConstellationName {
  text: string
  anchorId: string
  complete: boolean
  /** 星座名相对主星的世界坐标偏移（星座外接框底边中点） */
  dx: number
  dy: number
  /** 星座规模，用于名字避让时的优先级 */
  weight: number
}

type Pos = { x: number; y: number }
type Rect = { x1: number; y1: number; x2: number; y2: number }

const FONT_FACE = 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif'
const SERIF_FACE = '"Songti SC", "STSong", "Noto Serif SC", "Source Han Serif SC", "SimSun", serif'

/** 宣纸描红配色：朱色底稿，学会的部分用墨描出 */
const INK = '30, 26, 22'
const CINNABAR = '176, 50, 32'
const GUIDE = '196, 92, 70'
const PAPER = '#f7f1e3'

function fontFace(theme: 'sky' | 'paper'): string {
  return theme === 'paper' ? SERIF_FACE : FONT_FACE
}

/** 星座层级以上节点保持屏幕尺寸；继续放大时节点跟着长大，最多约 3 倍 */
function zoomBoost(scale: number): number {
  const ref = 0.55
  return scale <= ref ? 1 : Math.min(3, (scale / ref) ** 0.8)
}

/** 屏幕像素换算：ps 用于星点尺寸，pl 用于线宽，pt 用于文字 */
function units(scale: number): { px: number; ps: number; pl: number; pt: number } {
  const px = 1 / Math.max(scale, 0.0001)
  const boost = zoomBoost(scale)
  return { px, ps: px * boost, pl: px * Math.sqrt(boost), pt: px * boost ** 0.4 }
}

export function drawConstellationSegments(opts: {
  ctx: CanvasRenderingContext2D
  positions: Record<string, Pos | undefined>
  stars: DrawableStar[]
  segments: DrawableSegment[]
  theme: 'sky' | 'paper'
  lod: GraphLodLevel
  scale: number
  phase: number
  burstIds?: Set<string>
  reducedMotion?: boolean
}): void {
  const { ctx, positions, stars, segments, theme, lod, scale, phase } = opts
  const { ps, pl } = units(scale)
  const byId = new Map(stars.map((star) => [star.id, star]))
  for (const seg of segments) {
    if (!segmentVisible(seg, byId, lod)) continue
    const a = positions[seg.from]
    const b = positions[seg.to]
    const starA = byId.get(seg.from)
    const starB = byId.get(seg.to)
    if (!a || !b || !starA || !starB) continue
    const len = Math.hypot(b.x - a.x, b.y - a.y)
    const gapA = Math.min(markRadius(starA, lod, theme) * ps + 4 * pl, len * 0.3)
    const gapB = Math.min(markRadius(starB, lod, theme) * ps + 4 * pl, len * 0.3)
    const line = trimLine(a, b, gapA, gapB)
    if (!line) continue
    if (theme === 'paper' && seg.tone === 'flow') {
      drawInkWrittenLine(ctx, line, pl, opts.reducedMotion ? 1 : writeProgress(phase))
      continue
    }
    if (theme === 'paper' && (seg.tone === 'gold' || seg.tone === 'mid' || seg.tone === 'dim')) {
      drawTracedLine(ctx, line, pl, seg.tone === 'gold' ? 1 : seg.tone === 'mid' ? (starA.state === 'done' ? 0.45 : -0.45) : 0)
      continue
    }
    const style = segmentStyle(seg.tone, theme, lod)
    ctx.save()
    ctx.beginPath()
    ctx.moveTo(line.x1, line.y1)
    ctx.lineTo(line.x2, line.y2)
    ctx.strokeStyle = style.color
    ctx.lineWidth = style.width * pl
    ctx.lineCap = 'round'
    if (style.dash) {
      ctx.setLineDash(style.dash.map((n) => n * pl))
      if (seg.tone === 'flow') ctx.lineDashOffset = -phase * 6 * pl
    }
    ctx.stroke()
    ctx.restore()
  }
}

export function drawConstellationStars(opts: {
  ctx: CanvasRenderingContext2D
  positions: Record<string, Pos | undefined>
  stars: DrawableStar[]
  names: ConstellationName[]
  theme: 'sky' | 'paper'
  lod: GraphLodLevel
  scale: number
  phase: number
  hoveredId: string | null
  reducedMotion: boolean
  burstIds?: Set<string>
  /** 银河远景强度 0..1：越大课程名越淡 */
  galaxyFade?: number
}): void {
  const { ctx, positions, stars, names, theme, lod, scale, phase, hoveredId, reducedMotion, burstIds } = opts
  const { px, ps, pl, pt } = units(scale)
  const taken: Rect[] = []
  if (lod === 'galaxy') {
    drawStarClusters(ctx, positions, stars, names, theme, px, taken, opts.galaxyFade ?? 0, hoveredId)
    return
  }
  for (const star of stars) {
    const pos = positions[star.id]
    if (!pos) continue
    const burst = burstIds?.has(star.id) ?? false
    if (theme === 'paper') drawPaperStar(ctx, star, pos, lod, ps, pl, phase, reducedMotion, burst)
    else drawSkyStar(ctx, star, pos, lod, ps, phase, reducedMotion, burst)
  }

  const label = (star: DrawableStar, pos: Pos, force: boolean) =>
    drawStarLabel(ctx, star, pos, theme, { ps, pt }, taken, force)
  const hovered = hoveredId ? stars.find((star) => star.id === hoveredId) : undefined
  if (hovered) {
    const pos = positions[hovered.id]
    if (pos) label(hovered, pos, true)
  }
  if (lod === 'node') {
    for (const star of stars.slice().sort((a, b) => labelPriority(b) - labelPriority(a))) {
      if (star.id === hoveredId || star.state === 'locked') continue
      const pos = positions[star.id]
      if (pos) label(star, pos, false)
    }
  } else if (lod === 'constellation') {
    for (const star of stars) {
      if (star.state !== 'recommended' || star.id === hoveredId) continue
      const pos = positions[star.id]
      if (pos) label(star, pos, false)
    }
  }
  const nameAlpha = lod === 'node' ? 0.45 : 1
  for (const name of names.slice().sort((a, b) => Number(b.complete) - Number(a.complete) || b.weight - a.weight)) {
    const pos = positions[name.anchorId]
    if (!pos) continue
    drawConstellationName(ctx, name, { x: pos.x + name.dx, y: pos.y + name.dy }, theme, pt, taken, nameAlpha)
  }
}

function segmentVisible(seg: DrawableSegment, _stars: Map<string, DrawableStar>, lod: GraphLodLevel): boolean {
  if (seg.kind === 'prereq') return true
  return lod !== 'galaxy'
}

/** 全景：每门课画成一团星云加细碎星点，不画单星光晕和连线（远看时光晕会叠成实心圆） */
function drawStarClusters(
  ctx: CanvasRenderingContext2D,
  positions: Record<string, Pos | undefined>,
  stars: DrawableStar[],
  names: ConstellationName[],
  theme: 'sky' | 'paper',
  px: number,
  taken: Rect[],
  galaxyFade: number,
  hoveredId: string | null
): void {
  const byDomain = new Map<string, Array<{ star: DrawableStar; pos: Pos }>>()
  for (const star of stars) {
    const pos = positions[star.id]
    if (!pos) continue
    const list = byDomain.get(star.domainId) ?? []
    list.push({ star, pos })
    byDomain.set(star.domainId, list)
  }
  const clusters: Array<{ domainId: string; cx: number; cy: number; r: number; lit: number; complete: boolean; nameY: number }> = []
  for (const [domainId, members] of byDomain) {
    const cx = members.reduce((sum, m) => sum + m.pos.x, 0) / members.length
    const cy = members.reduce((sum, m) => sum + m.pos.y, 0) / members.length
    const spread = Math.max(...members.map((m) => Math.hypot(m.pos.x - cx, m.pos.y - cy)))
    const lit = members.filter((m) => m.star.state === 'done').length / members.length
    const complete = members.find((m) => m.star.role === 'domain')?.star.state === 'done'
    const r = Math.max(spread * 0.75, 10 * px)
    const nameY = cy + r * 0.9
    clusters.push({ domainId, cx, cy, r, lit, complete, nameY })

    if (theme === 'paper') {
      for (const { star, pos } of members) {
        const done = star.state === 'done'
        const dotR = (star.role === 'domain' ? 1.8 : star.role === 'module' ? 1.3 : 1) * (done ? 1.15 : 1) * px
        ctx.beginPath()
        ctx.arc(pos.x, pos.y, dotR, 0, Math.PI * 2)
        ctx.fillStyle = done
          ? `rgba(${INK}, 0.9)`
          : star.state === 'progress' || star.state === 'recommended'
            ? `rgba(${CINNABAR}, 0.9)`
            : `rgba(${GUIDE}, 0.4)`
        ctx.fill()
      }
      continue
    }
    const glowR = r * 1.5
    const g = ctx.createRadialGradient(cx, cy, 0, cx, cy, glowR)
    const tint = complete ? '246, 210, 120' : lit > 0 ? '200, 205, 235' : '150, 175, 235'
    const base = (0.16 + lit * 0.12) * (1 - galaxyFade * 0.5)
    for (const [stop, k] of [
      [0, 1],
      [0.2, 0.7],
      [0.45, 0.32],
      [0.7, 0.1],
      [1, 0],
    ] as const) {
      g.addColorStop(stop, `rgba(${tint}, ${base * k})`)
    }
    ctx.fillStyle = g
    ctx.beginPath()
    ctx.arc(cx, cy, glowR, 0, Math.PI * 2)
    ctx.fill()

    for (const { star, pos } of members) {
      const dotR = (star.role === 'domain' ? 1.6 : star.role === 'module' ? 1.1 : 0.75) * px
      ctx.beginPath()
      ctx.arc(pos.x, pos.y, dotR, 0, Math.PI * 2)
      if (star.state === 'done') {
        ctx.fillStyle = 'rgba(255, 236, 180, 0.95)'
      } else if (star.state === 'progress') {
        ctx.fillStyle = 'rgba(255, 200, 160, 0.9)'
      } else if (star.state === 'locked') {
        ctx.fillStyle = 'rgba(170, 185, 220, 0.35)'
      } else {
        ctx.fillStyle = 'rgba(225, 233, 255, 0.8)'
      }
      ctx.fill()
    }
  }

  const nameAlpha = 0.85 * (1 - galaxyFade) ** 2
  const nameOf = new Map(names.map((n) => [n.anchorId, n]))
  for (const c of clusters.slice().sort((a, b) => Number(b.complete) - Number(a.complete) || b.r - a.r)) {
    const anchorId = `domain:${c.domainId}`
    const name = nameOf.get(anchorId)
    if (!name) continue
    const hovered = hoveredId !== null && stars.some((s) => s.id === hoveredId && s.domainId === c.domainId)
    const alpha = hovered ? 0.95 : nameAlpha
    if (alpha < 0.06) continue
    drawConstellationName(ctx, name, { x: c.cx, y: c.nameY }, theme, px, taken, alpha)
  }
}

export interface GalaxyDust {
  extent: number
  dots: Array<{ x: number; y: number; size: number; a: number; warm: boolean }>
  haze: Array<{ x: number; y: number; r: number; a: number }>
}

function seeded(seed: number): () => number {
  let s = seed | 0
  return () => {
    s = (s + 0x6d2b79f5) | 0
    let t = Math.imul(s ^ (s >>> 15), 1 | s)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function gauss(rnd: () => number): number {
  return Math.sqrt(-2 * Math.log(Math.max(rnd(), 1e-9))) * Math.cos(2 * Math.PI * rnd())
}

/** 按课程所在旋臂生成银河尘埃；extent 为最外层课程到中心的距离 */
export function buildGalaxyDust(extent: number): GalaxyDust {
  const rnd = seeded(0x9a1a)
  const outer = Math.max(extent, GALAXY_CORE * 3)
  const dots: GalaxyDust['dots'] = []
  const count = Math.round(Math.min(4200, 1400 + outer * 0.45))
  for (let i = 0; i < count; i++) {
    const kind = rnd()
    let x: number
    let y: number
    let warm = false
    if (kind < 0.16) {
      const rr = Math.abs(gauss(rnd)) * GALAXY_CORE * 0.7
      const a = rnd() * Math.PI * 2
      x = rr * Math.cos(a)
      y = rr * Math.sin(a) * 0.85
      warm = rnd() < 0.7
    } else if (kind < 0.28) {
      const rr = Math.sqrt(rnd()) * outer * 1.1
      const a = rnd() * Math.PI * 2
      x = rr * Math.cos(a)
      y = rr * Math.sin(a)
    } else {
      const arm = Math.floor(rnd() * GALAXY_ARMS)
      const s = GALAXY_CORE * 0.5 + Math.pow(rnd(), 0.85) * (outer * 1.08 - GALAXY_CORE * 0.5)
      const p = galaxyArmPoint(arm, s, gauss(rnd) * (GALAXY_ARM_WIDTH * 0.7 + s * 0.06))
      x = p.x
      y = p.y
      warm = rnd() < 0.12
    }
    const bright = rnd() < 0.06
    dots.push({ x, y, size: bright ? 1.4 + rnd() * 0.6 : 0.6 + rnd() * 0.6, a: bright ? 0.7 : 0.18 + rnd() * 0.35, warm })
  }
  const haze: GalaxyDust['haze'] = []
  for (let arm = 0; arm < GALAXY_ARMS; arm++) {
    for (let s = GALAXY_CORE * 0.7; s < outer * 1.05; s += 140) {
      const p = galaxyArmPoint(arm, s, gauss(rnd) * GALAXY_ARM_WIDTH * 0.4)
      haze.push({ x: p.x, y: p.y, r: 260 + s * 0.09, a: 0.05 + rnd() * 0.03 })
    }
  }
  return { extent: outer, dots, haze }
}

/** 远景银河：核球、旋臂薄雾与尘埃星点；fade 为 0 时不画 */
export function drawGalaxy(ctx: CanvasRenderingContext2D, dust: GalaxyDust, scale: number, fade: number): void {
  if (fade <= 0.01) return
  const px = 1 / Math.max(scale, 0.0001)
  ctx.save()
  ctx.globalCompositeOperation = 'lighter'
  for (const h of dust.haze) {
    const g = ctx.createRadialGradient(h.x, h.y, 0, h.x, h.y, h.r)
    g.addColorStop(0, `rgba(120, 150, 230, ${h.a * fade})`)
    g.addColorStop(1, 'rgba(120, 150, 230, 0)')
    ctx.fillStyle = g
    ctx.fillRect(h.x - h.r, h.y - h.r, h.r * 2, h.r * 2)
  }
  const coreR = GALAXY_CORE * 1.9
  const core = ctx.createRadialGradient(0, 0, 0, 0, 0, coreR)
  core.addColorStop(0, `rgba(255, 236, 200, ${0.42 * fade})`)
  core.addColorStop(0.18, `rgba(250, 215, 170, ${0.2 * fade})`)
  core.addColorStop(0.5, `rgba(180, 170, 220, ${0.06 * fade})`)
  core.addColorStop(1, 'rgba(120, 130, 200, 0)')
  ctx.fillStyle = core
  ctx.fillRect(-coreR, -coreR, coreR * 2, coreR * 2)
  ctx.globalCompositeOperation = 'source-over'
  for (const d of dust.dots) {
    const s = d.size * px
    ctx.globalAlpha = d.a * fade
    ctx.fillStyle = d.warm ? '#ffe2bf' : '#d4defa'
    ctx.fillRect(d.x - s / 2, d.y - s / 2, s, s)
  }
  ctx.restore()
}

/** 描红格：外框双圈加内圈，像字帖的米字格底纹，只在远景隐约可见 */
export function drawStarChartPlate(ctx: CanvasRenderingContext2D, extent: number, scale: number, fade: number): void {
  if (fade <= 0.01) return
  const px = 1 / Math.max(scale, 0.0001)
  const outer = extent * 1.12
  ctx.save()
  ctx.strokeStyle = `rgba(${GUIDE}, ${0.22 * fade})`
  ctx.lineWidth = 1.2 * px
  ctx.beginPath()
  ctx.arc(0, 0, outer, 0, Math.PI * 2)
  ctx.stroke()
  ctx.lineWidth = 0.7 * px
  ctx.beginPath()
  ctx.arc(0, 0, outer + 6 * px, 0, Math.PI * 2)
  ctx.stroke()
  ctx.strokeStyle = `rgba(${GUIDE}, ${0.1 * fade})`
  ctx.setLineDash([5 * px, 7 * px])
  ctx.beginPath()
  ctx.arc(0, 0, outer * 0.55, 0, Math.PI * 2)
  for (let i = 0; i < 4; i++) {
    const a = (i / 4) * Math.PI + Math.PI / 4
    ctx.moveTo(Math.cos(a) * outer, Math.sin(a) * outer)
    ctx.lineTo(-Math.cos(a) * outer, -Math.sin(a) * outer)
  }
  ctx.stroke()
  ctx.restore()
}

const BACKDROP_TILE = 900
let backdropStars: Array<{ x: number; y: number; r: number; a: number; warm: boolean }> | null = null

function backdrop(): NonNullable<typeof backdropStars> {
  if (backdropStars) return backdropStars
  let seed = 0x5eed1234
  const rnd = () => {
    seed = (seed + 0x6d2b79f5) | 0
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
  backdropStars = Array.from({ length: 260 }, () => {
    const bright = rnd() < 0.08
    return {
      x: rnd() * BACKDROP_TILE,
      y: rnd() * BACKDROP_TILE,
      r: bright ? 0.9 + rnd() * 0.6 : 0.35 + rnd() * 0.45,
      a: bright ? 0.55 + rnd() * 0.3 : 0.12 + rnd() * 0.28,
      warm: rnd() < 0.15,
    }
  })
  return backdropStars
}

/** 星空背景：屏幕空间平铺的远景星点，随视图中心做很弱的视差移动 */
export function drawSkyBackdrop(ctx: CanvasRenderingContext2D, view: Pos, scale: number): void {
  const canvas = ctx.canvas
  const dpr = canvas.clientWidth > 0 ? canvas.width / canvas.clientWidth : 1
  const w = canvas.clientWidth
  const h = canvas.clientHeight
  const offX = -view.x * scale * 0.04
  const offY = -view.y * scale * 0.04
  ctx.save()
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  const vignette = ctx.createRadialGradient(w * 0.5, h * 0.45, 0, w * 0.5, h * 0.45, Math.max(w, h) * 0.75)
  vignette.addColorStop(0, 'rgba(60, 80, 140, 0.10)')
  vignette.addColorStop(0.6, 'rgba(30, 40, 80, 0.04)')
  vignette.addColorStop(1, 'rgba(0, 0, 0, 0)')
  ctx.fillStyle = vignette
  ctx.fillRect(0, 0, w, h)
  const stars = backdrop()
  const mod = (v: number) => ((v % BACKDROP_TILE) + BACKDROP_TILE) % BACKDROP_TILE
  for (let tx = 0; tx < w + BACKDROP_TILE; tx += BACKDROP_TILE) {
    for (let ty = 0; ty < h + BACKDROP_TILE; ty += BACKDROP_TILE) {
      for (const star of stars) {
        const x = tx + mod(star.x + offX) - BACKDROP_TILE / 2
        const y = ty + mod(star.y + offY) - BACKDROP_TILE / 2
        if (x < -2 || y < -2 || x > w + 2 || y > h + 2) continue
        ctx.beginPath()
        ctx.arc(x, y, star.r, 0, Math.PI * 2)
        ctx.fillStyle = star.warm ? `rgba(255, 225, 190, ${star.a})` : `rgba(210, 222, 255, ${star.a})`
        ctx.fill()
      }
    }
  }
  ctx.restore()
}

function labelPriority(star: DrawableStar): number {
  if (star.state === 'recommended') return 6
  if (star.role === 'domain') return 5
  if (star.role === 'module') return 4
  if (star.state === 'progress') return 3
  if (star.state === 'done') return 2
  return 1
}

/** 星核屏幕半径（像素） */
/** 宣纸毛笔圆圈的半径（屏幕像素，乘 ps 换算） */
const BRUSH_RADIUS = { domain: 16, module: 8.5 } as const

/** 节点在画面上的外径：连线留白、标签避让都以它为准 */
function markRadius(star: DrawableStar, lod: GraphLodLevel, theme: 'sky' | 'paper'): number {
  if (theme === 'paper' && lod !== 'galaxy' && star.role !== 'topic' && brushesReady()) {
    return BRUSH_RADIUS[star.role]
  }
  return starRadius(star, lod)
}

function starRadius(star: DrawableStar, lod: GraphLodLevel): number {
  const far = lod === 'galaxy'
  if (star.role === 'domain') return far ? 3.2 : 4
  if (star.role === 'module') return far ? 2 : 2.8
  if (star.state === 'locked') return 1.3
  if (star.state === 'recommended') return 2.6
  return 2.1
}

function segmentStyle(
  tone: DrawableSegment['tone'],
  theme: 'sky' | 'paper',
  lod: GraphLodLevel
): { color: string; width: number; dash?: number[] } {
  const far = lod === 'galaxy'
  if (theme === 'paper') {
    if (tone === 'gold') return { color: `rgba(${INK}, 0.82)`, width: 1.15 }
    if (tone === 'flow') return { color: `rgba(${CINNABAR}, 0.85)`, width: 1.3 }
    if (tone === 'parent') return { color: `rgba(${GUIDE}, 0.4)`, width: 0.8, dash: [1.5, 4] }
    if (tone === 'prereq') return { color: `rgba(${CINNABAR}, 0.6)`, width: 0.9, dash: [3, 3] }
    if (tone === 'mid') return { color: `rgba(${INK}, 0.45)`, width: 0.95 }
    return { color: `rgba(${INK}, ${far ? 0.28 : 0.2})`, width: 0.85 }
  }
  if (tone === 'gold') return { color: 'rgba(246, 214, 120, 0.62)', width: 1 }
  if (tone === 'flow') return { color: 'rgba(170, 205, 255, 0.85)', width: 1.1, dash: [3, 5] }
  if (tone === 'parent') return { color: 'rgba(200, 180, 120, 0.22)', width: 0.8, dash: [1.5, 5] }
  if (tone === 'prereq') return { color: 'rgba(170, 205, 255, 0.55)', width: 0.9, dash: [2, 4] }
  if (tone === 'mid') return { color: 'rgba(196, 210, 240, 0.32)', width: 0.9 }
  return { color: `rgba(196, 210, 240, ${far ? 0.42 : 0.2})`, width: far ? 0.9 : 0.8 }
}

/** 推荐线书写进度：写一遍后停在写完的样子 */
function writeProgress(phase: number): number {
  return Math.min(1, phase * 0.22)
}

function strokeGuide(ctx: CanvasRenderingContext2D, a: Pos, b: Pos, px: number): void {
  ctx.strokeStyle = `rgba(${GUIDE}, 0.38)`
  ctx.lineWidth = 0.9 * px
  ctx.beginPath()
  ctx.moveTo(a.x, a.y)
  ctx.lineTo(b.x, b.y)
  ctx.stroke()
}

/**
 * 描红连线：淡朱底稿上，学会的部分用墨描出。
 * traced 为描出比例：1 全部，0.45 从起点描到 45%，-0.45 从终点描回 45%，0 只有底稿。
 */
function drawTracedLine(ctx: CanvasRenderingContext2D, line: Rect, px: number, traced: number): void {
  const at = (t: number) => ({ x: line.x1 + (line.x2 - line.x1) * t, y: line.y1 + (line.y2 - line.y1) * t })
  const [inkFrom, inkTo] = traced > 0 ? [0, traced] : traced < 0 ? [1 + traced, 1] : [0, 0]
  const [guideFrom, guideTo] = traced > 0 ? [traced, 1] : traced < 0 ? [0, 1 + traced] : [0, 1]
  ctx.save()
  ctx.lineCap = 'round'
  if (guideTo > guideFrom) strokeGuide(ctx, at(guideFrom), at(guideTo), px)
  if (inkTo > inkFrom) {
    const a = at(inkFrom)
    const b = at(inkTo)
    ctx.strokeStyle = `rgba(${INK}, 0.88)`
    ctx.lineWidth = 1.5 * px
    ctx.beginPath()
    ctx.moveTo(a.x, a.y)
    ctx.lineTo(b.x, b.y)
    ctx.stroke()
  }
  ctx.restore()
}

/** 宣纸推荐线：淡朱底稿上，朱砂笔从起点一笔写到终点 */
function drawInkWrittenLine(ctx: CanvasRenderingContext2D, line: Rect, px: number, t: number): void {
  ctx.save()
  ctx.lineCap = 'round'
  strokeGuide(ctx, { x: line.x1, y: line.y1 }, { x: line.x2, y: line.y2 }, px)
  if (t > 0) {
    const x = line.x1 + (line.x2 - line.x1) * t
    const y = line.y1 + (line.y2 - line.y1) * t
    ctx.strokeStyle = `rgba(${CINNABAR}, 0.9)`
    ctx.lineWidth = 1.5 * px
    ctx.beginPath()
    ctx.moveTo(line.x1, line.y1)
    ctx.lineTo(x, y)
    ctx.stroke()
    if (t < 1) {
      ctx.fillStyle = `rgba(${CINNABAR}, 0.95)`
      ctx.beginPath()
      ctx.arc(x, y, 1.5 * px, 0, Math.PI * 2)
      ctx.fill()
    }
  }
  ctx.restore()
}

function trimLine(a: Pos, b: Pos, trimA: number, trimB: number): Rect | null {
  const dx = b.x - a.x
  const dy = b.y - a.y
  const len = Math.hypot(dx, dy)
  if (len <= trimA + trimB + 0.5) return null
  const ux = dx / len
  const uy = dy / len
  return { x1: a.x + ux * trimA, y1: a.y + uy * trimA, x2: b.x - ux * trimB, y2: b.y - uy * trimB }
}

type SkyTint = { core: string; glow: string; alpha: number }

function skyTint(star: DrawableStar): SkyTint {
  if (star.state === 'done') return { core: '#fff6d6', glow: '246, 210, 110', alpha: 0.55 }
  if (star.state === 'progress') return { core: '#ffd9b8', glow: '240, 140, 70', alpha: 0.5 }
  if (star.role === 'domain') return { core: '#f4f8ff', glow: '170, 200, 255', alpha: 0.5 }
  if (star.role === 'module') return { core: '#dfe8ff', glow: '150, 180, 245', alpha: 0.35 }
  if (star.state === 'recommended') return { core: '#e6f0ff', glow: '150, 190, 255', alpha: 0.5 }
  if (star.state === 'available') return { core: '#cfdcf5', glow: '150, 180, 240', alpha: 0.22 }
  return { core: '#5d6782', glow: '110, 125, 160', alpha: 0 }
}

function drawSkyStar(
  ctx: CanvasRenderingContext2D,
  star: DrawableStar,
  pos: Pos,
  lod: GraphLodLevel,
  px: number,
  phase: number,
  reducedMotion: boolean,
  burst: boolean
): void {
  const tint = skyTint(star)
  const r = starRadius(star, lod) * px
  const twinkle = reducedMotion ? 1 : 0.85 + 0.15 * Math.sin(phase * 0.9 + (hashString(star.id) % 628) / 100)
  ctx.save()
  ctx.translate(pos.x, pos.y)

  if (tint.alpha > 0) {
    const glowR = r * (star.role === 'domain' ? 6 : star.state === 'done' ? 4.5 : 3.6)
    const alpha = tint.alpha * (star.state === 'progress' ? twinkle : 1)
    const g = ctx.createRadialGradient(0, 0, 0, 0, 0, glowR)
    g.addColorStop(0, `rgba(${tint.glow}, ${alpha})`)
    g.addColorStop(0.35, `rgba(${tint.glow}, ${alpha * 0.28})`)
    g.addColorStop(1, `rgba(${tint.glow}, 0)`)
    ctx.fillStyle = g
    ctx.beginPath()
    ctx.arc(0, 0, glowR, 0, Math.PI * 2)
    ctx.fill()
  }

  if (star.role === 'domain' || star.state === 'done') {
    const len = r * (star.role === 'domain' ? 4.2 : 3)
    drawSpikes(ctx, len, `rgba(${tint.glow}, ${star.role === 'domain' ? 0.75 : 0.6})`, px * 0.9)
  }

  if (star.state === 'progress') {
    const sweep = Math.max(0.15, Math.min(1, star.mastery ?? 0.35)) * Math.PI * 2
    ctx.beginPath()
    ctx.arc(0, 0, r * 2.4, -Math.PI / 2, -Math.PI / 2 + sweep)
    ctx.strokeStyle = 'rgba(240, 150, 80, 0.75)'
    ctx.lineWidth = 1 * px
    ctx.stroke()
  } else if (star.state === 'recommended') {
    const breathe = reducedMotion ? 1 : 0.5 + 0.5 * Math.sin(phase * 1.4)
    ctx.beginPath()
    ctx.arc(0, 0, r * (2.6 + breathe * 0.6), 0, Math.PI * 2)
    ctx.strokeStyle = `rgba(170, 205, 255, ${0.45 + 0.3 * breathe})`
    ctx.lineWidth = 0.9 * px
    ctx.stroke()
  }

  if (burst && !reducedMotion) {
    const t = (phase % 6) / 6
    ctx.beginPath()
    ctx.arc(0, 0, r * (2 + t * 6), 0, Math.PI * 2)
    ctx.strokeStyle = `rgba(246, 214, 120, ${0.5 * (1 - t)})`
    ctx.lineWidth = 1 * px
    ctx.stroke()
  }

  ctx.beginPath()
  ctx.arc(0, 0, r, 0, Math.PI * 2)
  ctx.fillStyle = tint.core
  ctx.globalAlpha = star.state === 'locked' ? 0.7 : 1
  ctx.fill()
  ctx.restore()
}

/** 四角衍射芒：中间亮、两端渐隐的细线 */
function drawSpikes(ctx: CanvasRenderingContext2D, length: number, color: string, width: number): void {
  ctx.save()
  ctx.lineWidth = width
  ctx.lineCap = 'round'
  for (const [dx, dy] of [
    [1, 0],
    [0, 1],
  ] as const) {
    const g = ctx.createLinearGradient(-dx * length, -dy * length, dx * length, dy * length)
    g.addColorStop(0, 'rgba(0,0,0,0)')
    g.addColorStop(0.5, color)
    g.addColorStop(1, 'rgba(0,0,0,0)')
    ctx.strokeStyle = g
    ctx.beginPath()
    ctx.moveTo(-dx * length, -dy * length)
    ctx.lineTo(dx * length, dy * length)
    ctx.stroke()
  }
  ctx.restore()
}

/** 印文：英文开头取大写首字母（TypeScript → TS），否则取前两个汉字 */
function sealText(label: string): string {
  const trimmed = label.trim()
  const han = trimmed.match(/[\u4e00-\u9fff]/g)
  if (han?.length && !/^[A-Za-z]/.test(trimmed)) return han.slice(0, 2).join('')
  const words = trimmed.match(/[A-Za-z0-9]+/g) ?? []
  if (words.length >= 2 && !han?.length) return (words[0]![0]! + words[1]![0]!).toUpperCase()
  const first = words[0] ?? trimmed
  const caps = first.match(/[A-Z]/g)
  return caps && caps.length >= 2 ? caps.slice(0, 2).join('') : first.charAt(0).toUpperCase()
}

/** 学完整门课：朱砂小印，两个汉字竖排 */
function drawSeal(ctx: CanvasRenderingContext2D, star: DrawableStar, pos: Pos, px: number): void {
  const text = sealText(star.label)
  const vertical = /[\u4e00-\u9fff]/.test(text) && text.length === 2
  const w = (vertical ? 15 : 19) * px
  const h = (vertical ? 27 : 19) * px
  const tilt = ((hashString(star.id) % 7) - 3) * 0.03
  ctx.save()
  ctx.translate(pos.x, pos.y)
  ctx.rotate(tilt)
  ctx.fillStyle = `rgba(${CINNABAR}, 0.9)`
  ctx.beginPath()
  ctx.roundRect(-w / 2, -h / 2, w, h, 1.6 * px)
  ctx.fill()
  ctx.strokeStyle = PAPER
  ctx.lineWidth = 0.7 * px
  ctx.strokeRect(-w / 2 + 1.6 * px, -h / 2 + 1.6 * px, w - 3.2 * px, h - 3.2 * px)
  ctx.fillStyle = PAPER
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  if (vertical) {
    ctx.font = `700 ${9.5 * px}px ${SERIF_FACE}`
    ctx.fillText(text[0]!, 0, -h * 0.21)
    ctx.fillText(text[1]!, 0, h * 0.23)
  } else {
    ctx.font = `700 ${(text.length > 1 ? 8.5 : 11) * px}px ${SERIF_FACE}`
    ctx.fillText(text, 0, 0.3 * px)
  }
  ctx.restore()
}

/**
 * 课程与模块的毛笔圆相：淡朱底稿圈上，用墨按掌握比例从 12 点钟方向写出一段，写满即学完。
 * 整门课学完时圈中盖朱砂印。
 */
function drawBrushedStar(
  ctx: CanvasRenderingContext2D,
  star: DrawableStar,
  pos: Pos,
  ps: number,
  phase: number,
  reducedMotion: boolean,
  burst: boolean
): void {
  const role = star.role as 'domain' | 'module'
  const radius = BRUSH_RADIUS[role] * ps
  const seed = hashString(star.id)
  const done = star.state === 'done'
  const mastery = done ? 1 : Math.max(0, Math.min(1, star.mastery ?? 0))
  if (burst && !reducedMotion) {
    const t = (phase % 6) / 6
    ctx.save()
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, radius * (1.2 + t * 1.5), 0, Math.PI * 2)
    ctx.strokeStyle = `rgba(${INK}, ${0.35 * (1 - t)})`
    ctx.lineWidth = 1.2 * ps
    ctx.stroke()
    ctx.restore()
  }
  if (!done) drawBrushRing(ctx, { x: pos.x, y: pos.y, radius, role, seed, rgb: GUIDE, alpha: 0.42 })
  if (mastery > 0) drawBrushRing(ctx, { x: pos.x, y: pos.y, radius, role, seed, rgb: INK, alpha: 0.9, sweep: mastery })
  if (role === 'domain' && done) {
    drawSeal(ctx, star, pos, ps * 0.78)
  } else if (role === 'module' && done) {
    ctx.beginPath()
    ctx.arc(pos.x, pos.y, 2.2 * ps, 0, Math.PI * 2)
    ctx.fillStyle = `rgba(${INK}, 0.9)`
    ctx.fill()
  }
}

/** 宣纸描红：未学是淡朱底稿圈，学会的用墨描成实心点，朱砂用于在学与推荐 */
function drawPaperStar(
  ctx: CanvasRenderingContext2D,
  star: DrawableStar,
  pos: Pos,
  lod: GraphLodLevel,
  ps: number,
  pl: number,
  phase: number,
  reducedMotion: boolean,
  burst: boolean
): void {
  if (star.role !== 'topic' && brushesReady()) {
    drawBrushedStar(ctx, star, pos, ps, phase, reducedMotion, burst)
    return
  }
  if (star.role === 'domain' && star.state === 'done') {
    drawSeal(ctx, star, pos, ps)
    return
  }
  const r = starRadius(star, lod) * ps
  const ring = (radius: number, color: string, width: number) => {
    ctx.beginPath()
    ctx.arc(0, 0, radius, 0, Math.PI * 2)
    ctx.strokeStyle = color
    ctx.lineWidth = width * pl
    ctx.stroke()
  }
  const dot = (radius: number, color: string) => {
    ctx.beginPath()
    ctx.arc(0, 0, radius, 0, Math.PI * 2)
    ctx.fillStyle = color
    ctx.fill()
  }
  ctx.save()
  ctx.translate(pos.x, pos.y)
  if (burst && !reducedMotion) {
    const t = (phase % 6) / 6
    ring(r * (2 + t * 5), `rgba(${INK}, ${0.45 * (1 - t)})`, 1.2)
  }
  if (star.state === 'done') {
    if (star.role === 'module') {
      ring(r * 1.5, `rgba(${INK}, 0.85)`, 1.3)
      dot(r * 0.8, `rgba(${INK}, 0.92)`)
    } else {
      dot(r * 1.15, `rgba(${INK}, 0.92)`)
    }
  } else if (star.role === 'domain') {
    ring(r * 1.9, `rgba(${GUIDE}, 0.6)`, 1.1)
    ring(r * 0.8, `rgba(${GUIDE}, 0.6)`, 1.1)
  } else if (star.role === 'module') {
    ring(r * 1.2, `rgba(${GUIDE}, 0.55)`, 1)
    dot(r * 0.35, `rgba(${GUIDE}, 0.45)`)
  } else if (star.state === 'progress') {
    dot(r, `rgba(${CINNABAR}, 0.9)`)
    const sweep = Math.max(0.15, Math.min(1, star.mastery ?? 0.35)) * Math.PI * 2
    ctx.beginPath()
    ctx.arc(0, 0, r * 2.1, -Math.PI / 2, -Math.PI / 2 + sweep)
    ctx.strokeStyle = `rgba(${CINNABAR}, 0.65)`
    ctx.lineWidth = 1 * pl
    ctx.stroke()
  } else if (star.state === 'recommended') {
    ring(r, `rgba(${CINNABAR}, 0.95)`, 1.4)
    ring(r * 1.9, `rgba(${CINNABAR}, 0.3)`, 0.8)
  } else if (star.state === 'available') {
    ring(r, `rgba(${GUIDE}, 0.6)`, 1)
  } else {
    ring(r, `rgba(${GUIDE}, 0.3)`, 0.9)
  }
  ctx.restore()
}

function overlaps(rect: Rect, taken: Rect[]): boolean {
  return taken.some((o) => rect.x1 < o.x2 && rect.x2 > o.x1 && rect.y1 < o.y2 && rect.y2 > o.y1)
}

function truncate(text: string, max: number): string {
  return text.length > max ? `${text.slice(0, max - 1)}…` : text
}

function drawStarLabel(
  ctx: CanvasRenderingContext2D,
  star: DrawableStar,
  pos: Pos,
  theme: 'sky' | 'paper',
  unit: { ps: number; pt: number },
  taken: Rect[],
  force: boolean
): void {
  const px = unit.pt
  const size = (star.role === 'domain' ? 13 : star.role === 'module' ? 12 : 11) * px
  const text = truncate(star.label, star.role === 'topic' ? 14 : 16)
  ctx.save()
  ctx.font = `${star.role === 'topic' ? 400 : 500} ${size}px ${fontFace(theme)}`
  const width = ctx.measureText(text).width
  const brushed = theme === 'paper' && star.role !== 'topic' && brushesReady()
  const sealed = theme === 'paper' && star.role === 'domain' && star.state === 'done'
  const dist = brushed
    ? BRUSH_RADIUS[star.role as 'domain' | 'module'] * unit.ps + 4 * px
    : sealed
      ? 17 * unit.ps
      : starRadius(star, 'node') * unit.ps + 7 * px
  const cos = Math.cos(star.labelAngle)
  const sin = Math.sin(star.labelAngle)
  const ax = pos.x + cos * dist
  const ay = pos.y + sin * dist
  const align: CanvasTextAlign = Math.abs(cos) < 0.35 ? 'center' : cos > 0 ? 'left' : 'right'
  const x1 = align === 'left' ? ax : align === 'right' ? ax - width : ax - width / 2
  const yc = Math.abs(cos) < 0.35 ? ay + (sin > 0 ? size * 0.55 : -size * 0.55) : ay
  const rect = { x1: x1 - 3 * px, y1: yc - size * 0.65, x2: x1 + width + 3 * px, y2: yc + size * 0.65 }
  if (!force && overlaps(rect, taken)) {
    ctx.restore()
    return
  }
  taken.push(rect)
  ctx.textAlign = align
  ctx.textBaseline = 'middle'
  ctx.lineJoin = 'round'
  ctx.lineWidth = 3 * px
  ctx.strokeStyle = theme === 'paper' ? 'rgba(247, 241, 227, 0.9)' : 'rgba(9, 13, 24, 0.82)'
  ctx.fillStyle = labelColor(star, theme)
  ctx.strokeText(text, ax, yc)
  ctx.fillText(text, ax, yc)
  ctx.restore()
}

function labelColor(star: DrawableStar, theme: 'sky' | 'paper'): string {
  if (theme === 'paper') {
    if (star.state === 'done') return `rgba(${INK}, 0.92)`
    if (star.state === 'recommended' || star.state === 'progress') return `rgb(${CINNABAR})`
    if (star.state === 'locked') return `rgba(${GUIDE}, 0.45)`
    return `rgba(${GUIDE}, 0.85)`
  }
  if (star.state === 'done') return 'rgba(246, 222, 150, 0.95)'
  if (star.state === 'progress') return 'rgba(250, 200, 160, 0.95)'
  if (star.state === 'recommended') return '#cfe0ff'
  if (star.state === 'locked') return 'rgba(150, 160, 185, 0.6)'
  return star.role === 'topic' ? 'rgba(210, 218, 235, 0.78)' : 'rgba(232, 238, 250, 0.92)'
}

function drawConstellationName(
  ctx: CanvasRenderingContext2D,
  name: ConstellationName,
  pos: Pos,
  theme: 'sky' | 'paper',
  px: number,
  taken: Rect[],
  alpha: number
): void {
  const size = 12 * px
  const text = truncate(name.text, 14)
  ctx.save()
  ctx.font = `${theme === 'paper' ? 600 : 500} ${size}px ${fontFace(theme)}`
  const spacing = (theme === 'paper' ? 4 : 2.5) * px
  const chars = [...text]
  const latin = /[A-Za-z0-9]/
  const gaps = chars.map((ch, i) =>
    i === chars.length - 1
      ? 0
      : theme === 'paper' && latin.test(ch) && latin.test(chars[i + 1]!)
        ? spacing * 0.2
        : spacing
  )
  const width = ctx.measureText(text).width + gaps.reduce((sum, g) => sum + g, 0)
  const y = pos.y + 14 * px
  const rect = { x1: pos.x - width / 2 - 6 * px, y1: y - 2 * px, x2: pos.x + width / 2 + 6 * px, y2: y + size + 4 * px }
  if (overlaps(rect, taken)) {
    ctx.restore()
    return
  }
  taken.push(rect)
  ctx.globalAlpha = alpha
  ctx.textAlign = 'left'
  ctx.textBaseline = 'top'
  ctx.fillStyle = name.complete
    ? theme === 'paper'
      ? `rgba(${CINNABAR}, 0.92)`
      : 'rgba(246, 214, 120, 0.9)'
    : theme === 'paper'
      ? `rgba(${INK}, 0.62)`
      : 'rgba(190, 202, 228, 0.62)'
  let x = pos.x - width / 2
  chars.forEach((ch, i) => {
    ctx.fillText(ch, x, y)
    x += ctx.measureText(ch).width + gaps[i]!
  })
  ctx.restore()
}
