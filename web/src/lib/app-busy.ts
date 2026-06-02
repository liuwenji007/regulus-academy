/** 长耗时建课/生成期间，避免侧边栏与次要接口误报连接失败 */

let busy = false
let reason = ''

export function setAppBusy(next: boolean, why = ''): void {
  busy = next
  reason = next ? why : ''
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
  setAppBusy(false)
  return true
}
