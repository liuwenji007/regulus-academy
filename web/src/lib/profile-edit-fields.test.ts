import { describe, expect, it } from 'vitest'
import { resolveEditableProfileFields } from './profile-edit-fields'

describe('resolveEditableProfileFields', () => {
  it('uses structured columns when present', () => {
    const got = resolveEditableProfileFields({
      id: '1',
      displayName: '测试',
      profileBackground: '后端工程师',
      profileGoal: '学 Go',
      profilePreference: '偏实战',
      profileSummary: '【背景】旧数据不应覆盖',
    })
    expect(got).toEqual({
      background: '后端工程师',
      goal: '学 Go',
      preference: '偏实战',
    })
  })

  it('falls back to legacy summary sections', () => {
    const got = resolveEditableProfileFields({
      id: '1',
      displayName: '测试',
      profileSummary: '【背景】前端开发\n【进展】心理学已学人本主义',
    })
    expect(got.background).toBe('前端开发')
    expect(got.goal).toBe('')
    expect(got.preference).toBe('')
  })

  it('splits inline preference from background section', () => {
    const got = resolveEditableProfileFields({
      id: '1',
      displayName: '测试',
      profileSummary: '【背景】工程师；讲解偏好：先结构后细节\n【目标】掌握并发',
    })
    expect(got.background).toBe('工程师')
    expect(got.preference).toBe('先结构后细节')
    expect(got.goal).toBe('掌握并发')
  })
})
