import { describe, expect, it } from 'vitest'
import {
  choiceTextAlreadyHasLetter,
  formatChoiceSubmission,
} from './coach-exercise'

describe('choiceTextAlreadyHasLetter', () => {
  it('detects matching letter prefixes only', () => {
    expect(choiceTextAlreadyHasLetter('C. if 语句支持初始化', 'C')).toBe(true)
    expect(choiceTextAlreadyHasLetter('A、统一战线', 'A')).toBe(true)
    expect(choiceTextAlreadyHasLetter('B) 武装斗争', 'B')).toBe(true)
    expect(choiceTextAlreadyHasLetter('C++ 支持泛型', 'C')).toBe(false)
    expect(choiceTextAlreadyHasLetter('A. 看起来像前缀', 'B')).toBe(false)
    expect(choiceTextAlreadyHasLetter('统一战线', 'A')).toBe(false)
  })
})

describe('formatChoiceSubmission', () => {
  it('does not double-prefix when choice text already has the letter', () => {
    const choices = [
      'A. if 不能带初始化语句',
      'B. switch 不支持 fallthrough',
      'C. if 语句支持在条件部分使用初始化语句，如 if x := f(); x > 0 {}',
      'D. for 只能写成 C 风格三段式',
    ]
    const text = formatChoiceSubmission([choices[2]], choices, 'single')
    expect(text).toBe(
      '我选择：C. if 语句支持在条件部分使用初始化语句，如 if x := f(); x > 0 {}'
    )
    expect(text).not.toContain('C. C.')
  })

  it('still prefixes when choice text has no letter', () => {
    const choices = ['统一战线', '武装斗争']
    expect(formatChoiceSubmission([choices[1]], choices, 'single')).toBe('我选择：B. 武装斗争')
  })
})
