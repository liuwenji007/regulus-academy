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

  const findings = report.findings ?? []
  const fixableCount = findings.filter((f) => f.autoFixable).length
  const hasFindings = findings.length > 0
  const failCount = report.summary?.failCount ?? 0
  const warnCount = report.summary?.warnCount ?? 0
  const infoCount = report.summary?.infoCount ?? 0

  const dimCards = Object.entries(report.dimensions ?? {})
    .map(([key, dim]) => {
      const label = DIMENSION_LABELS[key] ?? key
      const low = dim.score < 60
      return `
        <div class="audit-dim-card${low ? ' audit-dim-card--low' : ''}">
          <div class="audit-dim-card-top">
            <span class="audit-dim-card-label">${escapeHtml(label)}</span>
            <span class="audit-dim-card-score">${dim.score}</span>
          </div>
          <div class="audit-dim-bar" role="progressbar" aria-valuenow="${dim.score}" aria-valuemin="0" aria-valuemax="100" aria-label="${escapeHtml(label)} ${dim.score} 分">
            <div class="audit-dim-fill" style="width:${dim.score}%"></div>
          </div>
        </div>
      `
    })
    .join('')

  const sorted = [...findings].sort((a, b) => {
    const order = { fail: 0, warn: 1, info: 2 }
    return (order[a.severity] ?? 3) - (order[b.severity] ?? 3)
  })

  const llmBlock = report.llmCritique?.feedback
    ? `<details class="course-audit-llm"${hasFindings ? '' : ' open'}>
        <summary class="course-audit-llm-summary">AI 总评</summary>
        <p class="course-audit-llm-text">${escapeHtml(report.llmCritique.feedback)}</p>
      </details>`
    : ''

  const findingsBlock = !hasFindings
    ? `<div class="course-audit-pass"><p>各项检查均已通过，暂无待处理建议。</p></div>`
    : `
      <section class="course-audit-findings" aria-labelledby="audit-findings-title">
        <div class="course-audit-findings-head">
          <h4 id="audit-findings-title" class="course-audit-findings-title">改进建议 <span class="course-audit-findings-count">${findings.length}</span></h4>
          ${
            fixableCount > 0
              ? `<label class="course-audit-select-all">
            <input type="checkbox" id="audit-select-all-fixable" />
            全选可自动优化（${fixableCount}）
          </label>`
              : `<span class="course-audit-findings-hint">暂无可一键自动优化项</span>`
          }
        </div>
        <ul class="course-audit-list" id="audit-findings-list">${sorted.map((f) => renderFindingRow(f)).join('')}</ul>
      </section>`

  overlay.innerHTML = `
    <div class="profile-modal card course-audit-modal${hasFindings ? ' course-audit-modal--detail' : ''}" role="dialog" aria-labelledby="audit-title">
      <header class="course-audit-header">
        <div class="course-audit-hero">
          <div class="course-audit-hero-text">
            <h3 id="audit-title" class="course-audit-title">课程体检报告</h3>
            <p class="course-audit-headline">${escapeHtml(report.summary?.headline ?? '')}</p>
          </div>
          <div class="course-audit-hero-score" aria-label="综合 ${report.summary?.score ?? 0} 分，等级 ${escapeHtml(report.summary?.grade ?? '—')}">
            <span class="course-audit-grade">${escapeHtml(report.summary?.grade ?? '—')}</span>
            <span class="course-audit-score-num">${report.summary?.score ?? 0}<small>分</small></span>
          </div>
        </div>
        <div class="course-audit-stats">
          ${failCount > 0 ? `<span class="audit-stat audit-stat--fail">严重 ${failCount}</span>` : ''}
          ${warnCount > 0 ? `<span class="audit-stat audit-stat--warn">待改进 ${warnCount}</span>` : ''}
          ${infoCount > 0 ? `<span class="audit-stat audit-stat--info">提示 ${infoCount}</span>` : ''}
          ${failCount === 0 && warnCount === 0 && infoCount === 0 ? `<span class="audit-stat audit-stat--ok">全部通过</span>` : ''}
        </div>
        ${dimCards ? `<div class="audit-dim-grid">${dimCards}</div>` : ''}
      </header>

      <div class="course-audit-body">
        ${llmBlock}
        ${findingsBlock}
      </div>

      <footer class="course-audit-footer profile-delete-actions">
        <button type="button" class="btn btn-ghost" id="audit-close-btn">关闭</button>
        ${fixableCount > 0 ? `<button type="button" class="btn btn-primary" id="audit-optimize-btn">应用选中优化</button>` : ''}
      </footer>
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
  const inputId = `finding-${escapeHtml(f.id)}`
  const check = f.autoFixable
    ? `<input type="checkbox" class="audit-finding-check" data-fixable="1" data-finding-id="${escapeHtml(f.id)}" id="${inputId}" />`
    : `<span class="audit-finding-check-spacer" aria-hidden="true"></span>`

  const labelAttrs = f.autoFixable ? ` for="${inputId}"` : ''
  const rowTag = f.autoFixable ? 'label' : 'div'

  return `
    <li class="course-audit-item course-audit-item--${f.severity}">
      <div class="course-audit-item-head">
        <${rowTag} class="course-audit-item-select"${labelAttrs}>
          ${check}
          <span class="course-audit-sev course-audit-sev--${f.severity}">${escapeHtml(sev)}</span>
        </${rowTag}>
        ${f.nodeKey ? `<code class="course-audit-node">${escapeHtml(f.nodeKey)}</code>` : ''}
      </div>
      <p class="course-audit-msg">${escapeHtml(f.message)}</p>
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
