import {
  getDomainTree,
  getUserProgress,
  getDomains,
  exportDomain,
  ApiError,
  type KnowledgeTree,
} from '../lib/api'
import { setAppBusy } from '../lib/app-busy'
import { clearPrefetchTree, peekPrefetchTree } from '../lib/course-prefetch'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import { graphNavLink } from '../lib/graph-link'
import { normalizeKnowledgeTree } from '../lib/tree-normalize'
import { startNodeSession } from '../lib/start-node-session'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'
import { showDomainConfirm } from '../components/domain-confirm'
import { handleDomainDelete, handleDomainRegenerate } from '../lib/domain-actions'
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

function treeLoadingHtml(hint: string): string {
  return `
    <section class="page page-tree">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>正在加载课程…</p>
        <p class="page-loading-hint">${hint}</p>
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
  container.innerHTML = treeLoadingHtml(
    prefetchedRaw ? '正在同步学习进度…' : '正在获取课程列表，请稍候'
  )

  const loadStartedAt = Date.now()

  try {
    const [treeRaw, progress, domains] = await Promise.all([
      loadTreeResilient(domainId, prefetchedRaw, stale),
      getUserProgress(domainId).catch(() => []),
      getDomains().catch(() => []),
    ])
    if (stale()) return

    const domainMeta = domains.find((d) => d.id === domainId)
    const tree = normalizeKnowledgeTree(treeRaw, domainId, domainMeta?.name)
    clearPrefetchTree(domainId)

    const canExport = domainMeta?.source === 'generated' || domainMeta?.source === 'personalized'
    localStorage.setItem('regulus:lastDomainId', domainId)

    const progressMap = new Map(progress.map((p) => [p.nodeKey, p]))
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
          .map((node) => {
            const st = progressMap.get(node.key)
            const statusClass = st?.status ?? 'pending'
            const resumeTag =
              statusClass === 'completed'
                ? '<span class="node-resume-tag node-resume-tag--review">复习</span>'
                : statusClass === 'in_progress'
                  ? '<span class="node-resume-tag">继续</span>'
                  : ''
            const isFocus = focusSet.has(node.key)
            const focusTag = isFocus ? '<span class="node-focus-tag">当前聚焦</span>' : ''
            return `
              <li class="node-item ${isFocus ? 'node-item--focus' : ''}" data-node="${node.key}" data-layer="${layer.key}" tabindex="0" role="button">
                <span class="node-status ${statusClass}" aria-hidden="true"></span>
                <span class="node-title">${escapeHtml(node.title)}</span>
                ${focusTag}
                ${resumeTag}
              </li>
            `
          })
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
                ${graphNavLink({ ariaLabel: '在知识图谱中查看本课程' })}
              </div>
            </div>
            <div class="domain-actions">
              ${canExport ? '<button type="button" class="btn btn-ghost btn-sm" id="domain-export-btn">导出 Skill 包</button>' : ''}
              <button type="button" class="btn btn-ghost btn-sm" id="domain-regenerate-btn">重新生成</button>
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

    const errEl = container.querySelector<HTMLDivElement>('#tree-error')!
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

    container.querySelector<HTMLButtonElement>('#domain-export-btn')?.addEventListener('click', () => {
      void (async () => {
        const btn = container.querySelector<HTMLButtonElement>('#domain-export-btn')
        if (!btn) return
        btn.disabled = true
        const prev = btn.textContent
        btn.textContent = '导出中…'
        try {
          const pkg = await exportDomain(domainId)
          const blob = new Blob([JSON.stringify(pkg, null, 2)], { type: 'application/json' })
          const url = URL.createObjectURL(blob)
          const a = document.createElement('a')
          a.href = url
          a.download = `${pkg.slug}-skill-export.json`
          a.click()
          URL.revokeObjectURL(url)
          errEl.innerHTML =
            '<div class="alert alert-success">已下载 Skill 包文件，解压后按 CONTRIBUTING.md 提交 PR</div>'
        } catch (e) {
          errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '导出失败'}</div>`
        } finally {
          btn.disabled = false
          btn.textContent = prev ?? '导出 Skill 包'
        }
      })()
    })

    container.querySelectorAll<HTMLElement>('.node-item').forEach((el) => {
      const nodeKey = el.dataset.node!
      const layer = el.dataset.layer!
      el.addEventListener('click', () => void openNode(nodeKey, layer))
      el.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          void openNode(nodeKey, layer)
        }
      })
    })

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
