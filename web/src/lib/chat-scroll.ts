/**
 * 对话区滚动：readable = 将最后一条助手消息滚到可视区开头；bottom = 滚到底部。
 * coach 在「新助手回复」或「打开会话且末条为助手」时触发 readable。
 * 流式输出：仅在用户贴近底部时跟随；上滑后暂停，回到底部附近再恢复。
 */
export type ChatScrollMode = 'readable' | 'bottom'

/** 距底部多少像素内视为「仍在跟随」 */
const NEAR_BOTTOM_PX = 96

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

export function isChatNearBottom(msgBox: HTMLElement, threshold = NEAR_BOTTOM_PX): boolean {
  return msgBox.scrollHeight - msgBox.scrollTop - msgBox.clientHeight <= threshold
}

/**
 * 流式跟随状态放在模块级：行动助手整页 innerHTML 会换掉 #messages，
 * WeakMap 绑在旧节点上会丢；同一时刻 SPA 只有一路对话在流式。
 */
let streamFollow = true
let streamIgnoreScroll = false
let streamBoundBox: HTMLElement | null = null
/** 上滑暂停时记住的 scrollTop，供 DOM 重建后恢复 */
let streamPausedScrollTop = 0

function onStreamScroll(this: HTMLElement): void {
  if (streamIgnoreScroll) return
  const near = isChatNearBottom(this)
  streamFollow = near
  if (!near) {
    streamPausedScrollTop = this.scrollTop
  }
}

function bindStreamScroll(msgBox: HTMLElement): void {
  if (streamBoundBox === msgBox) return
  if (streamBoundBox) {
    streamBoundBox.removeEventListener('scroll', onStreamScroll)
  }
  streamBoundBox = msgBox
  msgBox.addEventListener('scroll', onStreamScroll, { passive: true })
}

/**
 * 流式结束后首帧保护标记：终态整页 innerHTML 重建时，若仍走 readable，会把
 * 最后一条助手消息锚到视口顶（短会话甚至 scrollTop=0）。mark 后 renderCoachView
 * 改走 scrollChatDuringStream 保持跟读位置，消费后自动清除。
 */
let streamJustEnded = false

/** 流式输出结束时调用，避免紧随其后的 renderCoachView 做 readable 顶部锚定 */
export function markStreamJustEnded(): void {
  streamJustEnded = true
}

/** 新一轮流式开始时重置为跟随 */
export function resetChatStreamFollow(msgBox?: HTMLElement): void {
  streamFollow = true
  streamPausedScrollTop = 0
  streamJustEnded = false
  if (msgBox) bindStreamScroll(msgBox)
}

/**
 * 流式期间用户是否主动上滑过（true = 没动过 / 已回到底部，false = 上滑暂停中）。
 * 供流式结束后的 readable 锚定判断：用户上滑过则跳过锚定，保留阅读位置。
 */
export function getChatStreamFollow(): boolean {
  return streamFollow
}

/** 当前是否处于「流式刚结束」保护期 */
export function isChatStreamJustEnded(): boolean {
  return streamJustEnded
}

/** 消费「流式刚结束」标记（消费后清除） */
export function consumeStreamJustEnded(): boolean {
  const v = streamJustEnded
  streamJustEnded = false
  return v
}

/** 整页重绘前快照：用于行动助手一类会销毁 #messages 的路径 */
export function snapshotChatStreamScroll(msgBox: HTMLElement | null): void {
  if (!msgBox) return
  bindStreamScroll(msgBox)
  if (!isChatNearBottom(msgBox)) {
    streamFollow = false
    streamPausedScrollTop = msgBox.scrollTop
  }
}

/**
 * 流式增量滚动：用户未上滑时滚到底；已上滑则保留/恢复阅读位置。
 * 直接写 scrollTop，避免 readable 的多重 rAF/timeout 在 ~80ms patch 下叠跑。
 */
export function scrollChatDuringStream(msgBox: HTMLElement): void {
  bindStreamScroll(msgBox)
  requestAnimationFrame(() => {
    streamIgnoreScroll = true
    if (streamFollow) {
      msgBox.scrollTop = msgBox.scrollHeight
    } else {
      const max = Math.max(0, msgBox.scrollHeight - msgBox.clientHeight)
      msgBox.scrollTop = Math.min(streamPausedScrollTop, max)
    }
    requestAnimationFrame(() => {
      streamIgnoreScroll = false
    })
  })
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
