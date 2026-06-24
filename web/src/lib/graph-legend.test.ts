import { describe, expect, it } from 'vitest'
import { graphLegendLodHint, renderGraphLegendHtml } from './graph-legend'

describe('graphLegendLodHint', () => {
  it('describes multi-domain zoom levels', () => {
    expect(graphLegendLodHint(true)).toBe('领域总览 → 模块簇 → 节点路径')
  })

  it('describes single-domain zoom levels', () => {
    expect(graphLegendLodHint(false)).toBe('模块簇 → 节点路径')
  })
})

describe('renderGraphLegendHtml', () => {
  it('renders stamp thumbnails for paper theme', () => {
    const html = renderGraphLegendHtml('paper', true)
    expect(html).toContain('/graph/empty_3.png')
    expect(html).toContain('/graph/pre_1.png')
    expect(html).toContain('/graph/dot_5.png')
    expect(html).toContain('/graph/dot_1.png')
    expect(html).toContain('本领域学完')
    expect(html).toContain('大→小：领域 · 模块 · 节点')
    expect(html).not.toContain('tree-graph-swatch--domain')
  })

  it('renders color swatches for sky theme', () => {
    const html = renderGraphLegendHtml('sky', false)
    expect(html).toContain('tree-graph-swatch--domain')
    expect(html).toContain('tree-graph-swatch--done')
    expect(html).not.toContain('graph-stamp-swatch')
    expect(html).toContain('模块簇 → 节点路径')
  })
})
