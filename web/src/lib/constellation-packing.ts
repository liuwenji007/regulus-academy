import type { KnowledgeTree } from './api'
import { buildConstellation, shapeDistance, type ConstellationLayout, type ShapeFingerprint } from './constellation-layout'
import { topicRootKey } from './graph-constellation'

export const SIMILAR_THRESHOLD = 0.35
export const PACK_GAP = 120
export interface PackCourse {
  domainId: string
  slug?: string
  name: string
  createdAt?: string
  /** 所属领域；空则归入「未分类」 */
  field?: string
  nodeTotal?: number
  completed?: number
  /** 有知识点在学但尚未学完任何一个时也算开始；星图据此把课程放进「在学」圈 */
  started?: boolean
  tree: KnowledgeTree
}

export interface PackedConstellation {
  domainId: string
  salt: number
  layout: ConstellationLayout
  cx: number
  cy: number
  /** 旋臂布局为所在旋臂的起始角；星图布局为所在分野的中线角 */
  theta0: number
  field: string
  /** 领域槽位（预设顺序 / 自定义 / 未分类），决定领域配色 */
  slot: number
  /** 星图布局中的掌握圈层：0 已通，1 在学，2 未涉；旋臂布局恒为 0 */
  tier: number
}

/** spiral：星空的银河旋臂；chart：宣纸的盖天式星图（同心圈层 + 分野扇区） */
export type PackMode = 'spiral' | 'chart'

export function packConstellations(courses: PackCourse[], mode: PackMode = 'spiral'): PackedConstellation[] {
  const sorted = courses.slice().sort((a, b) => {
    const byTime = (a.createdAt ?? '').localeCompare(b.createdAt ?? '')
    return byTime || a.domainId.localeCompare(b.domainId)
  })
  const placed: PackedConstellation[] = []
  const familyPrints = new Map<string, ShapeFingerprint[]>()

  for (const course of sorted) {
    const family = topicRootKey(course.slug ?? '', course.name)
    const compare = [...(familyPrints.get(family) ?? []), ...placed.slice(-4).map((item) => item.layout.spineFingerprint)]
    let chosenSalt = 0
    let chosen = buildConstellation(course.domainId, course.tree, 0)
    let bestDistance = distanceTo(chosen.spineFingerprint, compare)
    for (let salt = 0; salt < 8; salt++) {
      const layout = salt === 0 ? chosen : buildConstellation(course.domainId, course.tree, salt)
      const distance = distanceTo(layout.spineFingerprint, compare)
      if (salt === 0 || distance > bestDistance) {
        bestDistance = distance
        chosenSalt = salt
        chosen = layout
      }
      if (distance > SIMILAR_THRESHOLD) {
        chosenSalt = salt
        chosen = layout
        break
      }
    }
    const prints = familyPrints.get(family) ?? []
    prints.push(chosen.spineFingerprint)
    familyPrints.set(family, prints)
    placed.push({ domainId: course.domainId, salt: chosenSalt, layout: chosen, cx: 0, cy: 0, theta0: 0, field: '', slot: 0, tier: 0 })
  }

  if (mode === 'chart') assignChartCenters(placed, sorted)
  else assignCenters(placed, sorted)
  return placed
}

function distanceTo(print: ShapeFingerprint, others: ShapeFingerprint[]): number {
  if (others.length === 0) return Infinity
  return Math.min(...others.map((other) => shapeDistance(print, other)))
}

/** 领域槽位数：决定领域排序与螺线形状（按 8 臂定螺旋度，领域少时只是臂间更空），核球内不放课程 */
export const GALAXY_ARMS = 8
export const GALAXY_CORE = 780
/**
 * 对数螺线：同一方向上相邻旋臂的半径比。间距随半径同比放大，臂宽也随之放大，
 * 星座按面积铺开，课程数翻 4 倍臂长只翻约 2 倍。小于 1.5 内圈挤，大于 1.8 不像旋涡
 */
export const GALAXY_ARM_RATIO = 1.6
const SPIRAL_B = Math.log(GALAXY_ARM_RATIO) / ((2 * Math.PI) / GALAXY_ARMS)
/** 旋臂在半径 GALAXY_CORE 处恰好指向起始角 */
const SPIRAL_R0 = GALAXY_CORE
/** 内圈旋臂的最小半宽 */
export const GALAXY_ARM_WIDTH = 150
/** 星座中心可偏离旋臂中心线的比例：占同方向臂间距不到四成，余下留作臂间暗带 */
const ARM_FILL = 0.38
const ARM_STEP = 40
const LATERAL_STEP = 75
const PLACE_START = GALAXY_CORE * 1.5
/** 领域名与任何星座圆之间的最小间距 */
const FIELD_LABEL_GAP = 220

/** 与 internal/domain/field.go 的 PresetFields 保持同一顺序 */
export const PRESET_FIELDS = ['编程语言', 'AI 与数据', '工程与架构', '运维与基础设施', '语言与沟通', '人文与通识'] as const
export const UNCLASSIFIED_FIELD = '未分类'

/** 旋臂上距中心 r、垂直旋臂方向偏移 lateral 的点。theta0 是这条旋臂的起始角 */
export function galaxyArmPoint(theta0: number, r: number, lateral = 0): { x: number; y: number } {
  const theta = theta0 + Math.log(Math.max(r, 1) / SPIRAL_R0) / SPIRAL_B
  const x = r * Math.cos(theta)
  const y = r * Math.sin(theta)
  const tx = Math.cos(theta) - Math.sin(theta) / SPIRAL_B
  const ty = Math.sin(theta) + Math.cos(theta) / SPIRAL_B
  const len = Math.hypot(tx, ty)
  return { x: x - (ty / len) * lateral, y: y + (tx / len) * lateral }
}

/** 半径 r 处星座中心可偏离旋臂中心线的最大距离 */
export function armHalfWidth(r: number): number {
  return Math.max(GALAXY_ARM_WIDTH, r * (GALAXY_ARM_RATIO - 1) * ARM_FILL * 0.5)
}

function lateralOffsets(r: number): number[] {
  const half = armHalfWidth(r)
  const out = [0]
  for (let d = LATERAL_STEP; d <= half; d += LATERAL_STEP) out.push(-d, d)
  return out
}

/** 预设领域各占一臂，所有自定义领域共用倒数第二臂，未分类固定最后一臂 */
export function fieldArmSlot(field: string): number {
  const name = field.trim()
  const preset = PRESET_FIELDS.indexOf(name as (typeof PRESET_FIELDS)[number])
  if (preset >= 0) return preset
  if (!name || name === UNCLASSIFIED_FIELD) return GALAXY_ARMS - 1
  return GALAXY_ARMS - 2
}

function fieldOf(course: PackCourse | undefined): string {
  return course?.field?.trim() || UNCLASSIFIED_FIELD
}

/** 银河外缘半径：最外层星座外沿，至少为核球的 3 倍 */
export function galaxyOuter(packed: PackedConstellation[]): number {
  return Math.max(GALAXY_CORE * 3, ...packed.map((item) => Math.hypot(item.cx, item.cy) + item.layout.radius))
}

export interface FieldAnchor {
  field: string
  slot: number
  x: number
  y: number
}

/** 领域名写在本臂中心线上、紧跟该臂最外侧星座之后，并避开所有星座；共用一臂的自定义领域合并成一个名字 */
export function fieldNameAnchors(packed: PackedConstellation[]): FieldAnchor[] {
  const arms = new Map<number, { fields: string[]; last: PackedConstellation }>()
  for (const item of packed) {
    const arm = arms.get(item.theta0) ?? { fields: [], last: item }
    if (!arm.fields.includes(item.field)) arm.fields.push(item.field)
    if (Math.hypot(item.cx, item.cy) > Math.hypot(arm.last.cx, arm.last.cy)) arm.last = item
    arms.set(item.theta0, arm)
  }
  const limit = galaxyOuter(packed) * 1.3
  const clear = (x: number, y: number, gap: number) =>
    packed.every((item) => Math.hypot(item.cx - x, item.cy - y) >= item.layout.radius + gap)
  return [...arms.entries()].map(([theta0, { fields, last }]) => {
    const field = fieldGroupName(fields)
    let point = galaxyArmPoint(theta0, limit)
    for (let r = Math.hypot(last.cx, last.cy); r < limit; r += ARM_STEP) {
      const p = galaxyArmPoint(theta0, r)
      if (clear(p.x, p.y, FIELD_LABEL_GAP)) {
        point = p
        break
      }
    }
    return { field, slot: last.slot, ...point }
  })
}

export interface ArmSpan {
  theta0: number
  slot: number
  /** 该旋臂最内侧星座的内沿半径 */
  start: number
  /** 该旋臂最外侧星座的外沿半径 */
  end: number
  /** 该旋臂上每个星座的中心与半径，光晕围绕它们聚集 */
  stars: Array<{ x: number; y: number; radius: number }>
}

/** 每条用到的旋臂上星座覆盖的半径段与各星座位置，供光晕只在有内容处生成 */
export function armSpans(packed: PackedConstellation[]): ArmSpan[] {
  const spans = new Map<number, ArmSpan>()
  for (const item of packed) {
    const r = Math.hypot(item.cx, item.cy)
    const span = spans.get(item.theta0) ?? { theta0: item.theta0, slot: item.slot, start: Infinity, end: 0, stars: [] }
    span.start = Math.min(span.start, r - item.layout.radius)
    span.end = Math.max(span.end, r + item.layout.radius)
    span.stars.push({ x: item.cx, y: item.cy, radius: item.layout.radius })
    spans.set(item.theta0, span)
  }
  return [...spans.values()]
}

/**
 * 领域掌握比例：>70% 内圈，30%～70% 中圈，其余外圈。
 * 各档按半径比分开，每档只多转半个扇区，不会转进相邻领域
 */
function tierStart(ratio: number): number {
  if (ratio > 0.7) return PLACE_START
  if (ratio >= 0.3) return PLACE_START * Math.sqrt(GALAXY_ARM_RATIO)
  return PLACE_START * GALAXY_ARM_RATIO
}

/**
 * 按领域分旋臂、按领域掌握比例分档；在旋臂带内由内向外找第一个空位，同一半径优先靠近中心线。
 * 旋臂只在实际出现的领域间平分一圈：领域集合变化时各臂整体转动，领域不变时新课程不挪动已有星座
 */
function assignCenters(placed: PackedConstellation[], courses: PackCourse[]): void {
  const courseById = new Map(courses.map((course) => [course.domainId, course]))
  const stats = new Map<string, { done: number; total: number }>()
  for (const course of courses) {
    const field = fieldOf(course)
    const stat = stats.get(field) ?? { done: 0, total: 0 }
    stat.done += course.completed ?? 0
    stat.total += course.nodeTotal ?? 0
    stats.set(field, stat)
  }
  const startOf = (field: string) => {
    const stat = stats.get(field)
    const ratio = stat && stat.total > 0 ? stat.done / stat.total : 0
    return tierStart(ratio)
  }
  const slots = presentSlots(courses)
  const done: PackedConstellation[] = []
  for (const item of placed) {
    const course = courseById.get(item.domainId)
    const field = fieldOf(course)
    const slot = fieldArmSlot(field)
    const theta0 = (slots.indexOf(slot) * 2 * Math.PI) / slots.length
    item.theta0 = theta0
    item.field = field
    item.slot = slot
    const fits = (x: number, y: number) =>
      done.every((other) => Math.hypot(other.cx - x, other.cy - y) >= other.layout.radius + item.layout.radius + PACK_GAP)
    let best: { x: number; y: number } | null = null
    for (let r = startOf(field); r < 200000 && !best; r += ARM_STEP) {
      for (const lateral of lateralOffsets(r)) {
        const point = galaxyArmPoint(theta0, r, lateral)
        if (fits(point.x, point.y)) {
          best = point
          break
        }
      }
    }
    item.cx = best?.x ?? 0
    item.cy = best?.y ?? 0
    done.push(item)
  }
}

/** 同一旋臂或分野里有多个领域时合并成一个名字，超过两个只列前两个 */
function fieldGroupName(fields: string[]): string {
  return fields.length > 2 ? `${fields.slice(0, 2).join('、')} 等` : fields.join('、')
}

function presentSlots(courses: PackCourse[]): number[] {
  return [...new Set(courses.map((course) => fieldArmSlot(fieldOf(course))))].sort((a, b) => a - b)
}

/** 星图内规半径：中心留白，不放课程 */
export const CHART_CORE = 700
/** 相邻圈层之间的留白，圈线画在正中 */
const CHART_RING_GAP = 280
/** 星座圆与分野界线之间的最小距离 */
const CHART_WEDGE_MARGIN = 60
/** 同一半径上沿弧线试探的步长 */
const CHART_ARC_STEP = 60
/** 第一个分野从正上方开始，顺时针排列 */
const CHART_START = -Math.PI / 2

/** 课程掌握比例：≥70% 已通（内圈），学过一些或正在学为在学（中圈），未开始为未涉（外圈） */
function courseTier(course: PackCourse | undefined): number {
  const total = course?.nodeTotal ?? 0
  const ratio = total > 0 ? (course?.completed ?? 0) / total : 0
  if (ratio >= 0.7) return 0
  if (ratio > 0 || course?.started) return 1
  return 2
}

function wedgeOf(slots: number[], slot: number): { a0: number; a1: number; mid: number } {
  const span = (2 * Math.PI) / slots.length
  const a0 = CHART_START + slots.indexOf(slot) * span
  return { a0, a1: a0 + span, mid: a0 + span / 2 }
}

/** 半径 r 处、星座半径 radius 时，相对分野中线可偏转的角度，按离中线由近到远排列 */
function chartOffsets(r: number, span: number, radius: number): number[] {
  const limit =
    span >= 2 * Math.PI ? Math.PI : span / 2 - Math.asin(Math.min(1, (radius + CHART_WEDGE_MARGIN) / Math.max(r, 1)))
  if (limit < 0) return []
  const step = CHART_ARC_STEP / Math.max(r, 1)
  const out = [0]
  for (let d = step; d <= limit; d += step) out.push(-d, d)
  return out
}

/**
 * 盖天式星图：每个领域占一块分野（扇区，在出现的领域间平分），按课程掌握程度分圈层，
 * 已通在内、在学居中、未涉在外。圈层内按创建顺序由内向外、靠近分野中线找第一个空位。
 * 圈层宽度随内容伸缩，课程数翻 4 倍半径约翻 2 倍。
 */
function assignChartCenters(placed: PackedConstellation[], courses: PackCourse[]): void {
  const courseById = new Map(courses.map((course) => [course.domainId, course]))
  const slots = presentSlots(courses)
  const span = (2 * Math.PI) / slots.length
  const order = placed
    .map((item, index) => ({ item, index, tier: courseTier(courseById.get(item.domainId)) }))
    .sort((a, b) => a.tier - b.tier || a.index - b.index)
  const done: PackedConstellation[] = []
  let floor = CHART_CORE
  for (let tier = 0; tier < 3; tier++) {
    const group = order.filter((entry) => entry.tier === tier)
    if (!group.length) continue
    let reach = floor
    for (const { item } of group) {
      const field = fieldOf(courseById.get(item.domainId))
      const slot = fieldArmSlot(field)
      const { mid } = wedgeOf(slots, slot)
      item.field = field
      item.slot = slot
      item.theta0 = mid
      item.tier = tier
      const radius = item.layout.radius
      const fits = (x: number, y: number) =>
        done.every((other) => Math.hypot(other.cx - x, other.cy - y) >= other.layout.radius + radius + PACK_GAP)
      let best: { x: number; y: number } | null = null
      for (let r = floor + radius + CHART_WEDGE_MARGIN; r < 200000 && !best; r += ARM_STEP) {
        for (const d of chartOffsets(r, span, radius)) {
          const x = r * Math.cos(mid + d)
          const y = r * Math.sin(mid + d)
          if (fits(x, y)) {
            best = { x, y }
            break
          }
        }
      }
      item.cx = best?.x ?? 0
      item.cy = best?.y ?? 0
      done.push(item)
      reach = Math.max(reach, Math.hypot(item.cx, item.cy) + radius)
    }
    floor = reach + CHART_RING_GAP
  }
}

export interface StarChart {
  /** 内规半径 */
  core: number
  /** 相邻圈层之间的分界圈半径 */
  rings: number[]
  /** 由内向外出现的圈层（0 已通，1 在学，2 未涉），比 rings 多一个 */
  tiers: number[]
  /** 外规半径 */
  outer: number
  /** 各分野的起止角与中线角；name 为该分野内出现的领域合并后的名字 */
  wedges: Array<{ a0: number; a1: number; mid: number; slot: number; name: string }>
}

/** 由星图布局结果推出要画的内规、圈层分界、外规与分野 */
export function starChart(packed: PackedConstellation[]): StarChart {
  const spans: Array<{ start: number; reach: number }> = []
  const tiers: number[] = []
  for (let tier = 0; tier < 3; tier++) {
    const group = packed.filter((item) => item.tier === tier)
    if (!group.length) continue
    tiers.push(tier)
    spans.push({
      start: Math.min(...group.map((item) => Math.hypot(item.cx, item.cy) - item.layout.radius)),
      reach: Math.max(...group.map((item) => Math.hypot(item.cx, item.cy) + item.layout.radius)),
    })
  }
  const rings = spans.slice(1).map((span, i) => (spans[i]!.reach + span.start) / 2)
  const outer = Math.max(CHART_CORE * 2, (spans.at(-1)?.reach ?? CHART_CORE) + CHART_RING_GAP * 0.8)
  const slots = [...new Set(packed.map((item) => item.slot))].sort((a, b) => a - b)
  const wedges = slots.map((slot) => {
    const fields: string[] = []
    for (const item of packed) if (item.slot === slot && !fields.includes(item.field)) fields.push(item.field)
    return { ...wedgeOf(slots, slot), slot, name: fieldGroupName(fields) }
  })
  return { core: CHART_CORE, rings, tiers, outer, wedges }
}
