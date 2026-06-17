import { describe, expect, it } from 'vitest'
import {
  DOMAIN_EMPTY_IDS,
  DOMAIN_STAMP_IDS,
  MODULE_STAMP_IDS,
  TOPIC_STAMP_IDS,
  pickInkStampId,
  pickPaperStampKey,
  stampDrawScale,
} from './ink-stamps'

describe('pickInkStampId', () => {
  it('domain uses dot_1～dot_3', () => {
    expect(pickInkStampId(0, 'domain')).toBe(1)
    expect(pickInkStampId(1, 'domain')).toBe(2)
    expect(pickInkStampId(2, 'domain')).toBe(3)
    expect(pickInkStampId(3, 'domain')).toBe(1)
    expect(DOMAIN_STAMP_IDS).toEqual([1, 2, 3])
  })

  it('module uses dot_5～dot_8', () => {
    expect(pickInkStampId(0, 'module')).toBe(5)
    expect(pickInkStampId(3, 'module')).toBe(8)
    expect(pickInkStampId(4, 'module')).toBe(5)
    expect(pickInkStampId(100, 'module')).toBe(5 + (100 % 4))
    expect(MODULE_STAMP_IDS).toEqual([5, 6, 7, 8])
  })

  it('topic uses dot_10～dot_16', () => {
    expect(pickInkStampId(0, 'topic')).toBe(10)
    expect(pickInkStampId(6, 'topic')).toBe(16)
    expect(pickInkStampId(7, 'topic')).toBe(10)
    expect(pickInkStampId(100, 'topic')).toBe(10 + (100 % 7))
    expect(TOPIC_STAMP_IDS).toEqual([10, 11, 12, 13, 14, 15, 16])
  })
})

describe('pickPaperStampKey', () => {
  it('lit maps to dot_*', () => {
    expect(pickPaperStampKey(0, 'module', 'lit')).toBe('dot_5')
    expect(pickPaperStampKey(0, 'topic', 'lit')).toBe('dot_10')
  })

  it('empty maps by role', () => {
    expect(pickPaperStampKey(0, 'domain', 'empty')).toBe('empty_1')
    expect(pickPaperStampKey(1, 'domain', 'empty')).toBe('empty_2')
    expect(pickPaperStampKey(0, 'module', 'empty')).toBe('empty_3')
    expect(pickPaperStampKey(0, 'topic', 'empty')).toBe('empty_4')
    expect(DOMAIN_EMPTY_IDS).toEqual([1, 2])
  })

  it('progress maps to pre_*', () => {
    expect(pickPaperStampKey(0, 'module', 'progress')).toBe('pre_1')
    expect(pickPaperStampKey(0, 'topic', 'progress')).toBe('pre_2')
  })
})

describe('stampDrawScale', () => {
  it('keeps lit scale as base draw scale', () => {
    expect(stampDrawScale('domain', 'lit')).toBe(1.45)
    expect(stampDrawScale('module', 'lit')).toBe(1.75)
    expect(stampDrawScale('topic', 'lit')).toBe(1.6)
  })

  it('aligns empty/progress module stamps to lit outer diameter', () => {
    expect(stampDrawScale('domain', 'empty')).toBeCloseTo(1.45 * 1.45, 5)
    expect(stampDrawScale('module', 'empty')).toBeCloseTo(1.75 * 0.74, 5)
    expect(stampDrawScale('module', 'progress')).toBeCloseTo(1.75 * 0.7, 5)
  })
})
