import {
  getDomains,
  getDomainTree,
  getUserProgress,
  ApiError,
  type UserProgress,
} from '../lib/api'
import { setAppBusy } from '../lib/app-busy'
import { navigateHash } from '../lib/navigate'
import { normalizeKnowledgeTree, resolveGraphModules } from '../lib/tree-normalize'
import {
  applyGraphCanvasTheme,
  getGraphCanvasTheme,
  graphCanvasThemeLabel,
  setGraphCanvasTheme,
  toggleGraphCanvasTheme,
  type GraphCanvasTheme,
} from '../lib/graph-canvas-theme'
import { bindGraphOutline, renderGraphOutlineHtml } from '../lib/graph-outline'
import {
  renderGraphViewToggleHtml,
  resolveGraphViewMode,
  setGraphViewMode,
  type GraphViewMode,
} from '../lib/graph-view-mode'
import { mountMultiDomainKnowledgeGraph, type KnowledgeGraphMount } from '../lib/knowledge-graph'
import { startNodeSession } from '../lib/start-node-session'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'
import { iconPanelLeft, iconX } from '../lib/icons'

const TREE_FOCUS_PREFIX = 'regulus:treeFocus:'

let graphRenderGen = 0
let activeGraphDestroy: (() => void) | null = null
let activeGraphMount: KnowledgeGraphMount | null = null
let graphUiAbort: AbortController | null = null

function disposeActiveGraph(): void {
  if (activeGraphDestroy) {
    try {
      activeGraphDestroy()
    } catch {
      /* ignore */
    }
    activeGraphDestroy = null
  }
  activeGraphMount = null
}

function graphThemeToggleLabel(current: GraphCanvasTheme): string {
  return current === 'paper' ? '星空' : '宣纸'
}

function readTreeFocus(domainId: string): Set<string> {
  try {
    const raw = sessionStorage.getItem(TREE_FOCUS_PREFIX + domainId)
    if (!raw) return new Set()
    const parsed = JSON.parse(raw) as { keys?: string[] }
    return new Set(Array.isArray(parsed.keys) ? parsed.keys : [])
  } catch {
    return new Set()
  }
}

function domainNavHtml(
  summaries: Array<{ id: string; name: string }>,
  chipIdPrefix: string,
  opts: { defaultCollapsed?: boolean; collapsible?: boolean } = {}
): string {
  const collapsible = opts.collapsible ?? true
  const defaultCollapsed = collapsible && (opts.defaultCollapsed ?? false)
  const collapsedClass = defaultCollapsed ? ' is-collapsed' : ''
  const staticClass = collapsible ? '' : ' graph-domain-nav--static'
  const expanded = !defaultCollapsed
  const toggleHtml = collapsible
    ? `<button type="button" class="graph-domain-nav-toggle" id="${chipIdPrefix}-domain-nav-toggle" title="${expanded ? '收起' : '展开'}" aria-expanded="${expanded ? 'true' : 'false'}" aria-label="${expanded ? '收起领域搜索' : '展开领域搜索'}">▴</button>`
    : ''
  return `
    <div class="graph-float-panel graph-domain-nav${staticClass}${collapsedClass}" id="${chipIdPrefix}-domain-nav">
      <div class="graph-domain-nav-header">
        <span class="graph-domain-nav-label">搜索领域</span>
        ${toggleHtml}
      </div>
      <div class="graph-domain-nav-body">
        <input
          type="search"
          id="${chipIdPrefix}-domain-search"
          class="input graph-domain-search"
          placeholder="搜索领域…"
          autocomplete="off"
          aria-label="搜索领域"
        />
        <div class="graph-domain-chips" id="${chipIdPrefix}-domain-chips" role="listbox" aria-label="领域列表">
          <button type="button" class="graph-domain-chip is-active" data-domain-id="" role="option">全部</button>
          ${summaries
            .map(
              (s) =>
                `<button type="button" class="graph-domain-chip" data-domain-id="${escapeHtml(s.id)}" role="option" title="${escapeHtml(s.name)}">${escapeHtml(shortDomainLabel(s.name))}</button>`
            )
            .join('')}
        </div>
      </div>
    </div>`
}

export async function renderGraph(container: HTMLElement): Promise<void> {
  const gen = ++graphRenderGen
  const stale = () => gen !== graphRenderGen

  graphUiAbort?.abort()
  graphUiAbort = new AbortController()
  const uiSignal = graphUiAbort.signal

  clearTreeSessionOverlay()
  disposeActiveGraph()

  void updateSidebar({ active: 'graph' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '知识图谱' },
  ])

  let canvasTheme = getGraphCanvasTheme()
  let viewMode: GraphViewMode = resolveGraphViewMode()

  container.innerHTML = `
    <section class="page page-graph page-graph--immersive">
      <div class="graph-stage graph-stage--loading" data-graph-theme="${canvasTheme}">
        <div class="graph-loading">
          <div class="spinner" aria-hidden="true"></div>
          <p>正在加载知识图谱…</p>
        </div>
      </div>
    </section>
  `

  try {
    const summaries = await getDomains()
    if (stale()) return

    if (summaries.length === 0) {
      container.innerHTML = `
        <section class="page page-graph">
          <header class="page-header">
            <h1 class="page-title">知识图谱</h1>
            <p class="page-sub">跨领域总览你的学习全景。创建第一门课后，各领域会在这里汇总展示。</p>
          </header>
          <div class="card graph-empty">
            <p>还没有课程</p>
            <a href="#/" class="btn btn-primary btn-sm">去开始学习</a>
          </div>
        </section>
      `
      return
    }

    const loaded = await Promise.all(
      summaries.map(async (summary) => {
        const [treeRaw, progress] = await Promise.all([
          getDomainTree(summary.id),
          getUserProgress(summary.id).catch(() => [] as UserProgress[]),
        ])
        const tree = normalizeKnowledgeTree(treeRaw, summary.id, summary.name)
        const progressMap = new Map(progress.map((p) => [p.nodeKey, p]))
        return {
          domainId: summary.id,
          slug: summary.slug,
          tree,
          progressMap,
          focusKeys: readTreeFocus(summary.id),
        }
      })
    )
    if (stale()) return

    const nodeTitleByKey = new Map<string, string>()
    let hasDerivedModules = false
    for (const entry of loaded) {
      const { isDerived } = resolveGraphModules(entry.tree)
      if (isDerived) hasDerivedModules = true
      for (const layer of entry.tree.layers) {
        for (const node of layer.nodes) {
          nodeTitleByKey.set(`${entry.domainId}:${node.key}`, node.title)
        }
      }
    }

    const derivedHint = hasDerivedModules
      ? ' · 部分课程按进度层临时分簇，重新生成可获得主题模块'
      : ''

    const showDomainNav = summaries.length > 1
    const galaxyHintTitle =
      summaries.length > 1
        ? `${summaries.length} 个领域。相关课程相邻排布，子模块环绕主领域。单击定位、双击领域进课程，拖动画布探索。滚轮缩放切换全景/模块/节点层级。${derivedHint}`
        : `模块扇形簇、节点沿路径点亮。单击模块定位，拖动画布探索，滚轮缩放查看细节。${derivedHint}`
    const galaxyHint =
      summaries.length > 1
        ? `${summaries.length} 个领域 · 拖动探索 · 单击定位 · 双击进课程`
        : `拖动探索 · 单击模块定位 · 滚轮缩放`
    const outlineHint =
      summaries.length > 1
        ? `${summaries.length} 门课 · 按领域与模块分层浏览 · 点击节点开始学习${derivedHint}`
        : `按模块分层浏览学习路径 · 点击节点开始微训练${derivedHint}`

    const galaxyDomainNav = showDomainNav
      ? domainNavHtml(summaries, 'graph', { collapsible: false })
      : ''
    const outlineDomainNav = showDomainNav ? domainNavHtml(summaries, 'graph-outline') : ''
    const lodHint =
      summaries.length > 1 ? '领域总览 → 模块簇 → 节点路径' : '模块簇 → 节点路径'
    const viewToggle = renderGraphViewToggleHtml(viewMode)
    const immersiveClass = viewMode === 'galaxy' ? 'page-graph--immersive' : 'page-graph--outline'

    container.innerHTML = `
      <section class="page page-graph ${immersiveClass}">
        <div id="graph-galaxy-panel" class="graph-galaxy-panel"${viewMode === 'outline' ? ' hidden' : ''}>
          <div class="graph-stage" data-graph-theme="${canvasTheme}">
            <div id="graph-canvas" class="graph-canvas" role="img" aria-label="多领域知识图谱"></div>

            <div class="graph-float graph-float--top">
              <div class="graph-hud-anchor" id="graph-hud-anchor">
                <div class="graph-hud-bar graph-float-panel">
                  <button
                    type="button"
                    class="graph-hud-toggle"
                    id="graph-hud-toggle"
                    aria-expanded="false"
                    aria-controls="graph-hud-drawer"
                    title="展开控制面板"
                    aria-label="展开知识图谱控制面板"
                  >${iconPanelLeft()}</button>
                  ${viewToggle}
                </div>
                <div
                  class="graph-hud-drawer graph-float-panel"
                  id="graph-hud-drawer"
                  aria-hidden="true"
                >
                  <div class="graph-hud-drawer-inner">
                    <div class="graph-hud-body" id="graph-hud-body">
                      <div class="graph-hud-body-header">
                        <div class="graph-header-row">
                          <h1 class="graph-title">知识图谱</h1>
                          <button
                            type="button"
                            class="graph-hud-close"
                            id="graph-hud-close"
                            aria-label="收起面板"
                            title="收起"
                          >${iconX()}</button>
                        </div>
                        <p class="graph-hint" title="${escapeHtml(galaxyHintTitle)}">${escapeHtml(galaxyHint)}</p>
                      </div>
                      <div class="graph-hud-toolbar">
                        <button
                          type="button"
                          class="btn btn-ghost btn-sm graph-theme-btn"
                          id="graph-theme-btn"
                          aria-pressed="${canvasTheme === 'sky' ? 'true' : 'false'}"
                          title="切换为${escapeHtml(graphCanvasThemeLabel(toggleGraphCanvasTheme(canvasTheme)))}主题"
                        >${escapeHtml(graphThemeToggleLabel(canvasTheme))}</button>
                        <button type="button" class="btn btn-ghost btn-sm" id="graph-fit-btn">重置视图</button>
                      </div>
                      ${galaxyDomainNav}
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="graph-float graph-float--legend-wrap" title="悬停查看图例">
              <span class="graph-legend-trigger graph-float-panel" id="graph-legend-trigger">图例</span>
              <div class="graph-float--legend graph-float-panel" id="graph-legend-panel">
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain"></i>领域</span>
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain-starlit"></i>圆满</span>
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--module"></i>模块</span>
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--pending"></i>未开始</span>
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--progress"></i>进行中</span>
                <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--done"></i>已学会</span>
                <span class="tree-graph-legend-item graph-legend-lod">缩放：${lodHint}</span>
              </div>
            </div>

            <div id="graph-error" class="graph-float graph-float--error"></div>
          </div>
        </div>

        <div id="graph-outline-panel" class="graph-outline-panel"${viewMode === 'galaxy' ? ' hidden' : ''}>
          <header class="graph-outline-header">
            <div class="graph-outline-header-main">
              <div class="graph-header-row">
                <h1 class="graph-title">知识图谱</h1>
                ${viewToggle}
              </div>
              <p class="graph-hint">${escapeHtml(outlineHint)}</p>
              ${outlineDomainNav}
            </div>
          </header>
          <div id="graph-outline-content" class="graph-outline-content"></div>
          <div id="graph-outline-error"></div>
        </div>
      </section>
    `

    if (stale()) return

    const pageEl = container.querySelector<HTMLElement>('.page-graph')!
    const galaxyPanel = container.querySelector<HTMLDivElement>('#graph-galaxy-panel')!
    const outlinePanel = container.querySelector<HTMLDivElement>('#graph-outline-panel')!
    const stageEl = container.querySelector<HTMLElement>('.graph-stage')!
    const errEl = container.querySelector<HTMLDivElement>('#graph-error')!
    const outlineErrEl = container.querySelector<HTMLDivElement>('#graph-outline-error')!
    const outlineContentEl = container.querySelector<HTMLDivElement>('#graph-outline-content')!
    const canvasEl = container.querySelector<HTMLDivElement>('#graph-canvas')!
    const themeBtn = container.querySelector<HTMLButtonElement>('#graph-theme-btn')
    const fitBtn = container.querySelector<HTMLButtonElement>('#graph-fit-btn')
    let sessionStarting = false
    let filterDomainId = ''

    const showError = (message: string) => {
      const html = `<div class="alert alert-error">${escapeHtml(message)}</div>`
      if (viewMode === 'galaxy') {
        errEl.innerHTML = html
      } else {
        outlineErrEl.innerHTML = html
      }
    }

    const startTopic = (domainId: string, nodeKey: string, layerKey: string) => {
      if (sessionStarting) return
      sessionStarting = true
      const nodeTitle = nodeTitleByKey.get(`${domainId}:${nodeKey}`) ?? '学习节点'
      if (viewMode === 'galaxy') errEl.innerHTML = ''
      else outlineErrEl.innerHTML = ''
      void startNodeSession({
        domainId,
        nodeKey,
        layer: layerKey,
        nodeTitle,
        pageEl,
        onError: showError,
      }).finally(() => {
        sessionStarting = false
      })
    }

    const updateThemeButton = (theme: GraphCanvasTheme) => {
      if (!themeBtn) return
      const next = toggleGraphCanvasTheme(theme)
      themeBtn.textContent = graphThemeToggleLabel(theme)
      themeBtn.setAttribute('aria-pressed', theme === 'sky' ? 'true' : 'false')
      themeBtn.title = `切换为${graphCanvasThemeLabel(next)}主题`
    }

    const mountGraph = (theme: GraphCanvasTheme) => {
      if (!stageEl || !canvasEl) return
      applyGraphCanvasTheme(stageEl, theme)
      disposeActiveGraph()
      canvasEl.innerHTML = ''
      errEl.innerHTML = ''
      try {
        const mount = mountMultiDomainKnowledgeGraph({
          container: canvasEl,
          domains: loaded,
          theme,
          onDomainClick: (domainId) => navigateHash(`/tree/${domainId}`),
          onTopicClick: startTopic,
        })
        activeGraphMount = mount
        activeGraphDestroy = mount.destroy
      } catch (e) {
        console.error('[graph] mount failed', e)
        canvasEl.innerHTML = '<p class="tree-graph-fallback">图谱暂时无法显示，请稍后重试</p>'
      }
      updateThemeButton(theme)
    }

    const paintOutline = () => {
      outlineContentEl.innerHTML = renderGraphOutlineHtml(loaded, filterDomainId)
      bindGraphOutline(outlineContentEl, startTopic, uiSignal)
    }

    const updateViewToggleUi = () => {
      container.querySelectorAll<HTMLButtonElement>('.graph-view-toggle-btn').forEach((btn) => {
        const mode = btn.dataset.viewMode as GraphViewMode
        const active = mode === viewMode
        btn.classList.toggle('is-active', active)
        btn.setAttribute('aria-selected', active ? 'true' : 'false')
      })
    }

    const syncDomainChips = (domainId: string) => {
      container.querySelectorAll<HTMLButtonElement>('.graph-domain-chip').forEach((c) => {
        c.classList.toggle('is-active', (c.dataset.domainId ?? '') === domainId)
      })
    }

    const applyViewMode = (next: GraphViewMode) => {
      viewMode = next
      setGraphViewMode(next)
      pageEl.classList.toggle('page-graph--immersive', next === 'galaxy')
      pageEl.classList.toggle('page-graph--outline', next === 'outline')
      galaxyPanel.hidden = next !== 'galaxy'
      outlinePanel.hidden = next !== 'outline'
      updateViewToggleUi()
      if (next === 'galaxy') {
        mountGraph(canvasTheme)
      } else {
        disposeActiveGraph()
        paintOutline()
      }
    }

    if (viewMode === 'galaxy') {
      mountGraph(canvasTheme)
    } else {
      paintOutline()
    }

    container.querySelectorAll<HTMLButtonElement>('.graph-view-toggle-btn').forEach((btn) => {
      btn.addEventListener(
        'click',
        () => {
          const next = btn.dataset.viewMode as GraphViewMode
          if (next === viewMode) return
          applyViewMode(next)
        },
        { signal: uiSignal }
      )
    })

    themeBtn?.addEventListener(
      'click',
      () => {
        const next = toggleGraphCanvasTheme(canvasTheme)
        setGraphCanvasTheme(next)
        canvasTheme = next
        mountGraph(next)
      },
      { signal: uiSignal }
    )

    fitBtn?.addEventListener(
      'click',
      () => {
        activeGraphMount?.fit()
        if (showDomainNav) {
          filterDomainId = ''
          syncDomainChips('')
        }
      },
      { signal: uiSignal }
    )

    const wireNavCollapse = (prefix: string) => {
      const navToggle = container.querySelector<HTMLButtonElement>(`#${prefix}-domain-nav-toggle`)
      const navEl = container.querySelector<HTMLDivElement>(`#${prefix}-domain-nav`)
      navToggle?.addEventListener(
        'click',
        (e) => {
          e.stopPropagation()
          if (!navEl) return
          const expanded = navEl.classList.contains('is-collapsed')
          setDomainNavExpanded(navEl, navToggle, expanded)
        },
        { signal: uiSignal }
      )
    }

    if (showDomainNav) {
      wireNavCollapse('graph-outline')
      wireDomainNav(
        container,
        summaries.map((s) => ({ id: s.id, name: s.name })),
        (domainId) => {
          filterDomainId = domainId
          syncDomainChips(domainId)
          if (viewMode === 'galaxy') {
            const graph = activeGraphMount
            if (!graph) return
            if (!domainId) {
              graph.fit()
              return
            }
            graph.focusDomain(domainId)
          } else {
            paintOutline()
          }
        },
        uiSignal
      )
    }

    wireGalaxyHud(stageEl, uiSignal)
  } catch (e) {
    if (stale()) return
    container.innerHTML = `
      <section class="page page-graph">
        <div class="alert alert-error">${e instanceof ApiError ? e.message : '加载失败'}</div>
        <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
          <button type="button" class="btn btn-secondary btn-sm" id="graph-retry-btn">重试</button>
        </p>
      </section>
    `
    container.querySelector<HTMLButtonElement>('#graph-retry-btn')?.addEventListener('click', () => {
      void renderGraph(container)
    })
  } finally {
    if (!stale()) {
      setAppBusy(false)
      refreshLLMStatusAfterBusy()
    }
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function shortDomainLabel(name: string): string {
  const t = name.trim()
  if (t.length <= 18) return t
  return t.slice(0, 17) + '…'
}

function setDomainNavExpanded(navEl: HTMLElement, navToggle: HTMLButtonElement | null, expanded: boolean): void {
  navEl.classList.toggle('is-collapsed', !expanded)
  if (navToggle) {
    navToggle.setAttribute('aria-expanded', expanded ? 'true' : 'false')
    navToggle.title = expanded ? '收起' : '展开'
    navToggle.setAttribute('aria-label', expanded ? '收起领域搜索' : '展开领域搜索')
  }
}

function setGraphHudExpanded(
  anchorEl: HTMLElement,
  hudToggle: HTMLButtonElement | null,
  drawerEl: HTMLElement | null,
  expanded: boolean
): void {
  anchorEl.classList.toggle('is-expanded', expanded)
  if (hudToggle) {
    hudToggle.setAttribute('aria-expanded', expanded ? 'true' : 'false')
    hudToggle.setAttribute('aria-label', expanded ? '收起知识图谱控制面板' : '展开知识图谱控制面板')
    hudToggle.title = expanded ? '收起控制面板' : '展开控制面板'
    hudToggle.classList.toggle('is-active', expanded)
  }
  if (drawerEl) drawerEl.setAttribute('aria-hidden', expanded ? 'false' : 'true')
}

function wireGalaxyHud(stageEl: HTMLElement, signal: AbortSignal): void {
  const topHud = stageEl.querySelector<HTMLElement>('.graph-float--top')
  const legendWrap = stageEl.querySelector<HTMLElement>('.graph-float--legend-wrap')
  const anchorEl = stageEl.querySelector<HTMLDivElement>('#graph-hud-anchor')
  const hudToggle = stageEl.querySelector<HTMLButtonElement>('#graph-hud-toggle')
  const hudClose = stageEl.querySelector<HTMLButtonElement>('#graph-hud-close')
  const drawerEl = stageEl.querySelector<HTMLDivElement>('#graph-hud-drawer')
  let idleTimer = 0

  const hudExpanded = () => {
    const hudOpen = anchorEl?.classList.contains('is-expanded') ?? false
    const nav = stageEl.querySelector<HTMLDivElement>('#graph-domain-nav')
    const navOpen = nav && !nav.classList.contains('is-collapsed')
    return Boolean(hudOpen || navOpen)
  }

  const syncHudIdle = () => {
    stageEl.classList.toggle('is-hud-idle', !hudExpanded())
  }

  const resetIdleTimer = () => {
    stageEl.classList.remove('is-hud-idle')
    window.clearTimeout(idleTimer)
    idleTimer = window.setTimeout(syncHudIdle, 2800)
  }

  const onActivity = () => resetIdleTimer()
  ;['mousemove', 'mousedown', 'wheel', 'touchstart', 'keydown'].forEach((eventName) => {
    stageEl.addEventListener(eventName, onActivity, { signal, passive: true })
  })

  hudToggle?.addEventListener(
    'click',
    () => {
      if (!anchorEl) return
      const expanded = !anchorEl.classList.contains('is-expanded')
      setGraphHudExpanded(anchorEl, hudToggle, drawerEl, expanded)
      resetIdleTimer()
    },
    { signal }
  )
  hudClose?.addEventListener(
    'click',
    () => {
      if (!anchorEl) return
      setGraphHudExpanded(anchorEl, hudToggle, drawerEl, false)
      resetIdleTimer()
    },
    { signal }
  )

  const navHeader = stageEl.querySelector<HTMLElement>('#graph-domain-nav .graph-domain-nav-header')
  const navToggle = stageEl.querySelector<HTMLButtonElement>('#graph-domain-nav-toggle')
  const navEl = stageEl.querySelector<HTMLDivElement>('#graph-domain-nav')
  if (navToggle && navEl) {
    navHeader?.addEventListener(
      'click',
      (e) => {
        if ((e.target as HTMLElement).closest('.graph-domain-nav-toggle')) return
        const expanded = navEl.classList.contains('is-collapsed')
        setDomainNavExpanded(navEl, navToggle, expanded)
        resetIdleTimer()
      },
      { signal }
    )
  }

  if (topHud) {
    topHud.addEventListener('mouseenter', () => stageEl.classList.remove('is-hud-idle'), { signal })
    topHud.addEventListener('focusin', () => stageEl.classList.remove('is-hud-idle'), { signal })
  }
  if (legendWrap) {
    legendWrap.addEventListener('mouseenter', () => stageEl.classList.remove('is-hud-idle'), { signal })
    legendWrap.addEventListener('focusin', () => stageEl.classList.remove('is-hud-idle'), { signal })
  }

  resetIdleTimer()
}

function wireDomainNav(
  container: HTMLElement,
  domains: Array<{ id: string; name: string }>,
  onSelect: (domainId: string) => void,
  signal: AbortSignal
): void {
  const searches = container.querySelectorAll<HTMLInputElement>('[id$="-domain-search"]')
  const chips = container.querySelectorAll<HTMLButtonElement>('.graph-domain-chip')

  searches.forEach((search) => {
    search.addEventListener(
      'input',
      () => {
        const q = search.value.trim().toLowerCase()
        chips.forEach((chip) => {
          const id = chip.dataset.domainId ?? ''
          if (!id) {
            chip.classList.remove('is-hidden')
            return
          }
          const meta = domains.find((d) => d.id === id)
          const label = (meta?.name ?? chip.textContent ?? '').toLowerCase()
          chip.classList.toggle('is-hidden', Boolean(q) && !label.includes(q))
        })
      },
      { signal }
    )
  })

  chips.forEach((chip) => {
    chip.addEventListener(
      'click',
      () => {
        onSelect(chip.dataset.domainId ?? '')
      },
      { signal }
    )
  })
}
