import { describe, expect, it } from 'vitest'
import { resolveCoachScrollMode, type ChatMessage } from './coach-view-state'

const assistantLast: ChatMessage[] = [{ role: 'assistant', content: '开场讲解' }]
const userThenAssistant: ChatMessage[] = [
  { role: 'user', content: '你好' },
  { role: 'assistant', content: '回复' },
]

describe('resolveCoachScrollMode', () => {
  it('anchors when opening with last assistant message', () => {
    expect(
      resolveCoachScrollMode({
        messages: assistantLast,
        sending: false,
        pending: null,
        preferReadableOnce: true,
      })
    ).toBe('readable')
  })

  it('anchors after new assistant reply', () => {
    expect(
      resolveCoachScrollMode({
        messages: userThenAssistant,
        sending: false,
        pending: null,
        preferReadableOnce: true,
      })
    ).toBe('readable')
  })

  it('scrolls to bottom while waiting for assistant', () => {
    expect(
      resolveCoachScrollMode({
        messages: [...userThenAssistant, { role: 'user', content: '追问' }],
        sending: true,
        pending: { userContent: '追问' },
        preferReadableOnce: true,
      })
    ).toBe('bottom')
  })

  it('does not anchor without preferReadableOnce flag', () => {
    expect(
      resolveCoachScrollMode({
        messages: assistantLast,
        sending: false,
        pending: null,
        preferReadableOnce: false,
      })
    ).toBe('bottom')
  })

  it('anchors new exercise question to the start', () => {
    expect(
      resolveCoachScrollMode({
        messages: [{ role: 'assistant', content: '题目…做完后直接把答案发给我' }],
        sending: false,
        pending: null,
        preferReadableOnce: true,
      })
    ).toBe('readable')
  })
})
