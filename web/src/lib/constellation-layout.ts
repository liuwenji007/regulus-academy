import type { KnowledgeTree } from './api'
import { resolveGraphModules } from './tree-normalize'

export type Archetype = 'chain' | 'twin' | 'sickle' | 'cross' | 'zigzag' | 'dipper'

export interface StarPlacement {
  id: string
  role: 'domain' | 'module' | 'topic'
  x: number
  y: number
  magnitude: number
  labelAngle: number
}

export interface ConstellationSegment {
  from: string
  to: string
  kind: 'spine' | 'limb'
}

export interface ShapeFingerprint {
  aspect: number
  hullFill: number
  turnHistogram: number[]
  degreeHistogram: number[]
}

export interface ConstellationLayout {
  stars: StarPlacement[]
  segments: ConstellationSegment[]
  radius: number
  archetype: Archetype
  spineFingerprint: ShapeFingerprint
}

export interface LayoutFeatures {
  M: number
  N: number
  maxShare: number
  cv: number
}

export const LIMB_STEP = 112
export const MIN_GAP = 58
export const SPINE_STEP = 188

const EPS = 1e-6
const FORK_AT = 6
const FORK_OFFSET = 76

type Pt = { x: number; y: number }
type Rng = () => number

interface ModPlan {
  key: string
  id: string
  topics: string[]
  layerRank: Map<string, number>
}

export function hashString(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = Math.imul(h, 31) + s.charCodeAt(i)
  return h >>> 0
}

export function mulberry32(seed: number): Rng {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) | 0
    let t = Math.imul(a ^ (a >>> 15), 1 | a)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function sub(a: Pt, b: Pt): Pt {
  return { x: a.x - b.x, y: a.y - b.y }
}
function dist(a: Pt, b: Pt): number {
  return Math.hypot(a.x - b.x, a.y - b.y)
}
function angleOf(p: Pt): number {
  return Math.atan2(p.y, p.x)
}
function fromAngle(a: number, length: number): Pt {
  return { x: Math.cos(a) * length, y: Math.sin(a) * length }
}
function angleDiff(a: number, b: number): number {
  let d = a - b
  while (d > Math.PI) d -= Math.PI * 2
  while (d < -Math.PI) d += Math.PI * 2
  return d
}
function cross(a: Pt, b: Pt): number {
  return a.x * b.y - a.y * b.x
}
function samePoint(a: Pt, b: Pt): boolean {
  return Math.abs(a.x - b.x) < 0.05 && Math.abs(a.y - b.y) < 0.05
}
function rot(p: Pt, theta: number): Pt {
  const c = Math.cos(theta)
  const s = Math.sin(theta)
  return { x: p.x * c - p.y * s, y: p.x * s + p.y * c }
}
function scaleAlong(p: Pt, axis: number, stretch: number): Pt {
  const c = Math.cos(axis)
  const s = Math.sin(axis)
  const u = p.x * c + p.y * s
  const v = -p.x * s + p.y * c
  const u2 = u * stretch
  return { x: u2 * c - v * s, y: u2 * s + v * c }
}
function deg(n: number): number {
  return (n * Math.PI) / 180
}

export function segmentsIntersectProper(a: Pt, b: Pt, c: Pt, d: Pt): boolean {
  if (samePoint(a, c) || samePoint(a, d) || samePoint(b, c) || samePoint(b, d)) return false
  const d1 = cross(sub(b, a), sub(c, a))
  const d2 = cross(sub(b, a), sub(d, a))
  const d3 = cross(sub(d, c), sub(a, c))
  const d4 = cross(sub(d, c), sub(b, c))
  if (((d1 > EPS && d2 < -EPS) || (d1 < -EPS && d2 > EPS)) && ((d3 > EPS && d4 < -EPS) || (d3 < -EPS && d4 > EPS))) {
    return true
  }
  const ab = sub(b, a)
  const cd = sub(d, c)
  if (Math.abs(cross(ab, cd)) > 1e-4) return false
  return pointInsideSegment(a, b, c) || pointInsideSegment(a, b, d) || pointInsideSegment(c, d, a) || pointInsideSegment(c, d, b)
}

function pointInsideSegment(a: Pt, b: Pt, p: Pt): boolean {
  if (Math.abs(cross(sub(b, a), sub(p, a))) > 1e-3) return false
  const dot = (p.x - a.x) * (b.x - a.x) + (p.y - a.y) * (b.y - a.y)
  if (dot <= EPS) return false
  const len2 = (b.x - a.x) ** 2 + (b.y - a.y) ** 2
  return dot < len2 - EPS
}

function modulePlans(domainId: string, tree: KnowledgeTree): ModPlan[] {
  const layerRank = new Map<string, number>()
  const orderIndex = new Map<string, number>()
  tree.layers.forEach((layer, li) => {
    layer.nodes.forEach((node, ni) => {
      layerRank.set(node.key, li)
      orderIndex.set(node.key, li * 1000 + ni)
    })
  })
  const requires = new Map<string, string[]>()
  for (const layer of tree.layers) {
    for (const node of layer.nodes) requires.set(node.key, node.requires ?? [])
  }
  return resolveGraphModules(tree)
    .modules.slice()
    .sort((a, b) => (a.order ?? 0) - (b.order ?? 0) || a.key.localeCompare(b.key))
    .map((mod) => ({
      key: mod.key,
      id: `module:${domainId}:${mod.key}`,
      topics: topoTopics(
        mod.nodes.filter((k) => layerRank.has(k)),
        requires,
        layerRank,
        orderIndex
      ),
      layerRank,
    }))
}

function topoTopics(
  keys: string[],
  requires: Map<string, string[]>,
  layerRank: Map<string, number>,
  orderIndex: Map<string, number>
): string[] {
  const set = new Set(keys)
  const indeg = new Map<string, number>()
  const children = new Map<string, string[]>()
  for (const k of keys) {
    indeg.set(k, 0)
    children.set(k, [])
  }
  for (const k of keys) {
    for (const req of requires.get(k) ?? []) {
      if (!set.has(req) || req === k) continue
      indeg.set(k, (indeg.get(k) ?? 0) + 1)
      children.get(req)!.push(k)
    }
  }
  const sortKeys = (list: string[]) =>
    list.sort(
      (a, b) =>
        (layerRank.get(a)! - layerRank.get(b)!) || (orderIndex.get(a)! - orderIndex.get(b)!) || a.localeCompare(b)
    )
  const pending = new Set(keys.filter((k) => (indeg.get(k) ?? 0) === 0))
  const out: string[] = []
  let steps = 0
  while (pending.size && steps++ <= keys.length) {
    const next = sortKeys([...pending])[0]!
    pending.delete(next)
    out.push(next)
    for (const child of children.get(next) ?? []) {
      const d = (indeg.get(child) ?? 1) - 1
      indeg.set(child, d)
      if (d === 0) pending.add(child)
    }
  }
  if (out.length < keys.length) out.push(...sortKeys(keys.filter((k) => !out.includes(k))))
  return out
}

export function layoutFeaturesOf(tree: KnowledgeTree): LayoutFeatures {
  const sizes = modulePlans(tree.domainId || 'd', tree).map((p) => p.topics.length)
  const M = sizes.length
  const N = sizes.reduce((s, n) => s + n, 0)
  const maxShare = N === 0 ? 1 : Math.max(0, ...sizes) / N
  const mean = M === 0 ? 0 : N / M
  const variance = M === 0 ? 0 : sizes.reduce((s, n) => s + (n - mean) ** 2, 0) / M
  const cv = mean === 0 ? 0 : Math.sqrt(variance) / mean
  return { M, N, maxShare, cv }
}

export function archetypeCandidates(f: LayoutFeatures): Archetype[] {
  const out: Archetype[] = []
  if (f.M <= 1) out.push('chain')
  if (f.M === 2) out.push('twin')
  if (f.M >= 3 && f.maxShare >= 0.5) out.push('sickle')
  if (f.M >= 3 && f.M <= 5 && f.cv < 0.35) out.push('cross')
  if (f.M >= 6 && f.N / f.M <= 3) out.push('zigzag')
  if (out.length === 0) out.push('dipper')
  return out
}

interface Built {
  at: Map<string, Pt>
  segments: Array<[string, string]>
  spineIds: string[]
}

function linkLimb(built: Built, anchorId: string, topicIds: string[], atIndex: (i: number) => Pt, forkAt: (i: number) => Pt): void {
  const mainCount = Math.min(topicIds.length, FORK_AT)
  let prev = anchorId
  for (let i = 0; i < mainCount; i++) {
    const id = topicIds[i]!
    built.at.set(id, atIndex(i))
    built.segments.push([prev, id])
    prev = id
  }
  if (topicIds.length <= FORK_AT) return
  const forkIdx = Math.min(2, mainCount - 1)
  prev = topicIds[forkIdx]!
  for (let i = FORK_AT; i < topicIds.length; i++) {
    const id = topicIds[i]!
    built.at.set(id, forkAt(i))
    built.segments.push([prev, id])
    prev = id
  }
}

function placeMonotone(rootId: string, plans: ModPlan[], yOf: (index: number) => number, signOf: (index: number) => number): Built {
  const built: Built = { at: new Map(), segments: [], spineIds: [rootId] }
  built.at.set(rootId, { x: 0, y: 0 })
  plans.forEach((mod, i) => {
    const at = { x: SPINE_STEP * (i + 1), y: yOf(i) }
    built.at.set(mod.id, at)
    built.spineIds.push(mod.id)
    const prev = i === 0 ? rootId : plans[i - 1]!.id
    built.segments.push([prev, mod.id])
    const sign = signOf(i)
    const ids = mod.topics.map((key) => topicId(mod, key))
    linkLimb(
      built,
      mod.id,
      ids,
      (ti) => ({ x: at.x, y: at.y + sign * LIMB_STEP * (ti + 1) }),
      (ti) => ({ x: at.x + FORK_OFFSET, y: at.y + sign * LIMB_STEP * (ti - FORK_AT + 3) })
    )
  })
  return built
}

function topicId(mod: ModPlan, key: string): string {
  const domain = mod.id.split(':')[1] ?? ''
  return `topic:${domain}:${key}`
}

function placeRays(
  rootId: string,
  arms: ModPlan[][],
  angleOfArm: (arm: number) => number,
  side: (arm: number, mod: number) => number
): Built {
  const built: Built = { at: new Map(), segments: [], spineIds: [rootId] }
  built.at.set(rootId, { x: 0, y: 0 })
  arms.forEach((arm, ai) => {
    const angle = angleOfArm(ai)
    let prev = rootId
    arm.forEach((mod, mi) => {
      const radius = SPINE_STEP * (mi + 1)
      const at = fromAngle(angle, radius)
      built.at.set(mod.id, at)
      built.spineIds.push(mod.id)
      built.segments.push([prev, mod.id])
      prev = mod.id
      const dir = fromAngle(angle + side(ai, mi), 1)
      const normal = fromAngle(angle + side(ai, mi) + Math.PI / 2, 1)
      const ids = mod.topics.map((key) => topicId(mod, key))
      linkLimb(
        built,
        mod.id,
        ids,
        (ti) => ({
          x: at.x + dir.x * LIMB_STEP * (ti + 1),
          y: at.y + dir.y * LIMB_STEP * (ti + 1),
        }),
        (ti) => ({
          x: at.x + dir.x * LIMB_STEP * (ti - FORK_AT + 3) + normal.x * FORK_OFFSET,
          y: at.y + dir.y * LIMB_STEP * (ti - FORK_AT + 3) + normal.y * FORK_OFFSET,
        })
      )
    })
  })
  return built
}

function buildLocal(rootId: string, plans: ModPlan[], archetype: Archetype, rnd: Rng): Built {
  if (archetype === 'twin' && plans.length >= 2) {
    const spread = deg(28 + rnd() * 12)
    const base = rnd() * Math.PI * 2
    return placeRays(rootId, [[plans[0]!], [plans[1]!]], (arm) => base + (arm === 0 ? -spread / 2 : spread / 2), () => 0)
  }
  if (archetype === 'cross') {
    const armCount = plans.length >= 4 ? 4 : Math.max(1, plans.length)
    const arms: ModPlan[][] = Array.from({ length: armCount }, () => [])
    if (plans.length === 5) {
      arms[0] = [plans[0]!, plans[1]!]
      arms[1] = [plans[2]!]
      arms[2] = [plans[3]!]
      arms[3] = [plans[4]!]
    } else {
      plans.forEach((mod, i) => arms[i]?.push(mod))
    }
    const base = rnd() * Math.PI * 2
    return placeRays(
      rootId,
      arms,
      (arm) => base + (arm * Math.PI * 2) / armCount,
      (_arm, mod) => (mod % 2 === 0 ? deg(32) : deg(-32))
    )
  }
  if (archetype === 'sickle') {
    let dom = 0
    plans.forEach((p, i) => {
      if (p.topics.length > plans[dom]!.topics.length) dom = i
    })
    const built: Built = { at: new Map(), segments: [], spineIds: [rootId] }
    built.at.set(rootId, { x: 0, y: 0 })
    const dominant = plans[dom]!
    built.at.set(dominant.id, { x: SPINE_STEP, y: 0 })
    built.spineIds.push(dominant.id)
    built.segments.push([rootId, dominant.id])
    const rise = 70 + rnd() * 50
    linkLimb(
      built,
      dominant.id,
      dominant.topics.map((key) => topicId(dominant, key)),
      (ti) => ({
        x: SPINE_STEP + LIMB_STEP * (ti + 1),
        y: rise * Math.sin(Math.min((ti + 1) * 0.42, 2.4)),
      }),
      (ti) => ({
        x: SPINE_STEP + LIMB_STEP * (ti - FORK_AT + 3),
        y: rise * Math.sin(Math.min((ti - FORK_AT + 3) * 0.42, 2.4)) + FORK_OFFSET,
      })
    )
    let prev = rootId
    let k = 1
    for (let i = 0; i < plans.length; i++) {
      if (i === dom) continue
      const mod = plans[i]!
      const at = { x: -SPINE_STEP * k, y: (k % 2 === 0 ? -1 : 1) * 36 }
      k++
      built.at.set(mod.id, at)
      built.spineIds.push(mod.id)
      built.segments.push([prev, mod.id])
      prev = mod.id
      const sign = at.y >= 0 ? 1 : -1
      linkLimb(
        built,
        mod.id,
        mod.topics.map((key) => topicId(mod, key)),
        (ti) => ({ x: at.x, y: at.y + sign * LIMB_STEP * (ti + 1) }),
        (ti) => ({ x: at.x - FORK_OFFSET, y: at.y + sign * LIMB_STEP * (ti - FORK_AT + 3) })
      )
    }
    return built
  }
  if (archetype === 'zigzag') {
    const amp = 80 + rnd() * 36
    return placeMonotone(
      rootId,
      plans,
      (i) => (i % 2 === 0 ? amp : -amp),
      (i) => (i % 2 === 0 ? 1 : -1)
    )
  }
  if (archetype === 'dipper') {
    const bowlCount = Math.min(plans.length, plans.length >= 5 ? 4 : Math.min(3, plans.length))
    const order = [...plans.slice(bowlCount), ...plans.slice(0, bowlCount)]
    const amp = 64 + rnd() * 28
    return placeMonotone(
      rootId,
      order,
      (i) => {
        const bowlStart = order.length - bowlCount
        if (i < bowlStart) return ((i + 1) / Math.max(1, bowlStart)) * 28
        const t = (i - bowlStart) / Math.max(1, bowlCount - 1)
        return 28 + Math.sin(t * Math.PI) * amp
      },
      () => 1
    )
  }
  return placeMonotone(
    rootId,
    plans,
    (i) => Math.sin(i * 0.9) * 36,
    () => 1
  )
}

function transformPoint(p: Pt, theta: number, axis: number, stretch: number): Pt {
  return scaleAlong(rot(p, theta), axis, stretch)
}

function minPair(pts: Pt[]): number {
  let m = Infinity
  for (let i = 0; i < pts.length; i++) {
    for (let j = i + 1; j < pts.length; j++) m = Math.min(m, dist(pts[i]!, pts[j]!))
  }
  return m
}

function layoutClean(at: Map<string, Pt>, segments: Array<[string, string]>): boolean {
  const pts = [...at.values()]
  if (pts.length > 1 && minPair(pts) < MIN_GAP * 0.8) return false
  const lines = segments.map(([a, b]) => [at.get(a)!, at.get(b)!] as [Pt, Pt])
  for (let i = 0; i < lines.length; i++) {
    for (let j = i + 1; j < lines.length; j++) {
      if (segmentsIntersectProper(lines[i]![0], lines[i]![1], lines[j]![0], lines[j]![1])) return false
    }
  }
  return true
}

function convexHull(pts: Pt[]): Pt[] {
  const p = pts.slice().sort((a, b) => a.x - b.x || a.y - b.y)
  if (p.length <= 2) return p
  const turn = (o: Pt, a: Pt, b: Pt) => (a.x - o.x) * (b.y - o.y) - (a.y - o.y) * (b.x - o.x)
  const lower: Pt[] = []
  for (const pt of p) {
    while (lower.length >= 2 && turn(lower[lower.length - 2]!, lower[lower.length - 1]!, pt) <= 0) lower.pop()
    lower.push(pt)
  }
  const upper: Pt[] = []
  for (let i = p.length - 1; i >= 0; i--) {
    const pt = p[i]!
    while (upper.length >= 2 && turn(upper[upper.length - 2]!, upper[upper.length - 1]!, pt) <= 0) upper.pop()
    upper.push(pt)
  }
  lower.pop()
  upper.pop()
  return lower.concat(upper)
}

function hullArea(h: Pt[]): number {
  let a = 0
  for (let i = 0; i < h.length; i++) {
    const p = h[i]!
    const q = h[(i + 1) % h.length]!
    a += p.x * q.y - q.x * p.y
  }
  return Math.abs(a) / 2
}

function center(pts: Pt[]): Pt {
  const n = pts.length || 1
  let x = 0
  let y = 0
  for (const p of pts) {
    x += p.x
    y += p.y
  }
  return { x: x / n, y: y / n }
}

function principalAngle(pts: Pt[]): number {
  const c = center(pts)
  let xx = 0
  let xy = 0
  let yy = 0
  for (const p of pts) {
    const x = p.x - c.x
    const y = p.y - c.y
    xx += x * x
    xy += x * y
    yy += y * y
  }
  return 0.5 * Math.atan2(2 * xy, xx - yy)
}

function collectTurns(pts: Pt[], segments: Array<[Pt, Pt]>): number[] {
  const neigh = new Map<number, number[]>()
  const find = (p: Pt) => {
    for (let i = 0; i < pts.length; i++) if (samePoint(pts[i]!, p)) return i
    return -1
  }
  for (const [a, b] of segments) {
    const i = find(a)
    const j = find(b)
    if (i < 0 || j < 0) continue
    neigh.set(i, [...(neigh.get(i) ?? []), j])
    neigh.set(j, [...(neigh.get(j) ?? []), i])
  }
  const turns: number[] = []
  for (const [i, list] of neigh) {
    if (list.length !== 2) continue
    const a = pts[list[0]!]!
    const b = pts[list[1]!]!
    const o = pts[i]!
    let interior = Math.abs(angleDiff(angleOf(sub(a, o)), angleOf(sub(b, o))))
    if (interior > Math.PI) interior = Math.PI * 2 - interior
    turns.push(Math.PI - interior)
  }
  return turns
}

export function spineFingerprint(stars: Pt[], segments: Array<[Pt, Pt]>, degrees: number[]): ShapeFingerprint {
  const pts = stars.length ? stars : [{ x: 0, y: 0 }]
  const c = center(pts)
  let xx = 0
  let xy = 0
  let yy = 0
  for (const p of pts) {
    const x = p.x - c.x
    const y = p.y - c.y
    xx += x * x
    xy += x * y
    yy += y * y
  }
  const tr = xx + yy
  const det = xx * yy - xy * xy
  const disc = Math.sqrt(Math.max(0, (tr * tr) / 4 - det))
  const l1 = tr / 2 + disc
  const l2 = Math.max(0, tr / 2 - disc)
  const aspect = l2 < 1e-6 ? 10 : Math.min(10, Math.sqrt(l1 / Math.max(l2, 1e-9)))
  const aligned = pts.map((p) => rot(sub(p, c), -principalAngle(pts)))
  const hull = convexHull(aligned)
  let minX = Infinity
  let maxX = -Infinity
  let minY = Infinity
  let maxY = -Infinity
  for (const p of aligned) {
    minX = Math.min(minX, p.x)
    maxX = Math.max(maxX, p.x)
    minY = Math.min(minY, p.y)
    maxY = Math.max(maxY, p.y)
  }
  const box = Math.max(0, maxX - minX) * Math.max(0, maxY - minY)
  const turns = new Array(6).fill(0) as number[]
  const nodeTurns = collectTurns(pts, segments)
  for (const t of nodeTurns) {
    const bin = Math.max(0, Math.min(5, Math.floor((t / Math.PI) * 6)))
    turns[bin] = (turns[bin] ?? 0) + 1
  }
  const degreeHistogram = [0, 0, 0, 0]
  for (const d of degrees) degreeHistogram[Math.max(1, Math.min(4, d)) - 1]! += 1
  const denom = degrees.length || 1
  return {
    aspect,
    hullFill: box < EPS ? 0 : Math.min(1, hullArea(hull) / box),
    turnHistogram: turns.map((n) => (nodeTurns.length === 0 ? 0 : n / nodeTurns.length)),
    degreeHistogram: degreeHistogram.map((n) => n / denom),
  }
}

export function shapeDistance(a: ShapeFingerprint, b: ShapeFingerprint): number {
  let turn = 0
  for (let i = 0; i < 6; i++) turn += Math.abs((a.turnHistogram[i] ?? 0) - (b.turnHistogram[i] ?? 0))
  let degSum = 0
  for (let i = 0; i < 4; i++) degSum += Math.abs((a.degreeHistogram[i] ?? 0) - (b.degreeHistogram[i] ?? 0))
  return (Math.abs(a.aspect - b.aspect) / 10) * 1.2 + Math.abs(a.hullFill - b.hullFill) * 1.4 + turn * 0.8 + degSum * 0.6
}

function labelAngles(ids: string[], at: Map<string, Pt>, segments: Array<[string, string]>): Map<string, number> {
  const out = new Map<string, number>()
  for (const id of ids) {
    let x = 0
    let y = 0
    for (const [a, b] of segments) {
      const other = a === id ? b : b === id ? a : ''
      if (!other || !at.has(other) || !at.has(id)) continue
      const v = sub(at.get(other)!, at.get(id)!)
      const l = Math.hypot(v.x, v.y) || 1
      x += v.x / l
      y += v.y / l
    }
    out.set(id, Math.hypot(x, y) < 0.2 ? Math.PI / 2 : angleOf({ x: -x, y: -y }))
  }
  return out
}

export function buildConstellation(domainId: string, tree: KnowledgeTree, salt = 0): ConstellationLayout {
  const plans = modulePlans(domainId, tree)
  const features = layoutFeaturesOf({ ...tree, domainId })
  const candidates = archetypeCandidates(features)
  const spineRng = mulberry32(hashString(`${domainId}:spine:${salt}`))
  const archetype = candidates.length === 1 ? candidates[0]! : candidates[Math.floor(spineRng() * candidates.length)]!
  const rootId = `domain:${domainId}`
  const local = buildLocal(rootId, plans, archetype, spineRng)
  const xRng = mulberry32(hashString(`${domainId}:xform:${salt}`))
  const theta = xRng() * Math.PI * 2
  const stretchPick = 0.7 + xRng() * 0.6
  const axis = xRng() * Math.PI * 2
  const apply = (stretch: number) => {
    const at = new Map<string, Pt>()
    for (const [id, p] of local.at) at.set(id, transformPoint(p, theta, axis, stretch))
    return at
  }
  let at = apply(stretchPick)
  if (!layoutClean(at, local.segments)) at = apply(1)

  const angles = labelAngles([...at.keys()], at, local.segments)
  const roleOf = (id: string): StarPlacement['role'] => (id === rootId ? 'domain' : id.startsWith('module:') ? 'module' : 'topic')
  const layerOf = new Map<string, number>()
  for (const mod of plans) {
    for (const key of mod.topics) layerOf.set(topicId(mod, key), mod.layerRank.get(key) ?? 1)
  }
  const stars: StarPlacement[] = []
  const root = at.get(rootId) ?? { x: 0, y: 0 }
  for (const [id, p] of at) {
    const role = roleOf(id)
    const count = plans.find((m) => m.id === id)?.topics.length ?? 0
    stars.push({
      id,
      role,
      x: p.x,
      y: p.y,
      magnitude: role === 'domain' ? 1 : role === 'module' ? (count >= 4 ? 2 : 3) : (layerOf.get(id) ?? 1) === 0 ? 4 : 5,
      labelAngle: angles.get(id) ?? Math.PI / 2,
    })
  }
  stars.sort((a, b) => roleRank(a.role) - roleRank(b.role) || a.id.localeCompare(b.id))
  const segments: ConstellationSegment[] = local.segments.map(([from, to]) => ({
    from,
    to,
    kind: from.startsWith('topic:') || to.startsWith('topic:') ? 'limb' : 'spine',
  }))
  let radius = 24
  for (const s of stars) radius = Math.max(radius, Math.hypot(s.x - root.x, s.y - root.y))
  const spinePts = local.spineIds.map((id) => at.get(id)!)
  const spineSegs: Array<[Pt, Pt]> = local.segments
    .filter(([a, b]) => local.spineIds.includes(a) && local.spineIds.includes(b))
    .map(([a, b]) => [at.get(a)!, at.get(b)!])
  const degMap = new Map<string, number>(local.spineIds.map((id) => [id, 0]))
  for (const [a, b] of local.segments) {
    if (!degMap.has(a) || !degMap.has(b)) continue
    degMap.set(a, (degMap.get(a) ?? 0) + 1)
    degMap.set(b, (degMap.get(b) ?? 0) + 1)
  }
  return {
    stars,
    segments,
    radius,
    archetype,
    spineFingerprint: spineFingerprint(
      spinePts,
      spineSegs,
      local.spineIds.map((id) => degMap.get(id) ?? 0)
    ),
  }
}

function roleRank(role: StarPlacement['role']): number {
  if (role === 'domain') return 0
  if (role === 'module') return 1
  return 2
}
