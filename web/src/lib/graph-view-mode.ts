export type GraphViewMode = 'galaxy' | 'outline'

export const GRAPH_VIEW_MODE_KEY = 'regulus:graphViewMode'

export function getGraphViewMode(): GraphViewMode {
  try {
    const raw = localStorage.getItem(GRAPH_VIEW_MODE_KEY)
    if (raw === 'galaxy' || raw === 'outline') return raw
  } catch {
    /* ignore */
  }
  return 'galaxy'
}

export function setGraphViewMode(mode: GraphViewMode): void {
  try {
    localStorage.setItem(GRAPH_VIEW_MODE_KEY, mode)
  } catch {
    /* ignore */
  }
}

/** URL `?view=` 优先，否则读 localStorage */
export function resolveGraphViewMode(): GraphViewMode {
  try {
    const hash = location.hash.slice(1) || '/'
    const query = hash.includes('?') ? hash.split('?')[1] : ''
    if (query) {
      const view = new URLSearchParams(query).get('view')
      if (view === 'outline' || view === 'galaxy') {
        setGraphViewMode(view)
        return view
      }
    }
  } catch {
    /* ignore */
  }
  return getGraphViewMode()
}

export function graphViewModeLabel(mode: GraphViewMode): string {
  return mode === 'galaxy' ? '银河' : '目录'
}

export function graphViewModeTitle(mode: GraphViewMode): string {
  return mode === 'galaxy' ? '探索全局关系' : '按清单浏览'
}

export function renderGraphViewToggleHtml(active: GraphViewMode): string {
  return `
    <div class="graph-view-toggle" role="tablist" aria-label="知识图谱视图">
      <button
        type="button"
        class="graph-view-toggle-btn${active === 'galaxy' ? ' is-active' : ''}"
        data-view-mode="galaxy"
        role="tab"
        aria-selected="${active === 'galaxy' ? 'true' : 'false'}"
        title="银河视图：探索全局关系"
      >银河</button>
      <button
        type="button"
        class="graph-view-toggle-btn${active === 'outline' ? ' is-active' : ''}"
        data-view-mode="outline"
        role="tab"
        aria-selected="${active === 'outline' ? 'true' : 'false'}"
        title="目录视图：按清单浏览"
      >目录</button>
    </div>
  `
}
