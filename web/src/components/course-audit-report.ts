import type { CourseAuditReport, AuditFinding } from '../lib/api'

export interface CourseAuditReportOptions {
  report: CourseAuditReport
  auditJobId: string
  onOptimize: (findingIds: string[], auditJobId: string) => void
}

type SeverityFilter = 'all' | 'fail' | 'warn' | 'info'

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

const SCORE_TIP =
  '综合分从 100 起按每条建议累加扣分（严重 −15、待改进 −5、提示 −1 且合计最多 −10），不是四维均分。某一维仍可能较高。'

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
      const tone = dim.score < 60 ? 'low' : dim.score < 85 ? 'mid' : 'ok'
      return `
        <div class="audit-dim-chip audit-dim-chip--${tone}" title="${escapeHtml(label)} ${dim.score} 分">
          <span class="audit-dim-chip-label">${escapeHtml(label)}</span>
          <span class="audit-dim-chip-score">${dim.score}</span>
          <span class="audit-dim-chip-track" aria-hidden="true">
            <span class="audit-dim-chip-fill" style="width:${dim.score}%"></span>
          </span>
        </div>
      `
    })
    .join('')

  const sorted = [...findings].sort((a, b) => {
    const order = { fail: 0, warn: 1, info: 2 }
    return (order[a.severity] ?? 3) - (order[b.severity] ?? 3)
  })

  const llmFeedback = report.llmCritique?.feedback?.trim() ?? ''
  const llmBlock = llmFeedback
    ? `<section class="course-audit-panel course-audit-panel--llm" aria-labelledby="audit-llm-title">
        <div class="course-audit-panel-head">
          <h4 id="audit-llm-title" class="course-audit-panel-title">AI 总评</h4>
          <span class="course-audit-panel-badge">整体看法</span>
        </div>
        <p class="course-audit-llm-text">${escapeHtml(llmFeedback)}</p>
      </section>`
    : ''

  const findingsBlock = !hasFindings
    ? `<section class="course-audit-panel course-audit-panel--pass">
        <p class="course-audit-pass-text">各项检查均已通过，暂无待处理建议。</p>
      </section>`
    : `
      <section class="course-audit-panel course-audit-panel--findings" aria-labelledby="audit-findings-title">
        <div class="course-audit-panel-head course-audit-findings-head">
          <div class="course-audit-panel-title-row">
            <h4 id="audit-findings-title" class="course-audit-panel-title">改进建议</h4>
            <span class="course-audit-findings-count" id="audit-findings-visible-count">${findings.length}</span>
          </div>
          ${
            fixableCount > 0
              ? `<label class="course-audit-select-all">
            <input type="checkbox" id="audit-select-all-fixable" />
            <span id="audit-select-all-label">全选可自动优化（${fixableCount}）</span>
          </label>`
              : `<span class="course-audit-findings-hint">暂无可一键自动优化项</span>`
          }
        </div>
        <p class="course-audit-panel-lead">勾选后可批量补全讲解节拍、练习思路等；不会删节点，也不改学习进度。</p>
        <ul class="course-audit-list" id="audit-findings-list">${sorted.map((f) => renderFindingRow(f)).join('')}</ul>
        <p class="course-audit-filter-empty" id="audit-filter-empty" hidden>当前筛选下暂无建议</p>
      </section>`

  const score = report.summary?.score ?? 0
  const grade = report.summary?.grade ?? '—'

  overlay.innerHTML = `
    <div class="profile-modal card course-audit-modal${hasFindings || llmFeedback ? ' course-audit-modal--detail' : ''}" role="dialog" aria-labelledby="audit-title">
      <header class="course-audit-header">
        <div class="course-audit-hero">
          <div class="course-audit-hero-text">
            <h3 id="audit-title" class="course-audit-title">课程体检报告</h3>
            <p class="course-audit-headline">${escapeHtml(report.summary?.headline ?? '')}</p>
          </div>
          <div class="course-audit-hero-score" title="${escapeHtml(SCORE_TIP)}" aria-label="综合 ${score} 分，等级 ${escapeHtml(grade)}。${escapeHtml(SCORE_TIP)}">
            <span class="course-audit-grade">${escapeHtml(grade)}</span>
            <span class="course-audit-score-num">${score}<small>分</small></span>
            <button type="button" class="course-audit-score-tip-btn" id="audit-score-tip-btn" aria-expanded="false" aria-controls="audit-score-tip">评分说明</button>
          </div>
        </div>
        <p class="course-audit-score-tip" id="audit-score-tip" hidden>${escapeHtml(SCORE_TIP)}</p>
        ${dimCards ? `<div class="audit-dim-rail" role="group" aria-label="各维度得分">${dimCards}</div>` : ''}
        <div class="course-audit-stats" role="group" aria-label="按严重程度筛选建议">
          ${
            hasFindings
              ? `<button type="button" class="audit-stat audit-stat--all is-active" data-severity-filter="all" aria-pressed="true">全部 ${findings.length}</button>`
              : ''
          }
          ${failCount > 0 ? `<button type="button" class="audit-stat audit-stat--fail" data-severity-filter="fail" aria-pressed="false">严重 ${failCount}</button>` : ''}
          ${warnCount > 0 ? `<button type="button" class="audit-stat audit-stat--warn" data-severity-filter="warn" aria-pressed="false">待改进 ${warnCount}</button>` : ''}
          ${infoCount > 0 ? `<button type="button" class="audit-stat audit-stat--info" data-severity-filter="info" aria-pressed="false">提示 ${infoCount}</button>` : ''}
          ${failCount === 0 && warnCount === 0 && infoCount === 0 ? `<span class="audit-stat audit-stat--ok">全部通过</span>` : ''}
        </div>
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

  const tipEl = overlay.querySelector<HTMLElement>('#audit-score-tip')
  const tipBtn = overlay.querySelector<HTMLButtonElement>('#audit-score-tip-btn')
  tipBtn?.addEventListener('click', () => {
    if (!tipEl || !tipBtn) return
    const open = tipEl.hasAttribute('hidden')
    if (open) tipEl.removeAttribute('hidden')
    else tipEl.setAttribute('hidden', '')
    tipBtn.setAttribute('aria-expanded', open ? 'true' : 'false')
  })

  const selectAll = overlay.querySelector<HTMLInputElement>('#audit-select-all-fixable')
  const selectAllLabel = overlay.querySelector('#audit-select-all-label')
  const visibleCountEl = overlay.querySelector('#audit-findings-visible-count')
  const filterEmptyEl = overlay.querySelector<HTMLElement>('#audit-filter-empty')
  const listEl = overlay.querySelector('#audit-findings-list')

  const applySeverityFilter = (filter: SeverityFilter) => {
    const items = overlay.querySelectorAll<HTMLLIElement>('.course-audit-item')
    let visible = 0
    let visibleFixable = 0
    items.forEach((li) => {
      const sev = (li.dataset.severity ?? '') as SeverityFilter
      const show = filter === 'all' || sev === filter
      li.hidden = !show
      if (show) {
        visible++
        if (li.querySelector('.audit-finding-check[data-fixable="1"]')) visibleFixable++
      }
    })
    if (visibleCountEl) visibleCountEl.textContent = String(visible)
    if (filterEmptyEl) {
      if (visible === 0) filterEmptyEl.removeAttribute('hidden')
      else filterEmptyEl.setAttribute('hidden', '')
    }
    if (listEl) {
      if (visible === 0) listEl.setAttribute('hidden', '')
      else listEl.removeAttribute('hidden')
    }
    if (selectAllLabel) {
      selectAllLabel.textContent = `全选可自动优化（${visibleFixable}）`
    }
    if (selectAll) {
      selectAll.checked = false
      selectAll.disabled = visibleFixable === 0
    }
    overlay.querySelectorAll<HTMLButtonElement>('[data-severity-filter]').forEach((btn) => {
      const active = btn.dataset.severityFilter === filter
      btn.classList.toggle('is-active', active)
      btn.setAttribute('aria-pressed', active ? 'true' : 'false')
    })
  }

  overlay.querySelectorAll<HTMLButtonElement>('[data-severity-filter]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const filter = (btn.dataset.severityFilter ?? 'all') as SeverityFilter
      applySeverityFilter(filter)
    })
  })

  selectAll?.addEventListener('change', () => {
    overlay.querySelectorAll<HTMLInputElement>('.audit-finding-check[data-fixable="1"]').forEach((cb) => {
      const li = cb.closest<HTMLLIElement>('.course-audit-item')
      if (li?.hidden) {
        cb.checked = false
        return
      }
      cb.checked = !!selectAll.checked
    })
  })

  overlay.querySelector('#audit-optimize-btn')?.addEventListener('click', () => {
    const ids: string[] = []
    overlay.querySelectorAll<HTMLInputElement>('.audit-finding-check:checked').forEach((cb) => {
      const li = cb.closest<HTMLLIElement>('.course-audit-item')
      if (li?.hidden) return
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
  const nodeLabel = f.nodeKey ? humanizeNodeKey(f.nodeKey) : ''

  return `
    <li class="course-audit-item course-audit-item--${f.severity}" data-severity="${escapeHtml(f.severity)}">
      <div class="course-audit-item-head">
        <${rowTag} class="course-audit-item-select"${labelAttrs}>
          ${check}
          <span class="course-audit-sev course-audit-sev--${f.severity}">${escapeHtml(sev)}</span>
          ${nodeLabel ? `<span class="course-audit-node">${escapeHtml(nodeLabel)}</span>` : ''}
        </${rowTag}>
      </div>
      <p class="course-audit-msg">${escapeHtml(humanizeFindingMessage(f.message))}</p>
      <p class="course-audit-suggestion">${escapeHtml(humanizeSuggestion(f.suggestion))}</p>
    </li>
  `
}

function humanizeNodeKey(key: string): string {
  return key.replace(/_/g, ' ')
}

function humanizeFindingMessage(msg: string): string {
  return msg
    .replace(/teaching_beats/g, '教学节拍')
    .replace(/must_teach/g, '必讲要点')
    .replace(/core_concepts/g, '核心概念')
    .replace(/exercise_ideas/g, '练习思路')
    .replace(/grading_hints/g, '批改要点')
    .replace(/common_mistakes/g, '常见误区')
    .replace(/fallback/g, '临时兜底')
}

function humanizeSuggestion(s: string): string {
  return s
    .replace(/按 core_concepts 补全教学节拍，对齐 Go 并发域标杆/g, '按核心概念补全教学节拍，让讲解与练习更对齐')
    .replace(/补充 must_teach 要点/g, '补齐必讲要点，讲解更完整')
    .replace(/teaching_beats/g, '教学节拍')
    .replace(/must_teach/g, '必讲要点')
    .replace(/core_concepts/g, '核心概念')
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
