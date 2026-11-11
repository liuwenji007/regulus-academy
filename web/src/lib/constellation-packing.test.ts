import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import { buildConstellation, shapeDistance } from './constellation-layout'
import {
  GALAXY_ARMS,
  GALAXY_ARM_RATIO,
  GALAXY_CORE,
  armHalfWidth,
  PACK_GAP,
  SIMILAR_THRESHOLD,
  UNCLASSIFIED_FIELD,
  armSpans,
  fieldNameAnchors,
  galaxyArmPoint,
  galaxyOuter,
  packConstellations,
  type PackCourse,
} from './constellation-packing'
import { buildGalaxyDust } from './constellation-render'

function course(spec: number[], domainId: string, name: string, createdAt: string, slug: string): PackCourse {
  const modules = spec.map((count, i) => ({
    key: `m${i}`,
    label: `${name}${i}`,
    order: i + 1,
    nodes: Array.from({ length: count }, (_, j) => `${domainId}-m${i}n${j}`),
  }))
  const tree: KnowledgeTree = {
    domainId,
    domainName: name,
    layers: [
      {
        key: 'intro',
        label: '入门',
        time: '',
        goal: '',
        nodes: modules.flatMap((m) => m.nodes.map((key) => ({ key, title: key }))),
      },
    ],
    modules,
  }
  return { domainId, slug, name, createdAt, tree }
}

describe('packConstellations', () => {
  const courses = [
    course([3, 3, 3, 3], 'a', 'Go 基础', '2024-01-01', 'go'),
    course([2, 2, 2, 2, 2, 2], 'b', 'Go 并发', '2024-02-01', 'go-advanced'),
    course([5, 4], 'c', 'Python 入门', '2024-03-01', 'python'),
    course([4, 4, 1], 'd', '架构入门', '2024-04-01', 'arch'),
  ]

  it('keeps earlier salts stable when a newer course is added', () => {
    const before = packConstellations(courses.slice(0, 2))
    const after = packConstellations(courses)
    expect(after.find((item) => item.domainId === 'a')?.salt).toBe(before[0]?.salt)
    expect(after.find((item) => item.domainId === 'b')?.salt).toBe(before[1]?.salt)
  })

  it('picks the first salt that clears the similarity threshold', () => {
    const packed = packConstellations(courses.slice(0, 2))
    const first = packed[0]!
    const second = packed[1]!
    let expected = 0
    for (let salt = 0; salt < 8; salt++) {
      const layout = buildConstellation('b', courses[1]!.tree, salt)
      const distance = shapeDistance(layout.spineFingerprint, first.layout.spineFingerprint)
      if (distance > SIMILAR_THRESHOLD) {
        expected = salt
        break
      }
      expected = salt
    }
    expect(second.salt).toBe(expected)
  })

  it('keeps earlier centers fixed when a newer course is added', () => {
    const before = packConstellations(courses.slice(0, 3))
    const after = packConstellations(courses)
    for (const item of before) {
      const next = after.find((other) => other.domainId === item.domainId)!
      expect(next.cx).toBe(item.cx)
      expect(next.cy).toBe(item.cy)
    }
  })

  it('lays many courses along the galaxy arms, outside the core', () => {
    const many = Array.from({ length: 36 }, (_, i) =>
      course([3, 2, 4].slice(0, 1 + (i % 3)), `k${i}`, `课程${i}`, `2024-01-${String(i + 1).padStart(2, '0')}`, `topic${i}`)
    )
    const packed = packConstellations(many)
    for (const item of packed) {
      const r = Math.hypot(item.cx, item.cy)
      expect(r).toBeGreaterThanOrEqual(GALAXY_CORE - 1)
      let nearest = Infinity
      for (let arm = 0; arm < GALAXY_ARMS; arm++) {
        const theta0 = (arm * 2 * Math.PI) / GALAXY_ARMS
        for (let s = GALAXY_CORE; s < r + 2000; s += 20) {
          const p = galaxyArmPoint(theta0, s)
          nearest = Math.min(nearest, Math.hypot(p.x - item.cx, p.y - item.cy))
        }
      }
      // 偏移按中心线上的半径取半宽，偏移后星座自身半径会略小
      expect(nearest).toBeLessThanOrEqual(armHalfWidth(r) * 1.2 + 25)
    }
  })

  it('puts courses of one field on the same arm and tiers by mastery', () => {
    const inner = course([3, 3], 'go', 'Go', '2024-01-01', 'go')
    inner.field = '编程语言'
    inner.nodeTotal = 10
    inner.completed = 9
    const sibling = course([2, 2], 'rust', 'Rust', '2024-02-01', 'rust')
    sibling.field = '编程语言'
    sibling.nodeTotal = 10
    sibling.completed = 8
    const outer = course([2, 2], 'eng', '英语', '2024-01-15', 'english')
    outer.field = '语言与沟通'
    outer.nodeTotal = 10
    outer.completed = 1
    const packed = packConstellations([inner, sibling, outer])
    const go = packed.find((item) => item.domainId === 'go')!
    const rust = packed.find((item) => item.domainId === 'rust')!
    const eng = packed.find((item) => item.domainId === 'eng')!
    expect(go.field).toBe('编程语言')
    expect(Math.abs(go.theta0 - rust.theta0)).toBeLessThan(0.001)
    expect(Math.abs(go.theta0 - eng.theta0)).toBeGreaterThan(0.2)
    expect(Math.hypot(go.cx, go.cy)).toBeLessThan(Math.hypot(eng.cx, eng.cy))
  })

  it('spreads the arms evenly over the fields that are present', () => {
    const list = ['编程语言', '语言与沟通', '人文与通识'].map((field, i) => {
      const item = course([3], `f${i}`, field, `2024-0${i + 1}-01`, `f${i}`)
      item.field = field
      return item
    })
    const angles = packConstellations(list).map((item) => item.theta0).sort((a, b) => a - b)
    expect(angles[0]).toBeCloseTo(0, 5)
    expect(angles[1]).toBeCloseTo((2 * Math.PI) / 3, 5)
    expect(angles[2]).toBeCloseTo((4 * Math.PI) / 3, 5)
  })

  it('does not move existing courses when a course joins an existing field', () => {
    const a = course([3], 'go', 'Go', '2024-01-01', 'go')
    a.field = '编程语言'
    const b = course([3], 'ai', 'AI', '2024-02-01', 'ai')
    b.field = 'AI 与数据'
    const before = packConstellations([a, b])
    const extra = course([3, 2], 'rust', 'Rust', '2024-03-01', 'rust')
    extra.field = '编程语言'
    const after = packConstellations([a, b, extra])
    for (const item of before) {
      const next = after.find((other) => other.domainId === item.domainId)!
      expect(next.cx).toBe(item.cx)
      expect(next.cy).toBe(item.cy)
      expect(next.theta0).toBe(item.theta0)
    }
  })

  it('leaves a dark lane between adjacent field arms that widens with radius', () => {
    for (const r of [2000, 5000, 12000]) {
      const a = galaxyArmPoint(0, r)
      const behind = ((GALAXY_ARMS - 1) * 2 * Math.PI) / GALAXY_ARMS
      const b = galaxyArmPoint(behind, r * GALAXY_ARM_RATIO)
      expect(Math.atan2(b.y, b.x)).toBeCloseTo(Math.atan2(a.y, a.x), 5)
      const gap = Math.hypot(a.x - b.x, a.y - b.y) - armHalfWidth(r) - armHalfWidth(r * GALAXY_ARM_RATIO)
      expect(gap).toBeGreaterThan(r * 0.15)
    }
  })

  it('grows an arm by area, not length, as one field gains courses', () => {
    const extentOf = (n: number) => {
      const list = Array.from({ length: n }, (_, i) => {
        const item = course([3, 2, 4].slice(0, 1 + (i % 3)), `g${i}`, `课程${i}`, `2024-01-01T00:${String(i).padStart(2, '0')}`, `g${i}`)
        item.field = 'AI 与数据'
        return item
      })
      return galaxyOuter(packConstellations(list))
    }
    expect(extentOf(48) / extentOf(12)).toBeLessThan(2.8)
  })

  it('writes each field name on its own arm, just past its outermost constellation', () => {
    const go = course([3, 3, 3], 'go', 'Go', '2024-01-01', 'go')
    go.field = '编程语言'
    const rust = course([3, 3], 'rust', 'Rust', '2024-02-01', 'rust')
    rust.field = '编程语言'
    const ai = course([2, 2], 'ai', 'AI', '2024-03-01', 'ai')
    ai.field = 'AI 与数据'
    const packed = packConstellations([go, rust, ai])
    const anchors = fieldNameAnchors(packed)
    expect(anchors.map((item) => item.field).sort()).toEqual(['AI 与数据', '编程语言'].sort())
    for (const anchor of anchors) {
      const members = packed.filter((item) => item.field === anchor.field)
      const r = Math.hypot(anchor.x, anchor.y)
      const onArm = galaxyArmPoint(members[0]!.theta0, r)
      expect(Math.hypot(onArm.x - anchor.x, onArm.y - anchor.y)).toBeLessThan(1)
      expect(r).toBeGreaterThanOrEqual(Math.max(...members.map((item) => Math.hypot(item.cx, item.cy))) - 1)
      expect(r).toBeLessThan(galaxyOuter(packed) * 1.3)
      for (const item of packed) {
        expect(Math.hypot(item.cx - anchor.x, item.cy - anchor.y)).toBeGreaterThanOrEqual(item.layout.radius + 200)
      }
    }
  })

  it('shares one arm and one name among custom fields', () => {
    const fields = ['烹饪', '摄影', '园艺']
    const custom = fields.map((field, i) => {
      const item = course([3], `c${i}`, field, `2024-0${i + 1}-01`, `c${i}`)
      item.field = field
      return item
    })
    const loose = course([3], 'u', '杂项', '2024-05-01', 'u')
    const packed = packConstellations([...custom, loose])
    const customArms = new Set(packed.filter((item) => fields.includes(item.field)).map((item) => item.theta0))
    expect(customArms.size).toBe(1)
    expect(packed.find((item) => item.domainId === 'u')!.theta0).not.toBe([...customArms][0])
    const names = fieldNameAnchors(packed).map((item) => item.field)
    expect(names).toContain('烹饪、摄影 等')
    expect(names).toContain(UNCLASSIFIED_FIELD)
  })

  it('spans each arm exactly over its constellations', () => {
    const packed = packConstellations(courses)
    for (const span of armSpans(packed)) {
      const onArm = packed.filter((item) => item.theta0 === span.theta0)
      const inner = Math.min(...onArm.map((item) => Math.hypot(item.cx, item.cy) - item.layout.radius))
      const outer = Math.max(...onArm.map((item) => Math.hypot(item.cx, item.cy) + item.layout.radius))
      expect(span.start).toBeCloseTo(inner, 5)
      expect(span.end).toBeCloseTo(outer, 5)
      expect(span.stars).toHaveLength(onArm.length)
    }
  })

  it('only glows where there are constellations', () => {
    const big = Array.from({ length: 20 }, (_, i) => {
      const item = course([3, 3, 2], `b${i}`, `课程${i}`, `2024-01-01T00:${String(i).padStart(2, '0')}`, `b${i}`)
      item.field = 'AI 与数据'
      return item
    })
    const lone = course([3], 'h', '历史', '2024-06-01', 'h')
    lone.field = '人文与通识'
    const packed = packConstellations([...big, lone])
    const dust = buildGalaxyDust(galaxyOuter(packed), armSpans(packed))
    const loneArm = armSpans(packed).find((span) => span.theta0 === packed.find((item) => item.domainId === 'h')!.theta0)!
    const reach = loneArm.end + armHalfWidth(loneArm.end) * 1.8
    const bigArm = armSpans(packed).find((span) => span.theta0 !== loneArm.theta0)!
    expect(bigArm.end).toBeGreaterThan(reach * 1.5)
    for (let s = reach + 200; s < galaxyOuter(packed) * 1.1; s += 200) {
      const p = galaxyArmPoint(loneArm.theta0, s)
      const lit = dust.haze.filter((h) => h.a > 0.005 && Math.hypot(h.x - p.x, h.y - p.y) < armHalfWidth(s) * 0.5)
      const fromBig = (h: { x: number; y: number }) =>
        packed.some((item) => item.domainId !== 'h' && Math.hypot(item.cx - h.x, item.cy - h.y) < item.layout.radius * 2 + 400)
      expect(lit.every(fromBig)).toBe(true)
    }
  })

  it('keeps constellation circles from overlapping', () => {
    const packed = packConstellations(courses)
    for (let i = 0; i < packed.length; i++) {
      for (let j = i + 1; j < packed.length; j++) {
        const a = packed[i]!
        const b = packed[j]!
        const dist = Math.hypot(a.cx - b.cx, a.cy - b.cy)
        expect(dist).toBeGreaterThanOrEqual(a.layout.radius + b.layout.radius + PACK_GAP - 0.5)
      }
    }
  })
})
