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
