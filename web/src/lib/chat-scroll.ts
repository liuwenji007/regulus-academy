/**
 * 对话区滚动：内容不足一屏时从顶部读；超过一屏时定位到最后一条消息的开头。
 * coach 页 render 会重建 DOM 并在同一次任务内恢复草稿、autosize 输入框等；单次 rAF
 * 在这些同步更新全部完成、首次绘制前执行，避免同步测量 stale rect 与重复 layout。
 */
export type ChatScrollMode = 'readable' | 'bottom'

function scrollToElementStart(msgBox: HTMLElement, target: HTMLElement): void {
  const boxRect = msgBox.getBoundingClientRect()
  const targetRect = target.getBoundingClientRect()
  const next = msgBox.scrollTop + (targetRect.top - boxRect.top) - 8
  msgBox.scrollTop = Math.max(0, next)
}

export function scrollChatMessages(
  msgBox: HTMLElement,
  mode: ChatScrollMode = 'readable'
): void {
  requestAnimationFrame(() => {
    if (mode === 'bottom') {
      msgBox.scrollTop = msgBox.scrollHeight
      return
    }

    const fitsOneScreen = msgBox.scrollHeight <= msgBox.clientHeight + 4
    if (fitsOneScreen) {
      msgBox.scrollTop = 0
      return
    }

    const assistants = msgBox.querySelectorAll<HTMLElement>('.bubble.assistant')
    const target =
      assistants.length > 0
        ? assistants[assistants.length - 1]!
        : msgBox.querySelector<HTMLElement>('.bubble:last-child')

    if (!target) {
      msgBox.scrollTop = 0
      return
    }

    scrollToElementStart(msgBox, target)
  })
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
