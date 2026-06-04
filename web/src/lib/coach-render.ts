import { phaseLabel } from './api'
import {
  extractEmbeddedExercise,
  renderExerciseComposer,
  restoreExerciseDraft,
  type ExerciseDraft,
} from './coach-exercise'
import {
  REAL_WORLD_CASE_PROMPT,
  SKIP_MASTERY_PROMPT,
  type ChatMessage,
  type CoachViewState,
} from './coach-view-state'
import { renderMarkdown } from './markdown'
import { scrollChatMessages } from './chat-scroll'

export interface CoachRenderChrome {
  /** 已完成时「继续 · xxx」标题（由课程树解析） */
  completedNextTitle: string
}

export function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

function formatBubbleContent(m: ChatMessage): string {
  if (m.role === 'assistant') {
    const { displayContent } = extractEmbeddedExercise(m.content)
    return `<div class="md-body">${renderMarkdown(displayContent)}</div>`
  }
  return escapeHtml(m.content)
}

export function coachLoadingHtml(hint: string): string {
  return `
    <section class="page page-coach">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>正在连接教练…</p>
        <p class="page-loading-hint">${escapeHtml(hint)}</p>
      </div>
    </section>
  `
}

export function coachErrorHtml(msg: string, domainId: string): string {
  return `
    <section class="page page-coach">
      <div class="alert alert-error">${escapeHtml(msg)}</div>
      <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
        <button type="button" class="btn btn-secondary btn-sm" id="coach-retry-btn">重试</button>
        ${domainId ? `<a class="btn btn-ghost btn-sm" href="#/tree/${domainId}" style="margin-left:0.5rem">返回课程</a>` : ''}
      </p>
    </section>
  `
}

function renderComposer(view: CoachViewState): string {
  const { sending, placeholder } = view
  const quickActions =
    view.composerMode === 'exercise_text' || view.composerMode === 'exercise_choice'
      ? `
          <div class="coach-quick-actions">
            <button type="button" class="coach-quick-btn" data-quick="不懂，回讲解">不懂，回讲解</button>
            <button type="button" class="coach-quick-btn" data-quick="换一题">换一题</button>
          </div>
        `
      : ''

  if (view.composerMode === 'exercise_choice' && view.exercise) {
    return renderExerciseComposer({
      exercise: view.exercise,
      placeholder,
      sending,
      quickActionsHtml: quickActions,
    })
  }

  if (view.composerMode === 'exercise_text') {
    return `
        <div class="coach-composer coach-composer--exercise">
          ${quickActions}
          <div class="coach-composer-head">
            <span class="coach-composer-label">练习作答</span>
            <span class="coach-composer-hint">Enter 换行 · Ctrl+Enter 提交</span>
          </div>
          <div class="coach-composer-body">
            <textarea
              class="input coach-answer-input"
              id="msg-input"
              rows="5"
              placeholder="${escapeHtml(placeholder)}"
              autocomplete="off"
              ${sending ? 'disabled' : ''}
              aria-label="练习作答"
            ></textarea>
            <button type="button" class="btn btn-primary coach-send-btn" id="send-btn" ${sending ? 'disabled' : ''}>${sending ? '…' : '提交答案'}</button>
          </div>
        </div>
      `
  }

  return `
        <div class="chat-input-row">
          <input class="input" id="msg-input" type="text" placeholder="${escapeHtml(placeholder)}" autocomplete="off" ${sending ? 'disabled' : ''} aria-label="消息输入" />
          <button type="button" class="btn btn-ghost" id="send-btn" ${sending ? 'disabled' : ''}>${sending ? '…' : '发送'}</button>
        </div>
      `
}

/** 完成态：输入与「下一节」合在同一底部 Dock，避免大卡片 + 第二条输入栏 */
function renderCompletedDock(view: CoachViewState, chrome: CoachRenderChrome): string {
  const { sending, placeholder, domainId } = view
  const nextTitle = chrome.completedNextTitle?.trim() ?? ''
  const nextTitleEsc = nextTitle ? escapeHtml(nextTitle) : ''
  const phEsc = escapeHtml(placeholder)

  const nextMeta = nextTitle
    ? `<p class="coach-completed-dock__next" title="${nextTitleEsc}">
            <span class="coach-completed-dock__next-kicker">下一节</span>
            <span class="coach-completed-dock__next-title">${nextTitleEsc}</span>
          </p>`
    : `<p class="coach-completed-dock__next coach-completed-dock__next--muted">本课程节点已全部完成</p>`

  const nextBtn = nextTitle
    ? `<button type="button" class="btn btn-primary coach-completed-dock__next-btn" id="next-node-btn" ${sending ? 'disabled' : ''} title="${nextTitleEsc}" aria-label="继续学习下一节：${nextTitleEsc}">
            ${sending ? '进入中…' : '下一节'}
          </button>`
    : `<a class="btn btn-primary coach-completed-dock__next-btn" href="#/tree/${domainId}">返回课程</a>`

  const chipLabel = nextTitle ? '本节已完成' : '全部完成'

  return `
        <div class="coach-completed-dock" role="region" aria-label="本节学习已完成">
          <div class="coach-completed-dock__meta">
            <span class="coach-completed-dock__chip">${chipLabel}</span>
            ${nextMeta}
            ${nextTitle ? `<a class="coach-completed-dock__back" href="#/tree/${domainId}">返回课程</a>` : ''}
          </div>
          <div class="coach-completed-dock__row">
            <input
              class="input coach-completed-dock__input"
              id="msg-input"
              type="text"
              placeholder="${phEsc}"
              autocomplete="off"
              ${sending ? 'disabled' : ''}
              aria-label="对本节提问"
            />
            <button type="button" class="btn btn-ghost coach-completed-dock__send" id="send-btn" ${sending ? 'disabled' : ''}>
              ${sending ? '…' : '发送'}
            </button>
            ${nextBtn}
          </div>
        </div>
      `
}

export function renderCoachView(
  container: HTMLElement,
  view: CoachViewState,
  chrome: CoachRenderChrome,
  draft: ExerciseDraft,
  opts?: { consumePreferReadable?: () => boolean }
): void {
  const lastIdx = view.messages.length - 1
  const bubbles = view.messages
    .map((m, i) => {
      const showInline =
        view.showInlinePracticeOnLast && i === lastIdx && m.role === 'assistant'
      const inlineBtn = showInline
        ? `
            <div class="bubble-cta">
              <div class="bubble-cta__leading">
              ${
                view.showInlineCaseOnLast
                  ? `<button type="button" class="coach-inline-case" data-case="${REAL_WORLD_CASE_PROMPT}" title="结合生产场景、代码与流程设计理解概念">
                实际案例
              </button>`
                  : ''
              }
              <button type="button" class="coach-inline-practice" data-practice="${view.practiceLabel}">
                ${escapeHtml(view.practiceLabel)}
              </button>
              </div>
              ${
                view.showInlineMasteryOnLast
                  ? `<div class="bubble-cta__trailing">
                <button type="button" class="coach-quick-btn coach-inline-mastery" data-quick="${SKIP_MASTERY_PROMPT}">
                ${escapeHtml(SKIP_MASTERY_PROMPT)}
              </button>
              </div>`
                  : ''
              }
            </div>
          `
        : ''
      return `<div class="bubble ${m.role}">${formatBubbleContent(m)}${inlineBtn}</div>`
    })
    .join('')

  const nodeCompleted = view.phase === 'completed'
  const footer = nodeCompleted ? renderCompletedDock(view, chrome) : renderComposer(view)

  const errorHtml = view.error
    ? `<div class="alert alert-error">${escapeHtml(view.error)}</div>`
    : ''

  container.innerHTML = `
      <section class="page page-coach">
        <header class="page-header page-header-compact">
          <h1 class="page-title">${escapeHtml(view.nodeTitle)}</h1>
          <span class="phase-badge">${phaseLabel(view.phase)}</span>
        </header>

        <div class="chat-panel card">
          <div class="chat-messages" id="messages" role="log" aria-live="polite">${bubbles}${view.sending ? '<div class="coach-loading">教练思考中…</div>' : ''}</div>
          <div class="coach-footer${nodeCompleted ? ' coach-footer--completed' : ''}">
            <div id="coach-error">${errorHtml}</div>
            ${nodeCompleted ? '' : `<div id="coach-toast">${view.toastHtml}</div>`}
            ${footer}
          </div>
        </div>
      </section>
    `

  let scrollMode = view.scrollMode
  if (opts?.consumePreferReadable && scrollMode === 'readable') {
    opts.consumePreferReadable()
  }

  const msgBox = container.querySelector<HTMLDivElement>('#messages')
  if (msgBox) {
    scrollChatMessages(msgBox, scrollMode)
  }

  if (!view.sending) {
    restoreExerciseDraft(container, draft, view.exercise)
    const input = container.querySelector<HTMLInputElement | HTMLTextAreaElement>('#msg-input')
    if (input) {
      if (input instanceof HTMLTextAreaElement) {
        input.style.height = 'auto'
        const max = Math.min(window.innerHeight * 0.38, 320)
        input.style.height = `${Math.min(input.scrollHeight, max)}px`
      }
      input.focus({ preventScroll: true })
      if (msgBox && scrollMode === 'readable') {
        scrollChatMessages(msgBox, 'readable')
      }
    }
  }
}

export function getMsgInput(container: HTMLElement): HTMLInputElement | HTMLTextAreaElement | null {
  return container.querySelector('#msg-input')
}
