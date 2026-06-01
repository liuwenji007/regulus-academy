import {
  getDomains,
  getDomainTree,
  getUserProgress,
  ApiError,
  type UserProgress,
} from '../lib/api'
import { setAppBusy } from '../lib/app-busy'
import { navigateHash } from '../lib/navigate'
import { normalizeKnowledgeTree } from '../lib/tree-normalize'
import { mountMultiDomainKnowledgeGraph, type KnowledgeGraphMount } from '../lib/knowledge-graph'
import { startNodeSession } from '../lib/start-node-session'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'

const TREE_FOCUS_PREFIX = 'regulus:treeFocus:'

let graphRenderGen = 0
let activeGraphDestroy: (() => void) | null = null

function disposeActiveGraph(): void {
  if (activeGraphDestroy) {
    try {
      activeGraphDestroy()
    } catch {
      /* ignore */
    }
    activeGraphDestroy = null
  }
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

  clearTreeSessionOverlay()
  disposeActiveGraph()

  void updateSidebar({ active: 'graph' })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '知识图谱' },
  ])

  container.innerHTML = `
    <section class="page page-graph page-graph--immersive">
      <div class="graph-stage graph-stage--loading">
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
            <p class="page-sub">跨领域总览你的学习路径。创建第一门课后，各领域的知识点会在这里汇聚展示。</p>
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
          tree,
          progressMap,
          focusKeys: readTreeFocus(summary.id),
        }
      })
    )
    if (stale()) return

    const nodeTitleByKey = new Map<string, string>()
    for (const entry of loaded) {
      for (const layer of entry.tree.layers) {
        for (const node of layer.nodes) {
          nodeTitleByKey.set(`${entry.domainId}:${node.key}`, node.title)
        }
      }
    }

    container.innerHTML = `
      <section class="page page-graph page-graph--immersive">
        <div class="graph-stage">
          <div id="graph-canvas" class="graph-canvas" role="img" aria-label="多领域知识图谱"></div>

          <div class="graph-float graph-float--top">
            <div class="graph-float-panel graph-float-title">
              <h1 class="graph-title">知识图谱</h1>
              <p class="graph-hint">${summaries.length} 个领域 · 拖拽缩放浏览 · 点击知识点进入学习</p>
            </div>
            <button type="button" class="btn btn-ghost btn-sm graph-float-panel" id="graph-fit-btn">重置视图</button>
          </div>

          <div class="graph-float graph-float--legend graph-float-panel" aria-hidden="true">
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--domain"></i>领域</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--pending"></i>未开始</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--progress"></i>进行中</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--done"></i>已学会</span>
            <span class="tree-graph-legend-item"><i class="tree-graph-swatch tree-graph-swatch--focus"></i>聚焦</span>
          </div>

          <div id="graph-error" class="graph-float graph-float--error"></div>
        </div>
      </section>
    `

    if (stale()) return

    const pageEl = container.querySelector<HTMLElement>('.page-graph')!
    const errEl = container.querySelector<HTMLDivElement>('#graph-error')!
    const canvasEl = container.querySelector<HTMLDivElement>('#graph-canvas')!
    let sessionStarting = false
    let graph: KnowledgeGraphMount | null = null

    try {
      graph = mountMultiDomainKnowledgeGraph({
        container: canvasEl,
        domains: loaded,
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
      activeGraphDestroy = graph.destroy
    } catch (e) {
      console.error('[graph] mount failed', e)
      canvasEl.innerHTML = '<p class="tree-graph-fallback">图谱暂时无法显示，请稍后重试</p>'
    }

    container.querySelector<HTMLButtonElement>('#graph-fit-btn')?.addEventListener('click', () => {
      graph?.fit()
    })
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
