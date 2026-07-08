import type { PlanningMessage, PlanningResult } from './api'
import { phaseLabel } from './api'
import { renderMarkdown } from './markdown'
import { scrollChatMessages } from './chat-scroll'
import { escapeHtml } from './utils'

export interface AssistantViewState {
  sessionId: string
  phase: string
  messages: PlanningMessage[]
  plan: PlanningResult | null
  sending: boolean
  synthesizing: boolean
  error: string
}

function formatBubbleContent(m: PlanningMessage): string {
  if (m.role === 'assistant') {
    return `<div class="md-body">${renderMarkdown(m.content)}</div>`
  }
  return escapeHtml(m.content)
}

export function assistantLoadingHtml(hint: string): string {
  return `
    <section class="page page-coach page-assistant">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>正在连接行动助手…</p>
        <p class="page-loading-hint">${escapeHtml(hint)}</p>
      </div>
    </section>
  `
}

export function assistantErrorHtml(msg: string): string {
  return `
    <section class="page page-coach page-assistant">
      <div class="alert alert-error">${escapeHtml(msg)}</div>
      <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
        <button type="button" class="btn btn-secondary btn-sm" id="assistant-retry-btn">重试</button>
        <a class="btn btn-ghost btn-sm" href="#/assistant" style="margin-left:0.5rem">重新开始</a>
      </p>
    </section>
  `
}

function renderMatrixSection(title: string, items: { title: string; why?: string; minutes?: number; next_step?: string; reason?: string }[]): string {
  if (!items?.length) return ''
  return `
    <div class="assistant-plan-section">
      <h3 class="assistant-plan-section-title">${escapeHtml(title)}</h3>
      <ul class="assistant-plan-list">
        ${items
          .map((item) => {
            const meta: string[] = []
            if (item.minutes) meta.push(`${item.minutes} 分钟`)
            if (item.next_step) meta.push(item.next_step)
            if (item.why) meta.push(item.why)
            if (item.reason) meta.push(item.reason)
            return `
              <li class="assistant-plan-item">
                <span class="assistant-plan-item-title">${escapeHtml(item.title)}</span>
                ${meta.length ? `<span class="assistant-plan-item-meta">${escapeHtml(meta.join(' · '))}</span>` : ''}
              </li>
            `
          })
          .join('')}
      </ul>
    </div>
  `
}

function renderPlanPanel(plan: PlanningResult): string {
  const today = plan.action_plan?.today ?? []
  const thisWeek = plan.action_plan?.this_week ?? []
  const learning = plan.learning_focus ?? []

  return `
    <aside class="assistant-plan-panel card" aria-label="行动规划">
      <header class="assistant-plan-header">
        <h2 class="assistant-plan-heading">行动方案</h2>
        ${plan.situation_summary ? `<p class="assistant-plan-summary">${escapeHtml(plan.situation_summary)}</p>` : ''}
      </header>

      ${today.length ? `
        <div class="assistant-plan-section assistant-plan-section--highlight">
          <h3 class="assistant-plan-section-title">今日行动</h3>
          <ul class="assistant-plan-list">
            ${today
              .map(
                (item) => `
              <li class="assistant-plan-item">
                <span class="assistant-plan-item-title">${escapeHtml(item.title)}</span>
                <span class="assistant-plan-item-meta">${item.minutes} 分钟 · ${item.kind === 'learning' ? '学习' : '事务'}${item.reason ? ` · ${escapeHtml(item.reason)}` : ''}</span>
              </li>
            `
              )
              .join('')}
          </ul>
        </div>
      ` : ''}

      ${renderMatrixSection('重要且紧急', plan.matrix?.important_urgent ?? [])}
      ${renderMatrixSection('重要不紧急', plan.matrix?.important_not_urgent ?? [])}
      ${renderMatrixSection('快速了结', plan.matrix?.quick_wins ?? [])}
      ${renderMatrixSection('建议暂缓', plan.matrix?.defer_or_drop ?? [])}

      ${thisWeek.length ? `
        <div class="assistant-plan-section">
          <h3 class="assistant-plan-section-title">本周补充</h3>
          <ul class="assistant-plan-list">
            ${thisWeek
              .map(
                (item) => `
              <li class="assistant-plan-item">
                <span class="assistant-plan-item-title">${escapeHtml(item.title)}</span>
                <span class="assistant-plan-item-meta">${item.minutes} 分钟</span>
              </li>
            `
              )
              .join('')}
          </ul>
        </div>
      ` : ''}

      ${learning.length ? `
        <div class="assistant-plan-section">
          <h3 class="assistant-plan-section-title">学习聚焦</h3>
          <ul class="assistant-plan-list">
            ${learning
              .map((lf) => {
                const canStart = lf.matched_domain_id && lf.matched_node_key
                return `
              <li class="assistant-plan-item assistant-plan-item--learning">
                <span class="assistant-plan-item-title">${escapeHtml(lf.area)}</span>
                <span class="assistant-plan-item-meta">${escapeHtml(lf.rationale)} · 建议 ${lf.suggested_minutes} 分钟</span>
                ${
                  canStart
                    ? `<button type="button" class="btn btn-primary btn-sm assistant-start-learn" data-domain-id="${escapeHtml(lf.matched_domain_id!)}" data-node-key="${escapeHtml(lf.matched_node_key!)}" data-node-title="${escapeHtml(lf.matched_node_title ?? lf.area)}">
                    开始 15 分钟
                  </button>`
                    : ''
                }
              </li>
            `
              })
              .join('')}
          </ul>
        </div>
      ` : ''}

      ${plan.mindset_note ? `<p class="assistant-plan-mindset">${escapeHtml(plan.mindset_note)}</p>` : ''}
    </aside>
  `
}

export function renderAssistantView(container: HTMLElement, view: AssistantViewState): void {
  const bubbles = view.messages
    .map((m) => `<div class="bubble ${m.role}">${formatBubbleContent(m)}</div>`)
    .join('')

  const loadingText = view.synthesizing ? '正在帮你整理行动方案…' : '思考中…'
  const planHtml = view.plan ? renderPlanPanel(view.plan) : ''

  container.innerHTML = `
    <section class="page page-coach page-assistant">
      <header class="page-header page-header-compact">
        <h1 class="page-title">行动助手</h1>
        <span class="phase-badge">${phaseLabel(view.phase)}</span>
      </header>

      <div class="assistant-layout">
        <div class="chat-panel card">
          <div class="chat-messages" id="messages" role="log" aria-live="polite">
            ${bubbles}
            ${view.sending ? `<div class="coach-loading">${escapeHtml(loadingText)}</div>` : ''}
          </div>
          <div class="coach-footer">
            <div id="assistant-error">${view.error ? `<div class="alert alert-error">${escapeHtml(view.error)}</div>` : ''}</div>
            <div class="coach-composer assistant-composer">
              <div class="assistant-composer-shell">
                <textarea
                  class="input assistant-composer-input"
                  id="msg-input"
                  rows="1"
                  placeholder="说说你现在脑子里装的事…"
                  ${view.sending ? 'disabled' : ''}
                  aria-label="输入消息"
                ></textarea>
                <button type="button" class="btn btn-primary assistant-composer-send" id="send-btn" ${view.sending ? 'disabled' : ''}>
                  ${view.sending ? '…' : '发送'}
                </button>
              </div>
              <p class="assistant-composer-hint">Enter 发送 · Shift+Enter 换行</p>
            </div>
            ${
              view.phase === 'plan_ready'
                ? `<div class="assistant-quick-actions">
                <button type="button" class="coach-quick-btn" data-quick="帮我把今日行动减到 2 条">精简今日行动</button>
                <button type="button" class="btn btn-ghost btn-sm" id="assistant-new-btn">新建规划</button>
              </div>`
                : `<div class="assistant-quick-actions">
                <button type="button" class="coach-quick-btn" data-quick="帮我整理出行动方案">帮我整理</button>
              </div>`
            }
          </div>
        </div>
        ${planHtml}
      </div>
    </section>
  `

  const msgBox = container.querySelector<HTMLDivElement>('#messages')
  if (msgBox) scrollChatMessages(msgBox, 'bottom')

  if (!view.sending) {
    const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
    input?.focus({ preventScroll: true })
  }
}

export function getAssistantInput(container: HTMLElement): HTMLTextAreaElement | null {
  return container.querySelector('#msg-input')
}
