import { describe, expect, it } from 'vitest'
import { deriveCoachViewState, isAnsweringExercise } from './coach-view-state'
import type { SessionDetail } from './api'

describe('isAnsweringExercise', () => {
  it('keeps textarea mode in review after wrong text answer', () => {
    expect(
      isAnsweringExercise(
        'review',
        '还没讲到栈大小差异，再想想。',
        { answerFormat: 'text' }
      )
    ).toBe(true)
  })

  it('uses chat mode in review when asking follow-up without exercise meta', () => {
    expect(isAnsweringExercise('review', '能再解释一下吗？', null)).toBe(false)
  })
})

describe('deriveCoachViewState', () => {
  it('shows exercise_text composer after wrong answer in exercise phase', () => {
    const server: SessionDetail = {
      sessionId: 's1',
      domainId: 'd1',
      nodeKey: 'n1',
      nodeTitle: '测试',
      phase: 'exercise',
      exercise: { answerFormat: 'text' },
      messages: [
        { id: 1, sessionId: 's1', role: 'assistant', content: '区别是什么？\n\n做完后直接把答案发给我。' },
        { id: 2, sessionId: 's1', role: 'user', content: '都是并发' },
        { id: 3, sessionId: 's1', role: 'assistant', content: '还没讲到栈大小差异，再想想。' },
      ],
    }
    const view = deriveCoachViewState({
      sessionId: 's1',
      server,
      bootstrap: null,
      pending: null,
      draft: { text: '', selectedChoices: [] },
      sending: false,
      error: '',
      toastHtml: '',
      preferReadableOnce: false,
    })
    expect(view.composerMode).toBe('exercise_text')
    expect(view.exercise?.answerFormat).toBe('text')
  })
})
