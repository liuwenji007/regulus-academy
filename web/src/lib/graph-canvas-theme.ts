import { applyGraphPaletteCssVars } from './graph-theme-palette'

export type GraphCanvasTheme = 'paper' | 'sky'

export const GRAPH_CANVAS_THEME_KEY = 'regulus:graphCanvasTheme'
export const GRAPH_CANVAS_THEME_ATTR = 'data-graph-theme'

export function getGraphCanvasTheme(): GraphCanvasTheme {
  try {
    const raw = localStorage.getItem(GRAPH_CANVAS_THEME_KEY)
    if (raw === 'paper' || raw === 'sky') return raw
  } catch {
    /* ignore */
  }
  return 'paper'
}

export function setGraphCanvasTheme(theme: GraphCanvasTheme): void {
  try {
    localStorage.setItem(GRAPH_CANVAS_THEME_KEY, theme)
  } catch {
    /* ignore */
  }
}

export function applyGraphCanvasTheme(host: HTMLElement, theme: GraphCanvasTheme): void {
  host.setAttribute(GRAPH_CANVAS_THEME_ATTR, theme)
  applyGraphPaletteCssVars(host, theme)
}

export function readGraphCanvasThemeFrom(el: HTMLElement): GraphCanvasTheme {
  const host = el.closest(`[${GRAPH_CANVAS_THEME_ATTR}]`)
  if (host) {
    const value = host.getAttribute(GRAPH_CANVAS_THEME_ATTR)
    if (value === 'paper' || value === 'sky') return value
  }
  return getGraphCanvasTheme()
}

export function graphCanvasThemeLabel(theme: GraphCanvasTheme): string {
  return theme === 'paper' ? '宣纸' : '星空'
}

/** 切换到另一主题 */
export function toggleGraphCanvasTheme(current: GraphCanvasTheme): GraphCanvasTheme {
  return current === 'paper' ? 'sky' : 'paper'
}

/** 顶栏快捷主题按钮文案：显示点击后将切换到的主题 */
export function graphCanvasThemeToggleLabel(current: GraphCanvasTheme): string {
  return graphCanvasThemeLabel(toggleGraphCanvasTheme(current))
}

export function renderGraphThemeToggleHtml(theme: GraphCanvasTheme, id = 'graph-theme-quick-btn'): string {
  const next = toggleGraphCanvasTheme(theme)
  const label = graphCanvasThemeLabel(next)
  return `
    <button
      type="button"
      class="graph-theme-quick-btn graph-theme-btn"
      id="${id}"
      aria-pressed="${theme === 'sky' ? 'true' : 'false'}"
      title="切换为${label}主题"
    >${label}</button>
  `
}
