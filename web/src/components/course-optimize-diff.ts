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

const FIELD_BENEFITS: Record<string, string> = {
  teaching_beats: '补齐讲解节拍与必讲要点，学这一节时教练更聚焦',
  boundaries: '划清本节不讲什么，减少重复与跑题',
  common_mistakes: '补充常见误区，练习会盯易错点',
  exercise_ideas: '丰富练习题思路，出题更贴考点',
  grading_hints: '补充批改要点，反馈更稳定',
  requires: '修正前置依赖，学习路径更顺',
}

export function showCourseOptimizeDiff(options: CourseOptimizeDiffOptions): void {
  const { domainId, jobId, patch, onApplied } = options
  const overlay = document.createElement('div')
  overlay.className = 'profile-overlay'

  const items = patch.patches ?? []
  const headline = (patch.headline || '').trim() || buildFallbackHeadline(items)
  const rows = items.map((p) => renderPatchRow(p)).join('')

  overlay.innerHTML = `
    <div class="profile-modal card course-optimize-modal" role="alertdialog" aria-labelledby="optimize-title">
      <h3 id="optimize-title" class="profile-modal-title">确认应用优化</h3>
      <p class="profile-modal-sub course-optimize-headline">${escapeHtml(headline)}</p>
      <p class="course-optimize-assurance">写入节点教学内容，不会删除节点，也不会改动学习进度。</p>
      <label class="course-audit-select-all">
        <input type="checkbox" id="optimize-select-all" checked />
        全选 ${items.length} 个知识点
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
  const title = (p.nodeTitle || '').trim() || humanizeNodeKey(p.nodeKey)
  const benefits = resolveBenefits(p)
  const benefitLines = benefits.map((b) => `<li>${escapeHtml(b)}</li>`).join('')
  const summary = (p.summary || benefits[0] || '补强节点教学内容').trim()

  return `
    <li class="course-optimize-item">
      <label class="course-optimize-item-label">
        <input type="checkbox" class="optimize-patch-check" data-patch-id="${escapeHtml(p.id)}" checked />
        <span class="course-optimize-item-main">
          <strong class="course-optimize-node-title">${escapeHtml(title)}</strong>
          <span class="course-optimize-item-summary">${escapeHtml(summary)}</span>
        </span>
      </label>
      ${benefitLines ? `<ul class="course-optimize-benefits">${benefitLines}</ul>` : ''}
    </li>
  `
}

function resolveBenefits(p: OptimizePatchItem): string[] {
  const fromApi = (p.benefits ?? []).map((b) => b.trim()).filter(Boolean)
  if (fromApi.length > 0) return dedupe(fromApi)

  const keys = Object.keys(p.after ?? {})
  const fromFields = keys
    .map((k) => FIELD_BENEFITS[k])
    .filter((v): v is string => Boolean(v))
  if (fromFields.length > 0) return dedupe(fromFields)

  if (p.summary?.trim()) return [p.summary.trim()]
  return ['补强本节教学内容，学习体验更稳']
}

function buildFallbackHeadline(items: OptimizePatchItem[]): string {
  const n = items.length
  if (n === 0) return '暂无可应用的优化。'
  const tags = new Set<string>()
  for (const p of items) {
    for (const b of resolveBenefits(p)) {
      if (/讲解|节拍|必讲/.test(b)) tags.add('讲解')
      if (/练习|出题/.test(b)) tags.add('练习')
      if (/批改|反馈/.test(b)) tags.add('批改')
      if (/误区|易错/.test(b)) tags.add('易错点')
      if (/边界|跑题|重复/.test(b)) tags.add('边界')
      if (/前置|路径/.test(b)) tags.add('路径')
    }
  }
  const order = ['讲解', '练习', '批改', '易错点', '边界', '路径']
  const parts = order.filter((k) => tags.has(k))
  if (parts.length === 0) {
    return `将对 ${n} 个知识点补强教学内容，不影响已有进度。`
  }
  return `将对 ${n} 个知识点提升${parts.join(' / ')}，写入后教练讲解、出题与反馈会更到位；不改节点、不影响已有进度。`
}

function humanizeNodeKey(key: string): string {
  return key.replace(/_/g, ' ')
}

function dedupe(items: string[]): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const item of items) {
    if (seen.has(item)) continue
    seen.add(item)
    out.push(item)
  }
  return out
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
