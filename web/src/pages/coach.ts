import { getSession, getDomainTree, sendMessage, phaseLabel, ApiError } from '../lib/api'
import { renderMarkdown } from '../lib/markdown'
import { setBreadcrumb, updateSidebar } from '../components/layout'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
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
  let sending = false

  try {
    const detail = await getSession(sessionId)
    phase = detail.phase
    nodeTitle = detail.nodeTitle
    domainId = detail.domainId
    const tree = await getDomainTree(domainId).catch(() => null)
    domainName = tree?.domainName ?? '当前课程'
    messages = detail.messages.map((m) => ({
      role: m.role === 'user' ? ('user' as const) : ('assistant' as const),
      content: m.content,
    }))
  } catch (e) {
    container.innerHTML = `
      <section class="page page-coach">
        <div class="alert alert-error">${e instanceof ApiError ? e.message : '加载失败'}</div>
      </section>
    `
    return
  }

  updateSidebar({
    active: 'coach',
    domainId,
    domainName,
    nodeTitle,
  })
  setBreadcrumb([
    { label: '开始学习', href: '#/' },
    { label: '知识树', href: `#/tree/${domainId}` },
    { label: nodeTitle },
  ])

  const render = () => {
    const bubbles = messages
      .map((m) => `<div class="bubble ${m.role}">${formatBubbleContent(m)}</div>`)
      .join('')

    const completed = phase === 'completed'
    const placeholder = completed
      ? '本节点已完成'
      : phase === 'exercise'
        ? '写下你的答案，或说「不懂」「换一题」'
        : '提问，或回复「开始练习」'

    const inputRow = completed
      ? `
        <div class="coach-completed-actions">
          <a class="btn btn-primary" href="#/tree/${domainId}">返回知识树</a>
        </div>
      `
      : `
        <div class="chat-input-row">
          <input class="input" id="msg-input" type="text" placeholder="${placeholder}" autocomplete="off" ${sending ? 'disabled' : ''} aria-label="消息输入" />
          <button class="btn btn-primary" id="send-btn" ${sending ? 'disabled' : ''}>${sending ? '…' : '发送'}</button>
        </div>
      `

    container.innerHTML = `
      <section class="page page-coach">
        <header class="page-header page-header-compact">
          <h1 class="page-title">${escapeHtml(nodeTitle)}</h1>
          <span class="phase-badge">${phaseLabel(phase)}</span>
        </header>

        <div class="chat-panel card">
          <div class="chat-messages" id="messages" role="log" aria-live="polite">${bubbles}${sending ? '<div class="coach-loading">教练思考中…</div>' : ''}</div>
          <div id="coach-error"></div>
          <div id="coach-toast"></div>
          ${inputRow}
        </div>
      </section>
    `

    const msgBox = container.querySelector<HTMLDivElement>('#messages')!
    msgBox.scrollTop = msgBox.scrollHeight

    if (completed) return

    const input = container.querySelector<HTMLInputElement>('#msg-input')!
    const btn = container.querySelector<HTMLButtonElement>('#send-btn')!
    const errEl = container.querySelector<HTMLDivElement>('#coach-error')!

    const send = async () => {
      const text = input.value.trim()
      if (!text || sending) return
      messages.push({ role: 'user', content: text })
      input.value = ''
      sending = true
      errEl.innerHTML = ''
      render()

      try {
        const reply = await sendMessage(sessionId, text)
        messages.push({ role: 'assistant', content: reply.content })
        phase = reply.phase
        if (reply.nodeCompleted) {
          const toast = container.querySelector<HTMLDivElement>('#coach-toast')!
          toast.innerHTML = '<div class="alert alert-success">节点已点亮</div>'
        }
      } catch (e) {
        messages.pop()
        errEl.innerHTML = `<div class="alert alert-error">${e instanceof ApiError ? e.message : '发送失败'}</div>`
      } finally {
        sending = false
        render()
        container.querySelector<HTMLInputElement>('#msg-input')?.focus()
      }
    }

    btn.addEventListener('click', send)
    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') send()
    })
    input.focus()
  }

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
