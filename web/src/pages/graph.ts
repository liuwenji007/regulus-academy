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
import { mountMultiDomainKnowledgeGraph, type KnowledgeGraphMount } from '../lib/knowledge-graph'
import { startNodeSession } from '../lib/start-node-session'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'

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
    { label: '知识银河' },
  ])

  let canvasTheme = getGraphCanvasTheme()

  container.innerHTML = `
    <section class="page page-graph page-graph--immersive">
      <div class="graph-stage graph-stage--loading" data-graph-theme="${canvasTheme}">
        <div class="graph-loading">
          <div class="spinner" aria-hidden="true"></div>
          <p>正在加载知识银河…</p>
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
            <h1 class="page-title">知识银河</h1>
            <p class="page-sub">跨领域总览你的学习全景。创建第一门课后，各领域会像星座一样在这里汇聚展示。</p>
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
    const graphHint =
      summaries.length > 1
        ? `${summaries.length} 个领域 · 相关课程相邻排布、子模块环绕主领域 · 单击定位 · 双击领域进课程 · 拖动力导向${derivedHint}`
        : `力导向知识图谱 · 模块扇形簇、节点沿路径点亮 · 单击模块定位 · 拖动力导向${derivedHint}`
    const domainNavHtml = showDomainNav
      ? `
          <div class="graph-float-panel graph-domain-nav" id="graph-domain-nav">
            <div class="graph-domain-nav-header">
              <span class="graph-domain-nav-label">搜索领域</span>
              <button type="button" class="graph-domain-nav-toggle" id="graph-domain-nav-toggle" title="收起" aria-expanded="true">▴</button>
            </div>
            <div class="graph-domain-nav-body">
              <input
                type="search"
                id="graph-domain-search"
                class="input graph-domain-search"
                placeholder="搜索领域…"
                autocomplete="off"
                aria-label="搜索领域"
              />
              <div class="graph-domain-chips" id="graph-domain-chips" role="listbox" aria-label="领域列表">
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
      : ''

    container.innerHTML = `
      <section class="page page-graph page-graph--immersive">
        <div class="graph-stage" data-graph-theme="${canvasTheme}">
          <div id="graph-canvas" class="graph-canvas" role="img" aria-label="多领域知识银河"></div>

          <div class="graph-float graph-float--top">
            <div class="graph-float-top-main">
              <div class="graph-float-panel graph-float-title">
                <h1 class="graph-title">知识银河</h1>
                <p class="graph-hint">${escapeHtml(graphHint)}</p>
              </div>
              ${domainNavHtml}
            </div>
            <div class="graph-float--actions">
              <button
                type="button"
                class="btn btn-ghost btn-sm graph-float-panel graph-theme-btn"
                id="graph-theme-btn"
                aria-pressed="${canvasTheme === 'sky' ? 'true' : 'false'}"
                title="切换为${escapeHtml(graphCanvasThemeLabel(toggleGraphCanvasTheme(canvasTheme)))}主题"
              >${escapeHtml(graphThemeToggleLabel(canvasTheme))}</button>
              <button type="button" class="btn btn-ghost btn-sm graph-float-panel" id="graph-fit-btn">重置视图</button>
            </div>
          </div>

          <div class="graph-float graph-float--legend graph-float-panel" aria-hidden="true">
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain"></i>领域</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain-starlit"></i>领域圆满</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--module"></i>模块</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--module-lit"></i>子领域学完</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--pending"></i>未开始</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--progress"></i>进行中</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--done"></i>已学会</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--focus"></i>聚焦</span>
            <span class="tree-graph-legend-item graph-legend-lod">缩放：${summaries.length > 1 ? '领域总览 → 模块簇 → 节点路径' : '模块簇 → 节点路径'}</span>
          </div>

          <div id="graph-error" class="graph-float graph-float--error"></div>
        </div>
      </section>
    `

    if (stale()) return

    const pageEl = container.querySelector<HTMLElement>('.page-graph')!
    const stageEl = container.querySelector<HTMLElement>('.graph-stage')!
    const errEl = container.querySelector<HTMLDivElement>('#graph-error')!
    const canvasEl = container.querySelector<HTMLDivElement>('#graph-canvas')!
    const themeBtn = container.querySelector<HTMLButtonElement>('#graph-theme-btn')
    const fitBtn = container.querySelector<HTMLButtonElement>('#graph-fit-btn')
    let sessionStarting = false

    const updateThemeButton = (theme: GraphCanvasTheme) => {
      if (!themeBtn) return
      const next = toggleGraphCanvasTheme(theme)
      themeBtn.textContent = graphThemeToggleLabel(theme)
      themeBtn.setAttribute('aria-pressed', theme === 'sky' ? 'true' : 'false')
      themeBtn.title = `切换为${graphCanvasThemeLabel(next)}主题`
    }

    const mountGraph = (theme: GraphCanvasTheme) => {
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
          onTopicClick: (domainId, nodeKey, layerKey) => {
            if (sessionStarting) return
            sessionStarting = true
            const nodeTitle = nodeTitleByKey.get(`${domainId}:${nodeKey}`) ?? '学习节点'
            errEl.innerHTML = ''
            void startNodeSession({
              domainId,
              nodeKey,
              layer: layerKey,
              nodeTitle,
              pageEl,
              onError: (message) => {
                errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(message)}</div>`
              },
            }).finally(() => {
              sessionStarting = false
            })
          },
        })
        activeGraphMount = mount
        activeGraphDestroy = mount.destroy
      } catch (e) {
        console.error('[graph] mount failed', e)
        canvasEl.innerHTML = '<p class="tree-graph-fallback">图谱暂时无法显示，请稍后重试</p>'
      }
      updateThemeButton(theme)
    }

    mountGraph(canvasTheme)

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

    const setActiveChip = (domainId: string) => {
      container.querySelectorAll<HTMLButtonElement>('.graph-domain-chip').forEach((c) => {
        c.classList.toggle('is-active', (c.dataset.domainId ?? '') === domainId)
      })
    }

    fitBtn?.addEventListener(
      'click',
      () => {
        activeGraphMount?.fit()
        if (showDomainNav) setActiveChip('')
      },
      { signal: uiSignal }
    )

    if (showDomainNav) {
      const navToggle = container.querySelector<HTMLButtonElement>('#graph-domain-nav-toggle')
      const navEl = container.querySelector<HTMLDivElement>('#graph-domain-nav')
      navToggle?.addEventListener('click', () => {
        const collapsed = navEl?.classList.toggle('is-collapsed')
        if (navToggle) {
          navToggle.setAttribute('aria-expanded', collapsed ? 'false' : 'true')
          navToggle.title = collapsed ? '展开' : '收起'
        }
      }, { signal: uiSignal })
    }

    if (showDomainNav) {
      wireDomainNav(
        container,
        summaries.map((s) => ({ id: s.id, name: s.name })),
        setActiveChip,
        uiSignal
      )
    }
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
  if (t.length <= 12) return t
  return t.slice(0, 11) + '…'
}

function wireDomainNav(
  container: HTMLElement,
  domains: Array<{ id: string; name: string }>,
  setActive: (domainId: string) => void,
  signal: AbortSignal
): void {
  const search = container.querySelector<HTMLInputElement>('#graph-domain-search')
  const chips = container.querySelectorAll<HTMLButtonElement>('.graph-domain-chip')

  search?.addEventListener(
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

  chips.forEach((chip) => {
    chip.addEventListener(
      'click',
      () => {
        const id = chip.dataset.domainId ?? ''
        setActive(id)
        const graph = activeGraphMount
        if (!graph) return
        if (!id) {
          graph.fit()
          return
        }
        graph.focusDomain(id)
      },
      { signal }
    )
  })
}
