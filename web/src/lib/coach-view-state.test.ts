import { describe, expect, it } from 'vitest'
import {
  buildDisplayMessages,
  deriveCoachViewState,
  findAssistantReplyAfterUser,
  isAnsweringExercise,
} from './coach-view-state'
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

describe('findAssistantReplyAfterUser', () => {
  const detail: SessionDetail = {
    sessionId: 's1',
    domainId: 'd1',
    nodeKey: 'n1',
    nodeTitle: '测试',
    phase: 'explain',
    messages: [
      { id: 1, sessionId: 's1', role: 'user', content: '为什么' },
      { id: 2, sessionId: 's1', role: 'assistant', content: '旧回答' },
      { id: 3, sessionId: 's1', role: 'user', content: '为什么' },
      { id: 4, sessionId: 's1', role: 'assistant', content: '新回答' },
    ],
  }

  it('ignores older same-text turns via watermark', () => {
    expect(findAssistantReplyAfterUser(detail, '为什么', 2)?.content).toBe('新回答')
    expect(findAssistantReplyAfterUser(detail, '为什么', 3)).toBeNull()
    expect(findAssistantReplyAfterUser(detail, '为什么', 0)?.content).toBe('新回答')
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

  it('shows streaming bubble and stage hint while sending', () => {
    const server: SessionDetail = {
      sessionId: 's1',
      domainId: 'd1',
      nodeKey: 'n1',
      nodeTitle: '测试',
      phase: 'explain',
      messages: [{ id: 1, sessionId: 's1', role: 'assistant', content: '开场' }],
    }
    const view = deriveCoachViewState({
      sessionId: 's1',
      server,
      bootstrap: null,
      pending: {
        userContent: '为什么？',
        streamingContent: '因为 goroutine',
        stageHint: '教练思考中…',
      },
      draft: { text: '', selectedChoices: [] },
      sending: true,
      error: '',
      toastHtml: '',
      preferReadableOnce: false,
    })
    expect(view.streaming).toBe(true)
    expect(view.stageHint).toBe('教练思考中…')
    expect(view.messages.at(-1)).toEqual({ role: 'assistant', content: '因为 goroutine' })
  })
})

describe('buildDisplayMessages', () => {
  it('prefers assistantContent over streamingContent after final packet', () => {
    const msgs = buildDisplayMessages(null, null, {
      userContent: 'q',
      streamingContent: 'partial',
      assistantContent: 'final answer',
    })
    expect(msgs).toEqual([
      { role: 'user', content: 'q' },
      { role: 'assistant', content: 'final answer' },
    ])
  })

  it('keeps frozen feedback when assistantContent set during mastery stage', () => {
    const msgs = buildDisplayMessages(null, null, {
      userContent: '答案',
      assistantContent: '这题答对了',
      stageHint: '正在评估掌握度…',
    })
    expect(msgs.at(-1)?.content).toBe('这题答对了')
  })
})
