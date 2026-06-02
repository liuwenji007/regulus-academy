import {
  getSession,
  getDomainTree,
  getUserProgress,
  sendMessage,
  phaseLabel,
  ApiError,
  type SessionDetail,
  type SessionExercise,
} from '../lib/api'
import {
  collectExerciseAnswer,
  exercisePlaceholder,
  normalizeSessionExercise,
  readExerciseDraft,
  renderExerciseComposer,
  restoreExerciseDraft,
  tryFormatJsonInTextarea,
  type ExerciseDraft,
} from '../lib/coach-exercise'
import { setAppBusy } from '../lib/app-busy'
import { clearTreeSessionOverlay } from '../lib/session-loading-overlay'
import {
  clearSessionBootstrap,
  peekSessionBootstrap,
  type SessionBootstrap,
} from '../lib/session-bootstrap'
import { scrollChatMessages } from '../lib/chat-scroll'
import { renderMarkdown } from '../lib/markdown'
import { setBreadcrumb, updateSidebar, refreshLLMStatusAfterBusy } from '../components/layout'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

const EXERCISE_MARKER = '做完后直接把答案发给我'
const REAL_WORLD_CASE_PROMPT = '实际案例'
const SKIP_MASTERY_PROMPT = '已经掌握，下一节'

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

let coachRenderGen = 0

function hasExerciseInHistory(messages: ChatMessage[]): boolean {
  return messages.some((m) => m.role === 'assistant' && m.content.includes(EXERCISE_MARKER))
}

function invitesPractice(content: string): boolean {
  return PRACTICE_INVITE_PATTERNS.some((p) => content.includes(p))
}

function shouldShowPracticeCTA(content: string, inExercise: boolean): boolean {
  if (!invitesPractice(content)) return false
  if (!inExercise) return true
  if (content.includes(EXERCISE_MARKER)) return false
  return (
    content.includes('再来一道') ||
    content.includes('继续练习') ||
    (content.includes('点击') && content.includes('开始练习'))
  )
}

function isAnsweringExercise(inExercise: boolean, lastAssistantContent: string): boolean {
  return inExercise && !shouldShowPracticeCTA(lastAssistantContent, true)
}

function getMsgInput(container: HTMLElement): HTMLInputElement | HTMLTextAreaElement | null {
  return container.querySelector('#msg-input')
}

function autosizeAnswerInput(el: HTMLTextAreaElement): void {
  el.style.height = 'auto'
  const max = Math.min(window.innerHeight * 0.38, 320)
  el.style.height = `${Math.min(el.scrollHeight, max)}px`
}

function coachLoadingHtml(hint: string): string {
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

function messagesFromDetail(detail: SessionDetail): ChatMessage[] {
  const raw = detail.messages ?? []
  return raw.map((m) => ({
    role: m.role === 'user' ? ('user' as const) : ('assistant' as const),
    content: m.content ?? '',
  }))
}

function messagesFromBootstrap(boot: SessionBootstrap): ChatMessage[] {
  if (boot.content?.trim()) {
    return [{ role: 'assistant', content: boot.content.trim() }]
  }
  return []
}

async function loadSessionResilient(
  sessionId: string,
  isStale: () => boolean
): Promise<SessionDetail> {
  const waits = [0, 300, 700, 1200]
  let lastErr: unknown
  for (const ms of waits) {
    if (isStale()) throw new DOMException('stale', 'AbortError')
    if (ms > 0) await new Promise((r) => setTimeout(r, ms))
    try {
      return await getSession(sessionId)
    } catch (e) {
      lastErr = e
      if (e instanceof ApiError && e.message.includes('无权')) throw e
    }
  }
  throw lastErr
}

function formatLoadError(e: unknown): string {
  if (e instanceof ApiError) return e.message
  if (e instanceof DOMException && e.name === 'AbortError') return ''
  if (e instanceof Error && e.message) return e.message
  return '加载失败，请稍后重试'
}

export async function renderCoach(container: HTMLElement, sessionId: string): Promise<void> {
  clearTreeSessionOverlay()

  const gen = ++coachRenderGen
  const stale = () => gen !== coachRenderGen

  const bootstrap = peekSessionBootstrap(sessionId)
  container.innerHTML = coachLoadingHtml(
    bootstrap?.content
      ? '正在同步对话记录…'
      : '首次讲解由 AI 生成，可能需要 30–60 秒，请稍候'
  )

  const loadStartedAt = Date.now()

  let messages: ChatMessage[] = []
  let phase = bootstrap?.phase ?? 'explain'
  let nodeTitle = ''
  let domainId = bootstrap?.domainId ?? ''
  let domainName = '当前课程'
  let domainNodeTotal = 0
  let domainCompleted = 0
  let sending = false
  let hasAttemptedExercise = false
  let currentExercise: SessionExercise | null = null
  let draft: ExerciseDraft = { text: '', selectedChoices: [] }
  let initialScrollDone = false
  let preferReadableOnce = false

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

  function buildCoachUI(): void {
    const dispatch = async (text: string) => {
      const trimmed = text.trim()
      if (!trimmed || sending) return

      const prevPhase = phase
      draft = { text: '', selectedChoices: [] }
      messages.push({ role: 'user', content: trimmed })
      sending = true
      clearError()
      render()

      try {
        const reply = await sendMessage(sessionId, trimmed)
        messages.push({ role: 'assistant', content: reply.content })
        phase = reply.phase
        if (reply.exercise) {
          currentExercise = normalizeSessionExercise(reply.exercise) ?? currentExercise
        } else if (phase !== 'exercise') {
          currentExercise = null
        }
        if (prevPhase === 'exercise' || reply.content.includes(EXERCISE_MARKER)) {
          hasAttemptedExercise = true
        }
        if (reply.nodeCompleted) {
          showToast('<div class="alert alert-success">节点已点亮</div>')
          preferReadableOnce = false
        }
        if (reply.content.includes(EXERCISE_MARKER) || reply.phase === 'explain') {
          preferReadableOnce = true
        }
        if (trimmed === REAL_WORLD_CASE_PROMPT) {
          preferReadableOnce = true
        }
      } catch (err) {
        messages.pop()
        draft = { text: trimmed, selectedChoices: [] }
        showError(err instanceof ApiError ? err.message : '发送失败，请重试')
      } finally {
        sending = false
        render()
        getMsgInput(container)?.focus()
      }
    }

    const answeringNow = () => {
      const lastIdx = messages.length - 1
      const lastAssistantContent =
        messages[lastIdx]?.role === 'assistant' ? messages[lastIdx].content : ''
      return isAnsweringExercise(phase === 'exercise', lastAssistantContent)
    }

    const submitAnswer = () => {
      if (answeringNow() && currentExercise) {
        const result = collectExerciseAnswer(container, currentExercise)
        if (!result.ok) {
          showError(result.message)
          return
        }
        void dispatch(result.text)
        return
      }
      const input = getMsgInput(container)
      if (input) void dispatch(input.value)
    }

    const bindEvents = () => {
      type CoachContainer = HTMLElement & {
        __coachClickBound?: boolean
        __coachDispatch?: (t: string) => void
      }
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

        const caseBtn = target.closest<HTMLButtonElement>('.coach-inline-case')
        if (caseBtn && !caseBtn.disabled) {
          e.preventDefault()
          root.__coachDispatch?.(caseBtn.dataset.case ?? REAL_WORLD_CASE_PROMPT)
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
          submitAnswer()
          return
        }

        const formatBtn = target.closest<HTMLButtonElement>('#json-format-btn')
        if (formatBtn && !formatBtn.disabled) {
          e.preventDefault()
          if (!tryFormatJsonInTextarea(container)) {
            showError('当前内容不是合法 JSON，请检查括号与引号')
          }
        }
      })

      container.addEventListener('keydown', (e) => {
        const input = e.target as HTMLElement
        if (input.id !== 'msg-input') return
        if (!(input instanceof HTMLInputElement || input instanceof HTMLTextAreaElement)) return
        if (e.isComposing || input.dataset.composing === '1') return

        if (e.key === 'Enter') {
          if (input instanceof HTMLTextAreaElement && e.shiftKey) return
          e.preventDefault()
          submitAnswer()
        }
      })

      container.addEventListener('input', (e) => {
        const input = e.target as HTMLElement
        if (input instanceof HTMLTextAreaElement && input.id === 'msg-input') {
          autosizeAnswerInput(input)
        }
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
      draft = readExerciseDraft(container, currentExercise)

      const completed = phase === 'completed'
      const inExercise = phase === 'exercise'
      const practiceLabel = hasAttemptedExercise ? '再来一道' : '开始练习'
      const lastIdx = messages.length - 1
      const lastAssistantContent =
        messages[lastIdx]?.role === 'assistant' ? messages[lastIdx].content : ''
      const answering = isAnsweringExercise(inExercise, lastAssistantContent)

      const resolveScrollMode = (): 'readable' | 'bottom' => {
        const inputFocused =
          document.activeElement instanceof HTMLElement &&
          container.contains(document.activeElement) &&
          (document.activeElement.id === 'msg-input' ||
            document.activeElement.classList.contains('coach-choice-input'))
        const hasDraft =
          draft.text.trim().length > 0 || draft.selectedChoices.length > 0

        if (sending || answering || inExercise || phase === 'review' || inputFocused || hasDraft) {
          return 'bottom'
        }
        if (preferReadableOnce) {
          preferReadableOnce = false
          return 'readable'
        }
        if (!initialScrollDone) {
          initialScrollDone = true
          return 'readable'
        }
        return 'bottom'
      }

      const bubbles = messages
        .map((m, i) => {
          const showInlinePractice =
            !completed &&
            !sending &&
            m.role === 'assistant' &&
            i === lastIdx &&
            shouldShowPracticeCTA(m.content, inExercise)
          const inlineBtn = showInlinePractice
            ? `
            <div class="bubble-cta">
              <button type="button" class="coach-inline-case" data-case="${REAL_WORLD_CASE_PROMPT}" title="结合生产场景、代码与流程设计理解概念">
                实际案例
              </button>
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
        : answering && currentExercise
          ? exercisePlaceholder(currentExercise.answerFormat)
          : '有疑问？在这里提问'

      const quickActions =
        answering && messages[lastIdx]
          ? `
          <div class="coach-quick-actions">
            <button type="button" class="coach-quick-btn" data-quick="不懂，回讲解">不懂，回讲解</button>
            <button type="button" class="coach-quick-btn" data-quick="换一题">换一题</button>
          </div>
        `
          : ''

      const explainQuickActions =
        !completed && !answering && !sending
          ? `
          <div class="coach-quick-actions coach-quick-actions--explain">
            <button type="button" class="coach-quick-btn" data-quick="${SKIP_MASTERY_PROMPT}">已掌握，下一节</button>
          </div>
        `
          : ''

      const composer =
        answering && currentExercise
          ? renderExerciseComposer({
              exercise: currentExercise,
              placeholder,
              sending,
              quickActionsHtml: quickActions,
            })
          : answering
            ? `
        <div class="coach-composer coach-composer--exercise">
          ${quickActions}
          <div class="coach-composer-head">
            <span class="coach-composer-label">练习作答</span>
            <span class="coach-composer-hint">Enter 发送 · Shift+Enter 换行</span>
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
            : `
        ${explainQuickActions}
        <div class="chat-input-row">
          <input class="input" id="msg-input" type="text" placeholder="${escapeHtml(placeholder)}" autocomplete="off" ${sending ? 'disabled' : ''} aria-label="消息输入" />
          <button type="button" class="btn btn-ghost" id="send-btn" ${sending ? 'disabled' : ''}>${sending ? '…' : '发送'}</button>
        </div>
      `

      const footer = completed
        ? `
        <div class="coach-completed-actions">
          <a class="btn btn-primary" href="#/tree/${domainId}">返回课程</a>
        </div>
      `
        : composer

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
      if (msgBox) {
        scrollChatMessages(msgBox, resolveScrollMode())
      }

      if (!completed && !sending) {
        restoreExerciseDraft(container, draft, currentExercise)
        const input = getMsgInput(container)
        if (input) {
          if (input instanceof HTMLTextAreaElement) autosizeAnswerInput(input)
          input.focus()
        }
      }
    }

    bindEvents()
    render()
  }

  try {
    if (bootstrap?.content && !bootstrap.resumed) {
      messages = messagesFromBootstrap(bootstrap)
      phase = bootstrap.phase
      domainId = bootstrap.domainId
      nodeTitle = bootstrap.nodeKey
      hasAttemptedExercise = phase === 'review' || phase === 'completed'
    }

    const detailPromise = loadSessionResilient(sessionId, stale)

    const domainMetaPromise = domainId
      ? Promise.all([
          getDomainTree(domainId).catch(() => null),
          getUserProgress(domainId).catch(() => []),
        ])
      : Promise.resolve([null, []] as const)

    if (messages.length > 0 && !stale()) {
      const [tree] = await domainMetaPromise
      if (tree) {
        domainName = tree.domainName ?? domainName
        domainNodeTotal = tree.layers?.reduce((n, l) => n + (l.nodes?.length ?? 0), 0) ?? 0
        nodeTitle =
          tree.layers.flatMap((l) => l.nodes).find((n) => n.key === bootstrap?.nodeKey)?.title ??
          nodeTitle
      }
      await updateSidebar({
        active: 'coach',
        domainId,
        domainName,
        domainNodeTotal,
        domainCompleted: 0,
        nodeTitle,
      })
      setBreadcrumb([
        { label: '开始学习', href: '#/' },
        { label: '我的课程', href: '#/courses' },
        { label: domainName, href: `#/tree/${domainId}` },
        { label: nodeTitle },
      ])
      buildCoachUI()
    }

    const [[tree, progress], detail] = await Promise.all([domainMetaPromise, detailPromise])
    if (stale()) return

    clearSessionBootstrap(sessionId)
    phase = detail.phase ?? phase
    nodeTitle = detail.nodeTitle || nodeTitle || '学习节点'
    domainId = detail.domainId || domainId
    domainName = tree?.domainName ?? domainName
    domainNodeTotal = tree?.layers?.reduce((n, l) => n + (l.nodes?.length ?? 0), 0) ?? 0
    domainCompleted = progress.filter((p) => p.status === 'completed').length
    messages = messagesFromDetail(detail)
    currentExercise = normalizeSessionExercise(detail.exercise)
    hasAttemptedExercise =
      phase === 'review' || phase === 'completed' || hasExerciseInHistory(messages)
  } catch (e) {
    if (stale()) return
    const msg = formatLoadError(e)
    if (!msg) return

    const minMs = 500
    const elapsed = Date.now() - loadStartedAt
    if (elapsed < minMs) await new Promise((r) => setTimeout(r, minMs - elapsed))
    if (stale()) return

    if (bootstrap?.content && messages.length > 0) {
      clearSessionBootstrap(sessionId)
      await updateSidebar({
        active: 'coach',
        domainId,
        domainName,
        domainNodeTotal,
        domainCompleted,
        nodeTitle: nodeTitle || bootstrap.nodeKey,
      })
      setBreadcrumb([
        { label: '开始学习', href: '#/' },
        { label: '我的课程', href: '#/courses' },
        { label: domainName, href: domainId ? `#/tree/${domainId}` : undefined },
        { label: nodeTitle || '教练对话' },
      ])
      buildCoachUI()
      showError(`${msg}（已显示本地缓存的讲解，发送消息将自动重试同步）`)
      return
    }

    container.innerHTML = `
      <section class="page page-coach">
        <div class="alert alert-error">${escapeHtml(msg)}</div>
        <p class="page-loading-hint" style="margin-top:1rem;text-align:center">
          <button type="button" class="btn btn-secondary btn-sm" id="coach-retry-btn">重试</button>
          ${domainId ? `<a class="btn btn-ghost btn-sm" href="#/tree/${domainId}" style="margin-left:0.5rem">返回课程</a>` : ''}
        </p>
      </section>
    `
    container.querySelector<HTMLButtonElement>('#coach-retry-btn')?.addEventListener('click', () => {
      void renderCoach(container, sessionId)
    })
    return
  } finally {
    if (!stale()) {
      setAppBusy(false)
      refreshLLMStatusAfterBusy()
    }
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
    { label: '我的课程', href: '#/courses' },
    { label: domainName, href: `#/tree/${domainId}` },
    { label: nodeTitle },
  ])

  buildCoachUI()
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
