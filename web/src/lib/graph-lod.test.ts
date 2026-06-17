import { describe, expect, it } from 'vitest'
import {
  lodFromScale,
  lodLabel,
  LOD_CONSTELLATION_TOPIC_SIZE_SCALE,
  PAPER_LOD_CONSTELLATION_MAX,
  PAPER_LOD_GALAXY_MAX,
  SKY_LOD_GALAXY_MAX,
  topicLabelsVisible,
  topicSizeForLod,
} from './graph-lod'

describe('lodFromScale — sky', () => {
  it('returns galaxy for zoomed-out views', () => {
    expect(lodFromScale(0.01, true, 'sky')).toBe('galaxy')
    expect(lodFromScale(SKY_LOD_GALAXY_MAX - 0.001, true, 'sky')).toBe('galaxy')
  })

  it('skips constellation and jumps to node when zoomed in', () => {
    expect(lodFromScale(SKY_LOD_GALAXY_MAX, true, 'sky')).toBe('node')
    expect(lodFromScale(1.5, true, 'sky')).toBe('node')
  })

  it('handles invalid scale', () => {
    expect(lodFromScale(0, true, 'sky')).toBe('galaxy')
    expect(lodFromScale(-1, true, 'sky')).toBe('galaxy')
  })
})

describe('lodFromScale — paper', () => {
  it('returns galaxy for zoomed-out views', () => {
    expect(lodFromScale(0.09, true, 'paper')).toBe('galaxy')
    expect(lodFromScale(PAPER_LOD_GALAXY_MAX - 0.01, true, 'paper')).toBe('galaxy')
  })

  it('returns constellation at mid zoom', () => {
    expect(lodFromScale(PAPER_LOD_GALAXY_MAX, true, 'paper')).toBe('constellation')
    expect(lodFromScale((PAPER_LOD_GALAXY_MAX + PAPER_LOD_CONSTELLATION_MAX) / 2, true, 'paper')).toBe(
      'constellation'
    )
    expect(lodFromScale(PAPER_LOD_CONSTELLATION_MAX - 0.01, true, 'paper')).toBe('constellation')
  })

  it('returns node when zoomed in', () => {
    expect(lodFromScale(PAPER_LOD_CONSTELLATION_MAX, true, 'paper')).toBe('node')
    expect(lodFromScale(1.5, true, 'paper')).toBe('node')
  })
})

describe('lodLabel', () => {
  it('maps levels to Chinese labels', () => {
    expect(lodLabel('galaxy')).toBe('全景')
    expect(lodLabel('constellation')).toBe('星座')
    expect(lodLabel('node')).toBe('节点')
  })
})

describe('topicSizeForLod', () => {
  it('shrinks topics at constellation LOD for paper only', () => {
    expect(topicSizeForLod(16, 'constellation', 'paper')).toBe(
      Math.max(6, Math.round(16 * LOD_CONSTELLATION_TOPIC_SIZE_SCALE))
    )
    expect(topicSizeForLod(16, 'constellation', 'sky')).toBe(16)
    expect(topicSizeForLod(12, 'node', 'paper')).toBe(12)
  })

  it('hides topic labels except at node LOD for paper', () => {
    expect(topicLabelsVisible('node', 'paper')).toBe(true)
    expect(topicLabelsVisible('constellation', 'paper')).toBe(false)
    expect(topicLabelsVisible('galaxy', 'paper')).toBe(false)
    expect(topicLabelsVisible('node', 'sky')).toBe(true)
  })
})
