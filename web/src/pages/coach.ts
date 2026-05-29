import {
  getSession,
  getDomainTree,
  getUserProgress,
  sendMessage,
  phaseLabel,
  ApiError,
} from '../lib/api'
import { renderMarkdown } from '../lib/markdown'
import { setBreadcrumb, updateSidebar } from '../components/layout'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

const EXERCISE_MARKER = '做完后直接把答案发给我'

const PRACTICE_INVITE_PATTERNS = [
  '开始练习',
  '再来一道',
  '再练一题',
  '再练一遍',
  '练一道题',
  '练一题',
  '再出一题',
  '继续练习',
]

function hasExerciseInHistory(messages: ChatMessage[]): boolean {
  return messages.some((m) => m.role === 'assistant' && m.content.includes(EXERCISE_MARKER))
}

function invitesPractice(content: string): boolean {
  return PRACTICE_INVITE_PATTERNS.some((p) => content.includes(p))
}

export async function renderCoach(container: HTMLElement, sessionId: string): Promise<void> {
  container.innerHTML = `
    <section class="page page-coach">
      <div class="page-loading">
        <div class="spinner" aria-hidden="true"></div>
        <p>加载对话…</p>
      </div>
    </section>
  `

  let messages: ChatMessage[] = []
  let phase = 'explain'
  let nodeTitle = ''
  let domainId = ''
  let domainName = ''
  let domainNodeTotal = 0
  let domainCompleted = 0
  let sending = false
  let hasAttemptedExercise = false

  const showError = (msg: string) => {
    const errEl = container.querySelector<HTMLDivElement>('#coach-error')
    if (errEl) errEl.innerHTML = `<div class="alert alert-error">${escapeHtml(msg)}</div>`
  }

  const clearError = () => {
    const errEl = container.querySelector<HTMLDivElement>('#coach-error')
    if (errEl) errEl.innerHTML = ''
  }

  const showToast = (html: string) => {
    const toastEl = container.querySelector<HTMLDivElement>('#coach-toast')
    if (toastEl) toastEl.innerHTML = html
  }

  try {
    const detail = await getSession(sessionId)
    phase = detail.phase
    nodeTitle = detail.nodeTitle
    domainId = detail.domainId
    const [tree, progress] = await Promise.all([
      getDomainTree(domainId).catch(() => null),
      getUserProgress(domainId).catch(() => []),
    ])
    domainName = tree?.domainName ?? '当前课程'
    domainNodeTotal = tree?.layers.reduce((n, l) => n + l.nodes.length, 0) ?? 0
    domainCompleted = progress.filter((p) => p.status === 'completed').length
    messages = detail.messages.map((m) => ({
      role: m.role === 'user' ? ('user' as const) : ('assistant' as const),
      content: m.content,
    }))
    hasAttemptedExercise =
      phase === 'review' || phase === 'completed' || hasExerciseInHistory(messages)
  } catch (e) {
    container.innerHTML = `
      <section class="page page-coach">
        <div class="alert alert-error">${e instanceof ApiError ? e.message : '加载失败'}</div>
      </section>
    `
    return
  }

  await updateSidebar({
    active: 'coach',
    domainId,
    domainName,
    domainNodeTotal,
    domainCompleted,
    nodeTitle,
  })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '知识树', href: `#/tree/${domainId}` },
    { label: nodeTitle },
  ])

  const dispatch = async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || sending) return

    const prevPhase = phase
    messages.push({ role: 'user', content: trimmed })
    sending = true
    clearError()
    render()

    try {
      const reply = await sendMessage(sessionId, trimmed)
      messages.push({ role: 'assistant', content: reply.content })
      phase = reply.phase
      if (prevPhase === 'exercise' || reply.content.includes(EXERCISE_MARKER)) {
        hasAttemptedExercise = true
      }
      if (reply.nodeCompleted) {
        showToast('<div class="alert alert-success">节点已点亮</div>')
      }
    } catch (e) {
      messages.pop()
      showError(e instanceof ApiError ? e.message : '发送失败，请重试')
    } finally {
      sending = false
      render()
      container.querySelector<HTMLInputElement>('#msg-input')?.focus()
    }
  }

  const bindEvents = () => {
    type CoachContainer = HTMLElement & { __coachClickBound?: boolean; __coachDispatch?: (t: string) => void }
    const root = container as CoachContainer
    root.__coachDispatch = (text: string) => {
      void dispatch(text)
    }

    if (root.__coachClickBound) return
    root.__coachClickBound = true

    container.addEventListener('click', (e) => {
      const target = e.target as HTMLElement

      const practiceBtn = target.closest<HTMLButtonElement>('.coach-inline-practice')
      if (practiceBtn && !practiceBtn.disabled) {
        e.preventDefault()
        root.__coachDispatch?.(practiceBtn.dataset.practice ?? '开始练习')
        return
      }

      const quickBtn = target.closest<HTMLButtonElement>('.coach-quick-btn')
      if (quickBtn && !quickBtn.disabled) {
        e.preventDefault()
        root.__coachDispatch?.(quickBtn.dataset.quick ?? '')
        return
      }

      const sendBtn = target.closest<HTMLButtonElement>('#send-btn')
      if (sendBtn && !sendBtn.disabled) {
        e.preventDefault()
        const input = container.querySelector<HTMLInputElement>('#msg-input')
        if (input) root.__coachDispatch?.(input.value)
      }
    })

    container.addEventListener('keydown', (e) => {
      if (e.key !== 'Enter') return
      const input = e.target as HTMLElement
      if (input.id !== 'msg-input' || !(input instanceof HTMLInputElement)) return
      if (e.isComposing || input.dataset.composing === '1') return
      e.preventDefault()
      root.__coachDispatch?.(input.value)
    })

    container.addEventListener('compositionstart', (e) => {
      const input = e.target as HTMLElement
      if (input.id === 'msg-input') input.dataset.composing = '1'
    })

    container.addEventListener('compositionend', (e) => {
      const input = e.target as HTMLElement
      if (input.id === 'msg-input') delete input.dataset.composing
    })
  }

  const render = () => {
    const completed = phase === 'completed'
    const inExercise = phase === 'exercise'
    const practiceLabel = hasAttemptedExercise ? '再来一道' : '开始练习'
    const lastIdx = messages.length - 1

    const bubbles = messages
      .map((m, i) => {
        const showInlinePractice =
          !completed &&
          !inExercise &&
          !sending &&
          m.role === 'assistant' &&
          i === lastIdx &&
          invitesPractice(m.content)
        const inlineBtn = showInlinePractice
          ? `
            <div class="bubble-cta">
              <button type="button" class="coach-inline-practice" data-practice="${practiceLabel}">
                ${escapeHtml(practiceLabel)}
              </button>
            </div>
          `
          : ''
        return `<div class="bubble ${m.role}">${formatBubbleContent(m)}${inlineBtn}</div>`
      })
      .join('')

    const placeholder = completed
      ? '本节点已完成'
      : inExercise
        ? '写下你的答案，或说「不懂」「换一题」'
        : '有疑问？在这里提问'

    const footer = completed
      ? `
        <div class="coach-completed-actions">
          <a class="btn btn-primary" href="#/tree/${domainId}">返回知识树</a>
        </div>
      `
      : `
        <div class="chat-input-row">
          <input class="input" id="msg-input" type="text" placeholder="${placeholder}" autocomplete="off" ${sending ? 'disabled' : ''} aria-label="消息输入" />
          <button type="button" class="btn btn-ghost" id="send-btn" ${sending ? 'disabled' : ''}>${sending ? '…' : '发送'}</button>
        </div>
        ${inExercise ? `
          <div class="coach-quick-actions">
            <button type="button" class="coach-quick-btn" data-quick="不懂，回讲解">不懂，回讲解</button>
            <button type="button" class="coach-quick-btn" data-quick="换一题">换一题</button>
          </div>
        ` : ''}
      `

    container.innerHTML = `
      <section class="page page-coach">
        <header class="page-header page-header-compact">
          <h1 class="page-title">${escapeHtml(nodeTitle)}</h1>
          <span class="phase-badge">${phaseLabel(phase)}</span>
        </header>

        <div class="chat-panel card">
          <div class="chat-messages" id="messages" role="log" aria-live="polite">${bubbles}${sending ? '<div class="coach-loading">教练思考中…</div>' : ''}</div>
          <div class="coach-footer">
            <div id="coach-error"></div>
            <div id="coach-toast"></div>
            ${footer}
          </div>
        </div>
      </section>
    `

    const msgBox = container.querySelector<HTMLDivElement>('#messages')
    if (msgBox) msgBox.scrollTop = msgBox.scrollHeight

    if (!completed && !sending) {
      container.querySelector<HTMLInputElement>('#msg-input')?.focus()
    }
  }

  bindEvents()
  render()
}

function formatBubbleContent(m: ChatMessage): string {
  if (m.role === 'assistant') {
    return `<div class="md-body">${renderMarkdown(m.content)}</div>`
  }
  return escapeHtml(m.content)
}

function escapeHtml(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}
