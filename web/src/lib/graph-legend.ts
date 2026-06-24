import type { GraphCanvasTheme } from './graph-canvas-theme'

/** 图例中的缩放层级说明 */
export function graphLegendLodHint(multiDomain: boolean): string {
  return multiDomain ? '领域总览 → 模块簇 → 节点路径' : '模块簇 → 节点路径'
}

function stampImg(src: string): string {
  return `<img class="graph-stamp-swatch" src="${src}" alt="" width="16" height="16" decoding="async" />`
}

function renderPaperLegend(lodHint: string): string {
  return `
<span class="tree-graph-legend-item">${stampImg('/graph/empty_3.png')}未学</span>
<span class="tree-graph-legend-item">${stampImg('/graph/pre_1.png')}进行中</span>
<span class="tree-graph-legend-item">${stampImg('/graph/dot_5.png')}已学完</span>
<span class="tree-graph-legend-item"><span class="graph-stamp-swatch-wrap graph-stamp-swatch-wrap--starlit">${stampImg('/graph/dot_1.png')}</span>本领域学完</span>
<span class="tree-graph-legend-item graph-legend-tier">大→小：领域 · 模块 · 节点</span>
<span class="tree-graph-legend-item graph-legend-lod">缩放：${lodHint}</span>`
}

function renderSkyLegend(lodHint: string): string {
  return `
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain"></i>领域</span>
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain-starlit"></i>圆满</span>
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--module"></i>模块</span>
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--pending"></i>未开始</span>
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--progress"></i>进行中</span>
<span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--done"></i>已学会</span>
<span class="tree-graph-legend-item graph-legend-lod">缩放：${lodHint}</span>`
}

/** 图谱视图左下角图例（随 canvas 主题切换） */
export function renderGraphLegendHtml(theme: GraphCanvasTheme, multiDomain: boolean): string {
  const lodHint = graphLegendLodHint(multiDomain)
  return theme === 'paper' ? renderPaperLegend(lodHint) : renderSkyLegend(lodHint)
}
