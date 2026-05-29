import { getSession, sendMessage, phaseLabel, ApiError } from '../lib/api'
import { renderMarkdown } from '../lib/markdown'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export async function renderCoach(container: HTMLElement, sessionId: string): Promise<void> {
  container.innerHTML = `<div class="page"><p class="page-sub">加载对话…</p></div>`

  let messages: ChatMessage[] = []
  let phase = 'explain'
  let nodeTitle = ''

  try {
    const detail = await getSession(sessionId)
    phase = detail.phase
    nodeTitle = detail.nodeTitle
    messages = detail.messages.map((m) => ({
      role: m.role === 'user' ? 'user' as const : 'assistant' as const,
      content: m.content,
    }))
  } catch (e) {
    container.innerHTML = `<div class="page"><div class="error-banner">${e instanceof ApiError ? e.message : '加载失败'}</div></div>`
    return
  }

  const render = () => {
    const bubbles = messages
      .map((m) => `<div class="bubble ${m.role}">${formatBubbleContent(m)}</div>`)
      .join('')

    const placeholder =
      phase === 'exercise' ? '写下你的答案…' : '提问，或回复「开始练习」'

    container.innerHTML = `
      <div class="page coach-page">
        <a href="javascript:history.back()" class="back-link">← 返回知识树</a>
        <h1 class="page-title">${escapeHtml(nodeTitle)}</h1>
        <span class="phase-badge">${phaseLabel(phase)}</span>
        <div class="chat-messages" id="messages">${bubbles}</div>
        <div id="coach-error"></div>
        <div id="coach-toast"></div>
        <div class="chat-input-row">
          <input class="input" id="msg-input" type="text" placeholder="${placeholder}" autocomplete="off" />
          <button class="btn" id="send-btn">发送</button>
        </div>
      </div>
    `

    const msgBox = container.querySelector<HTMLDivElement>('#messages')!
    msgBox.scrollTop = msgBox.scrollHeight

    const input = container.querySelector<HTMLInputElement>('#msg-input')!
    const btn = container.querySelector<HTMLButtonElement>('#send-btn')!
    const errEl = container.querySelector<HTMLDivElement>('#coach-error')!

    const send = async () => {
      const text = input.value.trim()
      if (!text) return
      messages.push({ role: 'user', content: text })
      input.value = ''
      btn.disabled = true
      errEl.innerHTML = ''
      render()

      try {
        const reply = await sendMessage(sessionId, text)
        messages.push({ role: 'assistant', content: reply.content })
        phase = reply.phase
        if (reply.nodeCompleted) {
          const toast = container.querySelector<HTMLDivElement>('#coach-toast')!
          toast.innerHTML = '<div class="toast-success">节点已点亮</div>'
        }
      } catch (e) {
        errEl.innerHTML = `<div class="error-banner">${e instanceof ApiError ? e.message : '发送失败'}</div>`
      } finally {
        btn.disabled = false
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
