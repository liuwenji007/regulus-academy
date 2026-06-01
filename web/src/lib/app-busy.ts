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
