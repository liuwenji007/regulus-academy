import { describe, expect, it } from 'vitest'
import {
  domainCompletionRatio,
  moduleCompletionRatio,
  pathEdgeOpacity,
  pathSegmentOpacity,
  PENDING_NODE_OPACITY,
} from './graph-progress-visual'
import type { UserProgress } from './api'

function progress(status: string): UserProgress {
  return {
    userId: 'u1',
    domainId: 'd1',
    nodeKey: 'x',
    layer: 'intro',
    status,
    mastery: status === 'completed' ? 1 : 0,
  }
}

describe('moduleCompletionRatio', () => {
  const mod = { key: 'm1', nodes: ['a', 'b', 'c', 'd'] }
  const valid = new Set(['a', 'b', 'c', 'd'])

  it('returns 0 when nothing completed', () => {
    const map = new Map<string, UserProgress>()
    expect(moduleCompletionRatio(mod, map, valid)).toBe(0)
  })

  it('returns 0.5 at half completion', () => {
    const map = new Map<string, UserProgress>([
      ['a', progress('completed')],
      ['b', progress('completed')],
    ])
    expect(moduleCompletionRatio(mod, map, valid)).toBe(0.5)
  })

  it('returns 1 when all completed', () => {
    const map = new Map<string, UserProgress>([
      ['a', progress('completed')],
      ['b', progress('completed')],
      ['c', progress('completed')],
      ['d', progress('completed')],
    ])
    expect(moduleCompletionRatio(mod, map, valid)).toBe(1)
  })

  it('ignores keys outside valid set', () => {
    const map = new Map<string, UserProgress>([['a', progress('completed')]])
    expect(moduleCompletionRatio(mod, map, new Set(['a']))).toBe(1)
  })
})

describe('domainCompletionRatio', () => {
  it('aggregates across modules', () => {
    const modules = [
      { key: 'm1', nodes: ['a', 'b'] },
      { key: 'm2', nodes: ['c', 'd'] },
    ]
    const valid = new Set(['a', 'b', 'c', 'd'])
    const map = new Map<string, UserProgress>([['a', progress('completed')]])
    expect(domainCompletionRatio(modules, map, valid)).toBe(0.25)
  })
})

describe('pathEdgeOpacity', () => {
  it('interpolates from low to high', () => {
    expect(pathEdgeOpacity(0)).toBeCloseTo(0.2)
    expect(pathEdgeOpacity(0.5)).toBeCloseTo(0.525)
    expect(pathEdgeOpacity(1)).toBeCloseTo(0.85)
  })

  it('clamps out-of-range values', () => {
    expect(pathEdgeOpacity(-1)).toBeCloseTo(0.2)
    expect(pathEdgeOpacity(2)).toBeCloseTo(0.85)
  })
})

describe('pathSegmentOpacity', () => {
  it('brightens when previous node is completed', () => {
    expect(pathSegmentOpacity(true, 0)).toBeGreaterThan(pathSegmentOpacity(false, 0))
    expect(pathSegmentOpacity(true, 1)).toBeCloseTo(0.9)
  })
})

describe('PENDING_NODE_OPACITY', () => {
  it('is dimmed but visible', () => {
    expect(PENDING_NODE_OPACITY).toBeGreaterThan(0.25)
    expect(PENDING_NODE_OPACITY).toBeLessThan(0.45)
  })
})
