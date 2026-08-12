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
import { consumeStreamJustEnded, getChatStreamFollow, markStreamJustEnded, resetChatStreamFollow, scrollChatDuringStream, scrollChatMessages, snapshotChatStreamScroll } from './chat-scroll'
import { escapeHtml } from './utils'

export interface CoachRenderChrome {
  /** 已完成时「继续 · xxx」标题（由课程树解析） */
  completedNextTitle: string
}

/** 流式时临时闭合未结束的 fence，减轻 Markdown 结构来回跳导致的闪烁 */
export function closeOpenMarkdownFences(text: string): string {
  const fences = text.match(/^```/gm)
  if (fences && fences.length % 2 === 1) {
    return `${text}\n\`\`\``
  }
  return text
}

export function renderStreamingMarkdown(text: string): string {
  return renderMarkdown(closeOpenMarkdownFences(text))
}

function formatBubbleContent(m: ChatMessage, opts?: { streaming?: boolean }): string {
  if (m.role === 'assistant') {
    const text = opts?.streaming ? m.content : extractEmbeddedExercise(m.content).displayContent
    const html = opts?.streaming ? renderStreamingMarkdown(text) : renderMarkdown(text)
    return `<div class="md-body">${html}</div>`
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
  const forbidden = msg.includes('无权')
  const actions = forbidden
    ? `<a class="btn btn-secondary btn-sm" href="#/courses">返回我的课程</a>
        <a class="btn btn-ghost btn-sm" href="#/" style="margin-left:0.5rem">开始学习</a>`
    : `<button type="button" class="btn btn-secondary btn-sm" id="coach-retry-btn">重试</button>
        ${domainId ? `<a class="btn btn-ghost btn-sm" href="#/tree/${domainId}" style="margin-left:0.5rem">返回课程</a>` : ''}`
  const hint = forbidden
    ? `<p class="page-loading-hint" style="margin-top:0.75rem;text-align:center">该对话属于其他学习角色，切换角色后无法继续打开。</p>`
    : ''
  return `
    <section class="page page-coach">
      <div class="alert alert-error">${escapeHtml(msg)}</div>
      ${hint}
      <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
        ${actions}
      </p>
    </section>
  `
}

function renderComposer(view: CoachViewState, draft?: ExerciseDraft): string {
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

  if (view.composerMode === 'exercise_text' && view.exercise) {
    return renderExerciseComposer({
      exercise: view.exercise,
      placeholder,
      sending,
      quickActionsHtml: quickActions,
      draftText: draft?.text,
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
  if (view.sending || view.streaming) {
    snapshotChatStreamScroll(container.querySelector('#messages'))
  }

  const lastIdx = view.messages.length - 1
  const bubbles = view.messages
    .map((m, i) => {
      const isStreamingBubble =
        view.streaming && i === lastIdx && m.role === 'assistant'
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
      return `<div class="bubble ${m.role}">${formatBubbleContent(m, { streaming: isStreamingBubble })}${inlineBtn}</div>`
    })
    .join('')

  const nodeCompleted = view.phase === 'completed'
  const footer = nodeCompleted ? renderCompletedDock(view, chrome) : renderComposer(view, draft)

  const errorHtml = view.error
    ? `<div class="alert alert-error">${escapeHtml(view.error)}</div>`
    : ''

  const loadingHint = view.stageHint.trim() || '教练思考中…'
  const showLoading = view.sending
  const loadingHtml = showLoading
    ? `<div class="coach-loading">${escapeHtml(loadingHint)}</div>`
    : ''

  container.innerHTML = `
      <section class="page page-coach">
        <header class="page-header page-header-compact">
          <h1 class="page-title">${escapeHtml(view.nodeTitle)}</h1>
          <span class="phase-badge">${phaseLabel(view.phase)}</span>
        </header>

        <div class="chat-panel card">
          <div class="chat-messages" id="messages" role="log" aria-live="polite">${bubbles}${loadingHtml}</div>
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
    if (view.sending || view.streaming) {
      // 流式进行中：跟随/暂停模式
      scrollChatDuringStream(msgBox)
    } else if (consumeStreamJustEnded()) {
      // 流式刚结束的首帧：用 scrollChatDuringStream 保持当前位置（streamFollow=true→底部，false→阅读位）
      // 不走 readable，避免整页 innerHTML 重建后 scrollTop 被同步重置为 0
      scrollChatDuringStream(msgBox)
    } else if (scrollMode === 'readable' && !getChatStreamFollow()) {
      // 用户在流式中主动上滑过：保留阅读位置，不强制锚到回复开头
    } else {
      scrollChatMessages(msgBox, scrollMode)
    }
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
    }
  }
}

/**
 * 流式增量更新：只改最后一条助手气泡与 loading 文案，避免整页 innerHTML 闪烁。
 * 若页面骨架尚未就绪则返回 false，调用方应回退到完整 renderCoachView。
 */
export function patchCoachStreamView(container: HTMLElement, view: CoachViewState): boolean {
  const root = container.querySelector('.page-coach')
  const msgBox = container.querySelector<HTMLDivElement>('#messages')
  if (!root || !msgBox) return false

  const loadingHint = view.stageHint.trim() || '教练思考中…'
  let loading = msgBox.querySelector<HTMLElement>('.coach-loading')
  if (view.sending) {
    if (!loading) {
      loading = document.createElement('div')
      loading.className = 'coach-loading'
      msgBox.appendChild(loading)
    }
    if (loading.textContent !== loadingHint) {
      loading.textContent = loadingHint
    }
  } else if (loading) {
    loading.remove()
    loading = null
    // 流式结束：保护后续整页重绘时不要同步跳到顶部
    markStreamJustEnded()
  }

  const last = view.messages[view.messages.length - 1]
  const wantAssistant = Boolean(last && last.role === 'assistant' && last.content)

  let streamBubble = msgBox.querySelector<HTMLElement>('.bubble.assistant.coach-stream-bubble')
  if (wantAssistant) {
    if (!streamBubble) {
      streamBubble = document.createElement('div')
      streamBubble.className = 'bubble assistant coach-stream-bubble'
      streamBubble.innerHTML = '<div class="md-body"></div>'
      if (loading) msgBox.insertBefore(streamBubble, loading)
      else msgBox.appendChild(streamBubble)
      // 新气泡：恢复跟随（用户上一轮上滑暂停后，新回复仍应从底部跟）
      resetChatStreamFollow(msgBox)
    }
    const mdBody = streamBubble.querySelector<HTMLElement>('.md-body')
    if (!mdBody) return false
    const prev = mdBody.dataset.streamText ?? ''
    if (prev !== last!.content) {
      mdBody.dataset.streamText = last!.content
      // 流式阶段不做 extractEmbedded，避免半截 JSON 误伤
      mdBody.innerHTML = renderStreamingMarkdown(last!.content)
    }
  }

  // 上滑暂停跟随，贴近底部才滚到底 —— 避免流式强制拽回底部
  scrollChatDuringStream(msgBox)
  return true
}

export function getMsgInput(container: HTMLElement): HTMLInputElement | HTMLTextAreaElement | null {
  return container.querySelector('#msg-input')
}
