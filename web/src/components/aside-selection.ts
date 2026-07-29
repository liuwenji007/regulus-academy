import { asideExplainSelection } from '../components/aside-panel'
import type { AsideIntent } from '../lib/api'

let bubbleEl: HTMLDivElement | null = null
let boundRoot: HTMLElement | null = null
let hideTimer: number | null = null

function ensureBubble(): HTMLDivElement {
  if (bubbleEl) return bubbleEl
  bubbleEl = document.createElement('div')
  bubbleEl.className = 'aside-selection-bubble'
  bubbleEl.hidden = true
  bubbleEl.innerHTML = `
    <button type="button" data-intent="what">这是什么</button>
    <button type="button" data-intent="reading">怎么读</button>
    <button type="button" data-intent="expand">展开讲</button>
  `
  document.body.appendChild(bubbleEl)

  bubbleEl.addEventListener('mousedown', (e) => {
    // 防止点击按钮时清掉 selection
    e.preventDefault()
  })
  bubbleEl.addEventListener('click', (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLElement>('[data-intent]')
    if (!btn?.dataset.intent) return
    const text = bubbleEl?.dataset.selection?.trim() || ''
    hideBubble()
    if (!text) return
    void asideExplainSelection(text, btn.dataset.intent as AsideIntent)
  })
  return bubbleEl
}

function hideBubble(): void {
  if (hideTimer) {
    window.clearTimeout(hideTimer)
    hideTimer = null
  }
  const el = ensureBubble()
  el.hidden = true
  delete el.dataset.selection
}

function showBubbleForSelection(): void {
  const sel = window.getSelection()
  if (!sel || sel.isCollapsed || !sel.rangeCount) {
    hideBubble()
    return
  }
  const text = sel.toString().trim()
  if (!text || text.length > 80 || text.length < 1) {
    hideBubble()
    return
  }
  // 必须在教练消息 Markdown 内
  const anchor = sel.anchorNode
  const node = anchor instanceof Element ? anchor : anchor?.parentElement
  if (!node || !boundRoot?.contains(node)) {
    hideBubble()
    return
  }
  if (!node.closest('.md-body, .bubble.assistant')) {
    hideBubble()
    return
  }

  const range = sel.getRangeAt(0)
  const rect = range.getBoundingClientRect()
  if (!rect.width && !rect.height) {
    hideBubble()
    return
  }

  const el = ensureBubble()
  el.dataset.selection = text
  el.hidden = false
  const top = rect.top + window.scrollY - el.offsetHeight - 8
  let left = rect.left + window.scrollX + rect.width / 2 - el.offsetWidth / 2
  left = Math.max(8, Math.min(left, window.innerWidth - el.offsetWidth - 8))
  el.style.top = `${Math.max(8, top)}px`
  el.style.left = `${left}px`
}

/** 在教练页容器上绑定划词气泡（可重复调用，幂等） */
export function bindCoachSelectionBubble(container: HTMLElement): void {
  boundRoot = container
  ensureBubble()

  const onUp = () => {
    if (hideTimer) window.clearTimeout(hideTimer)
    hideTimer = window.setTimeout(() => showBubbleForSelection(), 10)
  }
  const onDown = (e: MouseEvent) => {
    if ((e.target as HTMLElement).closest('.aside-selection-bubble')) return
    hideBubble()
  }

  // 用属性标记避免重复绑定
  const marked = container as HTMLElement & { __asideSelBound?: boolean }
  if (marked.__asideSelBound) return
  marked.__asideSelBound = true
  container.addEventListener('mouseup', onUp)
  document.addEventListener('mousedown', onDown)
  document.addEventListener('scroll', hideBubble, true)
}

export function unbindCoachSelectionBubble(): void {
  hideBubble()
  boundRoot = null
}
