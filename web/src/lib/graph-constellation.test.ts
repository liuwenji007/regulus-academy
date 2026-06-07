import { describe, expect, it } from 'vitest'
import {
  constellationSeparationLength,
  groupDomainsIntoConstellations,
  layoutDomainCentersByConstellation,
  topicRootKey,
} from './graph-constellation'

describe('topicRootKey', () => {
  it('maps go/python slugs to theme roots', () => {
    expect(topicRootKey('go', 'Go 入门')).toBe('go')
    expect(topicRootKey('golang', '')).toBe('go')
    expect(topicRootKey('python', 'Python')).toBe('python')
    expect(topicRootKey('', 'Python 进阶')).toBe('python')
  })

  it('falls back to slug head or other', () => {
    expect(topicRootKey('kubernetes-basics', 'K8s')).toBe('kubernetes')
    expect(topicRootKey('', '')).toBe('other')
  })
})

describe('groupDomainsIntoConstellations', () => {
  it('clusters related domains under one constellation', () => {
    const groups = groupDomainsIntoConstellations([
      { domainId: 'a', name: 'Go 基础', slug: 'go', nodeCount: 12 },
      { domainId: 'b', name: 'Go 并发', slug: 'go-advanced', nodeCount: 18 },
      { domainId: 'c', name: 'Python 入门', slug: 'python', nodeCount: 8 },
    ])
    expect(groups).toHaveLength(2)
    expect(groups.find((g) => g.key === 'go')?.nodeCount).toBe(30)
    const go = groups.find((g) => g.key === 'go')
    const py = groups.find((g) => g.key === 'python')
    expect(go?.domainIds).toEqual(['a', 'b'])
    expect(py?.domainIds).toEqual(['c'])
  })
})

describe('layoutDomainCentersByConstellation', () => {
  it('places domains in separate sectors by theme group', () => {
    const groups = groupDomainsIntoConstellations([
      { domainId: 'a', name: 'Go 基础', slug: 'go', nodeCount: 20 },
      { domainId: 'b', name: 'Go 并发', slug: 'go-advanced', nodeCount: 25 },
      { domainId: 'c', name: 'Python 入门', slug: 'python', nodeCount: 10 },
    ])
    const layout = layoutDomainCentersByConstellation(groups)
    const a = layout.get('a')!
    const b = layout.get('b')!
    const c = layout.get('c')!
    const goCx = (a.x + b.x) / 2
    const goCy = (a.y + b.y) / 2
    const withinGo = Math.hypot(a.x - b.x, a.y - b.y)
    const crossToPy = Math.hypot(c.x - goCx, c.y - goCy)
    expect(withinGo).toBeLessThan(crossToPy)
    expect(Math.hypot(a.x, a.y)).toBeGreaterThan(Math.hypot(c.x, c.y))
    for (const p of [a, b, c]) {
      expect(Math.hypot(p.x, p.y)).toBeGreaterThan(300)
    }
  })
})

describe('constellationSeparationLength', () => {
  it('uses longer repulsion for heavier cross-constellation pairs', () => {
    const light = { key: 'py', label: 'Python', domainIds: ['c'], nodeCount: 10 }
    const heavy = { key: 'go', label: 'Go', domainIds: ['a', 'b'], nodeCount: 45 }
    const cross = constellationSeparationLength(heavy, light)
    const within = constellationSeparationLength(heavy, heavy)
    expect(cross).toBeGreaterThan(within)
    expect(cross).toBeGreaterThan(constellationSeparationLength(light, light))
  })
})
