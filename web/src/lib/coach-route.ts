/** 当前 hash 是否在教练对话路由 */
export function isCoachRoute(hash = location.hash.slice(1) || '/'): boolean {
  return /^\/coach\/[^/]+/.test(hash)
}
