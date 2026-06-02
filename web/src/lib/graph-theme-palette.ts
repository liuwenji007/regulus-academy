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
