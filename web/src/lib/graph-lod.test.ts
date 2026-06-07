import { describe, expect, it } from 'vitest'
import { lodFromScale, lodLabel, LOD_CONSTELLATION_MAX, LOD_GALAXY_MAX } from './graph-lod'

describe('lodFromScale', () => {
  it('returns galaxy for zoomed-out views', () => {
    expect(lodFromScale(0.1)).toBe('galaxy')
    expect(lodFromScale(LOD_GALAXY_MAX - 0.01)).toBe('galaxy')
  })

  it('returns constellation at mid zoom', () => {
    expect(lodFromScale(LOD_GALAXY_MAX)).toBe('constellation')
    expect(lodFromScale((LOD_GALAXY_MAX + LOD_CONSTELLATION_MAX) / 2)).toBe('constellation')
    expect(lodFromScale(LOD_CONSTELLATION_MAX - 0.01)).toBe('constellation')
  })

  it('returns node when zoomed in', () => {
    expect(lodFromScale(LOD_CONSTELLATION_MAX)).toBe('node')
    expect(lodFromScale(1.5)).toBe('node')
  })

  it('handles invalid scale', () => {
    expect(lodFromScale(0)).toBe('galaxy')
    expect(lodFromScale(-1)).toBe('galaxy')
  })
})

describe('lodLabel', () => {
  it('maps levels to Chinese labels', () => {
    expect(lodLabel('galaxy')).toBe('全景')
    expect(lodLabel('constellation')).toBe('星座')
    expect(lodLabel('node')).toBe('节点')
  })
})
