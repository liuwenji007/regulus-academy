import { describe, expect, it } from 'vitest'
import type { KnowledgeTree } from './api'
import {
  archetypeCandidates,
  buildConstellation,
  layoutFeaturesOf,
  MIN_GAP,
  segmentsIntersectProper,
  type ConstellationLayout,
} from './constellation-layout'

function course(spec: number[], domainId = 'course'): KnowledgeTree {
  const modules = spec.map((count, i) => ({
    key: `m${i}`,
    label: `模块${i}`,
    order: i + 1,
    nodes: Array.from({ length: count }, (_, j) => `m${i}n${j}`),
  }))
  return {
    domainId,
    domainName: '测试课程',
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
}

function starAt(layout: ConstellationLayout, id: string) {
  const star = layout.stars.find((s) => s.id === id)
  if (!star) throw new Error(`missing ${id}`)
  return star
}

function assertConstellation(layout: ConstellationLayout) {
  const byId = new Map(layout.stars.map((s) => [s.id, s]))
  expect(layout.segments).toHaveLength(layout.stars.length - 1)
  const deg = new Map<string, number>()
  for (const s of layout.stars) deg.set(s.id, 0)
  for (const seg of layout.segments) {
    deg.set(seg.from, (deg.get(seg.from) ?? 0) + 1)
    deg.set(seg.to, (deg.get(seg.to) ?? 0) + 1)
    expect(byId.has(seg.from)).toBe(true)
    expect(byId.has(seg.to)).toBe(true)
  }
  for (const [id, d] of deg) {
    if (id.startsWith('domain:')) expect(d).toBeLessThanOrEqual(4)
    else expect(d).toBeLessThanOrEqual(3)
    expect(d).toBeGreaterThan(0)
  }
  const seen = new Set<string>([layout.stars[0]!.id])
  const queue = [layout.stars[0]!.id]
  const adj = new Map<string, string[]>()
  for (const seg of layout.segments) {
    const a = adj.get(seg.from) ?? []
    a.push(seg.to)
    adj.set(seg.from, a)
    const b = adj.get(seg.to) ?? []
    b.push(seg.from)
    adj.set(seg.to, b)
  }
  while (queue.length) {
    const id = queue.pop()!
    for (const next of adj.get(id) ?? []) {
      if (seen.has(next)) continue
      seen.add(next)
      queue.push(next)
    }
  }
  expect(seen.size).toBe(layout.stars.length)

  for (let i = 0; i < layout.stars.length; i++) {
    for (let j = i + 1; j < layout.stars.length; j++) {
      const a = layout.stars[i]!
      const b = layout.stars[j]!
      const d = Math.hypot(a.x - b.x, a.y - b.y)
      expect(d).toBeGreaterThanOrEqual(MIN_GAP * 0.8 - 1e-6)
    }
  }
  const segs = layout.segments.map((seg) => ({
    a: byId.get(seg.from)!,
    b: byId.get(seg.to)!,
  }))
  for (let i = 0; i < segs.length; i++) {
    for (let j = i + 1; j < segs.length; j++) {
      const hit = segmentsIntersectProper(segs[i]!.a, segs[i]!.b, segs[j]!.a, segs[j]!.b)
      if (hit) {
        throw new Error(
          `${layout.segments[i]!.from}->${layout.segments[i]!.to} x ${layout.segments[j]!.from}->${layout.segments[j]!.to}`
        )
      }
    }
  }
}

describe('archetypeCandidates', () => {
  it('picks chain, twin, cross, sickle, zigzag, or dipper from structure', () => {
    expect(archetypeCandidates(layoutFeaturesOf(course([4])))).toEqual(['chain'])
    expect(archetypeCandidates(layoutFeaturesOf(course([3, 3])))).toEqual(['twin'])
    expect(archetypeCandidates(layoutFeaturesOf(course([4, 4, 4, 4])))).toEqual(['cross'])
    expect(archetypeCandidates(layoutFeaturesOf(course([10, 1, 1, 1])))).toEqual(['sickle'])
    expect(archetypeCandidates(layoutFeaturesOf(course([2, 2, 2, 2, 2, 2])))).toEqual(['zigzag'])
    expect(archetypeCandidates(layoutFeaturesOf(course([4, 4, 1])))).toEqual(['dipper'])
    expect(buildConstellation('arch', course([4, 4, 4, 4]), 0).archetype).toBe('cross')
    expect(buildConstellation('arch', course([2, 2, 2, 2, 2, 2]), 1).archetype).toBe('zigzag')
  })
})

describe('buildConstellation', () => {
  it('is deterministic for the same domain and salt', () => {
    const tree = course([3, 2, 4, 1])
    expect(buildConstellation('det', tree, 2)).toEqual(buildConstellation('det', tree, 2))
  })

  it('keeps a crossing-free tree with bounded degree and spacing', () => {
    const specs = [1, 2, 4, 6, 9].flatMap((modules) =>
      [2, 8, 14].map((per) => Array.from({ length: modules }, () => per))
    )
    for (const spec of specs) {
      try {
        assertConstellation(buildConstellation(`m${spec.length}n${spec[0]}`, course(spec), 0))
      } catch (err) {
        const detail = err instanceof Error ? err.message : String(err)
        throw new Error(`spec ${spec.join('x')} failed: ${detail}`)
      }
    }
  })

  it('does not move existing stars when the last module gains a node', () => {
    const before = buildConstellation('grow', course([3, 3]), 0)
    const after = buildConstellation('grow', course([3, 4]), 0)
    for (const star of before.stars) {
      const next = starAt(after, star.id)
      expect(next.x).toBe(star.x)
      expect(next.y).toBe(star.y)
    }
  })

  it('does not move the spine when any module gains a node', () => {
    const before = buildConstellation('spine', course([2, 2, 2, 2]), 0)
    const after = buildConstellation('spine', course([3, 2, 2, 2]), 0)
    for (const star of before.stars) {
      if (star.role === 'topic') continue
      const next = starAt(after, star.id)
      expect(next.x).toBe(star.x)
      expect(next.y).toBe(star.y)
    }
  })
})
