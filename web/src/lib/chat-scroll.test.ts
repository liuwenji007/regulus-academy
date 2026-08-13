import { afterEach, describe, expect, it } from 'vitest'
import { consumeStreamJustEnded, markStreamJustEnded, resetChatStreamFollow } from './chat-scroll'

describe('chat-scroll stream end protection', () => {
  afterEach(() => {
    resetChatStreamFollow()
    consumeStreamJustEnded()
  })

  it('markStreamJustEnded is consumed once', () => {
    markStreamJustEnded()
    expect(consumeStreamJustEnded()).toBe(true)
    expect(consumeStreamJustEnded()).toBe(false)
  })

  it('resetChatStreamFollow clears the just-ended mark', () => {
    markStreamJustEnded()
    resetChatStreamFollow()
    expect(consumeStreamJustEnded()).toBe(false)
  })
})
