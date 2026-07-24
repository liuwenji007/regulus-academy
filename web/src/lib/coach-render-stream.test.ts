import { describe, expect, it } from 'vitest'
import { closeOpenMarkdownFences } from './coach-render'

describe('closeOpenMarkdownFences', () => {
  it('temporarily closes an open fence', () => {
    expect(closeOpenMarkdownFences('说明：\n\n```go\nfunc main() {')).toBe(
      '说明：\n\n```go\nfunc main() {\n```'
    )
  })

  it('does not alter already closed fences', () => {
    const src = '```go\nok\n```\n后文'
    expect(closeOpenMarkdownFences(src)).toBe(src)
  })
})
