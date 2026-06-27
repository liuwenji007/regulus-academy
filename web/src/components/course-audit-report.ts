import type { CourseAuditReport, AuditFinding } from '../lib/api'

export interface CourseAuditReportOptions {
  report: CourseAuditReport
  auditJobId: string
  onOptimize: (findingIds: string[], auditJobId: string) => void
}

const DIMENSION_LABELS: Record<string, string> = {
  structure: '结构',
  nodeCompleteness: '节点完整性',
  teachingAlignment: '教考对齐',
  prerequisites: '前置依赖',
}

const SEVERITY_LABELS: Record<string, string> = {
  fail: '严重',
  warn: '待改进',
  info: '提示',
}

export function showCourseAuditReport(options: CourseAuditReportOptions): void {
  const { report, auditJobId, onOptimize } = options
  const overlay = document.createElement('div')
  overlay.className = 'profile-overlay'

  const dimBars = Object.entries(report.dimensions ?? {})
    .map(([key, dim]) => {
      const label = DIMENSION_LABELS[key] ?? key
      return `
        <div class="audit-dim-row">
          <span class="audit-dim-label">${escapeHtml(label)}</span>
          <div class="audit-dim-bar" role="progressbar" aria-valuenow="${dim.score}" aria-valuemin="0" aria-valuemax="100">
            <div class="audit-dim-fill" style="width:${dim.score}%"></div>
          </div>
          <span class="audit-dim-score">${dim.score}</span>
        </div>
      `
    })
    .join('')

  const sorted = [...(report.findings ?? [])].sort((a, b) => {
    const order = { fail: 0, warn: 1, info: 2 }
    return (order[a.severity] ?? 3) - (order[b.severity] ?? 3)
  })

  const findingsHtml = sorted
    .map((f) => renderFindingRow(f))
    .join('')

  overlay.innerHTML = `
    <div class="profile-modal card course-audit-modal" role="dialog" aria-labelledby="audit-title">
      <div class="course-audit-header">
        <h3 id="audit-title" class="profile-modal-title">课程体检报告</h3>
        <p class="profile-modal-sub">${escapeHtml(report.summary?.headline ?? '')}</p>
        <div class="course-audit-scorecard">
          <span class="course-audit-grade">${escapeHtml(report.summary?.grade ?? '—')}</span>
          <span class="course-audit-score">${report.summary?.score ?? 0} 分</span>
          <span class="course-audit-counts">
            严重 ${report.summary?.failCount ?? 0} · 待改进 ${report.summary?.warnCount ?? 0} · 提示 ${report.summary?.infoCount ?? 0}
          </span>
        </div>
      </div>
      <div class="course-audit-dims">${dimBars}</div>
      ${
        report.llmCritique?.feedback
          ? `<div class="course-audit-llm card"><p class="course-audit-llm-label">AI 总评</p><p>${escapeHtml(report.llmCritique.feedback)}</p></div>`
          : ''
      }
      <div class="course-audit-findings">
        <div class="course-audit-findings-toolbar">
          <label class="course-audit-select-all">
            <input type="checkbox" id="audit-select-all-fixable" />
            全选可自动优化项
          </label>
        </div>
        <ul class="course-audit-list" id="audit-findings-list">${findingsHtml}</ul>
      </div>
      <div class="profile-delete-actions">
        <button type="button" class="btn btn-ghost" id="audit-close-btn">关闭</button>
        <button type="button" class="btn btn-primary" id="audit-optimize-btn">应用选中优化</button>
      </div>
    </div>
  `
  document.body.appendChild(overlay)

  const close = () => overlay.remove()
  overlay.querySelector('#audit-close-btn')?.addEventListener('click', close)
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) close()
  })

  const selectAll = overlay.querySelector<HTMLInputElement>('#audit-select-all-fixable')
  selectAll?.addEventListener('change', () => {
    overlay.querySelectorAll<HTMLInputElement>('.audit-finding-check[data-fixable="1"]').forEach((cb) => {
      cb.checked = !!selectAll.checked
    })
  })

  overlay.querySelector('#audit-optimize-btn')?.addEventListener('click', () => {
    const ids: string[] = []
    overlay.querySelectorAll<HTMLInputElement>('.audit-finding-check:checked').forEach((cb) => {
      const id = cb.dataset.findingId
      if (id) ids.push(id)
    })
    if (ids.length === 0) {
      alert('请至少勾选一项可优化建议')
      return
    }
    close()
    onOptimize(ids, auditJobId)
  })
}

function renderFindingRow(f: AuditFinding): string {
  const sev = SEVERITY_LABELS[f.severity] ?? f.severity
  const check =
    f.autoFixable
      ? `<input type="checkbox" class="audit-finding-check" data-fixable="1" data-finding-id="${escapeHtml(f.id)}" id="finding-${escapeHtml(f.id)}" />`
      : ''
  return `
    <li class="course-audit-item course-audit-item--${f.severity}">
      <label class="course-audit-item-label" for="finding-${escapeHtml(f.id)}">
        ${check}
        <span class="course-audit-sev">${escapeHtml(sev)}</span>
        <span class="course-audit-msg">${escapeHtml(f.message)}</span>
        ${f.nodeKey ? `<span class="course-audit-node">${escapeHtml(f.nodeKey)}</span>` : ''}
      </label>
      <p class="course-audit-suggestion">${escapeHtml(f.suggestion)}</p>
    </li>
  `
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
