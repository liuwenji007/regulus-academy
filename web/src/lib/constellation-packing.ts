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
  tree: KnowledgeTree
}

export interface PackedConstellation {
  domainId: string
  salt: number
  layout: ConstellationLayout
  cx: number
  cy: number
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
    placed.push({ domainId: course.domainId, salt: chosenSalt, layout: chosen, cx: 0, cy: 0 })
  }

  assignCenters(placed, sorted)
  return placed
}

function distanceTo(print: ShapeFingerprint, others: ShapeFingerprint[]): number {
  if (others.length === 0) return Infinity
  return Math.min(...others.map((other) => shapeDistance(print, other)))
}

/** 银河旋臂：两条阿基米德旋臂，核球内不放课程 */
export const GALAXY_ARMS = 2
export const GALAXY_CORE = 520
export const GALAXY_PITCH = 1250
export const GALAXY_ARM_WIDTH = 150
const ARM_STEP = 40
const ARM_LANES = [0, -1, 1, -2, 2]

/** 旋臂上距中心 r、垂直旋臂方向偏移 lateral 的点 */
export function galaxyArmPoint(arm: number, r: number, lateral = 0): { x: number; y: number } {
  const theta = (arm * 2 * Math.PI) / GALAXY_ARMS + r / GALAXY_PITCH
  const x = r * Math.cos(theta)
  const y = r * Math.sin(theta)
  const tx = Math.cos(theta) - (r / GALAXY_PITCH) * Math.sin(theta)
  const ty = Math.sin(theta) + (r / GALAXY_PITCH) * Math.cos(theta)
  const len = Math.hypot(tx, ty) || 1
  return { x: x - (ty / len) * lateral, y: y + (tx / len) * lateral }
}

function* armCandidates(): Generator<{ x: number; y: number; r: number }> {
  for (let r = GALAXY_CORE; r < 200000; r += ARM_STEP) {
    for (const lane of ARM_LANES) {
      for (let arm = 0; arm < GALAXY_ARMS; arm++) {
        const p = galaxyArmPoint(arm, r, lane * GALAXY_ARM_WIDTH * 0.5)
        yield { ...p, r }
      }
    }
  }
}

/** 按创建顺序沿旋臂由内向外落位；同族课程选离族质心最近的空位 */
function assignCenters(placed: PackedConstellation[], courses: PackCourse[]): void {
  const courseById = new Map(courses.map((course) => [course.domainId, course]))
  const familyMembers = new Map<string, PackedConstellation[]>()
  const done: PackedConstellation[] = []
  for (const item of placed) {
    const course = courseById.get(item.domainId)
    const family = topicRootKey(course?.slug ?? '', course?.name ?? item.domainId)
    const members = familyMembers.get(family) ?? []
    const fits = (x: number, y: number) =>
      done.every((other) => Math.hypot(other.cx - x, other.cy - y) >= other.layout.radius + item.layout.radius + PACK_GAP)
    let best: { x: number; y: number } | null = null
    if (members.length) {
      const fx = members.reduce((sum, m) => sum + m.cx, 0) / members.length
      const fy = members.reduce((sum, m) => sum + m.cy, 0) / members.length
      const focusR = Math.hypot(fx, fy)
      let bestDist = Infinity
      for (const c of armCandidates()) {
        if (c.r - focusR > bestDist) break
        const d = Math.hypot(c.x - fx, c.y - fy)
        if (d < bestDist && fits(c.x, c.y)) {
          bestDist = d
          best = c
        }
      }
    } else {
      for (const c of armCandidates()) {
        if (fits(c.x, c.y)) {
          best = c
          break
        }
      }
    }
    item.cx = best?.x ?? 0
    item.cy = best?.y ?? 0
    done.push(item)
    members.push(item)
    familyMembers.set(family, members)
  }
}
