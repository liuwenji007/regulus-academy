/** 当前 hash 是否在行动助手路由 */
export function isAssistantRoute(hash = location.hash.slice(1) || '/'): boolean {
  return /^\/assistant(?:\/|$|\?)/.test(hash)
}
