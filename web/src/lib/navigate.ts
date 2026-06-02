import { clearTreeSessionOverlay } from './session-loading-overlay'

/** 仅改 hash；浏览器在 hash 变化时会自动触发 hashchange */

export function navigateHash(hash: string, opts?: { reload?: boolean }): void {
  const next = hash.startsWith('#') ? hash : `#${hash}`
  if (location.hash === next) {
    if (opts?.reload) window.dispatchEvent(new HashChangeEvent('hashchange'))
    return
  }
  location.hash = next
}

/** 更新教练页 URL 中的 sessionId，不触发 hashchange（避免整页重载丢本地消息） */
export function replaceCoachHashSession(sessionId: string): void {
  const next = `#/coach/${sessionId}`
  if (location.hash === next) return
  history.replaceState(null, '', `${location.pathname}${location.search}${next}`)
}

/** 进入教练对话；若已在同一会话则强制刷新路由并去掉树页 loading 遮罩 */
export function navigateToCoach(sessionId: string): void {
  const next = `#/coach/${sessionId}`
  const same = location.hash === next
  clearTreeSessionOverlay()
  navigateHash(next, { reload: same })
}
