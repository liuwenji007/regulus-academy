/**
 * 对话区滚动：readable = 将最后一条助手消息滚到可视区开头；bottom = 滚到底部。
 * coach 在「新助手回复」或「打开会话且末条为助手」时触发 readable。
 */
export type ChatScrollMode = 'readable' | 'bottom'

function scrollToElementStart(msgBox: HTMLElement, target: HTMLElement): void {
  const boxRect = msgBox.getBoundingClientRect()
  const targetRect = target.getBoundingClientRect()
  const next = msgBox.scrollTop + (targetRect.top - boxRect.top) - 8
  msgBox.scrollTop = Math.max(0, next)
}

function applyReadableScroll(msgBox: HTMLElement): void {
  const fitsOneScreen = msgBox.scrollHeight <= msgBox.clientHeight + 4
  if (fitsOneScreen) {
    msgBox.scrollTop = 0
    return
  }

  const assistants = msgBox.querySelectorAll<HTMLElement>('.bubble.assistant')
  const target = assistants[assistants.length - 1]

  if (!target) {
    msgBox.scrollTop = 0
    return
  }

  scrollToElementStart(msgBox, target)
}

export function scrollChatMessages(
  msgBox: HTMLElement,
  mode: ChatScrollMode = 'readable'
): void {
  const run = () => {
    if (mode === 'bottom') {
      msgBox.scrollTop = msgBox.scrollHeight
      return
    }
    applyReadableScroll(msgBox)
  }

  if (mode === 'readable') {
    // 同步先顶到开头，避免在双 rAF 之前浏览器把长内容锚到底部
    applyReadableScroll(msgBox)
    requestAnimationFrame(() => {
      requestAnimationFrame(run)
      // Markdown 渲染 / 全屏 overlay 收起后高度可能再变，补一次延迟校正
      window.setTimeout(run, 0)
      window.setTimeout(run, 80)
    })
    return
  }
  requestAnimationFrame(run)
}

/** @deprecated 使用 scrollChatMessages */
export function scrollChatToReadablePosition(
  msgBox: HTMLElement,
  _opts?: { smooth?: boolean }
): void {
  scrollChatMessages(msgBox, 'readable')
}

export function scrollChatToBottom(msgBox: HTMLElement): void {
  scrollChatMessages(msgBox, 'bottom')
}
