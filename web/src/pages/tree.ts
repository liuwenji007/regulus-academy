import { getDomainTree, getUserProgress, startSession, ApiError } from '../lib/api'

export async function renderTree(container: HTMLElement, domainId: string): Promise<void> {
  container.innerHTML = `<div class="page"><p class="page-sub">加载知识树…</p></div>`

  try {
    const [tree, progress] = await Promise.all([
      getDomainTree(domainId),
      getUserProgress(domainId),
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
          nextHint = `推荐下一步：${node.title}`
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
            return `
              <li class="node-item" data-node="${node.key}" data-layer="${layer.key}">
                <span class="node-status ${statusClass}"></span>
                <span class="node-title">${escapeHtml(node.title)}</span>
              </li>
            `
          })
          .join('')
        return `
          <section class="layer">
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
      <div class="page">
        <a href="#/" class="back-link">← 返回</a>
        <h1 class="page-title">${escapeHtml(tree.domainName)}</h1>
        <div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div>
        <p class="page-sub">已完成 ${completed} / ${total} 个节点${nextHint ? ' · ' + escapeHtml(nextHint) : ''}</p>
        <div id="tree-error"></div>
        ${layersHtml}
      </div>
    `

    const errEl = container.querySelector<HTMLDivElement>('#tree-error')!
    container.querySelectorAll<HTMLElement>('.node-item').forEach((el) => {
      el.addEventListener('click', async () => {
        const nodeKey = el.dataset.node!
        const layer = el.dataset.layer!
        try {
          const res = await startSession(domainId, nodeKey, layer)
          location.hash = `#/coach/${res.sessionId}`
          window.dispatchEvent(new HashChangeEvent('hashchange'))
        } catch (e) {
          errEl.innerHTML = `<div class="error-banner">${e instanceof ApiError ? e.message : '启动会话失败'}</div>`
        }
      })
    })
  } catch (e) {
    container.innerHTML = `
      <div class="page">
        <a href="#/" class="back-link">← 返回</a>
        <div class="error-banner">${e instanceof ApiError ? e.message : '加载失败'}</div>
      </div>
    `
  }
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
