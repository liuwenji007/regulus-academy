import {
  getDomainTree,
  getUserProgress,
  getDomains,
  exportDomainSkillZip,
  getExtendEligibility,
  ApiError,
  type KnowledgeTree,
} from '../lib/api'
import { clearAppBusyIfAfter, getAppBusyReason } from '../lib/app-busy'
import { delayMs, fadeOutAndRemove, waitForNextPaint } from '../lib/loading-transition'
import { clearPrefetchTree, peekPrefetchTree } from '../lib/course-prefetch'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { bindNodeList, renderNodeItem } from '../lib/node-list'
import { normalizeKnowledgeTree, nodeTitleMap } from '../lib/tree-normalize'
import { startNodeSession } from '../lib/start-node-session'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'
import { showDomainConfirm } from '../components/domain-confirm'
import { showExtendConfirm } from '../components/extend-confirm'
import {
  consumeRegenerateToast,
  handleDomainDelete,
  handleDomainExtend,
  handleDomainRegenerate,
} from '../lib/domain-actions'
import type { NavKey } from '../components/sidebar'

const TREE_FOCUS_PREFIX = 'regulus:treeFocus:'

let treeRenderGen = 0

function isRetryableLoadError(e: unknown): boolean {
  if (e instanceof ApiError) {
    const m = e.message
    return (
      m.includes('请求失败') ||
      m.includes('无法解析') ||
      m.includes('not found') ||
      m.includes('未找到')
    )
  }
  if (e instanceof TypeError) return true
  return e instanceof Error && e.message === 'Failed to fetch'
}

async function loadTreeResilient(
  domainId: string,
  prefetched: KnowledgeTree | null,
  isStale: () => boolean
): Promise<KnowledgeTree> {
  if (prefetched) return prefetched

  const waits = [0, 400, 900, 1600, 2500]
  let lastErr: unknown
  for (const ms of waits) {
    if (isStale()) throw new DOMException('stale', 'AbortError')
    if (ms > 0) await new Promise((r) => setTimeout(r, ms))
    try {
      return await getDomainTree(domainId)
    } catch (e) {
      lastErr = e
      if (!isRetryableLoadError(e)) throw e
    }
  }
  throw lastErr
}

function formatLoadError(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof DOMException && e.name === 'AbortError') return ''
  if (e instanceof Error && e.message) return e.message
  return '加载失败，请稍后重试'
}

function treeLoadingHtml(title: string, hint?: string): string {
  const hintHtml = hint
    ? `<p class="page-loading-hint">${escapeHtml(hint)}</p>`
    : ''
  return `
    <section class="page page-tree">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>${escapeHtml(title)}</p>
        ${hintHtml}
      </div>
    </section>
  `
}

interface TreeFocusState {
  keys: string[]
  label: string
}

function readTreeFocus(domainId: string): TreeFocusState | null {
  try {
    const raw = sessionStorage.getItem(TREE_FOCUS_PREFIX + domainId)
    if (!raw) return null
    const parsed = JSON.parse(raw) as TreeFocusState
    if (!Array.isArray(parsed.keys) || parsed.keys.length === 0) return null
    return parsed
  } catch {
    return null
  }
}

export async function renderTree(
  container: HTMLElement,
  domainId: string,
  _nav: NavKey = 'tree'
): Promise<void> {
  const gen = ++treeRenderGen
  const stale = () => gen !== treeRenderGen

  clearTreeSessionOverlay()

  const prefetchedRaw = peekPrefetchTree(domainId)
  const buildHandoff = getAppBusyReason() === 'build'
  container.innerHTML = treeLoadingHtml(
    buildHandoff ? '正在生成知识树…' : '正在加载课程…',
    buildHandoff
      ? 'AI 正在规划学习路径，通常需要 30 秒～2 分钟，请稍候'
      : prefetchedRaw
        ? '正在同步学习进度…'
        : '正在获取课程列表，请稍候'
  )

  const loadStartedAt = Date.now()

  try {
    const [treeRaw, progress, domains, extendElig] = await Promise.all([
      loadTreeResilient(domainId, prefetchedRaw, stale),
      getUserProgress(domainId).catch(() => []),
      getDomains().catch(() => []),
      getExtendEligibility(domainId).catch(() => null),
    ])
    if (stale()) return

    const domainMeta = domains.find((d) => d.id === domainId)
    const tree = normalizeKnowledgeTree(treeRaw, domainId, domainMeta?.name)
    clearPrefetchTree(domainId)

    const canExport = domainMeta?.source === 'generated' || domainMeta?.source === 'personalized'
    localStorage.setItem('regulus:lastDomainId', domainId)

    const progressMap = new Map(progress.map((p) => [p.nodeKey, p]))
    const titleMap = nodeTitleMap(tree)
    const completed = progress.filter((p) => p.status === 'completed').length
    const total = tree.layers.reduce((n, l) => n + l.nodes.length, 0)

    await updateSidebar({
      active: 'tree',
      domainId,
      domainName: tree.domainName,
      domainNodeTotal: total,
      domainCompleted: completed,
    })
    if (stale()) return

    setBreadcrumb([
      { label: '开始学习', href: '#/' },
      { label: '我的课程', href: '#/courses' },
      { label: tree.domainName },
    ])

    const focus = readTreeFocus(domainId)
    const focusSet = new Set(focus?.keys ?? [])

    const pct = total > 0 ? Math.round((completed / total) * 100) : 0
    const extendEligible = extendElig?.eligible === true

    let nextHint = ''
    outer: for (const layer of tree.layers) {
      for (const node of layer.nodes) {
        const st = progressMap.get(node.key)
        if (!st || st.status !== 'completed') {
          nextHint = node.title
          break outer
        }
      }
    }

    const layersHtml = tree.layers
      .map((layer) => {
        const nodesHtml = layer.nodes
          .map((node) =>
            renderNodeItem({
              node,
              layerKey: layer.key,
              progressMap,
              focusSet,
              titleMap,
            })
          )
          .join('')
        return `
          <section class="layer card">
            <div class="layer-header">
              <span class="layer-label">${escapeHtml(layer.label)}</span>
              <span class="layer-meta">${escapeHtml(layer.time)}</span>
            </div>
            <p class="layer-goal">${escapeHtml(layer.goal)}</p>
            <ul class="node-list">${nodesHtml}</ul>
          </section>
        `
      })
      .join('')

    const minLoadingMs = 360
    const elapsed = Date.now() - loadStartedAt
    if (elapsed < minLoadingMs) {
      await delayMs(minLoadingMs - elapsed)
    }
    if (stale()) return

    const loadingEl = container.querySelector<HTMLElement>('.page-loading')
    if (loadingEl) await fadeOutAndRemove(loadingEl)

    container.innerHTML = `
      <section class="page page-tree">
        <header class="page-header">
          <div class="page-header-row">
            <div class="page-header-main">
              <h1 class="page-title">${escapeHtml(tree.domainName)}</h1>
              <div class="page-tree-meta">
                <p class="page-sub page-tree-hint">
                  ${
                    nextHint
                      ? `<span class="page-tree-hint-label">推荐下一步</span><span class="page-tree-hint-node">${escapeHtml(nextHint)}</span>`
                      : '点击节点开始微训练'
                  }
                </p>
              </div>
            </div>
            <div class="domain-actions">
              ${extendEligible ? '<button type="button" class="btn btn-primary btn-sm" id="domain-extend-btn" title="追加进阶学习节点">解锁进阶路径</button>' : ''}
              ${canExport ? '<button type="button" class="btn btn-ghost btn-sm" id="domain-export-btn">导出 Skill 包</button>' : ''}
              <button type="button" class="btn btn-ghost btn-sm" id="domain-regenerate-btn" title="按当前学习画像重新生成课程">重新生成</button>
              <button type="button" class="btn btn-ghost btn-sm btn-danger-text" id="domain-delete-btn">移除课程</button>
            </div>
          </div>
        </header>

        <div class="progress-card card">
          <div class="progress-stats">
            <span class="progress-label">学习进度</span>
            <span class="progress-value">${completed} / ${total} 节点 · ${pct}%</span>
          </div>
          <div class="progress-bar" role="progressbar" aria-valuenow="${pct}" aria-valuemin="0" aria-valuemax="100">
            <div class="progress-fill" style="width:${pct}%"></div>
          </div>
        </div>

        <div id="tree-toast"></div>
        <div id="tree-error"></div>

        ${focus?.label ? `
          <div class="tree-focus-banner card">
            <span class="tree-focus-banner-label">当前聚焦</span>
            <strong>${escapeHtml(focus.label)}</strong>
            <span class="tree-focus-banner-hint">完整知识树已展开，高亮节点为本次学习重点，其余节点可随时拓展</span>
          </div>
        ` : ''}

        <div class="tree-layers">${layersHtml}</div>
      </section>
    `

    if (stale()) return
    await waitForNextPaint()

    const errEl = container.querySelector<HTMLDivElement>('#tree-error')!
    const toastEl = container.querySelector<HTMLDivElement>('#tree-toast')!
    const regenToast = consumeRegenerateToast()
    if (regenToast && toastEl) {
      toastEl.innerHTML = `<div class="alert alert-success">${escapeHtml(regenToast)}</div>`
    }
    const pageEl = container.querySelector<HTMLElement>('.page-tree')!

    const openNode = (nodeKey: string, layer: string) => {
      const nodeTitle =
        tree.layers.flatMap((l) => l.nodes).find((n) => n.key === nodeKey)?.title ?? '学习节点'
      errEl.innerHTML = ''
      void startNodeSession({
        domainId,
        nodeKey,
        layer,
        nodeTitle,
        pageEl,
        onError: (message) => {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(message)}</div>`
        },
      })
    }

    const bindDomainAction = (
      btnId: string,
      action: 'delete' | 'regenerate'
    ) => {
      container.querySelector<HTMLButtonElement>(btnId)?.addEventListener('click', () => {
        void (async () => {
          const outcome = await showDomainConfirm({
            domainId,
            domainName: tree.domainName,
            action,
          })
          if (!outcome.ok) return
          if (outcome.action === 'delete') {
            await handleDomainDelete(domainId)
            return
          }
          await handleDomainRegenerate(domainId, outcome.result.tree!.domainId, outcome.result)
        })()
      })
    }
    bindDomainAction('#domain-delete-btn', 'delete')
    bindDomainAction('#domain-regenerate-btn', 'regenerate')

    container.querySelector<HTMLButtonElement>('#domain-extend-btn')?.addEventListener('click', () => {
      void (async () => {
        const outcome = await showExtendConfirm({
          domainName: tree.domainName,
          completed,
          total,
          minRatio: extendElig?.minRatio ?? 0.8,
        })
        if (!outcome.ok) return
        const btn = container.querySelector<HTMLButtonElement>('#domain-extend-btn')
        if (btn) btn.disabled = true
        try {
          await handleDomainExtend(domainId, outcome.goal || undefined)
          toastEl.innerHTML = '<div class="alert alert-success">进阶路径已解锁，页面即将刷新</div>'
          setTimeout(() => {
            void renderTree(container, domainId, _nav)
          }, 600)
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '扩展失败')}</div>`
          if (btn) btn.disabled = false
        }
      })()
    })

    container.querySelector<HTMLButtonElement>('#domain-export-btn')?.addEventListener('click', () => {
      void (async () => {
        const btn = container.querySelector<HTMLButtonElement>('#domain-export-btn')
        if (!btn) return
        btn.disabled = true
        const prev = btn.textContent
        btn.textContent = '导出中…'
        try {
          const { slug } = await exportDomainSkillZip(domainId)
          errEl.innerHTML = `<div class="alert alert-success">已下载 <code>${slug}-skill.zip</code>：解压后整目录放入 Agent 的 skills 目录即可练习；如需贡献社区，将其中 <code>domains/${slug}/</code> 按 CONTRIBUTING.md 提 PR</div>`
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(e instanceof ApiError ? e.message : '导出失败')}</div>`
        } finally {
          btn.disabled = false
          btn.textContent = prev ?? '导出 Skill 包'
        }
      })()
    })

    bindNodeList(container, (nodeKey, layer) => void openNode(nodeKey, layer))

    const firstFocus = container.querySelector<HTMLElement>('.node-item--focus')
    firstFocus?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  } catch (e) {
    if (stale()) return
    const msg = formatLoadError(e)
    if (!msg) return

    const minLoadingMs = 600
    const elapsed = Date.now() - loadStartedAt
    if (elapsed < minLoadingMs) {
      await new Promise((r) => setTimeout(r, minLoadingMs - elapsed))
    }
    if (stale()) return

    void updateSidebar({ active: 'tree', domainId })
    setBreadcrumb([
      { label: '开始学习', href: '#/' },
      { label: '我的课程', href: '#/courses' },
      { label: '课程' },
    ])
    container.innerHTML = `
      <section class="page page-tree">
        <div class="alert alert-error">${escapeHtml(msg)}</div>
        <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
          <button type="button" class="btn btn-secondary btn-sm" id="tree-retry-btn">重试</button>
        </p>
      </section>
    `
    container.querySelector<HTMLButtonElement>('#tree-retry-btn')?.addEventListener('click', () => {
      void renderTree(container, domainId, _nav)
    })
  } finally {
    if (!stale()) {
      clearAppBusyIfAfter('build', refreshLLMStatusAfterBusy)
    }
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
