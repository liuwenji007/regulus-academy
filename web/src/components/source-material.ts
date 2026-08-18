import { getDomainSourceMaterial, ApiError } from '../lib/api'

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

export function showSourceMaterial(domainId: string, domainName: string): void {
  const overlay = document.createElement('div')
  overlay.className = 'profile-overlay'
  overlay.innerHTML = `
    <div class="profile-modal card source-material-modal" role="dialog" aria-modal="true" aria-labelledby="source-material-title">
      <div class="source-material-header">
        <h3 id="source-material-title" class="profile-modal-title">导入原文</h3>
        <p class="profile-modal-sub">建课时从材料抽出的文本（不是模型摘要）。可对照检查抽字是否正确。</p>
      </div>
      <div class="source-material-body" id="source-material-body">
        <p class="source-material-loading">正在读取抽出文本…</p>
      </div>
      <div class="profile-delete-actions source-material-actions">
        <button type="button" class="btn btn-primary" id="source-material-close">关闭</button>
      </div>
    </div>
  `
  document.body.appendChild(overlay)

  const close = () => overlay.remove()
  overlay.querySelector('#source-material-close')?.addEventListener('click', close)
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) close()
  })

  void (async () => {
    const body = overlay.querySelector<HTMLElement>('#source-material-body')
    if (!body) return
    try {
      const mat = await getDomainSourceMaterial(domainId)
      const kindLabel = mat.kind === 'pdf' ? 'PDF' : mat.kind === 'url' ? '网页' : mat.kind
      const stats = [
        kindLabel,
        mat.label,
        mat.pageCount ? `${mat.pageCount} 页` : '',
        mat.charCount ? `${mat.charCount} 字` : '',
      ]
        .filter(Boolean)
        .join(' · ')
      body.innerHTML = `
        <p class="source-material-meta">${escapeHtml(stats || domainName)}</p>
        <pre class="source-material-pre">${escapeHtml(mat.text)}</pre>
      `
    } catch (e) {
      const msg = e instanceof ApiError ? e.message : '读取导入原文失败'
      body.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
    }
  })()
}
