import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import { buildConstellation, shapeDistance } from './constellation-layout'
import {
  GALAXY_ARMS,
  GALAXY_ARM_WIDTH,
  GALAXY_CORE,
  PACK_GAP,
  SIMILAR_THRESHOLD,
  galaxyArmPoint,
  packConstellations,
  type PackCourse,
} from './constellation-packing'

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
        for (let s = GALAXY_CORE; s < r + 2000; s += 20) {
          const p = galaxyArmPoint(arm, s)
          nearest = Math.min(nearest, Math.hypot(p.x - item.cx, p.y - item.cy))
        }
      }
      expect(nearest).toBeLessThanOrEqual(GALAXY_ARM_WIDTH + 25)
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
