import { describe, expect, it } from 'vitest'
import { graphLegendLodHint, renderGraphLegendHtml } from './graph-legend'

describe('graphLegendLodHint', () => {
  it('describes multi-domain zoom levels', () => {
    expect(graphLegendLodHint(true)).toBe('银河全景 → 星座 → 知识点')
  })

  it('describes single-domain zoom levels', () => {
    expect(graphLegendLodHint(false)).toBe('星座 → 知识点')
  })
})

describe('renderGraphLegendHtml', () => {
  it('lists the same states for both themes', () => {
    for (const theme of ['paper', 'sky'] as const) {
      const html = renderGraphLegendHtml(theme, true)
      for (const label of ['未学', '推荐', '在学', '已学', '模块进度', '课程学完', '未学连线', '已学连线', '推荐路线']) {
        expect(html).toContain(label)
      }
      expect(html).toContain('银河全景 → 星座 → 知识点')
      expect(html).not.toContain('/graph/')
      expect(html).not.toContain('tree-graph-swatch')
    }
  })

  it('shortens the zoom hint for a single course', () => {
    expect(renderGraphLegendHtml('sky', false)).toContain('缩放：星座 → 知识点')
  })
})
