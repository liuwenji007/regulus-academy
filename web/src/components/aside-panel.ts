import {
  asideAskStream,
  asideExplain,
  listAsideMessages,
  listAsideTerms,
  listKnowledgeGaps,
  resolveKnowledgeGap,
  type AsideIntent,
  type AsideMessageItem,
  type AsideTermItem,
  type KnowledgeGapItem,
} from '../lib/api'
import { renderMarkdown } from '../lib/markdown'
import { iconPanelRight, iconSparkles, iconX } from '../lib/icons'

export type AsideLessonContext = {
  domainId?: string
  nodeKey?: string
  coachSessionId?: string
  domainName?: string
  nodeTitle?: string
}

type AsideTab = 'chat' | 'terms' | 'gaps'

type ChatBubble = {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  streaming?: boolean
}

let rootEl: HTMLElement | null = null
let open = false
let activeTab: AsideTab = 'chat'
let lessonCtx: AsideLessonContext = {}
let bubbles: ChatBubble[] = []
let terms: AsideTermItem[] = []
let gaps: KnowledgeGapItem[] = []
let busy = false
let bubbleSeq = 0

function esc(s: string): string {
  const d = document.createElement('div')
  d.textContent = s
  return d.innerHTML
}

/** 挂载到 app-shell；只调用一次，跨路由存活 */
export function mountAsidePanel(shell: HTMLElement): void {
  if (rootEl) return

  let slot = shell.querySelector<HTMLElement>('#aside-slot')
  if (!slot) {
    slot = document.createElement('div')
    slot.id = 'aside-slot'
    shell.appendChild(slot)
  }

  slot.innerHTML = `
    <aside class="aside-panel" id="aside-panel" aria-label="学习旁路助手" hidden>
      <header class="aside-panel__header">
        <div class="aside-panel__title">
          <span class="aside-panel__icon">${iconSparkles()}</span>
          <div>
            <strong>旁路助手</strong>
            <p class="aside-panel__subtitle" id="aside-subtitle">划词提问 · 术语本 · 知识缺口</p>
          </div>
        </div>
        <button type="button" class="aside-panel__close" id="aside-close" aria-label="关闭">${iconX()}</button>
      </header>
      <nav class="aside-panel__tabs" role="tablist">
        <button type="button" class="aside-tab is-active" data-aside-tab="chat" role="tab">对话</button>
        <button type="button" class="aside-tab" data-aside-tab="terms" role="tab">术语本</button>
        <button type="button" class="aside-tab" data-aside-tab="gaps" role="tab">知识缺口</button>
      </nav>
      <div class="aside-panel__body" id="aside-body"></div>
      <footer class="aside-panel__footer" id="aside-footer"></footer>
    </aside>
    <div class="aside-backdrop" id="aside-backdrop" hidden></div>
  `

  rootEl = slot
  bindEvents(slot)
  paint()
}

function bindEvents(slot: HTMLElement): void {
  slot.addEventListener('click', (e) => {
    const t = e.target as HTMLElement
    if (t.closest('#aside-close') || t.closest('#aside-backdrop')) {
      setAsideOpen(false)
      return
    }
    const tabBtn = t.closest<HTMLElement>('[data-aside-tab]')
    if (tabBtn?.dataset.asideTab) {
      activeTab = tabBtn.dataset.asideTab as AsideTab
      void refreshTabData()
      paint()
      return
    }
    const resolveBtn = t.closest<HTMLElement>('[data-gap-resolve]')
    if (resolveBtn) {
      const id = Number(resolveBtn.dataset.gapResolve)
      if (id > 0) void onResolveGap(id)
      return
    }
    const termBtn = t.closest<HTMLElement>('[data-term-show]')
    if (termBtn) {
      const id = Number(termBtn.dataset.termShow)
      const item = terms.find((x) => x.id === id)
      if (item?.card) {
        activeTab = 'chat'
        appendBubble('assistant', formatCardLocal(item))
        paint()
      }
      return
    }
  })

  slot.addEventListener('keydown', (e) => {
    if (e.key !== 'Enter' || e.shiftKey) return
    const input = e.target as HTMLElement
    if (input instanceof HTMLTextAreaElement && input.id === 'aside-input') {
      if (input.dataset.composing === '1') return
      e.preventDefault()
      void submitAsk()
    }
  })

  slot.addEventListener('compositionstart', (e) => {
    const input = e.target as HTMLElement
    if (input.id === 'aside-input') input.dataset.composing = '1'
  })
  slot.addEventListener('compositionend', (e) => {
    const input = e.target as HTMLElement
    if (input.id === 'aside-input') delete input.dataset.composing
  })

  slot.addEventListener('click', (e) => {
    const send = (e.target as HTMLElement).closest('#aside-send')
    if (send) void submitAsk()
  })
}

function formatCardLocal(item: AsideTermItem): string {
  const c = item.card
  if (!c) return `**${item.term || item.normalizedTerm}**`
  const parts = [`### ${c.term || item.term}`, c.oneLiner ? `**${c.oneLiner}**` : '']
  if (c.ipa || c.readingCn) {
    parts.push(`**读音** ${c.ipa || ''} ${c.readingCn ? `（${c.readingCn}）` : ''}`)
  }
  if (c.explanation) parts.push(c.explanation)
  if (c.analogy) parts.push(`> 类比：${c.analogy}`)
  if (c.redirectHint) parts.push(`—\n*${c.redirectHint}*`)
  return parts.filter(Boolean).join('\n\n')
}

export function setAsideOpen(next: boolean): void {
  open = next
  const shell = document.getElementById('app-shell')
  shell?.classList.toggle('aside-open', open)
  const panel = rootEl?.querySelector<HTMLElement>('#aside-panel')
  const backdrop = rootEl?.querySelector<HTMLElement>('#aside-backdrop')
  if (panel) {
    if (open) panel.removeAttribute('hidden')
    else panel.setAttribute('hidden', '')
  }
  if (backdrop) {
    if (open && window.matchMedia('(max-width: 900px)').matches) backdrop.removeAttribute('hidden')
    else backdrop.setAttribute('hidden', '')
  }
  if (open) void refreshTabData()
}

export function toggleAsidePanel(): void {
  setAsideOpen(!open)
}

export function isAsideOpen(): boolean {
  return open
}

/** 同步主线课上下文（coach 页更新时调用） */
export function setAsideLessonContext(ctx: AsideLessonContext): void {
  lessonCtx = { ...ctx }
  updateSubtitle()
}

function updateSubtitle(): void {
  const el = rootEl?.querySelector('#aside-subtitle')
  if (!el) return
  const parts: string[] = []
  if (lessonCtx.domainName) parts.push(lessonCtx.domainName)
  if (lessonCtx.nodeTitle) parts.push(lessonCtx.nodeTitle)
  el.textContent = parts.length ? parts.join(' · ') : '划词提问 · 术语本 · 知识缺口'
}

function appendBubble(role: ChatBubble['role'], content: string, streaming = false): string {
  const id = `b-${++bubbleSeq}`
  bubbles.push({ id, role, content, streaming })
  if (bubbles.length > 80) bubbles = bubbles.slice(-60)
  return id
}

function updateBubble(id: string, content: string, streaming = false): void {
  const b = bubbles.find((x) => x.id === id)
  if (!b) return
  b.content = content
  b.streaming = streaming
}

/** 外部入口：划词解释 */
export async function asideExplainSelection(
  anchorText: string,
  intent: AsideIntent = 'what'
): Promise<void> {
  const text = anchorText.trim()
  if (!text || busy) return
  setAsideOpen(true)
  activeTab = 'chat'
  const label =
    intent === 'reading' ? '怎么读' : intent === 'expand' ? '展开讲' : '这是什么'
  appendBubble('user', `[${label}] ${text}`)
  const pendingId = appendBubble('assistant', '正在查询…', true)
  paint()
  busy = true
  try {
    const res = await asideExplain({
      domainId: lessonCtx.domainId,
      nodeKey: lessonCtx.nodeKey,
      coachSessionId: lessonCtx.coachSessionId,
      anchorText: text,
      intent,
    })
    updateBubble(pendingId, res.markdown || '（无内容）', false)
    void refreshTermsQuiet()
    void refreshGapsQuiet()
  } catch (err) {
    updateBubble(pendingId, `查询失败：${err instanceof Error ? err.message : String(err)}`, false)
  } finally {
    busy = false
    paint()
  }
}

async function submitAsk(): Promise<void> {
  const input = rootEl?.querySelector<HTMLTextAreaElement>('#aside-input')
  if (!input || busy) return
  const q = input.value.trim()
  if (!q) return
  input.value = ''
  input.style.height = 'auto'
  activeTab = 'chat'
  appendBubble('user', q)
  const pendingId = appendBubble('assistant', '', true)
  paint()
  busy = true
  let acc = ''
  try {
    await asideAskStream(
      {
        domainId: lessonCtx.domainId,
        nodeKey: lessonCtx.nodeKey,
        coachSessionId: lessonCtx.coachSessionId,
        question: q,
      },
      {
        onDelta: (t) => {
          acc += t
          updateBubble(pendingId, acc, true)
          paintMessagesOnly()
        },
      }
    )
    if (acc) updateBubble(pendingId, acc, false)
    void refreshGapsQuiet()
  } catch (err) {
    updateBubble(pendingId, `回复失败：${err instanceof Error ? err.message : String(err)}`, false)
  } finally {
    busy = false
    paint()
  }
}

async function refreshTabData(): Promise<void> {
  if (activeTab === 'terms') await refreshTermsQuiet()
  if (activeTab === 'gaps') await refreshGapsQuiet()
  if (activeTab === 'chat' && bubbles.length === 0 && lessonCtx.domainId) {
    try {
      const msgs = await listAsideMessages(lessonCtx.domainId)
      hydrateFromMessages(msgs)
    } catch {
      /* ignore */
    }
  }
  paint()
}

function hydrateFromMessages(msgs: AsideMessageItem[]): void {
  if (!msgs.length) return
  bubbles = msgs.slice(-40).map((m) => ({
    id: `hist-${m.id}`,
    role: m.role === 'user' ? 'user' : 'assistant',
    content: m.content,
  }))
}

async function refreshTermsQuiet(): Promise<void> {
  try {
    terms = await listAsideTerms(lessonCtx.domainId)
  } catch {
    terms = []
  }
}

async function refreshGapsQuiet(): Promise<void> {
  try {
    gaps = await listKnowledgeGaps(lessonCtx.domainId)
  } catch {
    gaps = []
  }
}

async function onResolveGap(id: number): Promise<void> {
  try {
    await resolveKnowledgeGap(id)
    gaps = gaps.filter((g) => g.id !== id)
    paint()
  } catch {
    /* ignore */
  }
}

function paint(): void {
  if (!rootEl) return
  rootEl.querySelectorAll('.aside-tab').forEach((el) => {
    el.classList.toggle('is-active', (el as HTMLElement).dataset.asideTab === activeTab)
  })
  const body = rootEl.querySelector('#aside-body')
  const footer = rootEl.querySelector('#aside-footer')
  if (!body || !footer) return

  if (activeTab === 'chat') {
    body.innerHTML = renderChatBody()
    footer.innerHTML = renderChatFooter()
    scrollChatToBottom()
  } else if (activeTab === 'terms') {
    body.innerHTML = renderTermsBody()
    footer.innerHTML = ''
  } else {
    body.innerHTML = renderGapsBody()
    footer.innerHTML = ''
  }
  updateSubtitle()
}

function paintMessagesOnly(): void {
  const list = rootEl?.querySelector('#aside-chat-list')
  if (!list) {
    paint()
    return
  }
  list.innerHTML = bubbles.map(renderBubble).join('')
  scrollChatToBottom()
}

function scrollChatToBottom(): void {
  const list = rootEl?.querySelector('#aside-chat-list')
  if (list) list.scrollTop = list.scrollHeight
}

function renderBubble(b: ChatBubble): string {
  const cls = `aside-bubble aside-bubble--${b.role}${b.streaming ? ' is-streaming' : ''}`
  const html =
    b.role === 'assistant' ? `<div class="md-body">${renderMarkdown(b.content || '…')}</div>` : esc(b.content)
  return `<div class="${cls}" data-bubble-id="${b.id}">${html}</div>`
}

function renderChatBody(): string {
  if (!bubbles.length) {
    return `<div class="aside-empty">
      <p>在左侧讲解里<strong>划选关键词</strong>，选「这是什么 / 怎么读 / 展开讲」；</p>
      <p>或在下方直接提问。旁路不会打断主线进度。</p>
    </div>
    <div class="aside-chat-list" id="aside-chat-list"></div>`
  }
  return `<div class="aside-chat-list" id="aside-chat-list">${bubbles.map(renderBubble).join('')}</div>`
}

function renderChatFooter(): string {
  return `
    <div class="aside-composer">
      <textarea id="aside-input" class="aside-input" rows="1" placeholder="随便问问（不会推进课程）…" ${busy ? 'disabled' : ''}></textarea>
      <button type="button" class="btn btn-primary aside-send" id="aside-send" ${busy ? 'disabled' : ''}>发送</button>
    </div>`
}

function renderTermsBody(): string {
  if (!terms.length) {
    return `<div class="aside-empty"><p>还没有查过的术语。划词查询后会出现在这里。</p></div>`
  }
  return `<ul class="aside-term-list">${terms
    .map(
      (t) => `
    <li>
      <button type="button" class="aside-term-item" data-term-show="${t.id}">
        <span class="aside-term-name">${esc(t.term || t.normalizedTerm)}</span>
        <span class="aside-term-meta">${esc(t.oneLiner || t.originalText || '')} · ${t.hitCount} 次</span>
      </button>
    </li>`
    )
    .join('')}</ul>`
}

function renderGapsBody(): string {
  if (!gaps.length) {
    return `<div class="aside-empty"><p>暂无知识缺口记录。划词、答错或跳级时会自动沉淀到这里，供你核对是否准确。</p></div>`
  }
  return `<ul class="aside-gap-list">${gaps
    .map((g) => {
      const src =
        g.source === 'mistake' ? '错题' : g.source === 'coach_gap' ? '掌握检测' : '旁路查询'
      return `
      <li class="aside-gap-item">
        <div class="aside-gap-main">
          <strong>${esc(g.concept)}</strong>
          <span class="aside-gap-meta">${src} · ${g.hitCount} 次${g.reason ? ` · ${esc(g.reason)}` : ''}</span>
        </div>
        <button type="button" class="btn btn-ghost aside-gap-resolve" data-gap-resolve="${g.id}">已懂</button>
      </li>`
    })
    .join('')}</ul>`
}

/** 头部切换按钮 HTML */
export function asideToggleButtonHtml(): string {
  return `<button type="button" class="header-aside-btn" id="header-aside-btn" aria-label="旁路助手" title="旁路助手（划词解释）">${iconPanelRight()}</button>`
}

export function bindAsideToggle(shell: HTMLElement): void {
  shell.addEventListener('click', (e) => {
    if ((e.target as HTMLElement).closest('#header-aside-btn')) {
      toggleAsidePanel()
    }
  })
  document.addEventListener('keydown', (e) => {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA') return
      e.preventDefault()
      toggleAsidePanel()
    }
  })
}