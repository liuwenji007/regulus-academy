/** 当前 hash 是否在行动助手路由 */
export function isAssistantRoute(): boolean {
  const hash = location.hash.slice(1) || '/'
  return /^\/assistant(?:\/|$|\?)/.test(hash)
}
