import { LOADING_FADE_MS } from './loading-transition'

/** 长耗时建课/生成期间，避免侧边栏与次要接口误报连接失败 */

let busy = false
let reason = ''
const listeners = new Set<() => void>()

function emitBusyChange(): void {
  for (const fn of listeners) fn()
}

/** 订阅 busy 变化（用于侧边栏「课程准备中」等 UI 同步） */
export function onAppBusyChange(fn: () => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function setAppBusy(next: boolean, why = ''): void {
  const was = busy
  busy = next
  reason = next ? why : ''
  if (was !== busy) emitBusyChange()
}

export function isAppBusy(): boolean {
  return busy
}

export function getAppBusyReason(): string {
  return reason
}

/** 仅当当前 busy 原因匹配时清除，避免误伤 session 等其他长耗时流程 */
export function clearAppBusyIf(why: string): boolean {
  if (!busy || reason !== why) return false
  setAppBusy(false, '')
  return true
}

/** handoff 完成后延迟清除，与遮罩淡出对齐 */
export function clearAppBusyIfAfter(
  why: string,
  onReleased?: () => void,
  delayMs = LOADING_FADE_MS + 80
): void {
  window.setTimeout(() => {
    if (clearAppBusyIf(why)) onReleased?.()
  }, delayMs)
}
