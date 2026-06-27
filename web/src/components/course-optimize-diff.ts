import {
  applyDomainOptimizePatches,
  type OptimizePatch,
  type OptimizePatchItem,
} from '../lib/api'

export interface CourseOptimizeDiffOptions {
  domainId: string
  jobId: string
  patch: OptimizePatch
  onApplied: (message?: string) => void
}

export function showCourseOptimizeDiff(options: CourseOptimizeDiffOptions): void {
  const { domainId, jobId, patch, onApplied } = options
  const overlay = document.createElement('div')
  overlay.className = 'profile-overlay'

  const items = patch.patches ?? []
  const rows = items.map((p) => renderPatchRow(p)).join('')

  overlay.innerHTML = `
    <div class="profile-modal card course-optimize-modal" role="alertdialog" aria-labelledby="optimize-title">
      <h3 id="optimize-title" class="profile-modal-title">确认应用优化</h3>
      <p class="profile-modal-sub">以下变更将写入节点内容，<strong>不会</strong>删除节点或改动学习进度。</p>
      <label class="course-audit-select-all">
        <input type="checkbox" id="optimize-select-all" checked />
        全选 ${items.length} 项
      </label>
      <ul class="course-optimize-list" id="optimize-patch-list">${rows}</ul>
      <div class="profile-delete-actions">
        <button type="button" class="btn btn-ghost" id="optimize-cancel-btn">取消</button>
        <button type="button" class="btn btn-primary" id="optimize-apply-btn">确认应用</button>
      </div>
    </div>
  `
  document.body.appendChild(overlay)

  const close = () => overlay.remove()
  overlay.querySelector('#optimize-cancel-btn')?.addEventListener('click', close)
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) close()
  })

  overlay.querySelector('#optimize-select-all')?.addEventListener('change', (e) => {
    const checked = (e.target as HTMLInputElement).checked
    overlay.querySelectorAll<HTMLInputElement>('.optimize-patch-check').forEach((cb) => {
      cb.checked = checked
    })
  })

  overlay.querySelector('#optimize-apply-btn')?.addEventListener('click', () => {
    void (async () => {
      const patchIds: string[] = []
      overlay.querySelectorAll<HTMLInputElement>('.optimize-patch-check:checked').forEach((cb) => {
        const id = cb.dataset.patchId
        if (id) patchIds.push(id)
      })
      if (patchIds.length === 0) {
        alert('请至少选择一项优化')
        return
      }
      const btn = overlay.querySelector<HTMLButtonElement>('#optimize-apply-btn')
      if (btn) {
        btn.disabled = true
        btn.textContent = '应用中…'
      }
      try {
        const result = await applyDomainOptimizePatches(domainId, jobId, patchIds)
        close()
        onApplied(result.message)
      } catch (err) {
        alert(err instanceof Error ? err.message : '应用失败')
        if (btn) {
          btn.disabled = false
          btn.textContent = '确认应用'
        }
      }
    })()
  })
}

function renderPatchRow(p: OptimizePatchItem): string {
  const fields = Object.keys(p.after ?? {})
  const diffLines = fields
    .map((key) => {
      const afterVal = formatField(p.after[key])
      return `<li><code>${escapeHtml(key)}</code> → ${escapeHtml(afterVal)}</li>`
    })
    .join('')
  return `
    <li class="course-optimize-item">
      <label>
        <input type="checkbox" class="optimize-patch-check" data-patch-id="${escapeHtml(p.id)}" checked />
        <strong>${escapeHtml(p.nodeKey)}</strong> — ${escapeHtml(p.summary)}
      </label>
      <ul class="course-optimize-diff">${diffLines}</ul>
    </li>
  `
}

function formatField(val: unknown): string {
  if (val == null) return '（空）'
  if (Array.isArray(val)) {
    return val.length ? `${val.length} 条` : '（空）'
  }
  return String(val)
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
