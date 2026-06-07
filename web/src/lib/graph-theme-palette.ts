import type { GraphCanvasTheme } from './graph-canvas-theme'

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
  }
  hover: {
    rootStroke: string
    moduleStroke: string
  }
}

const LABEL_FACE = 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif'

const SHARED_PROGRESS: Pick<GraphPalette, 'active' | 'done' | 'focus' | 'glow'> = {
  active: { fill: '#c45c26', border: '#9a3f18' },
  done: { fill: '#f5dc6a', border: '#c9a227' },
  focus: { fill: '#c45c26', border: '#ffffff' },
  glow: {
    focus: 'rgba(196, 92, 38, 0.5)',
    active: 'rgba(196, 92, 38, 0.42)',
    done: 'rgba(245, 220, 106, 0.55)',
    starlight: 'rgba(255, 248, 210, 0.65)',
  },
}

export const GRAPH_THEME_PALETTES: Record<GraphCanvasTheme, { label: GraphLabelStyle; palette: GraphPalette }> = {
  paper: {
    label: {
      text: '#0c0a09',
      stroke: '#fffef9',
      minPx: 12,
      face: LABEL_FACE,
    },
    palette: {
      canvas: '#faf8f4',
      root: { fill: '#44403c', border: '#44403c' },
      rootStarlit: { fill: '#f5dc6a', border: '#c9a227' },
      module: { fill: '#f0ebe3', border: '#78716c' },
      moduleLit: { fill: '#f5dc6a', border: '#c9a227' },
      moduleHover: { fill: '#e7e0d4', border: '#57534e' },
      pending: { fill: '#ffffff', border: '#a8a29e' },
      ...SHARED_PROGRESS,
      edge: {
        belong: 'rgba(28, 25, 23, 0.14)',
        domainModule: 'rgba(196, 92, 38, 0.32)',
        path: 'rgba(196, 92, 38, 0.28)',
        prerequisite: 'rgba(68, 64, 60, 0.42)',
        highlight: '#c45c26',
      },
      hover: {
        rootStroke: 'rgba(68, 64, 60, 0.45)',
        moduleStroke: 'rgba(120, 113, 108, 0.55)',
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
      root: { fill: '#d6d3d1', border: '#d6d3d1' },
      rootStarlit: { fill: '#f5dc6a', border: '#c9a227' },
      module: { fill: '#57534e', border: '#78716c' },
      moduleLit: { fill: '#f5dc6a', border: '#c9a227' },
      moduleHover: { fill: '#78716c', border: '#a8a29e' },
      pending: { fill: '#f5f3ef', border: '#a8a29e' },
      ...SHARED_PROGRESS,
      edge: {
        belong: 'rgba(255, 255, 255, 0.2)',
        domainModule: 'rgba(245, 200, 120, 0.45)',
        path: 'rgba(196, 92, 38, 0.45)',
        prerequisite: 'rgba(214, 211, 209, 0.38)',
        highlight: '#c45c26',
      },
      hover: {
        rootStroke: 'rgba(245, 243, 239, 0.45)',
        moduleStroke: 'rgba(245, 243, 239, 0.4)',
      },
    },
  },
}

export function getGraphThemeTokens(theme: GraphCanvasTheme) {
  return GRAPH_THEME_PALETTES[theme]
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
