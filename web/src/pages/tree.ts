import { getDomainTree, getUserProgress, getActiveSession, startSession, ApiError } from '../lib/api'
import { setBreadcrumb, updateSidebar } from '../components/layout'
import type { NavKey } from '../components/sidebar'

export async function renderTree(
  container: HTMLElement,
  domainId: string,
  _nav: NavKey = 'tree'
): Promise<void> {
  container.innerHTML = `
    <section class="page page-tree">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>加载知识树…</p>
      </div>
    </section>
  `

  try {
    const [tree, progress] = await Promise.all([
      getDomainTree(domainId),
      getUserProgress(domainId),
    ])
    localStorage.setItem('regulus:lastDomainId', domainId)
    updateSidebar({ active: 'tree', domainId, domainName: tree.domainName })
    setBreadcrumb([
      { label: '开始学习', href: '#/' },
      { label: tree.domainName },
    ])

    const progressMap = new Map(progress.map((p) => [p.nodeKey, p]))
    const completed = progress.filter((p) => p.status === 'completed').length
    const total = tree.layers.reduce((n, l) => n + l.nodes.length, 0)
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
              statusClass === 'in_progress'
                ? '<span class="node-resume-tag">继续</span>'
                : ''
            return `
              <li class="node-item" data-node="${node.key}" data-layer="${layer.key}" tabindex="0" role="button">
                <span class="node-status ${statusClass}" aria-hidden="true"></span>
                <span class="node-title">${escapeHtml(node.title)}</span>
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
          <h1 class="page-title">${escapeHtml(tree.domainName)}</h1>
          <p class="page-sub">${nextHint ? `推荐下一步：${escapeHtml(nextHint)}` : '选择一个节点开始微训练'}</p>
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
        <div class="tree-layers">${layersHtml}</div>
      </section>
    `

    const errEl = container.querySelector<HTMLDivElement>('#tree-error')!
    const openNode = async (nodeKey: string, layer: string) => {
      try {
        const active = await getActiveSession(domainId, nodeKey)
        if (active.sessionId) {
          location.hash = `#/coach/${active.sessionId}`
          window.dispatchEvent(new HashChangeEvent('hashchange'))
          return
        }
        const res = await startSession(domainId, nodeKey, layer)
        location.hash = `#/coach/${res.sessionId}`
        window.dispatchEvent(new HashChangeEvent('hashchange'))
      } catch (e) {
        errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '启动会话失败'}</div>`
      }
    }

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
  } catch (e) {
    updateSidebar({ active: 'tree' })
    setBreadcrumb([{ label: '开始学习', href: '#/' }, { label: '知识树' }])
    container.innerHTML = `
      <section class="page page-tree">
        <div class="alert alert-error">${e instanceof ApiError ? e.message : '加载失败'}</div>
      </section>
    `
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
