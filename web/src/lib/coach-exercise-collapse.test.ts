import { describe, expect, it } from 'vitest'
import {
  shouldCollapseExerciseComposer,
  shouldResetComposerExpanded,
} from './coach-exercise'

describe('shouldCollapseExerciseComposer', () => {
  it('does not collapse empty or short answers', () => {
    expect(shouldCollapseExerciseComposer('', 'text')).toBe(false)
    expect(shouldCollapseExerciseComposer('short\nanswer', 'text')).toBe(false)
  })

  it('collapses long line counts', () => {
    const draft = Array.from({ length: 8 }, (_, i) => `line ${i}`).join('\n')
    expect(shouldCollapseExerciseComposer(draft, 'text')).toBe(true)
  })

  it('collapses long single-block text', () => {
    expect(shouldCollapseExerciseComposer('x'.repeat(280), 'json')).toBe(true)
  })

  it('collapses choice when there are multiple options', () => {
    expect(shouldCollapseExerciseComposer('', 'choice', { choiceCount: 2 })).toBe(true)
    expect(shouldCollapseExerciseComposer('', 'choice', { choiceCount: 4 })).toBe(true)
  })

  it('does not collapse choice with fewer than two options', () => {
    expect(shouldCollapseExerciseComposer('', 'choice', { choiceCount: 1 })).toBe(false)
    expect(shouldCollapseExerciseComposer('', 'choice')).toBe(false)
  })
})

describe('shouldResetComposerExpanded', () => {
  it('resets when switching from text to choice', () => {
    expect(shouldResetComposerExpanded('text', 'choice')).toBe(true)
  })

  it('resets when switching between non-choice formats', () => {
    expect(shouldResetComposerExpanded('text', 'json')).toBe(true)
    expect(shouldResetComposerExpanded('json', 'text')).toBe(true)
  })

  it('does not reset when format stays the same', () => {
    expect(shouldResetComposerExpanded('text', 'text')).toBe(false)
    expect(shouldResetComposerExpanded('choice', 'choice')).toBe(false)
  })

  it('resets on first exercise format after empty', () => {
    expect(shouldResetComposerExpanded('', 'text')).toBe(true)
    expect(shouldResetComposerExpanded('', 'choice')).toBe(true)
  })
})
