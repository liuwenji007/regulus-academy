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
  tree: KnowledgeTree
}

export interface PackedConstellation {
  domainId: string
  salt: number
  layout: ConstellationLayout
  cx: number
  cy: number
  /** 所在旋臂的起始角 */
  theta0: number
  field: string
  /** 领域槽位（预设顺序 / 自定义 / 未分类），决定领域配色 */
  slot: number
}

export function packConstellations(courses: PackCourse[]): PackedConstellation[] {
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
    placed.push({ domainId: course.domainId, salt: chosenSalt, layout: chosen, cx: 0, cy: 0, theta0: 0, field: '', slot: 0 })
  }

  assignCenters(placed, sorted)
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
    const field = fields.length > 2 ? `${fields.slice(0, 2).join('、')} 等` : fields.join('、')
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
  const slots = [...new Set(courses.map((course) => fieldArmSlot(fieldOf(course))))].sort((a, b) => a - b)
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
