import type { PlanningMessage, PlanningResult } from './api'
import { phaseLabel } from './api'
import { renderMarkdown } from './markdown'
import { snapshotChatStreamScroll, scrollChatDuringStream, scrollChatMessages } from './chat-scroll'
import { escapeHtml } from './utils'

const PLAN_EXPAND_KEY = 'regulus.assistant.planExpanded'

export interface AssistantViewState {
  sessionId: string
  phase: string
  messages: PlanningMessage[]
  plan: PlanningResult | null
  planExpanded: boolean
  sending: boolean
  synthesizing: boolean
  error: string
}

export function readPlanExpandedPreference(): boolean {
  try {
    return sessionStorage.getItem(PLAN_EXPAND_KEY) === '1'
  } catch {
    return false
  }
}

export function writePlanExpandedPreference(expanded: boolean): void {
  try {
    sessionStorage.setItem(PLAN_EXPAND_KEY, expanded ? '1' : '0')
  } catch {
    /* ignore */
  }
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

function isChecked(plan: PlanningResult, key: string): boolean {
  return Boolean(plan.ui_state?.checked?.[key])
}

function renderCheckItem(opts: {
  checkKey: string
  title: string
  meta: string
  checked: boolean
}): string {
  return `
    <li class="assistant-plan-item assistant-plan-item--check${opts.checked ? ' is-checked' : ''}">
      <label class="assistant-check-row">
        <input
          type="checkbox"
          class="assistant-check"
          data-check-key="${escapeHtml(opts.checkKey)}"
          ${opts.checked ? 'checked' : ''}
        />
        <span class="assistant-check-body">
          <span class="assistant-plan-item-title">${escapeHtml(opts.title)}</span>
          ${opts.meta ? `<span class="assistant-plan-item-meta">${escapeHtml(opts.meta)}</span>` : ''}
        </span>
      </label>
    </li>
  `
}

function renderMatrixSection(
  title: string,
  items: { title: string; why?: string; minutes?: number; next_step?: string; reason?: string }[],
): string {
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

/** @internal exported for unit tests */
export function buildAssistantPlanPanelHtml(plan: PlanningResult, expanded: boolean): string {
  const focus = plan.focus
  const northStar = focus?.north_star?.trim() || plan.situation_summary || '专注当下最该推进的一条线'
  const pinned = Boolean(plan.ui_state?.north_star_pinned)
  const clearFirst = (plan.clear_first ?? []).slice(0, 2)
  const todayLearning = focus?.today_learning
  const canStart = Boolean(todayLearning?.matched_domain_id && todayLearning?.matched_node_key)
  const today = plan.action_plan?.today ?? []
  const thisWeek = plan.action_plan?.this_week ?? []

  const clearHtml = clearFirst.length
    ? `
      <div class="assistant-plan-section assistant-plan-section--highlight">
        <h3 class="assistant-plan-section-title">先清障</h3>
        <ul class="assistant-plan-list">
          ${clearFirst
            .map((item, i) => {
              const meta: string[] = []
              if (item.minutes) meta.push(`${item.minutes} 分钟`)
              if (item.next_step) meta.push(item.next_step)
              return renderCheckItem({
                checkKey: `clear:${i}`,
                title: item.title,
                meta: meta.join(' · '),
                checked: isChecked(plan, `clear:${i}`),
              })
            })
            .join('')}
        </ul>
      </div>
    `
    : ''

  const learningHtml = todayLearning
    ? `
      <div class="assistant-plan-section assistant-plan-section--focus-learn">
        <h3 class="assistant-plan-section-title">今日学习</h3>
        <ul class="assistant-plan-list">
          <li class="assistant-plan-item assistant-plan-item--learning">
            <span class="assistant-plan-item-title">${escapeHtml(todayLearning.title)}</span>
            <span class="assistant-plan-item-meta">${todayLearning.minutes || 15} 分钟</span>
            ${
              canStart
                ? `<button type="button" class="btn btn-primary btn-sm assistant-start-learn" data-domain-id="${escapeHtml(todayLearning.matched_domain_id!)}" data-node-key="${escapeHtml(todayLearning.matched_node_key!)}" data-node-title="${escapeHtml(todayLearning.matched_node_title ?? todayLearning.title)}">
                开始 15 分钟
              </button>`
                : `<a class="btn btn-secondary btn-sm" href="#/courses">去选课续练</a>`
            }
          </li>
        </ul>
      </div>
    `
    : ''

  const detailsHtml = expanded
    ? `
      <div class="assistant-plan-details">
        ${focus?.week_wedge ? `<p class="assistant-plan-wedge"><span class="assistant-plan-wedge-label">本周楔子</span>${escapeHtml(focus.week_wedge)}</p>` : ''}
        ${focus?.why ? `<p class="assistant-plan-summary">${escapeHtml(focus.why)}</p>` : ''}
        ${plan.situation_summary && plan.situation_summary !== northStar ? `<p class="assistant-plan-summary">${escapeHtml(plan.situation_summary)}</p>` : ''}

        ${
          today.length
            ? `
          <div class="assistant-plan-section">
            <h3 class="assistant-plan-section-title">今日行动</h3>
            <ul class="assistant-plan-list">
              ${today
                .map((item, i) =>
                  renderCheckItem({
                    checkKey: `today:${i}`,
                    title: item.title,
                    meta: `${item.minutes} 分钟 · ${item.kind === 'learning' ? '学习' : '事务'}${item.reason ? ` · ${item.reason}` : ''}`,
                    checked: isChecked(plan, `today:${i}`),
                  }),
                )
                .join('')}
            </ul>
          </div>
        `
            : ''
        }

        ${renderMatrixSection('重要且紧急', plan.matrix?.important_urgent ?? [])}
        ${renderMatrixSection('重要不紧急', plan.matrix?.important_not_urgent ?? [])}
        ${renderMatrixSection('快速了结', plan.matrix?.quick_wins ?? [])}
        ${renderMatrixSection('建议暂缓', plan.matrix?.defer_or_drop ?? [])}

        ${
          thisWeek.length
            ? `
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
              `,
                )
                .join('')}
            </ul>
          </div>
        `
            : ''
        }

        ${plan.mindset_note ? `<p class="assistant-plan-mindset">${escapeHtml(plan.mindset_note)}</p>` : ''}
      </div>
    `
    : ''

  return `
    <aside class="assistant-plan-panel card" aria-label="聚焦位">
      <header class="assistant-plan-header">
        <div class="assistant-plan-header-row">
          <h2 class="assistant-plan-heading">聚焦位</h2>
          <span class="assistant-pin-badge${pinned ? ' is-pinned' : ''}">${pinned ? '已钉住' : '待确认'}</span>
        </div>
        <p class="assistant-north-star">${escapeHtml(northStar)}</p>
        <div class="assistant-pin-actions">
          <button type="button" class="btn ${pinned ? 'btn-secondary' : 'btn-primary'} btn-sm" id="assistant-pin-btn">
            ${pinned ? '取消钉住' : '钉住这个目标'}
          </button>
          <button type="button" class="btn btn-ghost btn-sm" id="assistant-expand-btn" aria-expanded="${expanded ? 'true' : 'false'}">
            ${expanded ? '收起详情' : '展开详情'}
          </button>
        </div>
      </header>

      ${clearHtml}
      ${learningHtml}
      ${detailsHtml}
    </aside>
  `
}

export function renderAssistantView(container: HTMLElement, view: AssistantViewState): void {
  if (view.sending) {
    snapshotChatStreamScroll(container.querySelector('#messages'))
  }

  const bubbles = view.messages
    .map((m) => `<div class="bubble ${m.role}">${formatBubbleContent(m)}</div>`)
    .join('')

  const loadingText = view.synthesizing ? '正在帮你整理聚焦方案…' : '思考中…'
  // 仅在真正合成方案时展示右侧占位；普通倾听对话不占右栏，避免「以为要出方案了」
  const planHtml = view.plan
    ? buildAssistantPlanPanelHtml(view.plan, view.planExpanded)
    : view.synthesizing
      ? `<aside class="assistant-plan-panel card assistant-plan-panel--pending" aria-label="聚焦位">
          <header class="assistant-plan-header">
            <h2 class="assistant-plan-heading">聚焦位</h2>
          </header>
          <p class="assistant-plan-pending-text">正在整理北星、清障与今日行动…</p>
        </aside>`
      : ''

  container.innerHTML = `
    <section class="page page-coach page-assistant">
      <header class="page-header page-header-compact">
        <h1 class="page-title">行动助手</h1>
        <span class="phase-badge">${phaseLabel(view.phase)}</span>
      </header>

      <div class="assistant-layout${planHtml ? ' has-plan-panel' : ''}">
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
                  placeholder="脑子里装太多？先倒几条出来…"
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
                <button type="button" class="assistant-quick-btn" data-quick="帮我把今日行动减到 2 条">精简今日行动</button>
                <button type="button" class="btn btn-ghost btn-sm" id="assistant-new-btn">新建规划</button>
              </div>`
                : `<div class="assistant-quick-actions">
                <button type="button" class="assistant-quick-btn" data-quick="帮我整理出行动方案">帮我整理</button>
              </div>`
            }
          </div>
        </div>
        ${planHtml}
      </div>
    </section>
  `

  const msgBox = container.querySelector<HTMLDivElement>('#messages')
  if (msgBox) {
    if (view.sending) {
      // 流式整页重绘：上滑暂停跟随，贴近底部才滚到底
      scrollChatDuringStream(msgBox)
    } else {
      scrollChatMessages(msgBox, 'bottom')
    }
  }

  if (!view.sending) {
    const input = container.querySelector<HTMLTextAreaElement>('#msg-input')
    input?.focus({ preventScroll: true })
  }
}

export function getAssistantInput(container: HTMLElement): HTMLTextAreaElement | null {
  return container.querySelector('#msg-input')
}
