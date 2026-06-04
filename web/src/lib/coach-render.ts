import { phaseLabel } from './api'
import {
  extractEmbeddedExercise,
  renderExerciseComposer,
  restoreExerciseDraft,
  type ExerciseDraft,
} from './coach-exercise'
import {
  REAL_WORLD_CASE_PROMPT,
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

function renderCompletedFooter(view: CoachViewState, chrome: CoachRenderChrome): string {
  const { sending, domainId } = view
  const nextTitle = chrome.completedNextTitle
  if (nextTitle) {
    return `
        <div class="coach-completed-bar">
          <p class="coach-completed-bar__hint">本节点已完成</p>
          <div class="coach-completed-bar__actions">
            <button type="button" class="btn btn-primary" id="next-node-btn" ${sending ? 'disabled' : ''}>
              ${sending ? '进入中…' : `继续 · ${escapeHtml(nextTitle)}`}
            </button>
            <a class="btn btn-ghost btn-sm coach-completed-bar__back" href="#/tree/${domainId}">返回课程</a>
          </div>
        </div>
      `
  }
  return `
        <div class="coach-completed-bar">
          <p class="coach-completed-bar__hint">本课程节点已全部完成</p>
          <div class="coach-completed-bar__actions">
            <a class="btn btn-primary btn-sm" href="#/tree/${domainId}">返回课程</a>
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
              <button type="button" class="coach-inline-case" data-case="${REAL_WORLD_CASE_PROMPT}" title="结合生产场景、代码与流程设计理解概念">
                实际案例
              </button>
              <button type="button" class="coach-inline-practice" data-practice="${view.practiceLabel}">
                ${escapeHtml(view.practiceLabel)}
              </button>
            </div>
          `
        : ''
      return `<div class="bubble ${m.role}">${formatBubbleContent(m)}${inlineBtn}</div>`
    })
    .join('')

  const footer =
    view.composerMode === 'completed' ? renderCompletedFooter(view, chrome) : renderComposer(view)

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
          <div class="coach-footer">
            <div id="coach-error">${errorHtml}</div>
            <div id="coach-toast">${view.toastHtml}</div>
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

  if (view.composerMode !== 'completed' && !view.sending) {
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
