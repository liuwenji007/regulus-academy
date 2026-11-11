import type { GraphCanvasTheme } from './graph-canvas-theme'
import { renderLegendGlyph, type LegendGlyph } from './constellation-render'

/** 图例中的缩放层级说明 */
export function graphLegendLodHint(multiDomain: boolean): string {
  return multiDomain ? '银河全景 → 星座 → 知识点' : '星座 → 知识点'
}

const ITEMS: Array<{ kind: LegendGlyph; label: string }> = [
  { kind: 'locked', label: '未学' },
  { kind: 'recommended', label: '推荐' },
  { kind: 'progress', label: '在学' },
  { kind: 'done', label: '已学' },
  { kind: 'module', label: '模块进度' },
  { kind: 'domain-done', label: '课程学完' },
  { kind: 'line-dim', label: '未学连线' },
  { kind: 'line-gold', label: '已学连线' },
  { kind: 'line-flow', label: '推荐路线' },
]

function glyph(theme: GraphCanvasTheme, kind: LegendGlyph): string {
  const src = renderLegendGlyph(theme, kind)
  if (!src) return ''
  return `<img class="graph-legend-glyph" src="${src}" alt="" width="18" height="18" />`
}

/** 图谱视图左下角图例（随 canvas 主题切换，图标与画布同源） */
export function renderGraphLegendHtml(theme: GraphCanvasTheme, multiDomain: boolean): string {
  const items = ITEMS.map(
    (item) => `<span class="tree-graph-legend-item">${glyph(theme, item.kind)}${item.label}</span>`
  ).join('')
  return `${items}<span class="tree-graph-legend-item graph-legend-lod">缩放：${graphLegendLodHint(multiDomain)}</span>`
}
