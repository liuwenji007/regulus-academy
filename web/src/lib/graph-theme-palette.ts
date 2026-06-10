import type { GraphCanvasTheme } from './graph-canvas-theme'
import { PENDING_NODE_OPACITY } from './graph-progress-visual'

export type GraphLabelStyle = {
  text: string
  stroke: string
  minPx: number
  face: string
}

export type GraphPalette = {
  canvas: string
  root: { fill: string; border: string }
  rootStarlit: { fill: string; border: string }
  module: { fill: string; border: string }
  moduleLit: { fill: string; border: string }
  moduleHover: { fill: string; border: string }
  pending: { fill: string; border: string }
  active: { fill: string; border: string }
  done: { fill: string; border: string }
  focus: { fill: string; border: string }
  glow: {
    focus: string
    active: string
    done: string
    starlight: string
  }
  edge: {
    belong: string
    domainModule: string
    path: string
    prerequisite: string
    highlight: string
    domainRelated: string
    domainDistant: string
  }
  hover: {
    rootStroke: string
    moduleStroke: string
  }
}

const LABEL_FACE = 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif'

// 两套主题各自成体系（修改后请同步 style.css 中 .tree-graph-swatch-- 图例色板）：
// paper（宣纸·水墨）：浓墨领域、淡墨模块、纸白待学、朱砂进行中、描金点亮
// sky（星空·星辰）：恒星白领域、星团蓝模块、暗星待学、橙色恒星进行中、金色恒星点亮
export const GRAPH_THEME_PALETTES: Record<GraphCanvasTheme, { label: GraphLabelStyle; palette: GraphPalette }> = {
  paper: {
    label: {
      text: '#0c0a09',
      stroke: '#fffef9',
      minPx: 12,
      face: LABEL_FACE,
    },
    palette: {
      canvas: '#f8f5ee',
      root: { fill: '#3a3633', border: '#211d1a' },
      rootStarlit: { fill: '#f5dc6a', border: '#c9a227' },
      module: { fill: '#ddd5c6', border: '#6b645a' },
      moduleLit: { fill: '#f5dc6a', border: '#c9a227' },
      moduleHover: { fill: '#cfc6b4', border: '#57534e' },
      pending: { fill: '#fffdf6', border: '#9c958a' },
      active: { fill: '#c45c26', border: '#8f3a14' },
      done: { fill: '#f5dc6a', border: '#c9a227' },
      focus: { fill: '#c45c26', border: '#ffffff' },
      glow: {
        focus: 'rgba(196, 92, 38, 0.5)',
        active: 'rgba(196, 92, 38, 0.42)',
        done: 'rgba(245, 220, 106, 0.55)',
        starlight: 'rgba(255, 248, 210, 0.65)',
      },
      edge: {
        belong: 'rgba(33, 29, 26, 0.16)',
        domainModule: 'rgba(58, 54, 51, 0.38)',
        path: 'rgba(196, 92, 38, 0.3)',
        prerequisite: 'rgba(68, 64, 60, 0.45)',
        highlight: '#c45c26',
        domainRelated: 'rgba(87, 83, 78, 0.38)',
        domainDistant: 'rgba(87, 83, 78, 0.14)',
      },
      hover: {
        rootStroke: 'rgba(33, 29, 26, 0.5)',
        moduleStroke: 'rgba(87, 80, 72, 0.55)',
      },
    },
  },
  sky: {
    label: {
      text: '#f5f3ef',
      stroke: '#0a0f1c',
      minPx: 12,
      face: LABEL_FACE,
    },
    palette: {
      canvas: '#0f1830',
      root: { fill: '#f8f6f0', border: '#c9d4ea' },
      rootStarlit: { fill: '#f5dc6a', border: '#c9a227' },
      module: { fill: '#41507a', border: '#8da4d4' },
      moduleLit: { fill: '#f5dc6a', border: '#c9a227' },
      moduleHover: { fill: '#52639a', border: '#aebcdc' },
      pending: { fill: '#aebcd8', border: '#7484a3' },
      active: { fill: '#e8753a', border: '#c4531c' },
      done: { fill: '#f5dc6a', border: '#c9a227' },
      focus: { fill: '#e8753a', border: '#ffffff' },
      glow: {
        focus: 'rgba(232, 117, 58, 0.55)',
        active: 'rgba(232, 117, 58, 0.45)',
        done: 'rgba(245, 220, 106, 0.6)',
        starlight: 'rgba(255, 248, 210, 0.65)',
      },
      edge: {
        belong: 'rgba(168, 188, 224, 0.26)',
        domainModule: 'rgba(150, 175, 220, 0.5)',
        path: 'rgba(232, 117, 58, 0.5)',
        prerequisite: 'rgba(190, 205, 235, 0.42)',
        highlight: '#e8753a',
        domainRelated: 'rgba(205, 220, 248, 0.4)',
        domainDistant: 'rgba(205, 220, 248, 0.12)',
      },
      hover: {
        rootStroke: 'rgba(220, 230, 250, 0.5)',
        moduleStroke: 'rgba(190, 205, 235, 0.45)',
      },
    },
  },
}

export function getGraphThemeTokens(theme: GraphCanvasTheme) {
  return GRAPH_THEME_PALETTES[theme]
}

/** 将调色板写入 CSS 变量，图例色块与 canvas 节点共用同一套色值 */
export function applyGraphPaletteCssVars(host: HTMLElement, theme: GraphCanvasTheme): void {
  const { palette } = getGraphThemeTokens(theme)
  const pendingFill = hexWithAlpha(palette.pending.fill, PENDING_NODE_OPACITY)
  const pendingBorder = hexWithAlpha(palette.pending.border, PENDING_NODE_OPACITY)
  const s = host.style
  s.setProperty('--graph-swatch-domain-fill', palette.root.fill)
  s.setProperty('--graph-swatch-domain-border', palette.root.border)
  s.setProperty('--graph-swatch-domain-glow', theme === 'sky' ? '0 0 6px rgba(220, 230, 250, 0.55)' : 'none')
  s.setProperty('--graph-swatch-starlit-fill', palette.rootStarlit.fill)
  s.setProperty('--graph-swatch-starlit-border', palette.rootStarlit.border)
  s.setProperty('--graph-swatch-module-fill', palette.module.fill)
  s.setProperty('--graph-swatch-module-border', palette.module.border)
  s.setProperty('--graph-swatch-pending-fill', pendingFill)
  s.setProperty('--graph-swatch-pending-border', pendingBorder)
  s.setProperty('--graph-swatch-active-fill', palette.active.fill)
  s.setProperty('--graph-swatch-active-border', palette.active.border)
  s.setProperty('--graph-swatch-active-glow', hexWithAlpha(palette.active.fill, 0.45))
  s.setProperty('--graph-swatch-done-fill', palette.done.fill)
  s.setProperty('--graph-swatch-done-border', palette.done.border)
  s.setProperty('--graph-swatch-done-glow', hexWithAlpha(palette.done.fill, 0.55))
}

function parseHex(hex: string): [number, number, number] | null {
  const h = hex.replace('#', '').trim()
  if (h.length === 3) {
    return [
      parseInt(h[0]! + h[0], 16),
      parseInt(h[1]! + h[1], 16),
      parseInt(h[2]! + h[2], 16),
    ]
  }
  if (h.length === 6) {
    return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)]
  }
  return null
}

function toHex(r: number, g: number, b: number): string {
  const clamp = (n: number) => Math.max(0, Math.min(255, Math.round(n)))
  return `#${clamp(r).toString(16).padStart(2, '0')}${clamp(g).toString(16).padStart(2, '0')}${clamp(b).toString(16).padStart(2, '0')}`
}

/** 按完成率 0..1 在 base 与 lit 之间插值模块色 */
export function moduleColorAtRatio(
  base: { fill: string; border: string },
  lit: { fill: string; border: string },
  ratio: number
): { fill: string; border: string } {
  const r = Math.max(0, Math.min(1, ratio))
  if (r >= 1) return lit
  if (r <= 0) return base
  const bf = parseHex(base.fill)
  const bb = parseHex(base.border)
  const lf = parseHex(lit.fill)
  const lb = parseHex(lit.border)
  if (!bf || !bb || !lf || !lb) return r >= 0.5 ? lit : base
  return {
    fill: toHex(bf[0] + (lf[0] - bf[0]) * r, bf[1] + (lf[1] - bf[1]) * r, bf[2] + (lf[2] - bf[2]) * r),
    border: toHex(bb[0] + (lb[0] - bb[0]) * r, bb[1] + (lb[1] - bb[1]) * r, bb[2] + (lb[2] - bb[2]) * r),
  }
}

/** 将 #rrggbb 转为带 alpha 的 rgba */
export function hexWithAlpha(hex: string, alpha: number): string {
  const rgb = parseHex(hex)
  if (!rgb) return hex
  const a = Math.max(0, Math.min(1, alpha))
  return `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, ${a})`
}
