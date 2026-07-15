import { describe, expect, it } from 'vitest'
import { isCoachRoute } from './coach-route'
import { isAssistantRoute } from './assistant-route'

describe('route guards', () => {
  it('distinguishes coach vs assistant hash', () => {
    expect(isCoachRoute('/coach/abc')).toBe(true)
    expect(isCoachRoute('/assistant/abc')).toBe(false)
    expect(isAssistantRoute('/assistant')).toBe(true)
    expect(isAssistantRoute('/assistant/sid')).toBe(true)
    expect(isAssistantRoute('/coach/sid')).toBe(false)
  })
})
