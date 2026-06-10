import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  graphLayoutScopeKey,
  loadGraphLayout,
  persistGraphLayoutFromNetwork,
  resolveNodePlacement,
  saveGraphLayout,
} from './graph-layout-persist'

describe('graphLayoutScopeKey', () => {
  it('sorts domain ids for stable key', () => {
    expect(graphLayoutScopeKey(['b', 'a'], 'u1')).toBe('u1:a,b')
    expect(graphLayoutScopeKey(['a', 'b'], 'u1')).toBe('u1:a,b')
  })
})

describe('graph layout persist', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', {
      store: {} as Record<string, string>,
      getItem(key: string) {
        return this.store[key] ?? null
      },
      setItem(key: string, value: string) {
        this.store[key] = value
      },
      removeItem(key: string) {
        delete this.store[key]
      },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('round-trips domain root positions only', () => {
    saveGraphLayout(['d1', 'd2'], {
      'domain:d1': { x: 10, y: 20 },
      'topic:d1:n1': { x: -5, y: 8 },
    })
    const loaded = loadGraphLayout(['d2', 'd1'])
    expect(loaded).toEqual({
      'domain:d1': { x: 10, y: 20 },
    })
  })

  it('persistGraphLayoutFromNetwork saves domain roots only', () => {
    persistGraphLayoutFromNetwork(
      ['d1'],
      ['domain:d1', 'module:d1:m1'],
      (ids) => {
        const out: Record<string, { x: number; y: number }> = {}
        for (const id of ids) out[id] = { x: 1, y: 2 }
        return out
      }
    )
    expect(loadGraphLayout(['d1'])).toEqual({
      'domain:d1': { x: 1, y: 2 },
    })
  })
})

describe('resolveNodePlacement', () => {
  it('pins saved nodes', () => {
    expect(
      resolveNodePlacement('domain:a', { x: 0, y: 0 }, { 'domain:a': { x: 99, y: -3 } })
    ).toEqual({ x: 99, y: -3, fixed: { x: true, y: true } })
  })

  it('falls back to default', () => {
    expect(resolveNodePlacement('domain:a', { x: 1, y: 2 }, null)).toEqual({ x: 1, y: 2 })
  })
})
