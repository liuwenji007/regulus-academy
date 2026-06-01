/**
 * 对话区滚动：内容不足一屏时从顶部读；超过一屏时定位到最后一条消息的开头，
 * 避免长讲解直接沉底。
 */
export function scrollChatToReadablePosition(
  msgBox: HTMLElement,
  opts?: { smooth?: boolean }
): void {
  const behavior = opts?.smooth ? 'smooth' : 'auto'

  const apply = () => {
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

    target.scrollIntoView({ block: 'start', inline: 'nearest', behavior })
  }

  requestAnimationFrame(() => requestAnimationFrame(apply))
}
